package collab

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestServeChatHistory_nilStore(t *testing.T) {
	e := echo.New()
	e.GET("/chat", ServeChatHistory(nil))
	req := httptest.NewRequest(http.MethodGet, "/chat?projectId=x", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
