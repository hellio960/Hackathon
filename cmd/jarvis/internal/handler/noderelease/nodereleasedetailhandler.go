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

func NodeReleaseDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.NodeReleaseDetailReq
		if err := httpx.Parse(r, &req); err != nil {
			httpresp.HttpErr(w, r, errorx.NewStatCodeError(http.StatusBadRequest, 2, err.Error()))
			return
		}

		l := noderelease.NewNodeReleaseDetailLogic(r.Context(), svcCtx)
		resp, err := l.NodeReleaseDetail(&req)

		httpresp.Http(w, r, resp, err)

	}
}
