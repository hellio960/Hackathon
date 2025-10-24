package jarivsupd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"hackathon/cmd/jarvis/internal/model"
	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/cmd/jarvis/shared/noderelease"
	"hackathon/common/device"
	"hackathon/common/errorx"
	"hackathon/common/utils"
	"hackathon/sharedmodel"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	appOprAdd     = "add"
	appOprDel     = "delete"
	appOprUpd     = "update"       //升级
	appOprModManu = "modifyManual" //手动更新
)

var defaultUpdTimeRange = types.TimeRange{
	Start: "00:00",
	End:   "23:59",
}

type UpdateJarvisUpdConfLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateJarvisUpdConfLogic(ctx context.Context, svcCtx *svc.ServiceContext) UpdateJarvisUpdConfLogic {
	return UpdateJarvisUpdConfLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UpdateJarvisUpdConf 更新updConf
func (l *UpdateJarvisUpdConfLogic) UpdateJarvisUpdConf(req *types.PutJarvisUpdConfReq) error {
	// check client version
	if !utils.InSlice(req.ClientVer, AvailableVersions) {
		return errors.New("your client version has been deprecated, please update your client")
	}

	devType := device.DevType(req.DevType)
	switch {
	case devType.IsSmallBoxDev():
		if req.NodeType != sharedmodel.NodeTypeSmallBox.String() {
			return fmt.Errorf("%s not exist in %s", req.DevType, req.NodeType)
		}
	case devType.IsJarvisGroup():
		if req.NodeType != sharedmodel.NodeTypeNode.String() {
			return fmt.Errorf("%s not exist in %s", req.DevType, req.NodeType)
		}
	default:
		return fmt.Errorf("unknown devType: %s", req.DevType)
	}
	if appAllow, err := l.svcCtx.AllowAppsModel.FindByName(l.ctx, req.NodeType, req.Name); err != nil || appAllow == nil {
		err = errors.New(fmt.Sprintf("app %s is not define", req.Name))
		return err
	}

	appChg, err := l.implementUpdConf(req)
	if err != nil {
		return err
	}

	v, err := json.Marshal(req.UpdConf)
	if err != nil {
		l.Logger.Errorf("upd jarvisUpdConf marshal err: %s", err.Error())
		return err
	}

	// update db record
	if len(appChg) > 0 {
		var data model.UpdRecordTemplate
		data.ID = model.NewUpdRecordID()
		data.UpdConf = string(v)
		data.AppChange = appChg
		if err = l.svcCtx.UpdRecordModel.Insert(l.ctx, &data); err != nil {
			l.Logger.Errorf("insert UpdRecordModel err: %s", err.Error())
			return err
		}
	}

	updConfName := sharedmodel.JarvisUpdConf
	switch {
	case devType.IsAntGroup():
		updConfName = sharedmodel.AntUpdConf + "_" + req.DevType
	case devType.IsDroidGroup():
		updConfName = sharedmodel.DroidUpdConf + "_" + req.DevType
	case devType.IsJarvisGroup():
	default:
		return errorx.NewDefaultError("devType invalid")
	}

	return l.svcCtx.SysParamModel.Upsert(l.ctx, &sharedmodel.SysParam{
		Name:   updConfName,
		Value:  string(v),
		Remark: "",
	})

}

// implementUpdConf 当全量发布时，更新配置及发布记录
func (l *UpdateJarvisUpdConfLogic) implementUpdConf(req *types.PutJarvisUpdConfReq) (map[string]*model.AppChange, error) {
	appChg := make(map[string]*model.AppChange)

	//var nodeType string
	topic := fmt.Sprintf("%s_%s", noderelease.JarvisUpdConfAllowNodes, req.Name)

	devType := device.DevType(req.DevType)
	switch {
	case devType.IsSmallBoxDev():
		topic = fmt.Sprintf("%s_%s", noderelease.BoxUpdConfAllowNodes, req.Name)
	case devType.IsJarvisGroup():
	default:
		return nil, errors.New("nodeType invalid")
	}

	v, err := l.svcCtx.BizRedis.Exists(topic)
	if err != nil {
		l.Logger.Infof("refreshUpdConf get allowNodes in redis err: %s", err.Error())
		return appChg, err
	}

	if v == true {
		var nodes []string
		if nodes, err = l.svcCtx.BizRedis.Smembers(topic); err != nil {
			l.Logger.Errorf("get updConfAllowNodes err: %s", err.Error())
			return appChg, err
		}
		if len(nodes) > 0 {
			return appChg, nil
		}
	}

	switch req.OprType {
	case appOprUpd, appOprModManu:
		// 无白名单:统一main和alter，并添加发布记录
		cfg := req.UpdConf.Apps[req.Name]
		if cfg == nil || cfg.Alter == nil {
			l.Logger.Errorf("app %s or alter %#v not exist in req body", req.Name, cfg)
			return appChg, errors.New(fmt.Sprintf("app %s not exist in req body", req.Name))
		}

		if cfg.Alter == nil {
			return appChg, errors.New(fmt.Sprintf("app's alter cfg is nil"))
		}

		var verMain, verAlter string
		if cfg.Main != nil {
			if mainUrlSlice := strings.Split(cfg.Main.URL, "/"); len(mainUrlSlice) > 2 {
				verMain = mainUrlSlice[len(mainUrlSlice)-2]
			}
		}

		if alterUrlSlice := strings.Split(cfg.Alter.URL, "/"); len(alterUrlSlice) > 2 {
			verAlter = alterUrlSlice[len(alterUrlSlice)-2]
		}

		appChg[req.Name] = &model.AppChange{
			Previous: verMain,
			Latest:   verAlter,
		}

		cfg.Main = cfg.Alter

	case appOprAdd:
		cfg := req.UpdConf.Apps[req.Name]
		if cfg == nil || cfg.Alter == nil {
			l.Logger.Errorf("app %s or %#v alter not exist in req body", req.Name, cfg)
			return appChg, errors.New(fmt.Sprintf("app %s not exist in req body", req.Name))
		}

		var verAlter string
		if alterUrlSlice := strings.Split(cfg.Alter.URL, "/"); len(alterUrlSlice) > 2 {
			verAlter = alterUrlSlice[len(alterUrlSlice)-2]
		}

		appChg[req.Name] = &model.AppChange{
			Previous: "none",
			Latest:   verAlter,
		}

		cfg.Main = cfg.Alter

	case appOprDel:
		// 更新配置，添加发布记录，删除allowApps列表
		cfg := req.UpdConf.Apps[req.Name]
		if cfg == nil {
			l.Logger.Errorf("app %s not exist in req body", req.Name)
			return appChg, errors.New(fmt.Sprintf("app %s not exist in req body", req.Name))
		}
		var verMain string
		if cfg.Main != nil {
			if mainUrlSlice := strings.Split(cfg.Main.URL, "/"); len(mainUrlSlice) > 2 {
				verMain = mainUrlSlice[len(mainUrlSlice)-2]
			}
		}

		appChg[req.Name] = &model.AppChange{
			Previous: verMain,
			Latest:   "none",
		}

		delete(req.UpdConf.Apps, req.Name)

		//if _, err := l.svcCtx.BizRedis.Del(topic); err != nil {
		//	l.Logger.Errorf("fail del %s, %s", topic, err.Error())
		//}
		//
		//if err := l.svcCtx.AllowAppsModel.Delete(l.ctx, nodeType, req.Name); err != nil {
		//	l.Logger.Errorf("delete app from AllowAppsModel err: %s", err.Error())
		//}

	default:
		return appChg, errors.New(fmt.Sprintf("unknown oprtype %s", req.OprType))
	}

	return appChg, nil
}
