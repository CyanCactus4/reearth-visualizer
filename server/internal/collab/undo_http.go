package collab

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearthx/log"
)

type undoRedoBody struct {
	SceneID string `json:"sceneId"`
}

// ServeCollabUndo handles POST /api/collab/undo with JSON { "sceneId": "..." }.
func ServeCollabUndo(hub *Hub) echo.HandlerFunc {
	return serveCollabUndoRedo(hub, true)
}

// ServeCollabRedo handles POST /api/collab/redo with JSON { "sceneId": "..." }.
func ServeCollabRedo(hub *Hub) echo.HandlerFunc {
	return serveCollabUndoRedo(hub, false)
}

func serveCollabUndoRedo(hub *Hub, undo bool) echo.HandlerFunc {
	if hub == nil || hub.opStack == nil {
		return func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusNotFound, "collab undo/redo not configured")
		}
	}
	return func(c echo.Context) error {
		op := adapter.Operator(c.Request().Context())
		if op == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}
		u := adapter.User(c.Request().Context())
		if u == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}
		userID := u.ID().String()

		var body undoRedoBody
		if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.SceneID == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "sceneId required")
		}
		sid, err := id.SceneIDFrom(body.SceneID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid sceneId")
		}
		if !op.IsWritableScene(sid) {
			return echo.NewHTTPError(http.StatusForbidden, "scene not writable")
		}

		var rec *UndoableOpRecord
		if undo {
			rec, err = hub.opStack.Undo(c.Request().Context(), userID, body.SceneID)
		} else {
			rec, err = hub.opStack.Redo(c.Request().Context(), userID, body.SceneID)
		}
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		payload := rec.Inverse
		if !undo {
			payload = rec.Forward
		}

		sc, err2 := ExecuteCollabUndoJSON(c.Request().Context(), payload, op)
		if err2 != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err2.Error())
		}
		if sc == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "scene nil after undo")
		}

		if undo {
			uc := adapter.Usecases(c.Request().Context())
			if err := maybePatchRedoForwardAfterUndo(c.Request().Context(), hub.opStack, userID, body.SceneID, rec, sc, uc, op); err != nil {
				log.Warnfc(c.Request().Context(), "collab: patch redo forward: %v", err)
			}
		}

		pid := sc.Project().String()
		rev := sceneRevOf(sc)
		hub.publishSceneRevision(sc.ID().String(), rev)
		kind := "collab_redo"
		if undo {
			kind = "collab_undo"
		}
		appliedD := map[string]any{
			"kind":     kind,
			"userId":   userID,
			"sceneRev": rev,
			"sceneId":  sc.ID().String(),
		}
		if rec.Kind != "" {
			appliedD["opKind"] = rec.Kind
		}
		nb, errM := json.Marshal(serverMessage{
			V: 1,
			T: "applied",
			D: appliedD,
		})
		if errM == nil {
			hub.fanoutRoom(c.Request().Context(), pid, nb)
		}

		if hub.applyAudit != nil {
			userName := ""
			if usr := adapter.User(c.Request().Context()); usr != nil {
				userName = strings.TrimSpace(usr.Name())
			}
			audit := ApplyAuditRecord{
				ProjectID: pid,
				UserID:    userID,
				UserName:  userName,
				Kind:      kind,
				OpKind:    rec.Kind,
				SceneRev:  rev,
				SceneID:   sc.ID().String(),
			}
			go func() {
				pctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := hub.applyAudit.Append(pctx, audit); err != nil {
					log.Warnfc(pctx, "collab: apply audit: %v", err)
				}
			}()
		}
		return c.JSON(http.StatusOK, map[string]any{"v": 1, "sceneRev": rev, "sceneId": sc.ID().String()})
	}
}
