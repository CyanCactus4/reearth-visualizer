package collab

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/pkg/id"
)

// ServeSceneRevisionSSE streams `data: <sceneRevMs>\n\n` when collab applies update that scene (same-process hub).
func ServeSceneRevisionSSE(hub *Hub) echo.HandlerFunc {
	if hub == nil {
		return func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusNotFound, "collab hub not available")
		}
	}
	return func(c echo.Context) error {
		op := adapter.Operator(c.Request().Context())
		if op == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}
		sidStr := strings.TrimSpace(c.QueryParam("sceneId"))
		if sidStr == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "sceneId is required")
		}
		sid, err := id.SceneIDFrom(sidStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid sceneId")
		}
		uc := adapter.Usecases(c.Request().Context())
		if uc == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
		}
		scenes, err := uc.Scene.Fetch(c.Request().Context(), []id.SceneID{sid}, op)
		if err != nil || len(scenes) == 0 {
			return echo.NewHTTPError(http.StatusForbidden, "scene not accessible")
		}
		if !op.IsReadableScene(sid) {
			return echo.NewHTTPError(http.StatusForbidden, "scene not readable")
		}

		w := c.Response()
		w.Header().Set(echo.HeaderContentType, "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.Writer.(http.Flusher)
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError, "streaming unsupported")
		}
		flusher.Flush()

		ch, unsub := hub.SubscribeSceneRevision(sidStr)
		defer unsub()

		ctx := c.Request().Context()
		for {
			select {
			case <-ctx.Done():
				return nil
			case rev, ok := <-ch:
				if !ok {
					return nil
				}
				_, _ = fmt.Fprintf(w.Writer, "data: %d\n\n", rev)
				flusher.Flush()
			}
		}
	}
}
