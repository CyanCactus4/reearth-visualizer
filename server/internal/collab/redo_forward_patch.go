package collab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/reearth/reearth/server/internal/adapter/gql/gqlmodel"
	"github.com/reearth/reearth/server/internal/usecase"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/scene"
)

// maybePatchRedoForwardAfterUndo updates the Mongo redo head forward payload when undo
// recreated an entity with a new id (remove_widget / remove_style inverse used add_*).
func maybePatchRedoForwardAfterUndo(ctx context.Context, stack CollabOpStack, userID, sceneID string, rec *UndoableOpRecord, sc *scene.Scene, uc *interfaces.Container, op *usecase.Operator) error {
	if stack == nil || rec == nil || sc == nil {
		return nil
	}
	var newFwd json.RawMessage
	var err error
	switch rec.Kind {
	case "remove_widget":
		newFwd, err = patchedRedoRemoveWidgetForward(rec, sc)
	case "remove_style":
		if uc == nil {
			return nil
		}
		newFwd, err = patchedRedoRemoveStyleForward(ctx, uc, op, rec)
	case "remove_property_item":
		if uc == nil {
			return nil
		}
		newFwd, err = patchedRedoRemovePropertyItemForward(ctx, uc, op, rec)
	default:
		return nil
	}
	if err != nil || len(newFwd) == 0 {
		return err
	}
	return stack.PatchHeadRedoForward(ctx, userID, sceneID, newFwd)
}

func patchedRedoRemoveWidgetForward(rec *UndoableOpRecord, sc *scene.Scene) (json.RawMessage, error) {
	var inv applyAddWidget
	if err := json.Unmarshal(rec.Inverse, &inv); err != nil {
		return nil, err
	}
	pid, err := id.PluginIDFrom(inv.PluginID)
	if err != nil {
		return nil, err
	}
	ext := id.PluginExtensionID(inv.ExtensionID)
	widStr := pickWidgetIDForPluginExtension(sc, pid, ext)
	if widStr == "" {
		return nil, errors.New("no widget matching inverse add_widget")
	}
	var fwd applyRemoveWidget
	if err := json.Unmarshal(rec.Forward, &fwd); err != nil {
		return nil, err
	}
	fwd.WidgetID = widStr
	b, err := json.Marshal(fwd)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func pickWidgetIDForPluginExtension(sc *scene.Scene, pid id.PluginID, ext id.PluginExtensionID) string {
	if sc == nil || sc.Widgets() == nil {
		return ""
	}
	var best string
	for _, w := range sc.Widgets().Widgets() {
		if w == nil {
			continue
		}
		if !w.Plugin().Equal(pid) || w.Extension() != ext {
			continue
		}
		s := w.ID().String()
		if s > best {
			best = s
		}
	}
	return best
}

func patchedRedoRemoveStyleForward(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, rec *UndoableOpRecord) (json.RawMessage, error) {
	var inv applyAddStyle
	if err := json.Unmarshal(rec.Inverse, &inv); err != nil {
		return nil, err
	}
	sid, err := id.SceneIDFrom(inv.SceneID)
	if err != nil {
		return nil, err
	}
	wantVal, err := parseStyleValueRaw(inv.Value)
	if err != nil || wantVal == nil {
		return nil, fmt.Errorf("inverse add_style value")
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	list, err := uc.Style.FetchByScene(opCtx, sid, op)
	if err != nil || list == nil {
		return nil, err
	}
	var bestID id.StyleID
	var bestKey string
	for _, st := range *list {
		if st == nil || st.Name() != inv.Name {
			continue
		}
		if !styleValuesEqual(st.Value(), wantVal) {
			continue
		}
		candidate := st.ID().String()
		if candidate > bestKey {
			bestKey = candidate
			bestID = st.ID()
		}
	}
	if bestKey == "" {
		return nil, errors.New("no style matching inverse add_style")
	}
	var fwd applyRemoveStyle
	if err := json.Unmarshal(rec.Forward, &fwd); err != nil {
		return nil, err
	}
	fwd.StyleID = bestID.String()
	b, err := json.Marshal(fwd)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func patchedRedoRemovePropertyItemForward(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, rec *UndoableOpRecord) (json.RawMessage, error) {
	var inv applyAddPropertyItem
	if err := json.Unmarshal(rec.Inverse, &inv); err != nil {
		return nil, err
	}
	var fwd applyRemovePropertyItem
	if err := json.Unmarshal(rec.Forward, &fwd); err != nil {
		return nil, err
	}
	sid, err := id.SceneIDFrom(inv.SceneID)
	if err != nil {
		return nil, err
	}
	pid, err := gqlmodel.ToID[id.Property](gqlmodel.ID(inv.PropertyID))
	if err != nil {
		return nil, err
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	list, err := uc.Property.Fetch(opCtx, []id.PropertyID{pid}, op)
	if err != nil || len(list) == 0 || list[0] == nil {
		return nil, fmt.Errorf("property fetch after undo")
	}
	prop := list[0]
	if prop.Scene() != sid {
		return nil, fmt.Errorf("scene mismatch")
	}
	sg := id.PropertySchemaGroupID(inv.SchemaGroupID)
	idx := 0
	if inv.Index != nil {
		idx = *inv.Index
	}
	newItemID, err := propertyListGroupIDAtIndex(prop, sg, idx)
	if err != nil {
		return nil, err
	}
	fwd.ItemID = newItemID.String()
	b, err := json.Marshal(&fwd)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func styleValuesEqual(a, b *scene.StyleValue) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	ba, err1 := json.Marshal(map[string]any(*a))
	bb, err2 := json.Marshal(map[string]any(*b))
	return err1 == nil && err2 == nil && string(ba) == string(bb)
}
