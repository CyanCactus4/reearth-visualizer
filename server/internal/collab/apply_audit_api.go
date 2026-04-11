package collab

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/pkg/id"
)

// ServeApplyAudit serves GET /api/collab/apply-audit?projectId=&limit= with the same access rules as ServeChatHistory.
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
		if !op.IsReadableScene(pj.Scene()) {
			return echo.NewHTTPError(http.StatusForbidden, "scene not readable")
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

		recs, err := store.ListRecent(c.Request().Context(), pid.String(), limit)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to load apply audit")
		}
		return c.JSON(http.StatusOK, map[string]any{"v": 1, "entries": recs})
	}
}
