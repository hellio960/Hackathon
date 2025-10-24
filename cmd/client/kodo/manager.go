package kodo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"hackathon/common/errorx"
)

const (
	NiulinkBucket = "niulink"
)

var (
	ErrNeedUpload = fmt.Errorf("need to upload")
)

// 获取kodo文件的md5
func GetKodoFileMd5(fileUrl string) (string, error) {
	var md5 = struct {
		Hash  string `json:"hash"`
		Fsize int64  `json:"fsize"`
	}{}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s?qhash/md5", fileUrl), nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.StatusCode)
		return "", errorx.NewDefaultError("get md5 failed")
	}

	err = json.Unmarshal(b, &md5)
	if err != nil || md5.Hash == "" {
		return "", errorx.NewDefaultError("parse md5 failed")
	}

	return md5.Hash, nil
}
