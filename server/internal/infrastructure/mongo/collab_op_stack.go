package mongo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/reearth/reearth/server/internal/collab"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CollabOpStackStore implements collab.CollabOpStack with two collections: ops + per-(user,scene) stacks.
type CollabOpStackStore struct {
	ops   *mongo.Collection
	state *mongo.Collection
}

func NewCollabOpStack(ops, state *mongo.Collection) *CollabOpStackStore {
	if ops == nil || state == nil {
		return nil
	}
	return &CollabOpStackStore{ops: ops, state: state}
}

func (s *CollabOpStackStore) EnsureIndexes(ctx context.Context) error {
	if s == nil {
		return nil
	}
	_, err := s.ops.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "sceneId", Value: 1}, {Key: "userId", Value: 1}, {Key: "ts", Value: -1}},
	})
	if err != nil {
		return err
	}
	_, err = s.state.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "sceneId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

func stackKey(userID, sceneID string) string {
	return userID + "\x00" + sceneID
}

func (s *CollabOpStackStore) RecordUndoable(ctx context.Context, rec collab.UndoableOpRecord) error {
	if s == nil || rec.UserID == "" || rec.SceneID == "" {
		return nil
	}
	id := primitive.NewObjectID()
	doc := bson.M{
		"_id":         id,
		"projectId":   rec.ProjectID,
		"sceneId":     rec.SceneID,
		"userId":      rec.UserID,
		"kind":        rec.Kind,
		"forwardJson": string(rec.Forward),
		"inverseJson": string(rec.Inverse),
		"ts":          time.Now().UnixMilli(),
	}
	if _, err := s.ops.InsertOne(ctx, doc); err != nil {
		return err
	}
	_, err := s.state.UpdateOne(ctx,
		bson.M{"userId": rec.UserID, "sceneId": rec.SceneID},
		bson.M{
			"$setOnInsert": bson.M{"projectId": rec.ProjectID},
			"$push":        bson.M{"undo": id},
			"$set":         bson.M{"redo": []primitive.ObjectID{}},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

func (s *CollabOpStackStore) Undo(ctx context.Context, userID, sceneID string) (*collab.UndoableOpRecord, error) {
	if s == nil {
		return nil, errors.New("nil stack")
	}
	var st struct {
		Undo []primitive.ObjectID `bson:"undo"`
		Redo []primitive.ObjectID `bson:"redo"`
	}
	err := s.state.FindOne(ctx, bson.M{"userId": userID, "sceneId": sceneID}).Decode(&st)
	if err != nil {
		return nil, err
	}
	if len(st.Undo) == 0 {
		return nil, errors.New("nothing to undo")
	}
	last := st.Undo[len(st.Undo)-1]
	newUndo := st.Undo[:len(st.Undo)-1]
	newRedo := append(st.Redo, last)
	_, err = s.state.UpdateOne(ctx, bson.M{"userId": userID, "sceneId": sceneID}, bson.M{"$set": bson.M{"undo": newUndo, "redo": newRedo}})
	if err != nil {
		return nil, err
	}
	return s.loadOp(ctx, last)
}

func (s *CollabOpStackStore) Redo(ctx context.Context, userID, sceneID string) (*collab.UndoableOpRecord, error) {
	if s == nil {
		return nil, errors.New("nil stack")
	}
	var st struct {
		Undo []primitive.ObjectID `bson:"undo"`
		Redo []primitive.ObjectID `bson:"redo"`
	}
	err := s.state.FindOne(ctx, bson.M{"userId": userID, "sceneId": sceneID}).Decode(&st)
	if err != nil {
		return nil, err
	}
	if len(st.Redo) == 0 {
		return nil, errors.New("nothing to redo")
	}
	last := st.Redo[len(st.Redo)-1]
	newRedo := st.Redo[:len(st.Redo)-1]
	newUndo := append(st.Undo, last)
	_, err = s.state.UpdateOne(ctx, bson.M{"userId": userID, "sceneId": sceneID}, bson.M{"$set": bson.M{"undo": newUndo, "redo": newRedo}})
	if err != nil {
		return nil, err
	}
	return s.loadOp(ctx, last)
}

func (s *CollabOpStackStore) loadOp(ctx context.Context, id primitive.ObjectID) (*collab.UndoableOpRecord, error) {
	var doc struct {
		ProjectID   string `bson:"projectId"`
		SceneID     string `bson:"sceneId"`
		UserID      string `bson:"userId"`
		Kind        string `bson:"kind"`
		ForwardJSON string `bson:"forwardJson"`
		InverseJSON string `bson:"inverseJson"`
	}
	if err := s.ops.FindOne(ctx, bson.M{"_id": id}).Decode(&doc); err != nil {
		return nil, err
	}
	return &collab.UndoableOpRecord{
		ProjectID: doc.ProjectID,
		SceneID:   doc.SceneID,
		UserID:    doc.UserID,
		Kind:      doc.Kind,
		Forward:   json.RawMessage(doc.ForwardJSON),
		Inverse:   json.RawMessage(doc.InverseJSON),
	}, nil
}

// PatchHeadRedoForward updates forwardJson for the op at the tail of the redo stack.
func (s *CollabOpStackStore) PatchHeadRedoForward(ctx context.Context, userID, sceneID string, forward json.RawMessage) error {
	if s == nil {
		return nil
	}
	var st struct {
		Redo []primitive.ObjectID `bson:"redo"`
	}
	if err := s.state.FindOne(ctx, bson.M{"userId": userID, "sceneId": sceneID}).Decode(&st); err != nil {
		return err
	}
	if len(st.Redo) == 0 {
		return nil
	}
	last := st.Redo[len(st.Redo)-1]
	_, err := s.ops.UpdateOne(ctx, bson.M{"_id": last}, bson.M{"$set": bson.M{"forwardJson": string(forward)}})
	return err
}
