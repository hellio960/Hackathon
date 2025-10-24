package noderelease

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/svc"
	"hackathon/cmd/jarvis/internal/types"
	"hackathon/cmd/jarvis/shared/noderelease"
	"hackathon/common/device"
	"hackathon/common/errorx"
	"hackathon/sharedmodel"
)

type NodeReleaseFilterSwitchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewNodeReleaseFilterSwitchLogic(ctx context.Context, svcCtx *svc.ServiceContext) NodeReleaseFilterSwitchLogic {
	return NodeReleaseFilterSwitchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *NodeReleaseFilterSwitchLogic) NodeReleaseFilterSwitch(req *types.NodeReleaseFilterSwitchReq) error {

	// 检查参数
	if err := l.checkArgs(req); err != nil {
		return err
	}

	// 获取release
	release, err := l.svcCtx.NodeReleaseModel.Find(l.ctx, req.ReleaseID)
	if err != nil {
		return err
	}

	return l.switchFilter(req, release, operator)
}

func (l *NodeReleaseFilterSwitchLogic) checkArgs(req *types.NodeReleaseFilterSwitchReq) error {
	if req.ByFilter != nil && req.ByNodeIds {
		return errorx.NewDefaultError("不能同时指定过滤规则和指定节点")
	}

	if req.ByFilter == nil && !req.ByNodeIds {
		return errorx.NewDefaultError("必须指定过滤规则或指定节点")
	}

	return nil
}

func (l *NodeReleaseFilterSwitchLogic) switchFilter(req *types.NodeReleaseFilterSwitchReq, release *sharedmodel.NodeRelease, operator string) error {
	if req.ByFilter != nil {
		// 若曾从规则匹配切到指定节点, 此时又尝试切回规则匹配, 则要求匹配规则必须一致
		history, _, err := l.svcCtx.NodeReleaseHistoryModel.Search(l.ctx, &sharedmodel.NodeReleaseHistoryCond{
			ReleaseID: release.ID,
			Operation: []string{sharedmodel.NodeReleaseHistoryOperationSwitchFilter.String(), sharedmodel.NodeReleaseHistoryOperationCreate.String()},
		})
		if err != nil {
			return err
		}
		for _, h := range history {
			if h.GrayPolicyInfo != nil && h.GrayPolicyInfo.AfterFilter != nil {
				if !compareFilter(h.GrayPolicyInfo.AfterFilter, req.ByFilter) {
					return errorx.NewDefaultError("当前指定的规则与首次指定的规则不一致, 请检查")
				}
			}
		}

		return l.switchToByFilter(req, release, operator)
	}

	return l.switchToByNodeIds(release, operator)
}

func (l *NodeReleaseFilterSwitchLogic) switchToByFilter(req *types.NodeReleaseFilterSwitchReq, release *sharedmodel.NodeRelease, operator string) error {
	// 如果是指定规则灰度, 需要确保当前没有按规则灰度任务
	releaseList, err := l.svcCtx.NodeReleaseModel.FindProcessingRelease(l.ctx, release.App, nil, device.DevType(release.DeviceType))
	if err != nil {
		return err
	}

	for _, r := range releaseList {
		if r.GrayPolicy.Filter != nil {
			return errorx.NewDefaultError(fmt.Sprintf("当前存在按规则灰度任务%s, 请先完成该任务", r.ID))
		}
	}

	// 当前灰度节点不用变更, 只更新灰度策略
	nodeIdsCount := len(release.GrayPolicy.NodeIds)
	release.GrayPolicy.NodeIds = nil

	var filter sharedmodel.GrayFilter
	b, err := json.Marshal(req.ByFilter)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &filter); err != nil {
		return err
	}
	release.GrayPolicy.Filter = &filter
	release.GrayPolicy.NodeIds = nil

	// 计算匹配总数, 并更新统计信息
	total, err := l.calculateTotalCount(req, release)
	if err != nil {
		return err
	}

	if total == 0 {
		return errorx.NewDefaultError("没有符合要求的灰度节点")
	}

	if release.GrayPolicy.StatInfo == nil {
		return errorx.NewDefaultError("任务异常:未找到灰度策略统计信息")
	}

	grayPolicyBak := release.GrayPolicy
	// 切到规则匹配时, 灰度节点池中的节点不会清除, 这些节点也算入节点总数
	release.GrayPolicy.StatInfo.TotalCount = int64(total) + int64(nodeIdsCount)
	release.GrayPolicy.Percentage = int(release.GrayPolicy.StatInfo.AllowCount * 100 / release.GrayPolicy.StatInfo.TotalCount)
	release.GrayPolicy.StatInfo.TotalCountUpdateAt = time.Now()

	if err = l.svcCtx.NodeReleaseModel.Update(l.ctx, release); err != nil {
		return err
	}

	// 添加操作历史
	return l.svcCtx.NodeReleaseHistoryModel.Insert(l.ctx, &sharedmodel.NodeReleaseHistory{
		ReleaseID: release.ID,
		Operation: sharedmodel.NodeReleaseHistoryOperationSwitchFilter,
		Operator:  operator,
		OpTime:    time.Now(),
		CreateAt:  time.Now(),
		UpdateAt:  time.Now(),

		// 变更信息
		GrayPolicyInfo: &sharedmodel.GrayPolicyRemark{
			AfterFilter:      release.GrayPolicy.Filter,
			FilterChangeMode: sharedmodel.NodeReleaseFilterChangeModeNodes2Filter,
			BeforePercentage: grayPolicyBak.Percentage,
			AfterPercentage:  release.GrayPolicy.Percentage,
		},
	})
}

func (l *NodeReleaseFilterSwitchLogic) switchToByNodeIds(release *sharedmodel.NodeRelease, operator string) error {
	// 把当前灰度节点作为灰度节点池, 且灰度比例为100%
	nodeType := sharedmodel.NodeTypeNode
	if device.DevType(release.DeviceType).IsSmallBoxDev() {
		nodeType = sharedmodel.NodeTypeSmallBox
	}
	topic, err := noderelease.GetAllowNodesTopic(l.ctx, nil, l.svcCtx.AllowAppsModel, release.App, nodeType.String(), release.ID)
	if err != nil {
		return err
	}

	allowNodes, err := noderelease.GetAllowNodes(l.ctx, l.svcCtx.BizRedis, topic)
	if err != nil {
		return err
	}
	if len(allowNodes) == 0 {
		return errorx.NewDefaultError("当前任务无灰度中节点, 请检查")
	}

	if release.GrayPolicy.StatInfo == nil {
		return errorx.NewDefaultError("任务异常:未找到灰度策略统计信息")
	}
	grayPolicyBak := release.GrayPolicy
	release.GrayPolicy.NodeIds = allowNodes
	release.GrayPolicy.Percentage = 100
	release.GrayPolicy.Filter = nil
	release.GrayPolicy.StatInfo.TotalCount = int64(len(allowNodes))
	release.GrayPolicy.StatInfo.AllowCount = int64(len(allowNodes))
	release.GrayPolicy.StatInfo.TotalCountUpdateAt = time.Now()
	release.GrayPolicy.StatInfo.AllowCountUpdateAt = time.Now()
	release.GrayPolicy.StatInfo.MarkID = ""

	if err = l.svcCtx.NodeReleaseModel.Update(l.ctx, release); err != nil {
		return err
	}

	// 添加操作历史
	if err = l.svcCtx.NodeReleaseHistoryModel.Insert(l.ctx, &sharedmodel.NodeReleaseHistory{
		ReleaseID: release.ID,
		Operation: sharedmodel.NodeReleaseHistoryOperationSwitchFilter,
		Operator:  operator,
		OpTime:    time.Now(),
		CreateAt:  time.Now(),
		UpdateAt:  time.Now(),

		// 变更信息
		GrayPolicyInfo: &sharedmodel.GrayPolicyRemark{
			NodeIdsAdd:       release.GrayPolicy.NodeIds,
			FilterChangeMode: sharedmodel.NodeReleaseFilterChangeModeFilter2Nodes,
			BeforePercentage: grayPolicyBak.Percentage,
			AfterPercentage:  release.GrayPolicy.Percentage,
		},
	}); err != nil {
		return err
	}

	return nil
}

func (l *NodeReleaseFilterSwitchLogic) calculateTotalCount(req *types.NodeReleaseFilterSwitchReq, release *sharedmodel.NodeRelease) (int, error) {
	nodeType := sharedmodel.NodeTypeNode
	if device.DevType(release.DeviceType).IsSmallBoxDev() {
		nodeType = sharedmodel.NodeTypeSmallBox
	}

	pageParam := sharedmodel.PageParam{Size: 1}
	pageParam.SetMarkSort("_id", sharedmodel.SortTypeAsc)

	pageParam.SetMark("")
	cond := &sharedmodel.NodeSearchCond{
		PageParam:   pageParam,
		DeviceType:  []string{device.DevType(release.DeviceType).String()},
		State:       req.ByFilter.Status,
		CustomerIDs: req.ByFilter.CustomerIds,
		Stage:       strings.Join(req.ByFilter.Stages, ","), // todo 注意加索引, 带上sortId
		NodeType:    nodeType.String(),
	}

	_, _, total, err := l.svcCtx.NodeJoinModel.Search(l.ctx, cond)
	if err != nil {
		return 0, err
	}

	return total, nil
}

// compareSlice 比较两个切片是否包含相同的元素（忽略顺序）
func compareSlice[T comparable](ids1, ids2 []T) bool {
	if len(ids1) != len(ids2) {
		return false
	}

	seen := make(map[T]int)

	for _, id := range ids1 {
		seen[id]++
	}

	for _, id := range ids2 {
		if seen[id] == 0 {
			return false
		}
		seen[id]--
	}

	return true
}

func compareFilter(filter1 *sharedmodel.GrayFilter, filter2 *types.GrayFilter) bool {
	return compareSlice(filter1.CustomerIds, filter2.CustomerIds) &&
		filter1.DevType.String() == filter2.DevType &&
		filter1.Status == filter2.Status &&
		compareSlice(filter1.Stages, filter2.Stages)
}
