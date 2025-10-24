package noderelease

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/cmd/jarvis/shared/noderelease"
	"hackathon/common/device"
	"hackathon/common/errorx"
	"hackathon/common/utils"
	"hackathon/sharedmodel"
)

const operator = "system"

type NodeReleaseContinueLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext

	nodeIdPoolCache      map[string]map[string]struct{} // key: releaseId, value: {key:nodeId, val:struct{}}
	allowNodesTopicCache noderelease.AllowNodesTopicCache
}

func NewNodeReleaseContinueLogic(ctx context.Context, svcCtx *svc.ServiceContext) NodeReleaseContinueLogic {
	return NodeReleaseContinueLogic{
		Logger:               logx.WithContext(ctx),
		ctx:                  ctx,
		svcCtx:               svcCtx,
		nodeIdPoolCache:      make(map[string]map[string]struct{}),
		allowNodesTopicCache: noderelease.NewAllowNodesTopicCache(),
	}
}

func (l *NodeReleaseContinueLogic) NodeReleaseContinue(req *types.NodeReleaseContinueReq) error {

	// 参数校验
	releasePlan, err := l.checkBasicArgs(req)
	if err != nil {
		return err
	}

	// 检查发布任务状态
	if err := l.ensureReleaseState(releasePlan, req); err != nil {
		return err
	}

	nodeType := sharedmodel.NodeTypeNode
	if device.DevType(releasePlan.DeviceType).IsSmallBoxDev() {
		nodeType = sharedmodel.NodeTypeSmallBox
	}

	topic, err := noderelease.GetAllowNodesTopic(l.ctx, l.allowNodesTopicCache, l.svcCtx.AllowAppsModel, releasePlan.App, nodeType.String(), releasePlan.ID)
	if err != nil {
		return err
	}

	// 获取当前正在灰度中的任务
	processingTasks, _, err := l.svcCtx.NodeReleaseModel.Search(l.ctx, &sharedmodel.NodeReleaseListCond{
		States:      []string{sharedmodel.NodeReleaseStateProcessing.String()},
		DeviceTypes: []string{device.DevType(releasePlan.DeviceType).String()},
		App:         releasePlan.App,
	})
	if err != nil {
		return err
	}

	grayPolicyBackup := releasePlan.GrayPolicy
	if req.Percentage != releasePlan.GrayPolicy.Percentage {
		// 修改发布比例, 增加相应灰度节点
		if err = l.addAllowNodes2Redis(req, releasePlan, processingTasks); err != nil {
			return err
		}

		releasePlan.GrayPolicy.Percentage = req.Percentage
	}

	// 当增加或删除灰度节点时, 只进行节点池增加或存量删除
	if len(req.AddNodes) > 0 {
		// 确保灰度节点未被占用
		if tips, _, err := noderelease.EnsureNodesNotInOtherTasks(l.ctx, l.allowNodesTopicCache, l.svcCtx.AllowAppsModel, l.svcCtx.BizRedis, l.nodeIdPoolCache, releasePlan, processingTasks, req.AddNodes); err != nil {
			if len(tips) > 0 {
				return errorx.NewDefaultError(strings.Join(tips, "\n"))
			}
			return err
		}

		// 需要区分发布模式, 如果是规则过滤, 直接加到灰度列表, 如果是指定节点灰度, 加到节点池
		if releasePlan.GrayPolicy.Filter != nil {
			// 将节点加入灰度列表
			cnt, err := noderelease.AddAllowNodes(l.ctx, l.svcCtx.BizRedis, topic, req.AddNodes)
			if err != nil {
				return err
			}

			// 更新灰度比例
			releasePlan.GrayPolicy.StatInfo.AllowCount += int64(cnt)
			releasePlan.GrayPolicy.StatInfo.AllowCountUpdateAt = time.Now()
			if releasePlan.GrayPolicy.StatInfo.TotalCount > 0 {
				releasePlan.GrayPolicy.Percentage = int(releasePlan.GrayPolicy.StatInfo.AllowCount) * 100 / int(releasePlan.GrayPolicy.StatInfo.TotalCount)
			}
		} else {
			if err = l.addNewNodes2Pool(req.AddNodes, releasePlan, processingTasks, topic, operator); err != nil {
				return err
			}
		}
	}

	if len(req.DelNodes) > 0 {
		if releasePlan.GrayPolicy.Filter != nil {
			// 从灰度列表里删除指定节点ID
			cnt, err := noderelease.RemoveAllowNodes(l.ctx, l.svcCtx.BizRedis, topic, req.DelNodes)
			if err != nil {
				return err
			}

			// 更新灰度比例
			releasePlan.GrayPolicy.StatInfo.AllowCount -= int64(cnt)
			if releasePlan.GrayPolicy.StatInfo.AllowCount < 0 {
				releasePlan.GrayPolicy.StatInfo.AllowCount = 0
			}
			releasePlan.GrayPolicy.StatInfo.AllowCountUpdateAt = time.Now()
			if releasePlan.GrayPolicy.StatInfo.TotalCount > 0 {
				releasePlan.GrayPolicy.Percentage = int(releasePlan.GrayPolicy.StatInfo.AllowCount) * 100 / int(releasePlan.GrayPolicy.StatInfo.TotalCount)
			}
		} else {
			if err = l.delNodesWithPool(topic, req.DelNodes, releasePlan, operator); err != nil {
				return err
			}
		}
	}

	// 保存发布任务
	if err := l.svcCtx.NodeReleaseModel.Update(l.ctx, releasePlan); err != nil {
		return err
	}

	// 记录操作日志
	return l.svcCtx.NodeReleaseHistoryModel.Insert(l.ctx, &sharedmodel.NodeReleaseHistory{
		ReleaseID: releasePlan.ID,
		Operation: sharedmodel.NodeReleaseHistoryOperationContinue,
		Operator:  operator,
		OpTime:    time.Now(),
		CreateAt:  time.Now(),
		UpdateAt:  time.Now(),

		// 变更信息
		GrayPolicyInfo: &sharedmodel.GrayPolicyRemark{
			NodeIdsAdd:       req.AddNodes,
			NodeIdsDel:       req.DelNodes,
			BeforePercentage: grayPolicyBackup.Percentage,
			AfterPercentage:  releasePlan.GrayPolicy.Percentage,
		},
	})
}

func (l *NodeReleaseContinueLogic) checkBasicArgs(req *types.NodeReleaseContinueReq) (*sharedmodel.NodeRelease, error) {
	if req.ReleaseID == "" {
		return nil, errorx.NewDefaultError("releaseId参数无效, 请检查")
	}

	if req.Percentage <= 0 || req.Percentage > 100 {
		return nil, errorx.NewDefaultError("灰度比例无效, 请检查")
	}

	// 新增节点和删除节点不能存在交集
	if utils.CommonInAB(req.AddNodes, req.DelNodes) {
		return nil, errorx.NewDefaultError("新增节点和移除节点中存在交集, 请检查")
	}

	releasePlan, err := l.svcCtx.NodeReleaseModel.Find(l.ctx, req.ReleaseID)
	if err != nil {
		if err == sharedmodel.ErrNotFound {
			return nil, errorx.NewDefaultError("发布任务不存在, 请检查")
		}

		l.Logger.Errorf("find node release failed, releaseId: %s, err: %v", req.ReleaseID, err)
		return nil, err
	}

	if releasePlan == nil {
		return nil, errorx.NewDefaultError("获取发布任务失败, 请检查")
	}

	if req.Percentage != releasePlan.GrayPolicy.Percentage && (len(req.AddNodes) > 0 || len(req.DelNodes) > 0) {
		return nil, errorx.NewDefaultError("灰度比例和新增/删除节点不能同时修改")
	}

	return releasePlan, nil
}

func (l *NodeReleaseContinueLogic) ensureReleaseState(releasePlan *sharedmodel.NodeRelease, req *types.NodeReleaseContinueReq) error {
	if !utils.InAnySlice(releasePlan.State, []sharedmodel.NodeReleaseState{sharedmodel.NodeReleaseStateProcessing}) {
		return errorx.NewDefaultError("发布任务不在可发布状态, 请检查")
	}

	if releasePlan.GrayPolicy.Percentage > req.Percentage {
		return errorx.NewDefaultError("禁止调小灰度比例")
	}

	return nil
}

func (l *NodeReleaseContinueLogic) delNodesWithPool(topic string, delNodes []string, releasePlan *sharedmodel.NodeRelease, operator string) error {
	NodePoolSet := collection.NewSet()
	for _, node := range releasePlan.GrayPolicy.NodeIds {
		NodePoolSet.AddStr(node)
	}

	for _, node := range delNodes {
		NodePoolSet.Remove(node)
	}

	if _, err := noderelease.RemoveAllowNodes(l.ctx, l.svcCtx.BizRedis, topic, delNodes); err != nil {
		return err
	}

	// 保持原有节点id顺序
	var allNodes []string
	for _, node := range releasePlan.GrayPolicy.NodeIds {
		if NodePoolSet.Contains(node) {
			allNodes = append(allNodes, node)
		}
	}

	// 重新计算灰度节点数
	count, err := noderelease.GetAllowNodesCount(l.ctx, l.svcCtx.BizRedis, topic)
	if err != nil {
		return err
	}

	releasePlan.GrayPolicy.NodeIds = allNodes
	releasePlan.GrayPolicy.StatInfo.TotalCount = int64(len(allNodes))
	releasePlan.GrayPolicy.StatInfo.AllowCount = int64(count)
	releasePlan.GrayPolicy.StatInfo.AllowCountUpdateAt = time.Now()
	if releasePlan.GrayPolicy.StatInfo.TotalCount > 0 {
		releasePlan.GrayPolicy.Percentage = int(releasePlan.GrayPolicy.StatInfo.AllowCount) * 100 / int(releasePlan.GrayPolicy.StatInfo.TotalCount)
	}
	if err := l.recalculatePercentage(releasePlan, topic); err != nil {
		return err
	}
	return nil
}

func (l *NodeReleaseContinueLogic) addNewNodes2Pool(newNodes []string, releasePlan *sharedmodel.NodeRelease, processingTasks []*sharedmodel.NodeRelease, topic, operator string) error {
	NodePoolSet := collection.NewSet()
	NewNodesSet := collection.NewSet()
	for _, node := range releasePlan.GrayPolicy.NodeIds {
		NodePoolSet.AddStr(node)
	}

	var validNewNodes []string
	for _, node := range newNodes {
		// 忽略重复节点
		if NewNodesSet.Contains(node) {
			continue
		}

		// 忽略已添加节点
		if NodePoolSet.Contains(node) {
			continue
		}

		validNewNodes = append(validNewNodes, node)
		NewNodesSet.AddStr(node)
	}

	releasePlan.GrayPolicy.NodeIds = append(releasePlan.GrayPolicy.NodeIds, validNewNodes...)
	releasePlan.GrayPolicy.StatInfo.TotalCount = int64(len(releasePlan.GrayPolicy.NodeIds))

	if err := l.recalculatePercentage(releasePlan, topic); err != nil {
		return err
	}

	return nil
}

func (l *NodeReleaseContinueLogic) recalculatePercentage(releasePlan *sharedmodel.NodeRelease, topic string) error {

	count, err := noderelease.GetAllowNodesCount(l.ctx, l.svcCtx.BizRedis, topic)
	if err != nil {
		return err
	}

	// 暂时只支持指定节点的比例计算
	if len(releasePlan.GrayPolicy.NodeIds) == 0 {
		return errorx.NewDefaultError("指定灰度节点不能为空, 请检查")
	}

	releasePlan.GrayPolicy.Percentage = int(count) * 100 / len(releasePlan.GrayPolicy.NodeIds)

	if releasePlan.GrayPolicy.Percentage > 100 {
		releasePlan.GrayPolicy.Percentage = 100
	}

	if releasePlan.GrayPolicy.Percentage < 0 {
		releasePlan.GrayPolicy.Percentage = 0
	}

	return nil
}

func (l *NodeReleaseContinueLogic) ensureAllowNodesNotEqualDelNodes(topic string, delNodes []string) error {
	allowCount, err := noderelease.GetAllowNodesCount(l.ctx, l.svcCtx.BizRedis, topic)
	if err != nil {
		return err
	}

	// 当前灰度节点数大于待删除节点数, 不会出现灰度节点被清空的情况
	if allowCount > int64(len(delNodes)) {
		return nil
	}

	var toDelCount int64
	for _, node := range delNodes {
		exist, err := l.svcCtx.BizRedis.Sismember(topic, node)
		if err != nil {
			return err
		}
		if exist {
			toDelCount++
		}
	}

	if toDelCount >= allowCount {
		// 处理方案: 导出当前任务的灰度中节点, 与待删除节点对比, 避免全部删除
		return errorx.NewDefaultError(fmt.Sprintf("待删除的节点完全包含当前灰度节点, 禁止完全清空灰度中节点. 尝试删除节点数: %d, 包含灰度中节点数: %d, 当前灰度节点数: %d", len(delNodes), toDelCount, allowCount))
	}

	return nil

}

func (l *NodeReleaseContinueLogic) addAllowNodes2Redis(req *types.NodeReleaseContinueReq, releasePlan *sharedmodel.NodeRelease, processingTasks []*sharedmodel.NodeRelease) error {
	// 通过策略执行灰度
	if releasePlan.GrayPolicy.Filter != nil {
		return l.addAllowNodes2RedisByFilter(req, releasePlan, processingTasks)
	}

	// 通过指定节点执行灰度
	if len(releasePlan.GrayPolicy.NodeIds) > 0 {
		return l.addAllowNodes2RedisByNodeIds(req, releasePlan)
	}

	return nil
}

func (l *NodeReleaseContinueLogic) addAllowNodes2RedisByFilter(req *types.NodeReleaseContinueReq, releasePlan *sharedmodel.NodeRelease, processingTasks []*sharedmodel.NodeRelease) error {
	targetGrayCount := releasePlan.GrayPolicy.StatInfo.TotalCount * int64(req.Percentage) / 100
	mark := releasePlan.GrayPolicy.StatInfo.MarkID

	// 计算当前灰度节点数
	nodeType := sharedmodel.NodeTypeNode
	if device.DevType(releasePlan.DeviceType).IsSmallBoxDev() {
		nodeType = sharedmodel.NodeTypeSmallBox
	}
	topic, err := noderelease.GetAllowNodesTopic(l.ctx, l.allowNodesTopicCache, l.svcCtx.AllowAppsModel, releasePlan.App, nodeType.String(), releasePlan.ID)
	if err != nil {
		return err
	}
	currentGrayCount, err := noderelease.GetAllowNodesCount(l.ctx, l.svcCtx.BizRedis, topic)
	if err != nil {
		return err
	}

	if currentGrayCount >= targetGrayCount {
		return nil
	}

	pageParam := sharedmodel.PageParam{Size: 500}
	pageParam.SetMarkSort("_id", sharedmodel.SortTypeAsc)
	for {
		pageParam.SetMark(mark)
		cond := &sharedmodel.NodeSearchCond{
			PageParam:   pageParam,
			DeviceType:  []string{device.DevType(releasePlan.DeviceType).String()},
			State:       releasePlan.GrayPolicy.Filter.Status,
			CustomerIDs: releasePlan.GrayPolicy.Filter.CustomerIds,
			Stage:       strings.Join(releasePlan.GrayPolicy.Filter.Stages, ","),
			NodeType:    "all", // 查询条件已经包含devType, nodeType可忽略, 根据当前search接口实现, 此处填all, 实际查询时将不涉及nodeType字段, 减小索引压力
			NoCount:     true,
		}

		nodeJoins, markID, _, err := l.svcCtx.NodeJoinModel.Search(l.ctx, cond)
		if err != nil {
			return err
		}

		l.Infof("len nodeJoins %d markID %s", len(nodeJoins), markID)
		mark = markID

		l.Infof("leftGrayCount %d len nodeJoins %d", targetGrayCount-currentGrayCount, len(nodeJoins))
		if leftGrayCount := targetGrayCount - currentGrayCount; leftGrayCount > int64(len(nodeJoins)) {
			// 待灰度节点数大于本批次节点, 先把这批节点加入灰度
			var nodeIds []string
			for _, nodeJoin := range nodeJoins {
				// 检查节点是否已被其它任务占用
				tips, curUsed, err := noderelease.EnsureNodesNotInOtherTasks(l.ctx, l.allowNodesTopicCache, l.svcCtx.AllowAppsModel, l.svcCtx.BizRedis, l.nodeIdPoolCache, releasePlan, processingTasks, []string{nodeJoin.NodeId})
				if err != nil || len(tips) > 0 {
					continue
				}
				if curUsed {
					continue
				}
				nodeIds = append(nodeIds, nodeJoin.NodeId)
			}

			_, err := noderelease.AddAllowNodes(l.ctx, l.svcCtx.BizRedis, topic, nodeIds)
			if err != nil {
				return err
			}
		} else {
			// 待灰度节点数小于本批次节点, 把本批次部分节点加入灰度
			var nodeIds []string
			for i := int64(0); i < leftGrayCount; i++ {
				// 检查节点是否已被其它任务占用
				tips, curUsed, err := noderelease.EnsureNodesNotInOtherTasks(l.ctx, l.allowNodesTopicCache, l.svcCtx.AllowAppsModel, l.svcCtx.BizRedis, l.nodeIdPoolCache, releasePlan, processingTasks, []string{nodeJoins[i].NodeId})
				if err != nil || len(tips) > 0 {
					continue
				}
				if curUsed {
					continue
				}
				nodeIds = append(nodeIds, nodeJoins[i].NodeId)
				mark = nodeJoins[i].Id
			}

			var cnt int
			if len(nodeIds) > 0 {
				cnt, err = noderelease.AddAllowNodes(l.ctx, l.svcCtx.BizRedis, topic, nodeIds)
				if err != nil {
					return err
				}
			}

			// 待灰度节点数小于本批次, 无须继续遍历
			if int64(cnt) >= leftGrayCount {
				break
			}
		}

		currentGrayCount, err = noderelease.GetAllowNodesCount(l.ctx, l.svcCtx.BizRedis, topic)
		if err != nil {
			return err
		}

		l.Infof("currentGrayCount %d targetGrayCount %d", currentGrayCount, targetGrayCount)
		if currentGrayCount >= targetGrayCount {
			break
		}

		// mark为空说明已经遍历到最后一页
		if mark == "" {
			break
		}
	}

	releasePlan.GrayPolicy.StatInfo.MarkID = mark
	releasePlan.GrayPolicy.StatInfo.AllowCount = currentGrayCount
	releasePlan.GrayPolicy.StatInfo.AllowCountUpdateAt = time.Now()

	return nil
}

func (l *NodeReleaseContinueLogic) addAllowNodes2RedisByNodeIds(req *types.NodeReleaseContinueReq, releasePlan *sharedmodel.NodeRelease) error {

	nodeType := sharedmodel.NodeTypeNode
	if device.DevType(releasePlan.DeviceType).IsSmallBoxDev() {
		nodeType = sharedmodel.NodeTypeSmallBox
	}
	topic, err := noderelease.GetAllowNodesTopic(l.ctx, l.allowNodesTopicCache, l.svcCtx.AllowAppsModel, releasePlan.App, nodeType.String(), releasePlan.ID)
	if err != nil {
		return err
	}
	currentGrayCount, err := noderelease.GetAllowNodesCount(l.ctx, l.svcCtx.BizRedis, topic)
	if err != nil {
		return err
	}

	targetGrayCount := int64(len(releasePlan.GrayPolicy.NodeIds)) * int64(req.Percentage) / 100
	for _, nodeId := range releasePlan.GrayPolicy.NodeIds {
		if currentGrayCount >= targetGrayCount {
			break
		}

		add, err := l.svcCtx.BizRedis.Sadd(topic, nodeId)
		if err != nil {
			return err
		}

		if add > 0 {
			currentGrayCount++
		}

		if currentGrayCount >= targetGrayCount {
			break
		}
	}

	currentGrayCount, err = noderelease.GetAllowNodesCount(l.ctx, l.svcCtx.BizRedis, topic)
	if err != nil {
		return err
	}

	l.Infof("currentGrayCount %d targetGrayCount %d", currentGrayCount, targetGrayCount)
	releasePlan.GrayPolicy.StatInfo.AllowCount = currentGrayCount
	releasePlan.GrayPolicy.StatInfo.AllowCountUpdateAt = time.Now()

	return nil
}
