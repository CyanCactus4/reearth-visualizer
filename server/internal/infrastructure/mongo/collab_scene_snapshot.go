package mongo

import (
	"context"
	"errors"
	"time"

	"github.com/reearth/reearth/server/internal/collab"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CollabSceneSnapshotStore persists collab scene export blobs for maintainer restore.
type CollabSceneSnapshotStore struct {
	coll *mongo.Collection
}

func NewCollabSceneSnapshotStore(coll *mongo.Collection) *CollabSceneSnapshotStore {
	if coll == nil {
		return nil
	}
	return &CollabSceneSnapshotStore{coll: coll}
}

func (s *CollabSceneSnapshotStore) EnsureIndexes(ctx context.Context) error {
	if s == nil {
		return nil
	}
	_, err := s.coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "sceneId", Value: 1}, {Key: "sceneRev", Value: -1}},
	})
	return err
}

func (s *CollabSceneSnapshotStore) Append(ctx context.Context, rec collab.SceneSnapshotRecord) error {
	if s == nil || rec.SceneID == "" || rec.SceneRev <= 0 || len(rec.Data) == 0 {
		return nil
	}
	doc := bson.M{
		"projectId": rec.ProjectID,
		"sceneId":   rec.SceneID,
		"sceneRev":  rec.SceneRev,
		"data":      rec.Data,
		"ts":        rec.Ts,
	}
	if doc["ts"] == int64(0) {
		doc["ts"] = time.Now().UnixMilli()
	}
	_, err := s.coll.InsertOne(ctx, doc)
	return err
}

func (s *CollabSceneSnapshotStore) LoadClosestAtOrBelow(ctx context.Context, sceneID string, targetRev int64) ([]byte, int64, error) {
	if s == nil {
		return nil, 0, errors.New("nil snapshot store")
	}
	var doc struct {
		SceneRev int64  `bson:"sceneRev"`
		Data     []byte `bson:"data"`
	}
	err := s.coll.FindOne(ctx, bson.M{
		"sceneId":  sceneID,
		"sceneRev": bson.M{"$lte": targetRev},
	}, options.FindOne().SetSort(bson.D{{Key: "sceneRev", Value: -1}})).Decode(&doc)
	if err != nil {
		return nil, 0, err
	}
	return doc.Data, doc.SceneRev, nil
}
