package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"hackathon/common/device"
	"hackathon/sharedmodel"
)

const (
	mongoURL = "mongodb://localhost:27017"
	dbName   = "jarvis"
)

var (
	// 设备类型
	deviceTypes = []device.DevType{
		device.DevAntVerA,    // ant.A
		device.DevAntVerB,    // ant.B
		device.DevAntVerC,    // ant.C
		device.DevJarvisVerA, // jarvis.A
	}

	// 节点阶段
	stages = []string{
		sharedmodel.NodeStageRegister,
		sharedmodel.NodeStageSubmitted,
		sharedmodel.NodeStageCensored,
		sharedmodel.NodeStageUncensored,
		sharedmodel.NodeStageInservice,
	}

	// 业务ID
	customerIDs = []uint32{
		10001, // 业务A
		10002, // 业务B
		10003, // 业务C
		10004, // 业务D
	}
)

func main() {
	rand.Seed(time.Now().UnixNano())

	log.Println("开始连接MongoDB...")
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURL))
	if err != nil {
		log.Fatalf("连接MongoDB失败: %v", err)
	}
	defer client.Disconnect(context.Background())

	collection := client.Database(dbName).Collection(sharedmodel.CollectionNodeJoin)

	log.Println("开始生成数据...")

	var nodes []interface{}
	nodeCount := 0

	// 1. 为每个设备类型、每个阶段生成至少100条数据
	log.Println("生成阶段数据...")
	for _, devType := range deviceTypes {
		for _, stage := range stages {
			count := 100 + rand.Intn(20) // 100-120条
			for i := 0; i < count; i++ {
				node := generateNode(devType, stage, false)
				nodes = append(nodes, node)
				nodeCount++
			}
		}
	}

	// 2. 为每个设备类型、每个业务生成至少100条inservice数据（包含customerIDs）
	log.Println("生成业务数据...")
	for _, devType := range deviceTypes {
		for _, customerID := range customerIDs {
			count := 100 + rand.Intn(20) // 100-120条
			for i := 0; i < count; i++ {
				node := generateNodeWithCustomer(devType, customerID)
				nodes = append(nodes, node)
				nodeCount++
			}
		}
	}

	log.Printf("共生成 %d 条数据，开始插入数据库...", nodeCount)

	// 批量插入
	batchSize := 1000
	for i := 0; i < len(nodes); i += batchSize {
		end := i + batchSize
		if end > len(nodes) {
			end = len(nodes)
		}

		batch := nodes[i:end]
		_, err := collection.InsertMany(context.Background(), batch)
		if err != nil {
			log.Printf("插入数据失败: %v", err)
			continue
		}
		log.Printf("已插入 %d/%d 条数据", end, len(nodes))
	}

	log.Println("数据生成完成！")

	// 统计信息
	printStatistics(collection)
}

// generateNode 生成普通节点数据
func generateNode(devType device.DevType, stage string, withCustomer bool) *sharedmodel.NodeJoin {
	nodeType, _ := devType.NodeType()

	node := &sharedmodel.NodeJoin{
		Id:          uuid.NewString(),
		NodeId:      uuid.NewString(),
		DeviceType:  devType,
		Stage:       stage,
		NodeType:    sharedmodel.NodeType(nodeType),
		Status:      sharedmodel.DNodeStatusOnline,
		CustomerIDs: []uint32{},
	}

	return node
}

// generateNodeWithCustomer 生成带业务ID的节点数据（stage必须是inservice）
func generateNodeWithCustomer(devType device.DevType, customerID uint32) *sharedmodel.NodeJoin {
	nodeType, _ := devType.NodeType()

	node := &sharedmodel.NodeJoin{
		Id:          uuid.NewString(),
		NodeId:      uuid.NewString(),
		DeviceType:  devType,
		Stage:       sharedmodel.NodeStageInservice, // 有customerID时，stage必须是inservice
		NodeType:    sharedmodel.NodeType(nodeType),
		Status:      sharedmodel.DNodeStatusOnline,
		CustomerIDs: []uint32{customerID},
	}

	return node
}

// printStatistics 打印统计信息
func printStatistics(collection *mongo.Collection) {
	ctx := context.Background()

	log.Println("\n=== 数据统计 ===")

	// 总数
	total, _ := collection.CountDocuments(ctx, bson.M{})
	log.Printf("总记录数: %d", total)

	// 按设备类型统计
	log.Println("\n按设备类型统计:")
	for _, devType := range deviceTypes {
		count, _ := collection.CountDocuments(ctx, bson.M{"deviceType": devType})
		log.Printf("  %s: %d 条", devType, count)
	}

	// 按阶段统计
	log.Println("\n按阶段统计:")
	for _, stage := range stages {
		count, _ := collection.CountDocuments(ctx, bson.M{"stage": stage})
		log.Printf("  %s: %d 条", stage, count)
	}

	// 按业务统计
	log.Println("\n按业务统计:")
	for _, customerID := range customerIDs {
		count, _ := collection.CountDocuments(ctx, bson.M{"customerIDs": customerID})
		log.Printf("  业务%d: %d 条", customerID, count)
	}

	// 按设备类型和业务统计
	log.Println("\n按设备类型和业务统计:")
	for _, devType := range deviceTypes {
		log.Printf("  %s:", devType)
		for _, customerID := range customerIDs {
			count, _ := collection.CountDocuments(ctx, bson.M{
				"deviceType":  devType,
				"customerIDs": customerID,
			})
			log.Printf("    业务%d: %d 条", customerID, count)
		}
	}
}
