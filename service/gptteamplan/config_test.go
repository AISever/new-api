package gptteamplan

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestGetConfigUsesDefaults(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	oldOptionMap := common.OptionMap
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()
	defer func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = oldOptionMap
		common.OptionMapRWMutex.Unlock()
	}()

	cfg := GetConfig()
	require.False(t, cfg.Enabled)
	require.Equal(t, "http://gptteamplan.tech", cfg.BaseURL)
}

func TestGetConfigUsesCustomBaseURL(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	oldOptionMap := common.OptionMap
	common.OptionMap = map[string]string{
		"gptteamplan.enabled":  "true",
		"gptteamplan.base_url": "http://example.com/",
	}
	common.OptionMapRWMutex.Unlock()
	defer func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = oldOptionMap
		common.OptionMapRWMutex.Unlock()
	}()

	cfg := GetConfig()
	require.True(t, cfg.Enabled)
	require.Equal(t, "http://example.com", cfg.BaseURL)
	require.True(t, cfg.IsReady())
}
