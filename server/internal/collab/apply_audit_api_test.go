package collab

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeApplyAudit_nilStore(t *testing.T) {
	e := echo.New()
	e.GET("/apply-audit", ServeApplyAudit(nil))
	req := httptest.NewRequest(http.MethodGet, "/apply-audit?projectId=x", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestParseApplyAuditSceneFilterParam_empty(t *testing.T) {
	scene := id.MustSceneID("01fbpdqax0ttrftj3gb5gm4rw7")
	f, err := parseApplyAuditSceneFilterParam("  ", scene, func(id.SceneID) bool { return true })
	require.NoError(t, err)
	assert.Equal(t, "", f)
}

func TestParseApplyAuditSceneFilterParam_ok(t *testing.T) {
	scene := id.MustSceneID("01fbpdqax0ttrftj3gb5gm4rw7")
	f, err := parseApplyAuditSceneFilterParam(scene.String(), scene, func(s id.SceneID) bool {
		return s == scene
	})
	require.NoError(t, err)
	assert.Equal(t, scene.String(), f)
}

func TestParseApplyAuditSceneFilterParam_wrongScene(t *testing.T) {
	scene := id.MustSceneID("01fbpdqax0ttrftj3gb5gm4rw7")
	other := id.MustSceneID("01fbpdqax0ttrftj3gb5gm4rw8")
	_, err := parseApplyAuditSceneFilterParam(other.String(), scene, func(id.SceneID) bool { return true })
	var he *echo.HTTPError
	require.True(t, errors.As(err, &he))
	assert.Equal(t, http.StatusForbidden, he.Code)
}

func TestParseApplyAuditSceneFilterParam_invalidID(t *testing.T) {
	scene := id.MustSceneID("01fbpdqax0ttrftj3gb5gm4rw7")
	_, err := parseApplyAuditSceneFilterParam("not-a-scene-id", scene, func(id.SceneID) bool { return true })
	var he *echo.HTTPError
	require.True(t, errors.As(err, &he))
	assert.Equal(t, http.StatusBadRequest, he.Code)
}

func TestParseApplyAuditSceneFilterParam_notReadable(t *testing.T) {
	scene := id.MustSceneID("01fbpdqax0ttrftj3gb5gm4rw7")
	_, err := parseApplyAuditSceneFilterParam(scene.String(), scene, func(id.SceneID) bool { return false })
	var he *echo.HTTPError
	require.True(t, errors.As(err, &he))
	assert.Equal(t, http.StatusForbidden, he.Code)
}
