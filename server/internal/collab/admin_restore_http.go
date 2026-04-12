package collab

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/pkg/id"
	"go.mongodb.org/mongo-driver/mongo"
)

type adminRestoreBody struct {
	SceneID string `json:"sceneId"`
	// TargetSceneRev restores the newest snapshot with sceneRev <= this value (Unix ms, same basis as collab sceneRev).
	TargetSceneRev *int64 `json:"targetSceneRev,omitempty"`
}

// ServeCollabAdminRestore is POST /api/collab/admin/restore-scene (maintainer+ only).
// When snapshots is nil, returns 501 (contract reserved). Otherwise loads ExportSceneData JSON and runs Scene.ImportSceneData.
func ServeCollabAdminRestore(hub *Hub, snapshots SceneSnapshotStore) echo.HandlerFunc {
	if snapshots == nil {
		return adminRestoreNotImplemented()
	}
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
		if body.TargetSceneRev == nil || *body.TargetSceneRev <= 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "targetSceneRev required")
		}
		data, foundRev, err := snapshots.LoadClosestAtOrBelow(c.Request().Context(), body.SceneID, *body.TargetSceneRev)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return echo.NewHTTPError(http.StatusNotFound, "no snapshot at or before targetSceneRev")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if len(data) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, "no snapshot at or before targetSceneRev")
		}
		uc := adapter.Usecases(c.Request().Context())
		if uc == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "usecases unavailable")
		}
		scenes, err := uc.Scene.Fetch(c.Request().Context(), []id.SceneID{sid}, op)
		if err != nil || len(scenes) == 0 || scenes[0] == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "scene load failed")
		}
		sce := scenes[0]
		buf := append([]byte(nil), data...)
		out, err := uc.Scene.ImportSceneData(c.Request().Context(), sce, &buf)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if hub != nil && out != nil {
			hub.publishSceneRevision(out.ID().String(), sceneRevOf(out))
		}
		return c.JSON(http.StatusOK, map[string]any{
			"v":                  1,
			"sceneId":            body.SceneID,
			"sceneRev":           sceneRevOf(out),
			"snapshotRev":        foundRev,
			"requestedTargetRev": *body.TargetSceneRev,
		})
	}
}

func adminRestoreNotImplemented() echo.HandlerFunc {
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
			"error":   "collab scene snapshots not configured (set Mongo collection REEARTH_COLLAB_SCENE_SNAPSHOT_COLLECTION / defaults)",
			"sceneId": body.SceneID,
		})
	}
}
