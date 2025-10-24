package noderelease

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"

	"hackathon/cmd/jarvis/internal/localstorage"
	"hackathon/common/device"
	"hackathon/common/errorx"
	"hackathon/common/utils"
	"hackathon/sharedmodel"
)

const (
	JarvisUpdConfAllowNodes = "jarvis_upd_config_allow_nodes"
	BoxUpdConfAllowNodes    = "box_upd_config_allow_nodes"
)

type AllowNodesTopicCache map[topicCacheKey]string

type topicCacheKey struct {
	App       string
	NodeType  string
	ReleaseID string
}

func NewAllowNodesTopicCache() AllowNodesTopicCache {
	return make(AllowNodesTopicCache)
}

func GetAllowNodesTopicNoCheck(app, nodeType, releaseID string) (topic string, err error) {
	if len(app) == 0 {
		return "", errors.New("app name empty")
	}

	topicPrefix := JarvisUpdConfAllowNodes
	switch nodeType {
	case sharedmodel.NodeTypeSmallBox.String():
		topicPrefix = BoxUpdConfAllowNodes
	case sharedmodel.NodeTypeNode.String(), "":
	default:
		return "", errors.New("nodeType invalid")
	}

	// topicPrefix_{app}_{releaseID}
	topic = topicPrefix + "_" + app
	if releaseID != "" {
		topic = topic + "_" + releaseID
	}

	return
}

func GetAllowNodesTopic(ctx context.Context, cache AllowNodesTopicCache, allowAppsModel sharedmodel.AllowAppsModel, app, nodeType, releaseID string) (topic string, err error) {
	cacheKey := topicCacheKey{App: app, NodeType: nodeType, ReleaseID: releaseID}
	if val, ok := cache[cacheKey]; ok && val != "" {
		return val, nil
	}

	var appAllow *sharedmodel.AllowAppsTemplate
	if appAllow, err = allowAppsModel.FindByName(ctx, nodeType, app); err != nil || appAllow == nil {
		err = errors.New(fmt.Sprintf("app %s is not define", app))
		return
	}

	topic, err = GetAllowNodesTopicNoCheck(app, nodeType, releaseID)
	if err != nil {
		return "", err
	}

	if cache != nil {
		cache[cacheKey] = topic
	}

	return topic, nil
}

func AddAllowNodes(ctx context.Context, bizRedis *localstorage.LocalRedis, topic string, nodeIds []string) (addCount int, err error) {

	if bizRedis == nil {
		return 0, errors.New("bizRedis is nil")
	}

	if len(nodeIds) == 0 {
		return 0, errors.New("nodeIds is empty")
	}

	if strings.TrimSpace(topic) == "" {
		return 0, errors.New("topic is empty")
	}

	v := make([]interface{}, 0)
	for i := range nodeIds {
		v = append(v, nodeIds[i])
	}

	if addCount, err = bizRedis.SaddCtx(ctx, topic, v...); err != nil {
		err = errors.New(fmt.Sprintf("update %s fail, %s", topic, err.Error()))
		return 0, err
	}

	logx.WithContext(ctx).Infof("succeed add allowNodes count %d to topic %s", addCount, topic)
	return addCount, nil
}

func RemoveAllowNodes(ctx context.Context, bizRedis *localstorage.LocalRedis, topic string, nodeIds []string) (removeCount int, err error) {
	if bizRedis == nil {
		return 0, errors.New("bizRedis is nil")
	}

	if len(nodeIds) == 0 {
		return 0, errors.New("nodeIds is empty")
	}

	if strings.TrimSpace(topic) == "" {
		return 0, errors.New("topic is empty")
	}

	v := make([]interface{}, 0)
	for i := range nodeIds {
		v = append(v, nodeIds[i])
	}

	logx.Infof("remove allow nodes from topic: %s nodes count:%d", topic, len(v))
	if removeCount, err = bizRedis.SremCtx(ctx, topic, v...); err != nil {
		err = errors.New(fmt.Sprintf("update %s fail, %s", topic, err.Error()))
		return 0, err
	}

	return removeCount, nil
}

func GetAllowNodes(ctx context.Context, bizRedis *localstorage.LocalRedis, topic string) (nodes []string, err error) {
	if bizRedis == nil {
		return nil, errors.New("bizRedis is nil")
	}

	// 通过scan的方式获取灰度节点, 避免一次性获取太多节点导致内存占用过高
	var cursor uint64
	count := int64(200)
	nodeSet := collection.NewSet()
	for {
		keys, newCursor, err := bizRedis.SscanCtx(ctx, topic, cursor, "", count)
		if err != nil {
			err = errors.New(fmt.Sprintf("get %s nodes fail, %s", topic, err.Error()))
			return nil, err
		}

		logx.Errorf("got keys count %d", len(keys))

		// 去重
		for _, key := range keys {
			if nodeSet.Contains(key) {
				continue
			}
			nodeSet.Add(key)
			nodes = append(nodes, key)
		}

		if newCursor == 0 {
			break
		}
		cursor = newCursor
	}

	return nodes, nil
}

func GetAllowNodesCount(ctx context.Context, bizRedis *localstorage.LocalRedis, topic string) (count int64, err error) {
	if bizRedis == nil {
		return 0, errors.New("bizRedis is nil")
	}

	if count, err = bizRedis.ScardCtx(ctx, topic); err != nil {
		err = errors.New(fmt.Sprintf("get %s count fail, %s", topic, err.Error()))
		return 0, err
	}
	return count, nil
}

func GetAllowNodesWithLimit(ctx context.Context, bizRedis *localstorage.LocalRedis, topic string, limit int) (nodes []string, err error) {

	if bizRedis == nil {
		return nil, errors.New("bizRedis is nil")
	}

	if topic == "" {
		return nil, errors.New("topic is empty")
	}

	if limit == 0 {
		return nil, errors.New("limit is zero")
	}

	// 通过scan的方式获取灰度节点, 避免一次性获取太多节点导致内存占用过高
	var cursor uint64
	count := int64(200)
	nodeIdSet := collection.NewSet()
	for {
		// count表示扫描的数量
		keys, newCursor, err := bizRedis.SscanCtx(ctx, topic, cursor, "", count)
		if err != nil {
			err = errors.New(fmt.Sprintf("get %s nodes fail, %s", topic, err.Error()))
			return nil, err
		}

		for _, key := range keys {
			if nodeIdSet.Contains(key) {
				continue
			}

			nodeIdSet.Add(key)
			nodes = append(nodes, key)
			if len(nodes) >= limit {
				return nodes, nil
			}
		}

		// 如果cursor为0，说明扫描完成
		if newCursor == 0 {
			break
		}
		cursor = newCursor
	}

	return nodes, nil
}

func ClearAllowNodes(ctx context.Context, bizRedis *localstorage.LocalRedis, topic string) (err error) {

	if bizRedis == nil {
		return errors.New("bizRedis is nil")
	}

	if topic == "" {
		return errors.New("topic is empty")
	}

	if _, err = bizRedis.DelCtx(ctx, topic); err != nil {
		err = errors.New(fmt.Sprintf("clear %s fail: %s", topic, err.Error()))
		return err
	}
	return nil
}

func CheckNodeIdIsAllowed(ctx context.Context, bizRedis *localstorage.LocalRedis, topic string, nodeId string) (exist bool, err error) {
	if bizRedis == nil {
		return false, errors.New("bizRedis is nil")
	}
	if nodeId == "" {
		return false, errors.New("nodeId is empty")
	}

	return bizRedis.SismemberCtx(ctx, topic, nodeId)
}

func SaveAllowNodes(ctx context.Context, grayNodesModel sharedmodel.GrayNodesModel, bizRedis *localstorage.LocalRedis, topic string, releaseID string) (err error) {
	if bizRedis == nil {
		return errors.New("bizRedis is nil")
	}

	if topic == "" {
		return errors.New("topic is empty")
	}

	if releaseID == "" {
		return errors.New("releaseId is empty")
	}

	getallnodes, err := GetAllowNodes(ctx, bizRedis, topic)
	if err != nil {
		return err
	}

	// 一次保存100个节点
	for _, nodes := range utils.GroupAny(getallnodes, 100) {
		var allNodes []*sharedmodel.GrayNode
		for _, node := range nodes {
			if node == "" {
				continue
			}
			allNodes = append(allNodes, &sharedmodel.GrayNode{
				ReleaseID: releaseID,
				NodeId:    node,
				CreateAt:  time.Now(),
			})
		}

		if err = grayNodesModel.UpsertBulk(ctx, allNodes); err != nil {
			return err
		}
	}

	return nil
}

func EnsureNodesNotInOtherTasks(ctx context.Context, cache AllowNodesTopicCache, allowAppsModel sharedmodel.AllowAppsModel, bizRedis *localstorage.LocalRedis, nodeIdPoolCache map[string]map[string]struct{}, release *sharedmodel.NodeRelease, processingTasks []*sharedmodel.NodeRelease, nodeIds []string) (otherTaskTips []string, curUsed bool, err error) {
	if bizRedis == nil || nodeIdPoolCache == nil {
		return nil, false, errorx.NewDefaultError("redis or cache invalid")
	}

	if release == nil {
		return nil, false, errorx.NewDefaultError("release无效")
	}

	// 检查是否已在默认发布中使用
	releaseID := release.ID
	app := release.App
	nodeType := sharedmodel.NodeTypeNode
	if device.DevType(release.DeviceType).IsSmallBoxDev() {
		nodeType = sharedmodel.NodeTypeSmallBox
	}

	var topic string
	topic, err = GetAllowNodesTopic(ctx, cache, allowAppsModel, app, nodeType.String(), "")
	for _, nodeId := range nodeIds {
		used, err := CheckNodeIdIsAllowed(ctx, bizRedis, topic, nodeId)
		if err != nil {
			return otherTaskTips, false, err
		}
		if used {
			otherTaskTips = append(otherTaskTips, fmt.Sprintf("节点 %s 已被任务 工具发布任务 占用", nodeId))
			continue
		}
	}

	for _, grayTask := range processingTasks {
		// 当前任务占用, 忽略
		currentTask := grayTask.ID == releaseID
		if grayTask.GrayPolicy.Filter != nil {

			topic, err = GetAllowNodesTopic(ctx, cache, allowAppsModel, grayTask.App, nodeType.String(), grayTask.ID)
			if err != nil {
				return otherTaskTips, false, err
			}
		}

		for _, nodeId := range nodeIds {
			if grayTask.GrayPolicy.Filter != nil {
				used, err := CheckNodeIdIsAllowed(ctx, bizRedis, topic, nodeId)
				if err != nil {
					return otherTaskTips, false, err
				}
				if used {
					if currentTask {
						curUsed = true
					} else {
						otherTaskTips = append(otherTaskTips, fmt.Sprintf("节点 %s 已被任务 %s 占用", nodeId, grayTask.ID))
					}
					continue
				}

			} else {
				// 获取当前任务的灰度节点池Map
				if _, ok := nodeIdPoolCache[grayTask.ID]; !ok {
					nodeIdPoolCache[grayTask.ID] = make(map[string]struct{})
					for _, nodeId := range grayTask.GrayPolicy.NodeIds {
						nodeIdPoolCache[grayTask.ID][nodeId] = struct{}{}
					}
				}

				if _, ok := nodeIdPoolCache[grayTask.ID][nodeId]; ok {
					if currentTask {
						curUsed = true
					} else {
						otherTaskTips = append(otherTaskTips, fmt.Sprintf("节点 %s 已被任务 %s 占用", nodeId, grayTask.ID))
					}
					continue
				}
			}
		}
	}

	if len(otherTaskTips) > 0 {
		return otherTaskTips, curUsed, errorx.NewDefaultError("存在节点已被其它任务占用, 请检查")
	}
	return otherTaskTips, curUsed, nil
}
