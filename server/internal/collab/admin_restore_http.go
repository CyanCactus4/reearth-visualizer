package collab

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/pkg/id"
)

type adminRestoreBody struct {
	SceneID string `json:"sceneId"`
	// TargetSceneRev reserved for future point-in-time restore (Mongo snapshots / audit replay).
	TargetSceneRev *int64 `json:"targetSceneRev,omitempty"`
}

// ServeCollabAdminRestorePlaceholder is POST /api/collab/admin/restore-scene (maintainer+ only).
// Full "restore project to revision" is not implemented; this endpoint reserves the contract and returns 501.
func ServeCollabAdminRestorePlaceholder() echo.HandlerFunc {
	return func(c echo.Context) error {
		op := adapter.Operator(c.Request().Context())
		if op == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}
		var body adminRestoreBody
		if err := c.Bind(&body); err != nil || body.SceneID == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "sceneId required")
		}
		sid, err := id.SceneIDFrom(body.SceneID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid sceneId")
		}
		if !op.IsMaintainingScene(sid) {
			return echo.NewHTTPError(http.StatusForbidden, "maintainer role required for admin restore")
		}
		return c.JSON(http.StatusNotImplemented, map[string]any{
			"v":       1,
			"error":   "admin scene restore is not implemented yet (no historical scene snapshots in this build)",
			"sceneId": body.SceneID,
		})
	}
}
