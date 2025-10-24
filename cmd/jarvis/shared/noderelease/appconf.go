package noderelease

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"hackathon/common/device"
	"hackathon/sharedmodel"
)

type JarvisUpdConf struct {
	Name         string           `json:"name"`
	Apps         map[string]*Apps `json:"apps"`
	TTL          int              `json:"ttl"`                   // 更新间隔
	UpdTimeRange *TimeRange       `json:"updTimeRange,optional"` // 升级时间段
	Timestamp    int64            `json:"timestamp,optional"`    // 服务器时间戳
}

type TimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type Apps struct {
	Main  *sharedmodel.AppConfig `json:"main,omitempty,optional"`
	Alter *sharedmodel.AppConfig `json:"alter,omitempty,optional"`
}

func GetAppConfigName(devType device.DevType) string {
	// 示例:jarvis_upd_config, ant_upd_config_ant.A, droid_upd_config_droid.A
	if !devType.Valid() {
		return ""
	}

	if devType.IsJarvisGroup() {
		return "jarvis_upd_config"
	}

	if devType.IsAntGroup() {
		return fmt.Sprintf("ant_upd_config_%s", devType.String())
	}

	if devType.IsDroidGroup() {
		return fmt.Sprintf("droid_upd_config_%s", devType.String())
	}

	return ""
}

func GetAppMainConfig(ctx context.Context, sysParamModel sharedmodel.SysParamModel, devType device.DevType, appName string) (*sharedmodel.AppConfig, error) {

	devConfigName := GetAppConfigName(devType)
	if devConfigName == "" {
		return nil, fmt.Errorf("invalid devType: %s", devType.String())
	}

	config, err := sysParamModel.FindByName(ctx, devConfigName)
	if err != nil {
		if err == sharedmodel.ErrNotFound {
			logx.Errorf("not found %s", devConfigName)
			// 当前devType的配置不存在
			return nil, nil
		}
		return nil, err
	}

	updConfig := JarvisUpdConf{}
	if err = json.Unmarshal([]byte(config.Value), &updConfig); err != nil {
		return nil, err
	}

	for _, app := range updConfig.Apps {
		if app.Main != nil && app.Main.Cmd == appName {
			return app.Main, nil
		}
	}

	// 当前devType下没有配置appName
	return nil, nil
}

func GetAppConfig(ctx context.Context, sysParamModel sharedmodel.SysParamModel, devType device.DevType, appName string) (main, alter *sharedmodel.AppConfig, err error) {

	devConfigName := GetAppConfigName(devType)
	if devConfigName == "" {
		return nil, nil, fmt.Errorf("invalid devType: %s", devType.String())
	}

	config, err := sysParamModel.FindByName(ctx, devConfigName)
	if err != nil {
		if err == sharedmodel.ErrNotFound {
			logx.Errorf("not found %s", devConfigName)
			// 当前devType的配置不存在
			return nil, nil, nil
		}
		return nil, nil, err
	}

	updConfig := JarvisUpdConf{}
	if err = json.Unmarshal([]byte(config.Value), &updConfig); err != nil {
		return nil, nil, err
	}

	for _, app := range updConfig.Apps {
		if (app.Main != nil && app.Main.Cmd == appName) || (app.Alter != nil && app.Alter.Cmd == appName) {
			return app.Main, app.Alter, nil
		}
	}

	// 当前devType下没有配置appName
	return nil, nil, nil
}

// 将当前灰度配置更新为正式版本, 用于全量发布任务
func CompleteAppConfig(ctx context.Context, sysParamModel sharedmodel.SysParamModel, release *sharedmodel.NodeRelease, bizRedis *redis.Redis, defaultTopic string) (prevMain, afterMain *sharedmodel.AppConfig, err error) {
	devConfigName := GetAppConfigName(device.DevType(release.DeviceType))
	if devConfigName == "" {
		return nil, nil, fmt.Errorf("invalid devType: %s", release.DeviceType)
	}

	config, err := sysParamModel.FindByName(ctx, devConfigName)
	if err != nil {
		return nil, nil, fmt.Errorf("%s not found, err:%s", devConfigName, err.Error())
	}

	updConfig := JarvisUpdConf{}
	if err = json.Unmarshal([]byte(config.Value), &updConfig); err != nil {
		return nil, nil, err
	}

	var found bool
	for _, app := range updConfig.Apps {
		if (app.Alter != nil && app.Alter.Cmd == release.App) || (app.Main != nil && app.Main.Cmd == release.App) {
			found = true
			prevMain = app.Main
			afterMain = release.AlterConfig
			app.Main = release.AlterConfig

			// 当工具没有在发布时, 将sysParam中的alter做同步更新
			cnt, errN := GetAllowNodesCount(ctx, bizRedis, defaultTopic)
			if errN == nil && cnt == 0 {
				app.Alter = release.AlterConfig
			}
		}
	}

	if !found {
		// 若为新增组件, 把组件加入其中
		if release.OpType == sharedmodel.NodeReleaseOpTypeAddApp {
			updConfig.Apps[release.App] = &Apps{
				Main:  release.AlterConfig,
				Alter: release.AlterConfig,
			}
		} else {
			return nil, nil, fmt.Errorf("app %s not found", release.App)
		}
	}

	// 如果main和alter都为空, 移除相应app
	for _, app := range updConfig.Apps {
		if app.Main == nil && app.Alter == nil {
			delete(updConfig.Apps, release.App)
		}
	}

	cfg, err := json.Marshal(updConfig)
	if err != nil {
		return nil, nil, err
	}

	if err = sysParamModel.Upsert(ctx, &sharedmodel.SysParam{
		Name:  devConfigName,
		Value: string(cfg),
	}); err != nil {
		return nil, nil, err
	}

	return prevMain, afterMain, nil

}

func RollbackAppConfig(ctx context.Context, sysParamModel sharedmodel.SysParamModel, releasePlan *sharedmodel.NodeRelease) (prevMain, afterMain *sharedmodel.AppConfig, err error) {
	devConfigName := GetAppConfigName(device.DevType(releasePlan.DeviceType))
	if devConfigName == "" {
		return nil, nil, fmt.Errorf("invalid devType: %s", releasePlan.DeviceType)
	}

	config, err := sysParamModel.FindByName(ctx, devConfigName)
	if err != nil {
		return nil, nil, err
	}

	updConfig := JarvisUpdConf{}
	if err = json.Unmarshal([]byte(config.Value), &updConfig); err != nil {
		return nil, nil, err
	}

	for _, app := range updConfig.Apps {
		if app.Main != nil && app.Main.Cmd == releasePlan.App {
			prevMain = app.Main
			afterMain = releasePlan.MainConfig
			app.Main = releasePlan.MainConfig
			app.Alter = releasePlan.MainConfig
		}
	}

	cfg, err := json.Marshal(updConfig)
	if err != nil {
		return nil, nil, err
	}

	if err = sysParamModel.Upsert(ctx, &sharedmodel.SysParam{
		Name:  devConfigName,
		Value: string(cfg),
	}); err != nil {
		return nil, nil, err
	}

	return prevMain, afterMain, nil
}
