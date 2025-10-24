package sharedmodel

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	CollectionDeployConfig = "deployConfig"
)

type ExecutorType string

const (
	ExecutorTypeK8S      ExecutorType = "k8s"
	ExecutorTypePhysical ExecutorType = "physical"
)

type DeployConfig struct {
	ID           string       `bson:"_id" json:"id"`
	DeviceType   string       `bson:"deviceType" json:"deviceType"`
	ExecutorType ExecutorType `bson:"executorType" json:"executorType"`
	K8SConfig    *K8SConfig   `bson:"k8sConfig,omitempty" json:"k8sConfig,omitempty"`
	PhysicalCfg  *PhysicalCfg `bson:"physicalConfig,omitempty" json:"physicalConfig,omitempty"`
	AutoRollback bool         `bson:"autoRollback" json:"autoRollback"`
	HealthCheck  *HealthCfg   `bson:"healthCheck,omitempty" json:"healthCheck,omitempty"`
	CreateAt     time.Time    `bson:"createAt" json:"createAt"`
	UpdateAt     time.Time    `bson:"updateAt" json:"updateAt"`
}

type K8SConfig struct {
	KubeConfigPath string `bson:"kubeConfigPath" json:"kubeConfigPath"`
	Namespace      string `bson:"namespace" json:"namespace"`
	TimeoutSeconds int    `bson:"timeoutSeconds" json:"timeoutSeconds"`
}

type PhysicalCfg struct {
	SSHUser        string `bson:"sshUser" json:"sshUser"`
	SSHKeyPath     string `bson:"sshKeyPath" json:"sshKeyPath"`
	TimeoutSeconds int    `bson:"timeoutSeconds" json:"timeoutSeconds"`
}

type HealthCfg struct {
	CheckInterval   int     `bson:"checkInterval" json:"checkInterval"`
	HealthThreshold float64 `bson:"healthThreshold" json:"healthThreshold"`
	MaxRetries      int     `bson:"maxRetries" json:"maxRetries"`
}

type DeployConfigModel interface {
	Insert(ctx context.Context, data *DeployConfig) error
	Find(ctx context.Context, id string) (*DeployConfig, error)
	FindByDeviceType(ctx context.Context, deviceType string) (*DeployConfig, error)
	Update(ctx context.Context, data *DeployConfig) error
	Delete(ctx context.Context, id string) error
}

type defaultDeployConfigModel struct {
	model *mon.Model
}

func NewDeployConfigModel(url, db string) DeployConfigModel {
	return &defaultDeployConfigModel{
		model: mon.MustNewModel(url, db, CollectionDeployConfig),
	}
}

func (m *defaultDeployConfigModel) Insert(ctx context.Context, data *DeployConfig) error {
	if data.ID == "" {
		data.ID = primitive.NewObjectID().Hex()
	}
	if data.CreateAt.IsZero() {
		data.CreateAt = time.Now()
	}
	if data.UpdateAt.IsZero() {
		data.UpdateAt = time.Now()
	}

	_, err := m.model.InsertOne(ctx, data)
	return err
}

func (m *defaultDeployConfigModel) Find(ctx context.Context, id string) (*DeployConfig, error) {
	var data DeployConfig
	err := m.model.FindOne(ctx, &data, bson.M{"_id": id})
	switch err {
	case nil:
		return &data, nil
	case mongo.ErrNoDocuments:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *defaultDeployConfigModel) FindByDeviceType(ctx context.Context, deviceType string) (*DeployConfig, error) {
	var data DeployConfig
	err := m.model.FindOne(ctx, &data, bson.M{"deviceType": deviceType})
	switch err {
	case nil:
		return &data, nil
	case mongo.ErrNoDocuments:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *defaultDeployConfigModel) Update(ctx context.Context, data *DeployConfig) error {
	data.UpdateAt = time.Now()
	_, err := m.model.UpdateOne(ctx, bson.M{"_id": data.ID}, bson.M{"$set": data})
	return err
}

func (m *defaultDeployConfigModel) Delete(ctx context.Context, id string) error {
	_, err := m.model.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
