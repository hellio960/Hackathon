package sharedmodel

import "time"

type AppChange struct {
	Previous string `bson:"previous" json:"previous"`
	Latest   string `bson:"latest"   json:"latest"`
}

type UpdRecord struct {
	ID        string                `bson:"_id"            json:"id"`
	AppChange map[string]*AppChange `bson:"appChange"      json:"appChange"`
	Desc      string                `bson:"desc,omitempty" json:"desc,omitempty"`
	UpdConf   string                `bson:"updConf"        json:"updConf"`
	CreateAt  time.Time             `bson:"createAt"       json:"createAt"`
	UpdateAt  time.Time             `bson:"updateAt"       json:"updateAt"`
}

type UpdRecordListCond struct {
	IDs             []string
	CreateTimeRange *TimeRange
	Page            int
	Size            int
}
