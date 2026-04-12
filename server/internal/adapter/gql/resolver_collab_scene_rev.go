package gql

import (
	"context"

	"github.com/reearth/reearth/server/internal/collab"
	"github.com/reearth/reearth/server/pkg/id"
)

// publishCollabSceneRevisionForScene pushes scene.UpdatedAt (ms) to collabSceneRevision subscribers
// when the change came from GraphQL instead of the collab WebSocket apply path.
func publishCollabSceneRevisionForScene(ctx context.Context, sceneID id.SceneID) {
	if sceneID == (id.SceneID{}) {
		return
	}
	h := collab.HubFromContext(ctx)
	if h == nil {
		return
	}
	op := getOperator(ctx)
	if op == nil || !op.IsReadableScene(sceneID) {
		return
	}
	scenes, err := usecases(ctx).Scene.Fetch(ctx, []id.SceneID{sceneID}, op)
	if err != nil || len(scenes) == 0 || scenes[0] == nil {
		return
	}
	rev := scenes[0].UpdatedAt().UnixMilli()
	if rev > 0 {
		h.PublishSceneRevision(sceneID.String(), rev)
	}
}

func publishCollabSceneRevisionForLayer(ctx context.Context, lid id.NLSLayerID) {
	op := getOperator(ctx)
	if op == nil {
		return
	}
	list, err := usecases(ctx).NLSLayer.Fetch(ctx, id.NLSLayerIDList{lid}, op)
	if err != nil || len(list) == 0 || list[0] == nil {
		return
	}
	publishCollabSceneRevisionForScene(ctx, (*list[0]).Scene())
}
