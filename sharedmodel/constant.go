package sharedmodel

const (
	NodeTypeNode     NodeType = "node"     // 大节点
	NodeTypeSwitch   NodeType = "switch"   // 交换机
	NodeTypeSmallBox NodeType = "smallBox" // 小盒子

	DNodeStatusOnline  string = "online"  // 在线
	DNodeStatusOutline string = "outline" // 离线
)

type (
	NodeType string // 节点类型
)

func (n NodeType) String() string {
	return string(n)
}
