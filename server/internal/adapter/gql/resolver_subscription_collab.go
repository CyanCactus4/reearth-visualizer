package gql

import (
	"context"
	"errors"

	"github.com/reearth/reearth/server/internal/adapter/gql/gqlmodel"
	"github.com/reearth/reearth/server/internal/collab"
	"github.com/reearth/reearth/server/pkg/id"
)

func (r *Resolver) Subscription() SubscriptionResolver {
	return &subscriptionResolver{r}
}

type subscriptionResolver struct{ *Resolver }

func (r *subscriptionResolver) CollabSceneRevision(ctx context.Context, sceneID gqlmodel.ID) (<-chan int, error) {
	op := getOperator(ctx)
	if op == nil {
		return nil, ErrUnauthorized
	}
	sid, err := id.SceneIDFrom(string(sceneID))
	if err != nil || !op.IsReadableScene(sid) {
		return nil, ErrUnauthorized
	}
	hub := collab.HubFromContext(ctx)
	if hub == nil {
		return nil, errors.New("collab hub unavailable")
	}
	ch, cancel := hub.SubscribeSceneRevision(sid.String())
	out := make(chan int, 32)
	go func() {
		defer cancel()
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-ch:
				if !ok {
					return
				}
				select {
				case out <- int(v):
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}
