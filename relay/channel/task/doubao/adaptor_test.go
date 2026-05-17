package doubao

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestBuildRequestURLUsesYunwuNewAPIVolcPath(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://yunwu.ai"}

	got, err := adaptor.BuildRequestURL(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ApiKey: "sk-test-key"},
	})

	require.NoError(t, err)
	require.Equal(t, "https://yunwu.ai/volc/v1/contents/generations/tasks", got)
}

func TestBuildRequestURLKeepsOfficialDoubaoPath(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://ark.cn-beijing.volces.com"}

	got, err := adaptor.BuildRequestURL(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ApiKey: "official-key"},
	})

	require.NoError(t, err)
	require.Equal(t, "https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks", got)
}
