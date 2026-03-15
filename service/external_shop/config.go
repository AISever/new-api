package externalshop

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	ProviderLDXP = "ldxp"
)

type Config struct {
	Enabled            bool
	BaseURL            string
	ShopToken          string
	ShopName           string
	AllowedCategoryIDs map[int]struct{}
	AllowedGoodsKeys   map[string]struct{}
}

func GetConfig() Config {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()

	cfg := Config{
		Enabled:   strings.EqualFold(common.OptionMap["external_shop.ldxp.enabled"], "true"),
		BaseURL:   strings.TrimRight(strings.TrimSpace(common.OptionMap["external_shop.ldxp.base_url"]), "/"),
		ShopToken: strings.TrimSpace(common.OptionMap["external_shop.ldxp.shop_token"]),
		ShopName:  strings.TrimSpace(common.OptionMap["external_shop.ldxp.shop_name"]),
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://pay.ldxp.cn"
	}
	cfg.AllowedCategoryIDs = parseIntSet(common.OptionMap["external_shop.ldxp.allowed_category_ids"])
	cfg.AllowedGoodsKeys = parseStringSet(common.OptionMap["external_shop.ldxp.allowed_goods_keys"])
	return cfg
}

func (c Config) IsReady() bool {
	return c.Enabled && c.BaseURL != "" && c.ShopToken != ""
}

func parseIntSet(raw string) map[int]struct{} {
	result := make(map[int]struct{})
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		intValue := common.String2Int(value)
		if intValue <= 0 {
			continue
		}
		result[intValue] = struct{}{}
	}
	return result
}

func parseStringSet(raw string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		result[value] = struct{}{}
	}
	return result
}
