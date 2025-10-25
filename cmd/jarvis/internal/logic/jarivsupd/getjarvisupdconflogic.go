package jarivsupd

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/cmd/jarvis/shared/noderelease"
	"hackathon/common/device"
	"hackathon/sharedmodel"
)

type GetJarvisUpdConfLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetJarvisUpdConfLogic(ctx context.Context, svcCtx *svc.ServiceContext) GetJarvisUpdConfLogic {
	return GetJarvisUpdConfLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetJarvisUpdConfLogic) GetJarvisUpdConf(req types.JarvisUpdConfReq) (resp *types.JarvisUpdConf, err error) {
	// 公共逻辑抽象到cmd/jarvis/internal/logic/shared/noderelease/rediskey.go
	updConfKey, err := noderelease.GenUpdConfKey(device.DevType(req.DevType))
	if err != nil {
		return nil, err
	}

	resp = &types.JarvisUpdConf{}
	sysParam, err := l.svcCtx.SysParamModel.FindByName(l.ctx, updConfKey)
	if err == sharedmodel.ErrNotFound {
		// 配置不存在时，创建默认空配置并插入数据库
		l.Infof("[GetJarvisUpdConf] 配置不存在, 创建默认空配置: %s", updConfKey)

		// 创建默认空配置
		defaultConf := &types.JarvisUpdConf{
			Name:         updConfKey,
			Apps:         make(map[string]*types.Apps),
			TTL:          300,
			UpdTimeRange: &defaultUpdTimeRange,
			Timestamp:    time.Now().UnixMilli(),
		}

		// 序列化为JSON
		confBytes, err := json.Marshal(defaultConf)
		if err != nil {
			l.Errorf("[GetJarvisUpdConf] 序列化默认配置失败: %v", err)
			return nil, err
		}

		// 插入到数据库
		sysParam := &sharedmodel.SysParam{
			Name:     updConfKey,
			Value:    string(confBytes),
			Remark:   "自动创建的默认空配置",
			CreateAt: time.Now(),
			UpdateAt: time.Now(),
		}

		if err := l.svcCtx.SysParamModel.Upsert(l.ctx, sysParam); err != nil {
			l.Errorf("[GetJarvisUpdConf] 插入默认配置失败: %v", err)
			return nil, err
		}

		l.Infof("[GetJarvisUpdConf] 默认配置创建成功: %s", updConfKey)
		return defaultConf, nil
	}

	if err != nil {
		l.Errorf("can not find %s, %s", updConfKey, err.Error())
		return
	}

	if err = json.Unmarshal([]byte(sysParam.Value), resp); err != nil {
		l.Errorf("unmarshal err: %s", err.Error())
		return
	}

	// 默认填充升级时间段
	if resp.Name == "" {
		resp.Name = updConfKey
	}
	resp.Timestamp = time.Now().UnixMilli()
	if resp.UpdTimeRange == nil {
		resp.UpdTimeRange = &defaultUpdTimeRange
	}

	return
}
