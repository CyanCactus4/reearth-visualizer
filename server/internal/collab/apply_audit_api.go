package collab

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/pkg/id"
)

// ServeApplyAudit serves GET /api/collab/apply-audit?projectId=&limit=&sceneId= with the same access rules as ServeChatHistory.
// Optional sceneId must match the project's scene and limits rows to that scene.
// Rows are newest-first (by persisted ts).
func ServeApplyAudit(store ApplyAuditStore) echo.HandlerFunc {
	if store == nil {
		return func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusNotFound, "apply audit not configured")
		}
	}
	return func(c echo.Context) error {
		op := adapter.Operator(c.Request().Context())
		if op == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}

		pidStr := strings.TrimSpace(c.QueryParam("projectId"))
		if pidStr == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "projectId is required")
		}
		pid, err := id.ProjectIDFrom(pidStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid projectId")
		}

		uc := adapter.Usecases(c.Request().Context())
		if uc == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
		}

		pj, err := uc.Project.FindActiveById(c.Request().Context(), pid, op)
		if err != nil || pj == nil {
			return echo.NewHTTPError(http.StatusForbidden, "project not accessible")
		}
		projectScene, err := resolveProjectSceneForAccess(c.Request().Context(), uc, op, pj, pid)
		if err != nil {
			return err
		}

		sceneFilter, errScene := parseApplyAuditSceneFilterParam(
			c.QueryParam("sceneId"),
			projectScene,
			func(s id.SceneID) bool { return op.IsReadableScene(s) },
		)
		if errScene != nil {
			return errScene
		}

		limit := 100
		if q := strings.TrimSpace(c.QueryParam("limit")); q != "" {
			if n, e := strconv.Atoi(q); e == nil && n > 0 {
				limit = n
			}
		}
		if limit > 500 {
			limit = 500
		}

		recs, err := store.ListRecent(c.Request().Context(), pid.String(), sceneFilter, limit)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to load apply audit")
		}
		return c.JSON(http.StatusOK, map[string]any{"v": 1, "entries": recs})
	}
}

// parseApplyAuditSceneFilterParam returns a Mongo sceneId filter, or empty when the query is omitted.
func parseApplyAuditSceneFilterParam(
	sceneParam string,
	projectScene id.SceneID,
	isReadable func(id.SceneID) bool,
) (filter string, err error) {
	q := strings.TrimSpace(sceneParam)
	if q == "" {
		return "", nil
	}
	sid, e := id.SceneIDFrom(q)
	if e != nil {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid sceneId")
	}
	if sid != projectScene {
		return "", echo.NewHTTPError(http.StatusForbidden, "sceneId does not match project")
	}
	if !isReadable(sid) {
		return "", echo.NewHTTPError(http.StatusForbidden, "scene not readable")
	}
	return sid.String(), nil
}
