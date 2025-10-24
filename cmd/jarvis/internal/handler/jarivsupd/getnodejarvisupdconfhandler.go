package jarivsupd

import (
	"net/http"

	"hackathon/cmd/jarvis/internal/logic/jarivsupd"
	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/common/errorx"
	"hackathon/common/httpresp"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetNodeJarvisUpdConfHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetJarvisUpdConfReq
		if err := httpx.Parse(r, &req); err != nil {
			httpresp.HttpErr(w, r, errorx.NewStatCodeError(http.StatusBadRequest, 2, err.Error()))
			return
		}

		l := jarivsupd.NewGetNodeJarvisUpdConfLogic(r.Context(), svcCtx)
		resp, err := l.GetNodeJarvisUpdConf(req)

		httpresp.Http(w, r, resp, err)

	}
}
