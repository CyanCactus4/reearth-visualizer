package collab

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/nlslayer"
	"github.com/reearth/reearthx/log"
)

type applyCreateNLSInfobox struct {
	Kind         string `json:"kind"`
	SceneID      string `json:"sceneId"`
	LayerID      string `json:"layerId"`
	BaseSceneRev *int64 `json:"baseSceneRev,omitempty"`
}

func applyCreateNLSInfoboxOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyCreateNLSInfobox
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
	_, err2 := uc.NLSLayer.CreateNLSInfobox(opCtx, lid, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "create_nls_infobox", map[string]any{
		"sceneId": p.SceneID,
		"layerId": p.LayerID,
	}, sc)
	if hub != nil && hub.opStack != nil {
		inv := applyRemoveNLSInfobox{
			Kind:    "remove_nls_infobox",
			SceneID: p.SceneID,
			LayerID: p.LayerID,
		}
		if invJSON, errM := json.Marshal(inv); errM == nil {
			rec := UndoableOpRecord{
				ProjectID: from.projectID,
				SceneID:   sid.String(),
				UserID:    actorUserID(from),
				Kind:      "create_nls_infobox",
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

type applyRemoveNLSInfobox struct {
	Kind         string `json:"kind"`
	SceneID      string `json:"sceneId"`
	LayerID      string `json:"layerId"`
	BaseSceneRev *int64 `json:"baseSceneRev,omitempty"`
}

func applyRemoveNLSInfoboxOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyRemoveNLSInfobox
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
	_, err2 := uc.NLSLayer.RemoveNLSInfobox(opCtx, lid, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "remove_nls_infobox", map[string]any{
		"sceneId": p.SceneID,
		"layerId": p.LayerID,
	}, sc)
	if hub != nil && hub.opStack != nil {
		inv := applyCreateNLSInfobox{
			Kind:    "create_nls_infobox",
			SceneID: p.SceneID,
			LayerID: p.LayerID,
		}
		if invJSON, errM := json.Marshal(inv); errM == nil {
			rec := UndoableOpRecord{
				ProjectID: from.projectID,
				SceneID:   sid.String(),
				UserID:    actorUserID(from),
				Kind:      "remove_nls_infobox",
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

type applyAddNLSInfoboxBlock struct {
	Kind         string `json:"kind"`
	SceneID      string `json:"sceneId"`
	LayerID      string `json:"layerId"`
	PluginID     string `json:"pluginId"`
	ExtensionID  string `json:"extensionId"`
	Index        *int   `json:"index,omitempty"`
	BaseSceneRev *int64 `json:"baseSceneRev,omitempty"`
}

type applyMoveNLSInfoboxBlock struct {
	Kind           string `json:"kind"`
	SceneID        string `json:"sceneId"`
	LayerID        string `json:"layerId"`
	InfoboxBlockID string `json:"infoboxBlockId"`
	Index          int    `json:"index"`
	BaseSceneRev   *int64 `json:"baseSceneRev,omitempty"`
}

type applyRemoveNLSInfoboxBlock struct {
	Kind           string `json:"kind"`
	SceneID        string `json:"sceneId"`
	LayerID        string `json:"layerId"`
	InfoboxBlockID string `json:"infoboxBlockId"`
	BaseSceneRev   *int64 `json:"baseSceneRev,omitempty"`
}

func applyAddNLSInfoboxBlockOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyAddNLSInfoboxBlock
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
	pid, err := id.PluginIDFrom(p.PluginID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_plugin", "message": err.Error()}})
		return nil
	}
	eid := id.PluginExtensionID(p.ExtensionID)
	if string(eid) == "" {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_extension", "message": "extensionId required"}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	block, _, err2 := uc.NLSLayer.AddNLSInfoboxBlock(opCtx, interfaces.AddNLSInfoboxBlockParam{
		LayerID:     lid,
		PluginID:    pid,
		ExtensionID: eid,
		Index:       p.Index,
	}, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	extra := map[string]any{
		"sceneId":     p.SceneID,
		"layerId":     p.LayerID,
		"pluginId":    p.PluginID,
		"extensionId": p.ExtensionID,
	}
	if block != nil {
		extra["blockId"] = block.ID().String()
	}
	broadcastApplied(ctx, hub, from, "add_nls_infobox_block", extra, sc)
	if hub != nil && hub.opStack != nil && block != nil {
		inv := applyRemoveNLSInfoboxBlock{
			Kind:           "remove_nls_infobox_block",
			SceneID:        p.SceneID,
			LayerID:        p.LayerID,
			InfoboxBlockID: block.ID().String(),
		}
		if invJSON, errM := json.Marshal(inv); errM == nil {
			rec := UndoableOpRecord{
				ProjectID: from.projectID,
				SceneID:   sid.String(),
				UserID:    actorUserID(from),
				Kind:      "add_nls_infobox_block",
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

func applyMoveNLSInfoboxBlockOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyMoveNLSInfoboxBlock
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
	bid, err := id.InfoboxBlockIDFrom(p.InfoboxBlockID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_block", "message": err.Error()}})
		return nil
	}
	var invJSON json.RawMessage
	if hub != nil && hub.opStack != nil {
		preCtx, preCancel := context.WithTimeout(ctx, applyOpTimeout)
		listL, errL := uc.NLSLayer.Fetch(preCtx, id.NLSLayerIDList{lid}, op)
		preCancel()
		if errL == nil && len(listL) > 0 && listL[0] != nil {
			ly := *listL[0]
			prevIdx, errI := infoboxBlockIndex(ly, bid)
			if errI == nil && prevIdx != p.Index {
				inv := applyMoveNLSInfoboxBlock{
					Kind:           "move_nls_infobox_block",
					SceneID:        p.SceneID,
					LayerID:        p.LayerID,
					InfoboxBlockID: p.InfoboxBlockID,
					Index:          prevIdx,
				}
				invJSON, _ = json.Marshal(inv)
			}
		}
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, _, _, err2 := uc.NLSLayer.MoveNLSInfoboxBlock(opCtx, interfaces.MoveNLSInfoboxBlockParam{
		LayerID:        lid,
		InfoboxBlockID: bid,
		Index:          p.Index,
	}, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "move_nls_infobox_block", map[string]any{
		"sceneId": p.SceneID,
		"layerId": p.LayerID,
		"blockId": p.InfoboxBlockID,
		"index":   p.Index,
	}, sc)
	if hub != nil && hub.opStack != nil && len(invJSON) > 0 {
		rec := UndoableOpRecord{
			ProjectID: from.projectID,
			SceneID:   sid.String(),
			UserID:    actorUserID(from),
			Kind:      "move_nls_infobox_block",
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
	return nil
}

func applyRemoveNLSInfoboxBlockOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyRemoveNLSInfoboxBlock
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
	bid, err := id.InfoboxBlockIDFrom(p.InfoboxBlockID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_block", "message": err.Error()}})
		return nil
	}
	var invJSON json.RawMessage
	if hub != nil && hub.opStack != nil {
		preCtx, preCancel := context.WithTimeout(ctx, applyOpTimeout)
		listL, errL := uc.NLSLayer.Fetch(preCtx, id.NLSLayerIDList{lid}, op)
		preCancel()
		if errL == nil && len(listL) > 0 && listL[0] != nil {
			ly := *listL[0]
			ib := ly.Infobox()
			if ib != nil {
				b := ib.Block(bid)
				if b != nil {
					idx, errI := infoboxBlockIndex(ly, bid)
					if errI == nil {
						idxCopy := idx
						inv := applyAddNLSInfoboxBlock{
							Kind:        "add_nls_infobox_block",
							SceneID:     p.SceneID,
							LayerID:     p.LayerID,
							PluginID:    b.Plugin().String(),
							ExtensionID: string(b.Extension()),
							Index:       &idxCopy,
						}
						invJSON, _ = json.Marshal(inv)
					}
				}
			}
		}
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, _, err2 := uc.NLSLayer.RemoveNLSInfoboxBlock(opCtx, interfaces.RemoveNLSInfoboxBlockParam{
		LayerID:        lid,
		InfoboxBlockID: bid,
	}, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "remove_nls_infobox_block", map[string]any{
		"sceneId": p.SceneID,
		"layerId": p.LayerID,
		"blockId": p.InfoboxBlockID,
	}, sc)
	if hub != nil && hub.opStack != nil && len(invJSON) > 0 {
		rec := UndoableOpRecord{
			ProjectID: from.projectID,
			SceneID:   sid.String(),
			UserID:    actorUserID(from),
			Kind:      "remove_nls_infobox_block",
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
	return nil
}

func infoboxBlockIndex(layer nlslayer.NLSLayer, bid id.InfoboxBlockID) (int, error) {
	ib := layer.Infobox()
	if ib == nil {
		return 0, fmt.Errorf("no infobox")
	}
	for i, b := range ib.Blocks() {
		if b != nil && b.ID() == bid {
			return i, nil
		}
	}
	return 0, fmt.Errorf("block not found")
}
