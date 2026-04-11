package mongo

import (
	"context"
	"slices"

	"github.com/reearth/reearth/server/internal/collab"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CollabChatHistory stores collab room chat lines in a dedicated collection (_id = server-assigned message id).
type CollabChatHistory struct {
	coll *mongo.Collection
}

func NewCollabChatHistory(coll *mongo.Collection) *CollabChatHistory {
	if coll == nil {
		return nil
	}
	return &CollabChatHistory{coll: coll}
}

// EnsureIndexes creates a compound index for ListRecent (projectId + ts desc).
func (s *CollabChatHistory) EnsureIndexes(ctx context.Context) error {
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

func (s *CollabChatHistory) Append(ctx context.Context, projectID, userID, text string, tsUnix int64, messageID string) error {
	_, err := s.coll.InsertOne(ctx, bson.M{
		"_id":       messageID,
		"projectId": projectID,
		"userId":    userID,
		"text":      text,
		"ts":        tsUnix,
	})
	return err
}

func (s *CollabChatHistory) ListRecent(ctx context.Context, projectID string, limit int) ([]collab.ChatMessageRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "ts", Value: -1}}).
		SetLimit(int64(limit))

	cur, err := s.coll.Find(ctx, bson.M{"projectId": projectID}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var docs []bson.M
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}

	out := make([]collab.ChatMessageRecord, 0, len(docs))
	for _, d := range docs {
		out = append(out, docToChatRecord(d))
	}
	slices.Reverse(out)
	return out, nil
}

func docToChatRecord(d bson.M) collab.ChatMessageRecord {
	var id string
	switch v := d["_id"].(type) {
	case string:
		id = v
	case primitive.ObjectID:
		id = v.Hex()
	default:
		id = ""
	}
	uid, _ := d["userId"].(string)
	txt, _ := d["text"].(string)
	ts := int64(0)
	switch v := d["ts"].(type) {
	case int32:
		ts = int64(v)
	case int64:
		ts = v
	case float64:
		ts = int64(v)
	}
	return collab.ChatMessageRecord{ID: id, UserID: uid, Text: txt, Ts: ts}
}
