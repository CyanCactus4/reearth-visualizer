package collab

import (
	"encoding/json"

	"github.com/reearth/reearth/server/pkg/property"
)

// buildUpdatePropertyValueInverseJSON builds an apply payload that restores the field
// to its state before the forward operation (same shape as client `update_property_value`).
func buildUpdatePropertyValueInverseJSON(prop *property.Property, forward *applyUpdatePropertyValue, ptr *property.Pointer) json.RawMessage {
	if prop == nil || forward == nil || ptr == nil {
		return nil
	}
	inv := applyUpdatePropertyValue{
		Kind:       "update_property_value",
		SceneID:    forward.SceneID,
		PropertyID: forward.PropertyID,
		FieldID:    forward.FieldID,
		Type:       forward.Type,
	}
	if forward.SchemaGroupID != nil && *forward.SchemaGroupID != "" {
		s := *forward.SchemaGroupID
		inv.SchemaGroupID = &s
	}
	if forward.ItemID != nil && *forward.ItemID != "" {
		s := *forward.ItemID
		inv.ItemID = &s
	}
	f, _, _ := prop.Field(ptr)
	if f != nil {
		val := f.Value()
		if val != nil && !val.IsEmpty() {
			raw, err := json.Marshal(val.Interface())
			if err == nil {
				inv.Value = raw
			}
		}
	}
	b, err := json.Marshal(inv)
	if err != nil {
		return nil
	}
	return json.RawMessage(b)
}
