package middleware

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOAuthLogsRedactQueryWithoutChangingHandler(t *testing.T) {
	old := gin.DefaultWriter
	var output bytes.Buffer
	gin.DefaultWriter = &output
	t.Cleanup(func() { gin.DefaultWriter = old })
	r := gin.New()
	SetUpLogger(r)
	r.GET("/api/oauth/github", func(c *gin.Context) { require.Equal(t, "secret-code", c.Query("code")); c.Status(http.StatusOK) })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/oauth/github?code=secret-code&state=secret-state", nil))
	require.NotContains(t, output.String(), "secret-code")
	require.NotContains(t, output.String(), "secret-state")
	require.Contains(t, output.String(), "/api/oauth/github")
}
