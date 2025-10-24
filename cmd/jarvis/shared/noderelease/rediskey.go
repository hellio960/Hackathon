package noderelease

import (
	"hackathon/common/device"
	"hackathon/common/errorx"
	"hackathon/sharedmodel"
)

func GenUpdConfKey(devType device.DevType) (string, error) {
	switch {
	case devType.IsAntGroup():
		return sharedmodel.AntUpdConf + "_" + devType.String(), nil
	case devType.IsDroidGroup():
		return sharedmodel.DroidUpdConf + "_" + devType.String(), nil
	case devType.IsJarvisGroup():
		return sharedmodel.JarvisUpdConf, nil
	default:
		return "", errorx.NewDefaultError("unknown device type")
	}
}
