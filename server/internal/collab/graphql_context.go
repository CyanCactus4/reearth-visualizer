package collab

import "context"

type hubCtxKey struct{}

// AttachHub stores the collab hub on ctx for GraphQL subscription resolvers.
func AttachHub(ctx context.Context, h *Hub) context.Context {
	if h == nil {
		return ctx
	}
	return context.WithValue(ctx, hubCtxKey{}, h)
}

// HubFromContext returns the hub attached with AttachHub, or nil.
func HubFromContext(ctx context.Context) *Hub {
	v, _ := ctx.Value(hubCtxKey{}).(*Hub)
	return v
}
