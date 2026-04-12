package collab

import "testing"

func FuzzDecodeApplyUpdatePropertyValue(f *testing.F) {
	f.Add([]byte(`{"kind":"update_property_value","sceneId":"00000000-0000-0000-0000-000000000001","propertyId":"00000000-0000-0000-0000-000000000002","fieldId":"f","type":"string","value":"x"}`))
	f.Add([]byte(`{"kind":"update_property_value","sceneId":"x","propertyId":"y","fieldId":"z","type":"number","value":1}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 8192 {
			return
		}
		_, _, _, _ = decodeApplyUpdatePropertyValue(data)
	})
}
