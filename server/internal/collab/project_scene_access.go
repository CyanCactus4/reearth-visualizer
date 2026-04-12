package collab

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/reearth/reearth/server/internal/usecase"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/project"
)

// resolveProjectSceneForAccess returns the scene ID used for collab ACL.
// Some project documents omit the denormalized `scene` field even when a scene
// row exists (see interactor.Scene.Create); fall back to Scene.FindByProject.
func resolveProjectSceneForAccess(
	ctx context.Context,
	uc *interfaces.Container,
	op *usecase.Operator,
	pj *project.Project,
	pid id.ProjectID,
) (id.SceneID, error) {
	if sid := pj.Scene(); op.IsReadableScene(sid) {
		return sid, nil
	}
	sc, err := uc.Scene.FindByProject(ctx, pid, op)
	if err != nil || sc == nil {
		return id.SceneID{}, echo.NewHTTPError(http.StatusForbidden, "scene not readable")
	}
	sid := sc.ID()
	if !op.IsReadableScene(sid) {
		return id.SceneID{}, echo.NewHTTPError(http.StatusForbidden, "scene not readable")
	}
	return sid, nil
}
