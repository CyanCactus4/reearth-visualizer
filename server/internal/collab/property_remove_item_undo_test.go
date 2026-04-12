package collab

import (
	"context"
	"testing"
)

func TestBuildInverseAddPropertyItemJSONAfterRemove_nilArgs(t *testing.T) {
	t.Parallel()
	_, err := buildInverseAddPropertyItemJSONAfterRemove(
		context.Background(),
		nil,
		nil,
		nil,
		&applyRemovePropertyItem{Kind: "remove_property_item"},
		0,
	)
	if err == nil {
		t.Fatal("expected error")
	}
}
