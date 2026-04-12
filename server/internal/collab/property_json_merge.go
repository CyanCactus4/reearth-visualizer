package collab

import (
	"encoding/json"
	"strings"

	"github.com/reearth/reearth/server/internal/adapter/gql/gqlmodel"
	"github.com/reearth/reearth/server/pkg/property"
)

// flatPropertyFieldKey matches client `propertyFieldClockKey` without propertyId prefix.
func flatPropertyFieldKey(schemaGroupID, itemID, fieldID string) string {
	return schemaGroupID + "\x00" + itemID + "\x00" + fieldID
}

func parseFlatPropertyFieldKey(k string) (schemaGroupID, itemID, fieldID string, ok bool) {
	parts := strings.Split(k, "\x00")
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// flattenPropertyValueLeaves maps composite field keys to {"type","value"} JSON-friendly blobs.
func flattenPropertyValueLeaves(p *property.Property) (map[string]map[string]any, error) {
	out := make(map[string]map[string]any)
	if p == nil {
		return out, nil
	}
	for _, it := range p.Items() {
		switch x := it.(type) {
		case *property.Group:
			collectGroupFieldsInto(out, x.SchemaGroup().String(), x.ID().String(), x)
		case *property.GroupList:
			sg := x.SchemaGroup().String()
			for _, g := range x.Groups() {
				collectGroupFieldsInto(out, sg, g.ID().String(), g)
			}
		}
	}
	return out, nil
}

func collectGroupFieldsInto(out map[string]map[string]any, schemaGroupID, itemID string, g *property.Group) {
	if g == nil {
		return
	}
	for _, f := range g.Fields(nil) {
		if f == nil {
			continue
		}
		k := flatPropertyFieldKey(schemaGroupID, itemID, f.Field().String())
		cell := map[string]any{
			"type":  gqlmodel.ValueType(strings.ToUpper(string(f.Type()))),
			"value": gqlmodel.ToPropertyValue(f.Value()),
		}
		out[k] = cell
	}
}

func cloneJSONMapStringAny(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	if out == nil {
		return map[string]any{}
	}
	return out
}

// jsonMergePatchObject applies RFC 7396 merge patch semantics for JSON objects (top-level keys).
func jsonMergePatchObject(dst, patch map[string]any) map[string]any {
	out := cloneJSONMapStringAny(dst)
	if patch == nil {
		return out
	}
	for k, pv := range patch {
		if pv == nil {
			delete(out, k)
			continue
		}
		pm, ok := pv.(map[string]any)
		if !ok {
			out[k] = pv
			continue
		}
		if cur, ok2 := out[k].(map[string]any); ok2 {
			out[k] = jsonMergePatchObject(cur, pm)
		} else {
			out[k] = cloneJSONMapStringAny(pm)
		}
	}
	return out
}

func leafMapsEqualJSON(a, b map[string]any) bool {
	if a == nil && b == nil {
		return true
	}
	ab, e1 := json.Marshal(a)
	bb, e2 := json.Marshal(b)
	if e1 != nil || e2 != nil {
		return false
	}
	return string(ab) == string(bb)
}
