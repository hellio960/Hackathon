package jarivsupd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/model"
	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/cmd/jarvis/shared/noderelease"
	"hackathon/common/utils"
	"hackathon/sharedmodel"
)

var AvailableVersions = []string{"v1.6"}

type UpdateAllowNodesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateAllowNodesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAllowNodesLogic {
	return &UpdateAllowNodesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UpdateAllowNodes 修改白名单列表
func (l *UpdateAllowNodesLogic) UpdateAllowNodes(req *types.PutAllowNodeReq) error {
	// check client version
	if !utils.InSlice(req.ClientVer, AvailableVersions) {
		return errors.New("your client version has been deprecated, please update your client")
	}

	req.App = strings.ToLower(req.App)
	allowNodesTopic, err := noderelease.GetAllowNodesTopic(l.ctx, nil, l.svcCtx.AllowAppsModel, req.App, req.NodeType, "")
	if err != nil {
		return err
	}

	if len(req.NodeIds) <= 0 {
		if _, err = l.svcCtx.BizRedis.Del(allowNodesTopic); err != nil {
			l.Logger.Errorf("fail del %s, %s", allowNodesTopic, err.Error())
			return err
		}

		return l.implementUpdConf(req.NodeType, req.App)
	}

	v := make([]interface{}, 0)
	for i := range req.NodeIds {
		v = append(v, req.NodeIds[i])
	}

	if _, err = l.svcCtx.BizRedis.Sadd(allowNodesTopic, v...); err != nil {
		l.Logger.Error("update %s fail, %s", allowNodesTopic, err.Error())
		return err
	}
	return nil
}

// implementUpdConf 当全量发布时，更新配置及发布记录
func (l *UpdateAllowNodesLogic) implementUpdConf(nodeType, appName string) error {

	var updConfNamePattern []string
	switch nodeType {
	case sharedmodel.NodeTypeSmallBox.String():
		// 相关节点类型的所有版本都要刷
		updConfNamePattern = []string{fmt.Sprintf("%s.*", sharedmodel.AntUpdConf), fmt.Sprintf("%s.*", sharedmodel.DroidUpdConf)}
	case sharedmodel.NodeTypeNode.String():
		updConfNamePattern = []string{sharedmodel.JarvisUpdConf}
	default:
		return errors.New("invalid devtype")
	}

	var sysParamList []*sharedmodel.SysParam
	for _, pattern := range updConfNamePattern {
		list, err := l.svcCtx.SysParamModel.FindByNamePattern(l.ctx, pattern)
		if err == sharedmodel.ErrNotFound {
			continue
		}
		if err != nil {
			l.Logger.Errorf("can not find %s, %s", pattern, err.Error())
			continue
		}
		sysParamList = append(sysParamList, list...)
	}

	for _, sysParam := range sysParamList {
		resp := &types.JarvisUpdConf{}
		if err := json.Unmarshal([]byte(sysParam.Value), resp); err != nil {
			l.Logger.Errorf("unmarshal updconf err: %s", err.Error())
			return err
		}

		appChg := make(map[string]*model.AppChange)
		if app := resp.Apps[appName]; app != nil {
			if app.Main == nil && app.Alter != nil {
				//增加app
				var verAlter string
				if alterUrlSlice := strings.Split(app.Alter.URL, "/"); len(alterUrlSlice) > 2 {
					verAlter = alterUrlSlice[len(alterUrlSlice)-2]
				}
				appChg[appName] = &model.AppChange{
					Previous: "none",
					Latest:   verAlter,
				}
				app.Main = app.Alter

			} else if app.Main != nil && app.Alter == nil {
				//移除app
				var verMain string
				if mainUrlSlice := strings.Split(app.Main.URL, "/"); len(mainUrlSlice) > 2 {
					verMain = mainUrlSlice[len(mainUrlSlice)-2]
				}

				appChg[appName] = &model.AppChange{
					Previous: verMain,
					Latest:   "none",
				}
				delete(resp.Apps, appName)

				//if err_ := l.svcCtx.AllowAppsModel.Delete(l.ctx, nodeType, appName); err_ != nil {
				//	l.Logger.Errorf("delete app from AllowAppsModel err: %s", err_.Error())
				//}

			} else if app.Main == nil && app.Alter == nil {
				//灰度添加app时回滚
				delete(resp.Apps, appName)
			} else {
				//修改app
				if app.Main.URL != app.Alter.URL {
					var verMain, verAlter string
					if mainUrlSlice := strings.Split(app.Main.URL, "/"); len(mainUrlSlice) > 2 {
						verMain = mainUrlSlice[len(mainUrlSlice)-2]
					}
					if alterUrlSlice := strings.Split(app.Alter.URL, "/"); len(alterUrlSlice) > 2 {
						verAlter = alterUrlSlice[len(alterUrlSlice)-2]
					}
					appChg[appName] = &model.AppChange{
						Previous: verMain,
						Latest:   verAlter,
					}
				}
				app.Main = app.Alter
			}
		}

		cfg, err := json.Marshal(resp)
		if err != nil {
			l.Logger.Errorf("marshal updconf err: %s", err.Error())
			return err
		}

		if len(appChg) > 0 {
			var data model.UpdRecordTemplate
			data.ID = model.NewUpdRecordID()
			data.UpdConf = string(cfg)
			data.AppChange = appChg
			if err = l.svcCtx.UpdRecordModel.Insert(l.ctx, &data); err != nil {
				l.Logger.Errorf("insert UpdRecordModel err: %s", err.Error())
				return err
			}
		}

		if err = l.svcCtx.SysParamModel.Upsert(l.ctx, &sharedmodel.SysParam{
			Name:   sysParam.Name,
			Value:  string(cfg),
			Remark: "",
		}); err != nil {
			return err
		}

		l.Logger.Infof("update %s done", sysParam.Name)
	}

	return nil
}
