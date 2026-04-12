package collab

import (
	"context"
	"encoding/json"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/pkg/id"
)

// applyUpdateSceneCamera is a dedicated collab kind for the scene root "camera" field
// (schema group + field id "camera", type CAMERA). It resolves propertyId and itemId
// from the scene document so clients do not need to repeat that wiring.
type applyUpdateSceneCamera struct {
	Kind         string          `json:"kind"`
	SceneID      string          `json:"sceneId"`
	Value        json.RawMessage `json:"value"`
	BaseSceneRev *int64          `json:"baseSceneRev,omitempty"`
	FieldClock   *int64          `json:"fieldClock,omitempty"`
	FieldHLC     *fieldHLCWire   `json:"fieldHlc,omitempty"`
}

func applyUpdateSceneCameraOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyUpdateSceneCamera
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
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	scenes, err := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err != nil || len(scenes) == 0 || scenes[0] == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": "scene not found"}})
		return nil
	}
	sc := scenes[0]
	pid := sc.Property()
	prop := fetchPropertyForCollabApply(ctx, uc, op, sid, pid, from)
	if prop == nil {
		return nil
	}
	const camGroup = "camera"
	it := prop.ItemBySchema(id.PropertySchemaGroupID(camGroup))
	if it == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "camera_item_not_found", "message": "scene has no camera property group"}})
		return nil
	}
	itemID := it.ID().String()
	sg := camGroup
	inner := applyUpdatePropertyValue{
		Kind:          "update_property_value",
		SceneID:       p.SceneID,
		PropertyID:    pid.String(),
		SchemaGroupID: &sg,
		ItemID:        &itemID,
		FieldID:       camGroup,
		Type:          "CAMERA",
		Value:         p.Value,
		BaseSceneRev:  p.BaseSceneRev,
		FieldClock:    p.FieldClock,
		FieldHLC:      p.FieldHLC,
	}
	b, err := json.Marshal(inner)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": err.Error()}})
		return nil
	}
	return applyUpdatePropertyValueOp(ctx, hub, from, b)
}
