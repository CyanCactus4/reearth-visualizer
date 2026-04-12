package collab

import (
	"context"
	"encoding/json"
	"time"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearthx/log"
)

type applyCreateNLSPhotoOverlay struct {
	Kind         string `json:"kind"`
	SceneID      string `json:"sceneId"`
	LayerID      string `json:"layerId"`
	BaseSceneRev *int64 `json:"baseSceneRev,omitempty"`
}

type applyRemoveNLSPhotoOverlay struct {
	Kind         string `json:"kind"`
	SceneID      string `json:"sceneId"`
	LayerID      string `json:"layerId"`
	BaseSceneRev *int64 `json:"baseSceneRev,omitempty"`
}

func applyCreateNLSPhotoOverlayOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyCreateNLSPhotoOverlay
	if err := json.Unmarshal(d, &p); err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	op := from.operator
	if op == nil || !op.IsWritableScene(from.sceneID) {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "forbidden", "message": "write not allowed"}})
		return nil
	}
	sid, err := id.SceneIDFrom(p.SceneID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_scene", "message": err.Error()}})
		return nil
	}
	if sid != from.sceneID {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "scene_mismatch", "message": "scene does not belong to this room"}})
		return nil
	}
	uc := adapter.Usecases(ctx)
	if uc == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": "usecases unavailable"}})
		return nil
	}
	if !assertSceneRevIfPresent(ctx, uc, op, sid, from, d) {
		return nil
	}
	lid, err := id.NLSLayerIDFrom(p.LayerID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_layer", "message": err.Error()}})
		return nil
	}
	if !nlsLayerBelongsToScene(ctx, uc, op, sid, lid, from) {
		return nil
	}
	if !nlsLayerMustNotBeLockedByPeer(ctx, hub, from, lid) {
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, err2 := uc.NLSLayer.CreateNLSPhotoOverlay(opCtx, lid, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "create_nls_photo_overlay", map[string]any{
		"sceneId": p.SceneID,
		"layerId": p.LayerID,
	}, sc)
	if hub != nil && hub.opStack != nil {
		inv := applyRemoveNLSPhotoOverlay{
			Kind:    "remove_nls_photo_overlay",
			SceneID: p.SceneID,
			LayerID: p.LayerID,
		}
		if invJSON, errM := json.Marshal(inv); errM == nil {
			rec := UndoableOpRecord{
				ProjectID: from.projectID,
				SceneID:   sid.String(),
				UserID:    actorUserID(from),
				Kind:      "create_nls_photo_overlay",
				Forward:   d,
				Inverse:   invJSON,
			}
			go func() {
				pctx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel2()
				if err := hub.opStack.RecordUndoable(pctx, rec); err != nil {
					log.Warnfc(pctx, "collab: undo stack: %v", err)
				}
			}()
		}
	}
	return nil
}

func applyRemoveNLSPhotoOverlayOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyRemoveNLSPhotoOverlay
	if err := json.Unmarshal(d, &p); err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	op := from.operator
	if op == nil || !op.IsWritableScene(from.sceneID) {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "forbidden", "message": "write not allowed"}})
		return nil
	}
	sid, err := id.SceneIDFrom(p.SceneID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_scene", "message": err.Error()}})
		return nil
	}
	if sid != from.sceneID {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "scene_mismatch", "message": "scene does not belong to this room"}})
		return nil
	}
	uc := adapter.Usecases(ctx)
	if uc == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": "usecases unavailable"}})
		return nil
	}
	if !assertSceneRevIfPresent(ctx, uc, op, sid, from, d) {
		return nil
	}
	lid, err := id.NLSLayerIDFrom(p.LayerID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_layer", "message": err.Error()}})
		return nil
	}
	if !nlsLayerBelongsToScene(ctx, uc, op, sid, lid, from) {
		return nil
	}
	if !nlsLayerMustNotBeLockedByPeer(ctx, hub, from, lid) {
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, err2 := uc.NLSLayer.RemoveNLSPhotoOverlay(opCtx, lid, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "remove_nls_photo_overlay", map[string]any{
		"sceneId": p.SceneID,
		"layerId": p.LayerID,
	}, sc)
	if hub != nil && hub.opStack != nil {
		inv := applyCreateNLSPhotoOverlay{
			Kind:    "create_nls_photo_overlay",
			SceneID: p.SceneID,
			LayerID: p.LayerID,
		}
		if invJSON, errM := json.Marshal(inv); errM == nil {
			rec := UndoableOpRecord{
				ProjectID: from.projectID,
				SceneID:   sid.String(),
				UserID:    actorUserID(from),
				Kind:      "remove_nls_photo_overlay",
				Forward:   d,
				Inverse:   invJSON,
			}
			go func() {
				pctx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel2()
				if err := hub.opStack.RecordUndoable(pctx, rec); err != nil {
					log.Warnfc(pctx, "collab: undo stack: %v", err)
				}
			}()
		}
	}
	return nil
}
