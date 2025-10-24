package jarivsupd

import (
	"net/http"

	"hackathon/cmd/jarvis/internal/logic/jarivsupd"
	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateAllowNodesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PutAllowNodeReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := jarivsupd.NewUpdateAllowNodesLogic(r.Context(), svcCtx)
		err := l.UpdateAllowNodes(&req)
		if err != nil {
			httpx.Error(w, err)
		} else {
			httpx.Ok(w)
		}
	}
}
