package noderelease

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"hackathon/cmd/jarvis/internal/logic/noderelease"
	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/common/httpresp"
)

func NodesSearchHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.NodesSearchReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := noderelease.NewNodesSearchLogic(r.Context(), svcCtx)
		resp, err := l.NodesSearch(&req)
		httpresp.Http(w, r, resp, err)
	}
}
