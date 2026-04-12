package repo

import (
	"context"
	"errors"

	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/project"
)

// ErrProjectMetadataNotFound is returned by ProjectMetadata.FindByProjectID when no row exists.
// Callers that load a project (e.g. collab WS) may treat it like GraphQL Fetch: metadata is optional.
var ErrProjectMetadataNotFound = errors.New("project metadata not found")

type ProjectMetadata interface {
	Filtered(WorkspaceFilter) ProjectMetadata
	FindByProjectID(context.Context, id.ProjectID) (*project.ProjectMetadata, error)
	FindByProjectIDList(context.Context, id.ProjectIDList) ([]*project.ProjectMetadata, error)
	Save(context.Context, *project.ProjectMetadata) error
	Remove(context.Context, id.ProjectID) error
}
