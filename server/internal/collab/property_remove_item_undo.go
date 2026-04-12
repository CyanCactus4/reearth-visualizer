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
	"github.com/reearth/reearth/server/pkg/property"
	"github.com/reearth/reearth/server/pkg/value"
)

// buildInverseAddPropertyItemJSONAfterRemove builds undo inverse (re-add list item) from state before removal.
func buildInverseAddPropertyItemJSONAfterRemove(
	ctx context.Context,
	uc *interfaces.Container,
	op *usecase.Operator,
	prop *property.Property,
	p *applyRemovePropertyItem,
	listIndex int,
) (json.RawMessage, error) {
	if prop == nil || uc == nil || p == nil {
		return nil, errors.New("missing args")
	}
	sgID := id.PropertySchemaGroupID(p.SchemaGroupID)
	itemIid, err := id.PropertyItemIDFrom(p.ItemID)
	if err != nil {
		return nil, err
	}
	grp := findListGroup(prop, sgID, itemIid)
	if grp == nil {
		return nil, fmt.Errorf("property list item not found")
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	schemas, err := uc.Property.FetchSchema(opCtx, []id.PropertySchemaID{prop.Schema()}, op)
	if err != nil || len(schemas) == 0 || schemas[0] == nil {
		return nil, fmt.Errorf("property schema: %w", err)
	}
	ps := schemas[0]
	idx := listIndex
	add := applyAddPropertyItem{
		Kind:           "add_property_item",
		SceneID:        p.SceneID,
		PropertyID:     p.PropertyID,
		SchemaGroupID:  p.SchemaGroupID,
		Index:          &idx,
		NameFieldType:  nil,
		NameFieldValue: nil,
		BaseSceneRev:   nil,
	}
	rep := grp.RepresentativeField(ps)
	if rep != nil && !rep.IsEmpty() {
		v := rep.Value()
		if v != nil && !v.IsEmpty() {
			vt := gqlmodel.ToValueType(value.Type(rep.Type()))
			tstr := string(vt)
			add.NameFieldType = &tstr
			pv := gqlmodel.ToPropertyValue(v)
			if pv == nil {
				return nil, fmt.Errorf("cannot serialize representative field")
			}
			raw, err := json.Marshal(*pv)
			if err != nil {
				return nil, err
			}
			add.NameFieldValue = raw
		}
	}
	b, err := json.Marshal(&add)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func findListGroup(prop *property.Property, sg id.PropertySchemaGroupID, itemID id.PropertyItemID) *property.Group {
	if prop == nil {
		return nil
	}
	for _, it := range prop.Items() {
		gl, ok := it.(*property.GroupList)
		if !ok || gl.SchemaGroup() != sg {
			continue
		}
		if g := gl.Group(itemID); g != nil {
			return g
		}
	}
	return nil
}

func propertyListGroupIDAtIndex(p *property.Property, sg id.PropertySchemaGroupID, idx int) (id.PropertyItemID, error) {
	if p == nil {
		return id.PropertyItemID{}, errors.New("nil property")
	}
	for _, it := range p.Items() {
		gl, ok := it.(*property.GroupList)
		if !ok || gl.SchemaGroup() != sg {
			continue
		}
		gr := gl.Groups()
		if idx < 0 || idx >= len(gr) || gr[idx] == nil {
			return id.PropertyItemID{}, fmt.Errorf("list index out of range")
		}
		return gr[idx].ID(), nil
	}
	return id.PropertyItemID{}, errors.New("group list not found")
}
