package mongo

import (
	"context"
	"time"

	"github.com/reearth/reearth/server/internal/collab"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	doc := bson.M{
		"_id":       primitive.NewObjectID(),
		"projectId": rec.ProjectID,
		"userId":    rec.UserID,
		"kind":      rec.Kind,
		"sceneRev":  rec.SceneRev,
		"sceneId":   rec.SceneID,
		"widgetId":  rec.WidgetID,
		"ts":        time.Now().UnixMilli(),
	}
	if rec.StoryID != "" {
		doc["storyId"] = rec.StoryID
	}
	if rec.PageID != "" {
		doc["pageId"] = rec.PageID
	}
	if rec.BlockID != "" {
		doc["blockId"] = rec.BlockID
	}
	if rec.PropertyID != "" {
		doc["propertyId"] = rec.PropertyID
	}
	if rec.FieldID != "" {
		doc["fieldId"] = rec.FieldID
	}
	if rec.StyleID != "" {
		doc["styleId"] = rec.StyleID
	}
	_, err := s.coll.InsertOne(ctx, doc)
	return err
}

func (s *CollabApplyAudit) ListRecent(ctx context.Context, projectID string, limit int) ([]collab.ApplyAuditListRow, error) {
	if s == nil || s.coll == nil {
		return []collab.ApplyAuditListRow{}, nil
	}
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
	out := make([]collab.ApplyAuditListRow, 0, len(docs))
	for _, d := range docs {
		out = append(out, docToApplyAuditRow(d))
	}
	return out, nil
}

func docToApplyAuditRow(d bson.M) collab.ApplyAuditListRow {
	var id string
	switch v := d["_id"].(type) {
	case primitive.ObjectID:
		id = v.Hex()
	case string:
		id = v
	default:
		id = ""
	}
	uid, _ := d["userId"].(string)
	kind, _ := d["kind"].(string)
	sceneID, _ := d["sceneId"].(string)
	widgetID, _ := d["widgetId"].(string)
	storyID, _ := d["storyId"].(string)
	pageID, _ := d["pageId"].(string)
	blockID, _ := d["blockId"].(string)
	propID, _ := d["propertyId"].(string)
	fieldID, _ := d["fieldId"].(string)
	styleID, _ := d["styleId"].(string)
	sceneRev := int64(0)
	switch v := d["sceneRev"].(type) {
	case int32:
		sceneRev = int64(v)
	case int64:
		sceneRev = v
	case float64:
		sceneRev = int64(v)
	}
	ts := int64(0)
	switch v := d["ts"].(type) {
	case int32:
		ts = int64(v)
	case int64:
		ts = v
	case float64:
		ts = int64(v)
	}
	return collab.ApplyAuditListRow{
		ID: id, UserID: uid, Kind: kind, SceneRev: sceneRev,
		SceneID: sceneID, WidgetID: widgetID, StoryID: storyID, PageID: pageID, BlockID: blockID,
		PropertyID: propID, FieldID: fieldID, StyleID: styleID, Ts: ts,
	}
}
