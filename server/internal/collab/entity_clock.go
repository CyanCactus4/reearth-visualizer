package collab

func widgetClockKey(sceneID, widgetID, field string) string {
	return sceneID + "\x00" + widgetID + "\x00" + field
}

// WidgetFieldClock returns the server LWW clock for a widget field (0 if unseen).
func (h *Hub) WidgetFieldClock(sceneID, widgetID, field string) int64 {
	if h == nil || sceneID == "" || widgetID == "" || field == "" {
		return 0
	}
	h.widgetClockMu.Lock()
	defer h.widgetClockMu.Unlock()
	return h.widgetClocks[widgetClockKey(sceneID, widgetID, field)]
}

// BumpWidgetFieldClocks increments clocks for the given fields and returns the new values.
func (h *Hub) BumpWidgetFieldClocks(sceneID, widgetID string, fields []string) map[string]int64 {
	out := make(map[string]int64)
	if h == nil || sceneID == "" || widgetID == "" {
		return out
	}
	h.widgetClockMu.Lock()
	defer h.widgetClockMu.Unlock()
	for _, f := range fields {
		if f == "" {
			continue
		}
		k := widgetClockKey(sceneID, widgetID, f)
		h.widgetClocks[k]++
		out[f] = h.widgetClocks[k]
	}
	return out
}
