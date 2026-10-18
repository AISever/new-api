package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAnonymousRequestLimitRejectsKnownAndChunkedBodies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	before := constant.AnonymousRequestBodyLimitKB
	constant.AnonymousRequestBodyLimitKB = 1
	t.Cleanup(func() { constant.AnonymousRequestBodyLimitKB = before })
	for _, chunked := range []bool{false, true} {
		t.Run(map[bool]string{false: "content-length", true: "chunked"}[chunked], func(t *testing.T) {
			called := false
			r := gin.New()
			r.POST("/webhook", AnonymousRequestBodyLimit(), func(c *gin.Context) { called = true })
			request := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(strings.Repeat("x", 1025)))
			if chunked {
				request.ContentLength = -1
				request.TransferEncoding = []string{"chunked"}
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)
			require.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
			require.False(t, called, "reject before signature parsing or payload logging")
		})
	}
}

func TestAnonymousRequestLimitPreservesSignedBodyAtBoundary(t *testing.T) {
	before := constant.AnonymousRequestBodyLimitKB
	constant.AnonymousRequestBodyLimitKB = 1
	t.Cleanup(func() { constant.AnonymousRequestBodyLimitKB = before })
	body := strings.Repeat("a", 1024)
	r := gin.New()
	r.POST("/webhook", AnonymousRequestBodyLimit(), func(c *gin.Context) {
		got, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		require.Equal(t, body, string(got), "payment signature bytes must not change")
		require.Equal(t, int64(1024), c.Request.ContentLength)
		c.Status(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body)))
	require.Equal(t, http.StatusNoContent, w.Code)
}
