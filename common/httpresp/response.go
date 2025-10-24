package httpresp

import (
	"net/http"
	"os"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"hackathon/common/errorx"
)

const (
	netUnavailableCode  = 2
	netUnavailableError = "网络开小差，请稍后再试试吧"
)

type errResp struct {
	Code int    `json:"code"`
	Desc string `json:"desc,omitempty"`
}

func (e *errResp) Error() string {
	return strconv.Itoa(e.Code) + ":" + e.Desc
}

// HttpHeader 写入http header
func HttpHeader(w http.ResponseWriter) {
	// 写入处理机器的信息，如果请求访问的机器链路是A->B->C，那么Nl_Host: C:B:A
	if hostName, err := os.Hostname(); err == nil {
		lastHost := w.Header().Get("Nl_Host")
		if lastHost != "" {
			hostName = lastHost + ":" + hostName
		}
		w.Header().Set("Nl_Host", hostName)
	}
}
func Http(w http.ResponseWriter, r *http.Request, data interface{}, err error) {
	if err != nil {
		HttpErr(w, r, err)
	} else {
		HttpOkJson(w, r, data)
	}
	HttpHeader(w)
}

func errDesc(codeErr errorx.CodeError, r *http.Request) string {
	return codeErr.Error()
}

func HttpErr(w http.ResponseWriter, r *http.Request, err error) {
	codeErr, ok := errorx.FromError(err)
	if ok {
		httpx.WriteJson(w, codeErr.Status(), errResp{
			Code: codeErr.Code(),
			Desc: errDesc(codeErr, r),
		})
	} else {
		httpx.WriteJson(w, http.StatusInternalServerError, errResp{
			Code: netUnavailableCode,
			Desc: netUnavailableError,
		})
		logx.WithContext(r.Context()).Error(err)
	}
}

func HttpOk(w http.ResponseWriter, r *http.Request) {
	httpx.Ok(w)
}

func HttpOkJson(w http.ResponseWriter, r *http.Request, data interface{}) {
	if data == nil {
		HttpOk(w, r)
	} else {
		httpx.OkJson(w, data)
	}
}
