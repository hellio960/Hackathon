package jarivsupd

import (
	"context"
	"encoding/json"
	"time"

	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/cmd/jarvis/shared/noderelease"
	"hackathon/common/device"
	"hackathon/sharedmodel"

	"github.com/zeromicro/go-zero/core/logx"
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
		return nil, nil
	}
	if err != nil {
		l.Logger.Errorf("can not find %s, %s", updConfKey, err.Error())
		return
	}
	if err = json.Unmarshal([]byte(sysParam.Value), resp); err != nil {
		l.Logger.Errorf("unmarshal err: %s", err.Error())
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
