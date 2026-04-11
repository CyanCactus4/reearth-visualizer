package collab

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/reearth/reearth/server/pkg/id"
	"github.com/stretchr/testify/assert"
)

func TestActivityUnknownKindErrors(t *testing.T) {
	h := NewHub(Options{})
	c := &Conn{hub: h, projectID: "p", userID: "u", sceneID: id.SceneID{}, send: make(chan []byte, 8)}
	d, _ := json.Marshal(map[string]string{"kind": "dance"})
	assert.NoError(t, dispatchActivity(context.Background(), h, c, d))
	select {
	case msg := <-c.send:
		var sm serverMessage
		_ = json.Unmarshal(msg, &sm)
		assert.Equal(t, "error", sm.T)
	default:
		t.Fatal("expected error")
	}
}
