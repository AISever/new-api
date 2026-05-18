package vidu

import (
	"net/http"
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBuildRequestHeaderDoesNotInferAggregatorFromKeyPrefix(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := httptest.NewRequest(http.MethodPost, "https://example.test/ent/v2/text2video", nil)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	err := adaptor.BuildRequestHeader(ctx, req, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ApiKey: "sk-test-key"},
	})

	require.NoError(t, err)
	require.Equal(t, "Token sk-test-key", req.Header.Get("Authorization"))
	require.Equal(t, "application/json", req.Header.Get("Content-Type"))
	require.Equal(t, "application/json", req.Header.Get("Accept"))
}
