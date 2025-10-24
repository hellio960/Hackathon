package device

// 说明：本模块会被所有节点端组件引用，不要引用三方包和有平台依赖的包，以保证跨平台的兼容性和性能开销
import (
	"errors"
)

type (
	DevType         string // 设备类型，以系统支持的设备类型的组合来定义
)

const (
	DevAntVerA    DevType = "ant.A"    // linux+arm32
	DevAntVerB    DevType = "ant.B"    // linux+arm64
	DevAntVerC    DevType = "ant.C"    // linux+amd64
	DevJarvisVerA DevType = "jarvis.A" // linux+amd64
	DevDroidVerA  DevType = "droid.A"  // android+arm64
	DevDroidVerB  DevType = "droid.B"  // android+arm32
	DevDroidVerC  DevType = "droid.C"  // android+amd64
	DevMicroVerA  DevType = "micro.A"  // linux+arm32+resourceLimited
	DevMicroVerB  DevType = "micro.B"  // linux+arm64+resourceLimited
	DevMicroVerC  DevType = "micro.C"  // windows+amd64+resourceLimited
)

// GetAllDeviceTypes 返回所有DevType
func GetAllDevTypes() []DevType {
	return []DevType{
		DevJarvisVerA,

		DevAntVerA,
		DevAntVerB,
		DevAntVerC,
		DevDroidVerA,
		DevDroidVerB,
		DevDroidVerC,
		DevMicroVerA,
		DevMicroVerB,
		DevMicroVerC,
	}
}

func (d DevType) Valid() bool {
	return d.IsJarvisGroup() || d.IsAntGroup() || d.IsDroidGroup() || d.IsMicroGroup()
}

func (d DevType) String() string {
	return string(d)
}

func (d DevType) NodeType() (string, error) {
	if d.IsAntGroup() || d.IsDroidGroup() {
		return "smallBox", nil
	}
	if d.IsJarvisGroup() {
		return "node", nil
	}
	return "", errors.New("unknown device type")
}

// ---------- 按大小节点区分 (upd配置管理使用, 其它场景建议优先使用nodeType)---------
func (d DevType) IsSmallBoxDev() bool {
	return d.IsAntGroup() || d.IsDroidGroup() || d.IsMicroGroup()
}

// ----------按具体设备区分----------
func (d DevType) IsAntGroup() bool {
	switch d {
	case DevAntVerA,
		DevAntVerB,
		DevAntVerC:
		return true
	default:
		return false
	}
}

func (d DevType) IsDroidGroup() bool {
	switch d {
	case DevDroidVerA, DevDroidVerB, DevDroidVerC:
		return true
	default:
		return false
	}
}

func (d DevType) IsMicroGroup() bool {
	switch d {
	case DevMicroVerA, DevMicroVerB, DevMicroVerC:
		return true
	default:
		return false
	}
}

func (d DevType) IsJarvisGroup() bool {
	switch d {
	case DevJarvisVerA:
		return true
	default:
		return false
	}
}
