package jarivsupd

import (
	"net/http"

	"hackathon/cmd/jarvis/internal/logic/jarivsupd"
	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetAllowNodesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetAllowNodesReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := jarivsupd.NewGetAllowNodesLogic(r.Context(), svcCtx)
		resp, err := l.GetAllowNodes(&req)
		if err != nil {
			httpx.Error(w, err)
		} else {
			httpx.OkJson(w, resp)
		}
	}
}
