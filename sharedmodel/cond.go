package sharedmodel

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ========================================= page

type SortType string

const (
	SortTypeDesc SortType = "desc" // 降序
	SortTypeAsc  SortType = "asc"  // 升序

	MarkFirstPage = "firstpage" // 标记第一页
)

type (
	PageParam struct {
		MarkPage // 使用Mark来翻页，和Page是互斥的，二选一; 当前支持对 time.Time/uint32/int64/string/primitive.ObjectID, 如果有变动请参考和更新 IsMarkFirstPage

		Page            int      // 页码, 和Mark互斥，二选一
		Size            int      // 每页数量
		SortKey         string   // 排序字段，多个字段排序使用英文,分隔;
		SortType        SortType // 排序方向
		IsSortByChinese bool
		// 如果查询参数仅有 _id，且无排序需求，同时表中存在 _id&排序字段的联合索引；那推荐不要 sort，直接筛选 _id，如果加了 sort，会导致优先匹配上联合索引
		// 所以在明确无排序需求时，推荐设置 NoSort 为 true
		NoDefaultSort bool // 不需要排序，默认需要
	}
	MarkPage struct {
		Mark         interface{} // mark字段的value, 无论升序或者降序，mark都应该有一个正确的值，尤其是第一页时, 比如uint32的升序，则第一页的mark应该传递0
		MarkSortKey  string      // 指定mark字段名
		MarkSortType SortType    // 指定mark字段的排序方向
	}
	TimeRange struct {
		Start time.Time `bson:"start,omitempty" json:"start,omitempty"` // 开始时间
		End   time.Time `bson:"end,omitempty"   json:"end,omitempty"`   // 结束时间
		Days  int       `bson:"days,omitempty"  json:"days,omitempty"`  // 天数
	}
	FieldsCond []string
	TimeCond   struct {
		Start time.Time
		End   time.Time
	}
)

func (page PageParam) GeneratePageOption(opts ...*options.FindOptions) *options.FindOptions {
	var option = options.Find()
	// 使用传入的options做默认值
	if len(opts) > 0 {
		option = opts[0]
	}
	// sort
	switch page.SortType {
	case SortTypeDesc:
		option.SetSort(bson.M{page.SortKey: -1})
	case SortTypeAsc:
		option.SetSort(bson.M{page.SortKey: 1})
	}
	if page.IsSortByChinese {
		option.SetCollation(&options.Collation{Locale: "zh", CaseLevel: true})
	}
	// 默认不设置 skip & limit
	if page.IsMarkPage() {
		option.SetLimit(int64(page.Size))
	}
	if page.IsPage() {
		option.SetSkip(page.Skip())
		option.SetLimit(page.Limit())
	}

	return option
}

func (page PageParam) IsMarkPage() bool {
	if page.Mark != nil && page.Size > 0 {
		return true
	}
	return false
}

func (page PageParam) IsPage() bool {
	if page.Page > 0 && page.Size > 0 {
		return true
	}

	return false
}

func (page PageParam) Skip() int64 {
	if page.IsPage() {
		return int64((page.Page - 1) * page.Size)
	}
	return 0
}

func (page PageParam) Limit() int64 {
	if page.IsPage() {
		return int64(page.Size)
	}
	return 1000
}

func (fc FieldsCond) SetProjection(opts *options.FindOptions) {
	if len(fc) == 0 || opts == nil {
		return
	}
	projection := make(map[string]int)
	for _, field := range fc {
		projection[field] = 1
	}
	opts.SetProjection(projection)
}

func (page PageParam) MarkCond(m bson.M) {
	if page.IsMarkPage() && !page.IsMarkFirstPage() {
		if page.MarkSortType == SortTypeDesc {
			m[page.MarkSortKey] = bson.M{"$lt": page.Mark}
		} else if page.MarkSortType == SortTypeAsc {
			m[page.MarkSortKey] = bson.M{"$gt": page.Mark}
		}
	}
	return
}

func (page PageParam) IsMarkFirstPage() bool {
	if s, ok := page.Mark.(string); ok {
		if s != "" && s != MarkFirstPage {
			return false
		}
	}
	if u, ok := page.Mark.(uint32); ok {
		if u != 0 {
			return false
		}
	}
	if u, ok := page.Mark.(int64); ok {
		if u != 0 {
			return false
		}
	}
	if t, ok := page.Mark.(time.Time); ok {
		if !t.IsZero() {
			return false
		}
	}
	if o, ok := page.Mark.(primitive.ObjectID); ok {
		if !o.IsZero() {
			return false
		}
	}
	return true
}

func (page *PageParam) SetMarkSort(key string, sortType SortType) {
	page.MarkSortKey = key
	page.MarkSortType = sortType
}

func (page *PageParam) SetMark(value interface{}) {
	page.Mark = value
}
