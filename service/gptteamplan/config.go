package gptteamplan

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
)

type Config struct {
	Enabled bool
	BaseURL string
}

func GetConfig() Config {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()

	baseURL := strings.TrimRight(strings.TrimSpace(common.OptionMap["gptteamplan.base_url"]), "/")
	if baseURL == "" {
		baseURL = "http://gptteamplan.tech"
	}

	return Config{
		Enabled: strings.EqualFold(strings.TrimSpace(common.OptionMap["gptteamplan.enabled"]), "true"),
		BaseURL: baseURL,
	}
}

func (c Config) IsReady() bool {
	return c.Enabled && c.BaseURL != ""
}
