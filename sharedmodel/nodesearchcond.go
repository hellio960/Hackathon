package sharedmodel

import (
	"time"
)

type (
	QueryType = string

	QueryTimeInterval struct {
		Type      string    // 时间筛选类型
		TimeStart time.Time // 时间筛选开始时间
		TimeEnd   time.Time // 时间筛选结束时间
	}
	DiskSearchType = string
)

var (
	searchHDDType       DiskSearchType = "hdd"
	searchSSDType       DiskSearchType = "ssd"
	searchSystemType    DiskSearchType = "system"
	searchDiskTotalType DiskSearchType = "total"
)

type NodeSearchCond struct {
	PageParam
	FieldsCond

	DeviceType  []string
	State       string   // 网络状态
	CustomerIDs []uint32 // 业务方 ID数组
	Stage       string   // 节点流程
	NodeType    string   // 节点类型
	NoCount     bool
}
