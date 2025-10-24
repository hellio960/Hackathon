package noderelease

import (
	"net/http"

	"hackathon/cmd/jarvis/internal/logic/noderelease"
	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/common/errorx"
	"hackathon/common/httpresp"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func NodeReleasePackagesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.NodeReleasePackagesReq
		if err := httpx.Parse(r, &req); err != nil {
			httpresp.HttpErr(w, r, errorx.NewStatCodeError(http.StatusBadRequest, 2, err.Error()))
			return
		}

		l := noderelease.NewNodeReleasePackagesLogic(r.Context(), svcCtx)
		resp, err := l.NodeReleasePackages(&req)

		httpresp.Http(w, r, resp, err)

	}
}
