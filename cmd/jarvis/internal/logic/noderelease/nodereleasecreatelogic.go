package noderelease

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hackathon/common/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"hackathon/cmd/client/kodo"
	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/cmd/jarvis/shared/noderelease"
	"hackathon/common/device"
	"hackathon/common/errorx"
	"hackathon/sharedmodel"
)

type NodeReleaseCreateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext

	nodeIdCache          map[string]map[string]struct{}
	allowNodesTopicCache noderelease.AllowNodesTopicCache
}

func NewNodeReleaseCreateLogic(ctx context.Context, svcCtx *svc.ServiceContext) NodeReleaseCreateLogic {
	return NodeReleaseCreateLogic{
		Logger:               logx.WithContext(ctx),
		ctx:                  ctx,
		svcCtx:               svcCtx,
		nodeIdCache:          make(map[string]map[string]struct{}),
		allowNodesTopicCache: noderelease.NewAllowNodesTopicCache(),
	}
}

func (l *NodeReleaseCreateLogic) NodeReleaseCreate(req *types.NodeReleaseCreateReq) (err error) {
	// 参数校验
	if err := l.checkBasicArgs(req); err != nil {
		return err
	}

	// 检查发布类型, 正式发布任务只能同时存在一个
	// 检查灰度策略, 禁止多版本并行灰度时存在多个条件匹配(防止节点无法准确分配)
	if err := l.ensureReleaseCompatible(req); err != nil {
		return err
	}

	// 检查包合法性
	if err := l.ensurePackageValid(req); err != nil {
		return err
	}

	// 检查灰度计划合法性
	if err := l.ensureGrayPolicyValid(req); err != nil {
		return err
	}

	releasePlan := &sharedmodel.NodeRelease{
		ID:          primitive.NewObjectID().Hex(),
		App:         req.AppName,
		DeviceType:  req.DevType,
		ReleaseType: sharedmodel.ReleaseType(req.ReleaseType),
		OpType:      sharedmodel.NodeReleaseOpType(req.OpType),
		State:       sharedmodel.NodeReleaseStateProcessing,
		GrayPolicy: sharedmodel.GrayPolicy{
			Percentage: req.GrayPolicy.Percentage,
		},
		MainConfig:  nil,
		AlterConfig: nil,
		Operator:    operator,
		Describe:    req.Describe,
		CreateAt:    time.Now(),
		UpdateAt:    time.Now(),
	}

	if err = utils.TransformStruct(req.GrayPolicy, &releasePlan.GrayPolicy); err != nil {
		return err
	}

	// alter配置从请求获取
	if req.AppConfig != nil {
		if err = utils.TransformStruct(req.AppConfig, &releasePlan.AlterConfig); err != nil {
			return err
		}
	}

	// main配置从sysparam表获取
	mainConfig, err := noderelease.GetAppMainConfig(l.ctx, l.svcCtx.SysParamModel, device.DevType(req.DevType), req.AppName)
	if err != nil {
		return err
	}

	// 新增组件时, main为空
	if mainConfig != nil {
		releasePlan.MainConfig = mainConfig
	}

	// 首次添加灰度节点, 根据灰度策略, 结合灰度比例, 获取当前批次灰度节点
	allowCount, totalCount, markID, err := l.firstAddAllowNodes(req, releasePlan.ID)
	if err != nil {
		return err
	}

	// 保存灰度信息
	releasePlan.GrayPolicy.StatInfo = &sharedmodel.StatInfo{
		MarkID:             markID,
		TotalCount:         totalCount,
		AllowCount:         allowCount,
		TotalCountUpdateAt: time.Now(),
		AllowCountUpdateAt: time.Now(),
	}

	// 保存发布任务
	if err := l.svcCtx.NodeReleaseModel.Insert(l.ctx, releasePlan); err != nil {
		return err
	}

	// 记录操作历史
	if err := l.svcCtx.NodeReleaseHistoryModel.Insert(l.ctx, &sharedmodel.NodeReleaseHistory{
		ReleaseID: releasePlan.ID,
		Operation: sharedmodel.NodeReleaseHistoryOperationCreate,
		Operator:  operator,
		OpTime:    time.Now(),
		Remark:    req.Describe,
		CreateAt:  time.Now(),
		UpdateAt:  time.Now(),
		// 变更信息
		AfterState: sharedmodel.NodeReleaseStateProcessing,
		GrayPolicyInfo: &sharedmodel.GrayPolicyRemark{
			NodeIdsAdd:      releasePlan.GrayPolicy.NodeIds,
			AfterFilter:     releasePlan.GrayPolicy.Filter,
			AfterPercentage: releasePlan.GrayPolicy.Percentage,
		},
	}); err != nil {
		return err
	}

	return nil
}

func (l *NodeReleaseCreateLogic) firstAddAllowNodes(req *types.NodeReleaseCreateReq, releaseID string) (allowCount int64, totalCount int64, markID string, err error) {

	nodeType := sharedmodel.NodeTypeNode
	if device.DevType(req.DevType).IsSmallBoxDev() {
		nodeType = sharedmodel.NodeTypeSmallBox
	}

	// 获取灰度节点topic
	topic, err := noderelease.GetAllowNodesTopic(l.ctx, l.allowNodesTopicCache, l.svcCtx.AllowAppsModel, req.AppName, nodeType.String(), releaseID)
	if err != nil {
		return 0, 0, "", err
	}

	if req.GrayPolicy.Percentage == 0 {
		return 0, 0, "", errorx.NewDefaultError("灰度比例不能为0")
	}

	if len(req.GrayPolicy.NodeIds) > 0 {
		totalCount = int64(len(req.GrayPolicy.NodeIds))
		allowCount, err = l.addGrayNodesByNodePool(req, topic)
		if err != nil {
			return 0, 0, "", err
		}

		// 记录到发布历史
		return allowCount, totalCount, "", nil
	}

	if req.GrayPolicy.Filter != nil {
		allowCount, totalCount, markID, err = l.addGrayNodesByFilter(req, topic)
		if err != nil {
			return 0, 0, "", err
		}

	}

	return allowCount, totalCount, markID, nil
}

func (l *NodeReleaseCreateLogic) addGrayNodesByNodePool(req *types.NodeReleaseCreateReq, topic string) (allowCount int64, err error) {
	// 首次添加, 请求即为全部灰度节点
	targetCount := len(req.GrayPolicy.NodeIds) * req.GrayPolicy.Percentage / 100
	if targetCount == 0 {
		// 至少加入一个灰度节点
		targetCount = 1
	}

	if allowCount, err = noderelease.GetAllowNodesCount(l.ctx, l.svcCtx.BizRedis, topic); err != nil {
		return 0, err
	}

	currentOffset := 0
	targetOffset := targetCount
	for currentOffset < targetOffset && targetOffset <= len(req.GrayPolicy.NodeIds) {
		grayNodes := req.GrayPolicy.NodeIds[currentOffset:targetOffset]

		// add to redis
		_, err := noderelease.AddAllowNodes(l.ctx, l.svcCtx.BizRedis, topic, grayNodes)
		if err != nil {
			return 0, err
		}

		currentOffset = targetOffset

		// 获取最新的灰度节点数
		allowCount, err = noderelease.GetAllowNodesCount(l.ctx, l.svcCtx.BizRedis, topic)
		if err != nil {
			return 0, err
		}

		if allowCount < int64(targetCount) {
			targetOffset += targetCount - int(allowCount)
		}
	}

	return allowCount, nil
}

func (l *NodeReleaseCreateLogic) addGrayNodesByFilter(req *types.NodeReleaseCreateReq, topic string) (allowCount int64, totalCount int64, markId string, err error) {

	cond := &sharedmodel.NodeSearchCond{
		DeviceType: []string{device.DevType(req.DevType).String()},
	}

	if req.GrayPolicy.Filter.Status != "" {
		cond.State = req.GrayPolicy.Filter.Status
	}

	if req.GrayPolicy.Filter.CustomerIds != nil {
		cond.CustomerIDs = req.GrayPolicy.Filter.CustomerIds
	}

	if len(req.GrayPolicy.Filter.Stages) > 0 {
		cond.Stage = strings.Join(req.GrayPolicy.Filter.Stages, ",")
	}

	// 先取首页, 获取总数
	var mark string
	pageParam := sharedmodel.PageParam{Size: 100}
	pageParam.SetMarkSort("_id", sharedmodel.SortTypeAsc)

	// 计算本次需要添加的灰度节点数
	var targetGrayCount int
	var currentGrayCount int64

	for {
		pageParam.SetMark(mark)
		cond := &sharedmodel.NodeSearchCond{
			PageParam:   pageParam,
			DeviceType:  []string{device.DevType(req.DevType).String()},
			State:       sharedmodel.DNodeStatusOnline,
			CustomerIDs: req.GrayPolicy.Filter.CustomerIds,
			Stage:       strings.Join(req.GrayPolicy.Filter.Stages, ","),
			NodeType:    "all", // 查询条件已经包含devType, nodeType可忽略, 根据当前search接口实现, 此处填all, 实际查询时将不涉及nodeType字段, 减小索引压力
		}

		// 如果已获取到节点总数, 无须再获取
		if mark != "" {
			cond.NoCount = true
		}

		nodeJoins, markID, total, err := l.svcCtx.NodeJoinModel.Search(l.ctx, cond)
		if err != nil {
			return 0, 0, "", err
		}
		if total == 0 && targetGrayCount == 0 {
			return 0, 0, "", errorx.NewDefaultError("没有符合要求的灰度节点")
		}

		mark = markID
		// 初始化本次灰度的目标节点数
		if total > 0 && targetGrayCount == 0 {
			targetGrayCount = total * req.GrayPolicy.Percentage / 100
			if targetGrayCount == 0 {
				// 至少加入一个灰度节点
				targetGrayCount = 1
			}
			totalCount = int64(total)
		}

		if leftGrayCount := targetGrayCount - int(currentGrayCount); leftGrayCount > len(nodeJoins) {
			// 待灰度节点数大于本批次节点, 先把这批节点加入灰度
			var nodeIds []string
			for _, nodeJoin := range nodeJoins {
				nodeIds = append(nodeIds, nodeJoin.NodeId)
			}

			if len(nodeIds) > 0 {
				_, err = noderelease.AddAllowNodes(l.ctx, l.svcCtx.BizRedis, topic, nodeIds)
				if err != nil {
					return 0, 0, "", err
				}
			}
		} else {
			// 待灰度节点数小于本批次节点, 把本批次部分节点加入灰度
			var nodeIds []string
			for i := 0; i < leftGrayCount; i++ {
				nodeIds = append(nodeIds, nodeJoins[i].NodeId)
				mark = nodeJoins[i].Id
			}

			if len(nodeIds) > 0 {
				_, err := noderelease.AddAllowNodes(l.ctx, l.svcCtx.BizRedis, topic, nodeIds)
				if err != nil {
					return 0, 0, "", err
				}
			}
		}

		cnt, err := noderelease.GetAllowNodesCount(l.ctx, l.svcCtx.BizRedis, topic)
		if err != nil {
			return 0, 0, "", err
		}

		currentGrayCount = cnt
		if currentGrayCount >= int64(targetGrayCount) {
			break
		}
	}

	return currentGrayCount, totalCount, mark, nil
}

func (l *NodeReleaseCreateLogic) ensureGrayPolicyValid(req *types.NodeReleaseCreateReq) error {
	if req.GrayPolicy.Percentage <= 0 || req.GrayPolicy.Percentage > 100 {
		return errorx.NewDefaultError("灰度比例范围错误, 请检查")
	}

	if req.GrayPolicy.Filter != nil && req.GrayPolicy.Filter.DevType == "" {
		return errorx.NewDefaultError("灰度策略设备类型不能为空, 请检查")
	}

	if req.GrayPolicy.Filter == nil && len(req.GrayPolicy.NodeIds) == 0 {
		return errorx.NewDefaultError("灰度策略&灰度节点不能同时为空, 请检查")
	}

	if req.GrayPolicy.Filter != nil && req.GrayPolicy.Filter.DevType != req.DevType {
		return errorx.NewDefaultError("灰度策略设备类型错误, 请检查")
	}

	// 如果是指定规则灰度, 需要确保当前没有按规则灰度任务
	release, err := l.svcCtx.NodeReleaseModel.FindProcessingRelease(l.ctx, req.AppName, nil, device.DevType(req.DevType))
	if err != nil {
		return err
	}

	for _, r := range release {
		if r.GrayPolicy.Filter != nil {
			return errorx.NewDefaultError(fmt.Sprintf("当前存在按规则灰度任务%s, 请先完成该任务", r.ID))
		}
	}

	// 若指定节点id, 检查id是否被占用
	if len(req.GrayPolicy.NodeIds) > 0 {
		fakeRelease := &sharedmodel.NodeRelease{
			App:        req.AppName,
			DeviceType: req.DevType,
		}
		if tips, _, err := noderelease.EnsureNodesNotInOtherTasks(l.ctx, l.allowNodesTopicCache, l.svcCtx.AllowAppsModel, l.svcCtx.BizRedis, l.nodeIdCache, fakeRelease, release, req.GrayPolicy.NodeIds); err != nil {
			if len(tips) > 0 {
				return errorx.NewDefaultError(strings.Join(tips, "\n"))
			}
			return err
		}
	}

	return nil
}

func (l *NodeReleaseCreateLogic) ensurePackageValid(req *types.NodeReleaseCreateReq) error {
	if req.AppConfig == nil {
		return nil
	}
	md5, err := kodo.GetKodoFileMd5(req.AppConfig.PackageUrl)
	if err != nil {
		return errorx.NewDefaultError("url无效, 请检查")
	}

	if md5 != req.AppConfig.PackageMD5 {
		return errorx.NewDefaultError("包md5错误, 请检查")
	}

	return nil
}

func (l *NodeReleaseCreateLogic) ensureReleaseCompatible(req *types.NodeReleaseCreateReq) error {
	switch req.ReleaseType {
	case sharedmodel.ReleaseTypeFormal.String():
		// 检查正式发布任务是否存在
		release, err := l.svcCtx.NodeReleaseModel.FindProcessingRelease(l.ctx, req.AppName, []sharedmodel.ReleaseType{sharedmodel.ReleaseTypeFormal}, device.DevType(req.DevType))
		if err != nil {
			return err
		}
		if release != nil {
			return errorx.NewDefaultError(fmt.Sprintf("同设备类型&应用(%s&%s)存在进行中正式发布任务, 请先完成该任务", req.DevType, req.AppName))
		}

	case sharedmodel.ReleaseTypeBeta.String():
		// 不能存在指定过滤规则的发布任务
		release, err := l.svcCtx.NodeReleaseModel.FindProcessingRelease(l.ctx, req.AppName, nil, device.DevType(req.DevType))
		if err != nil {
			return err
		}

		for _, r := range release {
			if r.GrayPolicy.Filter != nil {
				return errorx.NewDefaultError(fmt.Sprintf("同设备类型&应用(%s&%s)存在按规则过滤灰度任务, 请先完成该任务或切换为指定节点灰度", req.DevType, req.AppName))
			}
		}
	default:
		return errorx.NewDefaultError("发布类型无效, 请检查")
	}

	return nil
}

func (l *NodeReleaseCreateLogic) checkBasicArgs(req *types.NodeReleaseCreateReq) error {

	if !device.DevType(req.DevType).Valid() {
		return errorx.NewDefaultError("devType参数无效, 请检查")
	}

	if req.AppName == "" {
		return errorx.NewDefaultError("appName参数无效, 请检查")
	}

	// 检查app配置有效性
	if err := l.appConfigValidate(req.AppConfig, req.OpType, req.DevType, req.ReleaseType); err != nil {
		return err
	}

	return nil
}

func (l *NodeReleaseCreateLogic) appConfigValidate(appConfig *types.AppConfig, opType string, devType string, releaseType string) error {

	switch sharedmodel.NodeReleaseOpType(opType) {
	case sharedmodel.NodeReleaseOpTypeAddApp:
		if appConfig == nil {
			return errorx.NewDefaultError("新增组件请指定组件配置")
		}

		err := checkAppConfig(appConfig, device.DevType(devType))
		if err != nil {
			return err
		}

		// 新增应用时, 禁止创建功能验证任务
		if sharedmodel.ReleaseTypeBeta.String() == releaseType {
			return errorx.NewDefaultError("新增应用时, 禁止创建功能验证任务")
		}

		// 确保allowApps中已创建组件
		exist, err := l.ensureAllowAppsExist(appConfig.Cmd, devType)
		if err != nil {
			return err
		}

		if !exist {
			return errorx.NewDefaultError("组件不存在, 请先创建组件")
		}

		// 新增组件时确保配置不存在, 避免误操作
		mainConfig, alterConfig, err := noderelease.GetAppConfig(l.ctx, l.svcCtx.SysParamModel, device.DevType(devType), appConfig.Cmd)
		if err != nil && err != sharedmodel.ErrNotFound {
			return err
		}
		if mainConfig != nil || alterConfig != nil {
			return errorx.NewDefaultError("新增组件配置已存在, 请检查")
		}

	case sharedmodel.NodeReleaseOpTypeUpdateApp:
		if appConfig == nil {
			return errorx.NewDefaultError("升级组件请指定组件配置")
		}

		err := checkAppConfig(appConfig, device.DevType(devType))
		if err != nil {
			return err
		}

		// 确保allowApps中已创建组件
		exist, err := l.ensureAllowAppsExist(appConfig.Cmd, devType)
		if err != nil {
			return err
		}

		if !exist {
			return errorx.NewDefaultError("组件不存在, 请先创建组件")
		}

		// 升级组件时必须存在main配置, 避免误操作
		mainConfig, err := noderelease.GetAppMainConfig(l.ctx, l.svcCtx.SysParamModel, device.DevType(devType), appConfig.Cmd)
		if err != nil {
			return err
		}
		if mainConfig == nil {
			return errorx.NewDefaultError("升级应用时必须存在应用配置, 请检查")
		}

	case sharedmodel.NodeReleaseOpTypeDeleteApp:
		// 新增应用时, 禁止创建功能验证任务
		if sharedmodel.ReleaseTypeBeta.String() == releaseType {
			return errorx.NewDefaultError("删除应用时, 禁止创建功能验证任务")
		}

		return nil

	default:
		return errorx.NewDefaultError("操作类型无效")
	}

	return nil
}

func checkAppConfig(appConfig *types.AppConfig, devType device.DevType) error {

	if appConfig.PackageUrl == "" {
		return errorx.NewDefaultError("包url不能为空, 请检查")
	}
	if appConfig.PackageType == "" {
		return errorx.NewDefaultError("包类型不能为空, 请检查")
	}
	if appConfig.Cmd == "" {
		return errorx.NewDefaultError("cmd不能为空, 请检查")
	}

	// PCDN-20836 安卓部分组件不指定workDir, 调整为只针对非安卓做检查
	if appConfig.WorkDir == "" && !devType.IsDroidGroup() {
		return errorx.NewDefaultError("workDir不能为空, 请检查")
	}

	if appConfig.PackageMD5 == "" {
		return errorx.NewDefaultError("包md5不能为空, 请检查")
	}

	return nil
}

func (l *NodeReleaseCreateLogic) ensureAllowAppsExist(appName string, devType string) (bool, error) {
	nodeType := sharedmodel.NodeTypeNode
	if device.DevType(devType).IsSmallBoxDev() {
		nodeType = sharedmodel.NodeTypeSmallBox
	}

	allowApps, _, err := l.svcCtx.AllowAppsModel.Search(l.ctx, sharedmodel.AllowAppsListCond{
		Names:     []string{appName},
		NodeTypes: []string{nodeType.String()},
	})
	if err != nil {
		return false, err
	}

	if len(allowApps) == 0 {
		return false, nil
	}

	return true, nil
}
