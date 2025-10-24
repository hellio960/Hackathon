package sharedmodel

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	CollectionHealthCheck = "healthCheck"
)

type HealthCheckRecord struct {
	ID             string    `bson:"_id" json:"id"`
	ReleaseID      string    `bson:"releaseId" json:"releaseId"`
	AppName        string    `bson:"appName" json:"appName"`
	DeviceType     string    `bson:"deviceType" json:"deviceType"`
	HealthyNodes   []string  `bson:"healthyNodes" json:"healthyNodes"`
	UnhealthyNodes []string  `bson:"unhealthyNodes" json:"unhealthyNodes"`
	TotalNodes     int       `bson:"totalNodes" json:"totalNodes"`
	HealthyCount   int       `bson:"healthyCount" json:"healthyCount"`
	UnhealthyCount int       `bson:"unhealthyCount" json:"unhealthyCount"`
	HealthRate     float64   `bson:"healthRate" json:"healthRate"`
	CheckTime      time.Time `bson:"checkTime" json:"checkTime"`
	CreateAt       time.Time `bson:"createAt" json:"createAt"`
}

type HealthCheckModel interface {
	Insert(ctx context.Context, data *HealthCheckRecord) error
	Find(ctx context.Context, id string) (*HealthCheckRecord, error)
	FindByReleaseID(ctx context.Context, releaseID string, limit int) ([]*HealthCheckRecord, error)
	Delete(ctx context.Context, id string) error
}

type defaultHealthCheckModel struct {
	model *mon.Model
}

func NewHealthCheckModel(url, db string) HealthCheckModel {
	return &defaultHealthCheckModel{
		model: mon.MustNewModel(url, db, CollectionHealthCheck),
	}
}

func (m *defaultHealthCheckModel) Insert(ctx context.Context, data *HealthCheckRecord) error {
	if data.ID == "" {
		data.ID = primitive.NewObjectID().Hex()
	}
	if data.CreateAt.IsZero() {
		data.CreateAt = time.Now()
	}

	_, err := m.model.InsertOne(ctx, data)
	return err
}

func (m *defaultHealthCheckModel) Find(ctx context.Context, id string) (*HealthCheckRecord, error) {
	var data HealthCheckRecord
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

func (m *defaultHealthCheckModel) FindByReleaseID(ctx context.Context, releaseID string, limit int) ([]*HealthCheckRecord, error) {
	if limit <= 0 {
		limit = 100
	}

	option := options.Find()
	option.SetSort(bson.M{"checkTime": -1})
	option.SetLimit(int64(limit))

	var records []*HealthCheckRecord
	err := m.model.Find(ctx, &records, bson.M{"releaseId": releaseID}, option)
	return records, err
}

func (m *defaultHealthCheckModel) Delete(ctx context.Context, id string) error {
	_, err := m.model.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
