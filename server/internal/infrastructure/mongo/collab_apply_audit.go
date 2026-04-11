package mongo

import (
	"context"
	"time"

	"github.com/reearth/reearth/server/internal/collab"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// CollabApplyAudit stores successful collab apply events (append-only).
type CollabApplyAudit struct {
	coll *mongo.Collection
}

func NewCollabApplyAudit(coll *mongo.Collection) *CollabApplyAudit {
	if coll == nil {
		return nil
	}
	return &CollabApplyAudit{coll: coll}
}

func (s *CollabApplyAudit) EnsureIndexes(ctx context.Context) error {
	if s == nil || s.coll == nil {
		return nil
	}
	_, err := s.coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "projectId", Value: 1},
			{Key: "ts", Value: -1},
		},
	})
	return err
}

func (s *CollabApplyAudit) Append(ctx context.Context, rec collab.ApplyAuditRecord) error {
	if s == nil || s.coll == nil {
		return nil
	}
	_, err := s.coll.InsertOne(ctx, bson.M{
		"_id":       primitive.NewObjectID(),
		"projectId": rec.ProjectID,
		"userId":    rec.UserID,
		"kind":      rec.Kind,
		"sceneRev":  rec.SceneRev,
		"sceneId":   rec.SceneID,
		"widgetId":  rec.WidgetID,
		"ts":        time.Now().UnixMilli(),
	})
	return err
}
