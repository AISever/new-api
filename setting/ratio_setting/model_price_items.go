package ratio_setting

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/types"
)

type ModelPriceItem struct {
	Label      string            `json:"label"`
	Price      float64           `json:"price"`
	Unit       string            `json:"unit,omitempty"`
	Conditions map[string]string `json:"conditions,omitempty"`
}

var modelPriceItemsMap = types.NewRWMap[string, []ModelPriceItem]()
var modelPricingProfilesMap = types.NewRWMap[string, ModelPricingProfile]()

type ModelPricingProfile struct {
	Shape            string           `json:"shape,omitempty"`
	Unit             string           `json:"unit,omitempty"`
	PricePerUnit     float64          `json:"price_per_unit,omitempty"`
	DurationSelector string           `json:"duration_selector,omitempty"`
	BillingMode      string           `json:"billing_mode,omitempty"`
	BillingExpr      string           `json:"billing_expr,omitempty"`
	Items            []ModelPriceItem `json:"items,omitempty"`
}

type ModelPricingProfileApplyResult struct {
	BillingMode map[string]string           `json:"billing_mode"`
	BillingExpr map[string]string           `json:"billing_expr"`
	PriceItems  map[string][]ModelPriceItem `json:"price_items"`
}

func ModelPriceItems2JSONString() string {
	return modelPriceItemsMap.MarshalJSONString()
}

func UpdateModelPriceItemsByJSONString(jsonStr string) error {
	return types.LoadFromJsonStringWithCallback(modelPriceItemsMap, jsonStr, InvalidateExposedDataCache)
}

func ModelPricingProfiles2JSONString() string {
	return modelPricingProfilesMap.MarshalJSONString()
}

func GetModelPricingProfile(name string) (ModelPricingProfile, bool) {
	name = FormatMatchingModelName(name)
	profile, ok := modelPricingProfilesMap.Get(name)
	return profile, ok
}

func ApplyModelPricingProfilesByJSONString(jsonStr string) (ModelPricingProfileApplyResult, error) {
	result := ModelPricingProfileApplyResult{
		BillingMode: make(map[string]string),
		BillingExpr: make(map[string]string),
		PriceItems:  make(map[string][]ModelPriceItem),
	}

	profiles := types.NewRWMap[string, ModelPricingProfile]()
	if err := types.LoadFromJsonString(profiles, jsonStr); err != nil {
		return result, err
	}

	for model, profile := range profiles.ReadAll() {
		profile = normalizeModelPricingProfile(profile)
		if err := validateModelPricingProfile(model, profile); err != nil {
			return result, err
		}
		if len(profile.Items) > 0 {
			result.PriceItems[model] = cloneModelPriceItems(profile.Items)
		}
		if strings.TrimSpace(profile.BillingMode) != "" {
			result.BillingMode[model] = strings.TrimSpace(profile.BillingMode)
		}
		if strings.TrimSpace(profile.BillingExpr) != "" {
			result.BillingExpr[model] = profile.BillingExpr
		}
		profiles.Set(model, profile)
	}

	modelPricingProfilesMap = profiles
	for model, items := range result.PriceItems {
		modelPriceItemsMap.Set(model, cloneModelPriceItems(items))
	}
	billing_setting.MergeBillingConfig(result.BillingMode, result.BillingExpr)
	InvalidateExposedDataCache()
	return result, nil
}

func normalizeModelPricingProfile(profile ModelPricingProfile) ModelPricingProfile {
	if strings.TrimSpace(profile.BillingMode) != "" || strings.TrimSpace(profile.BillingExpr) != "" {
		return profile
	}
	if strings.TrimSpace(profile.Shape) != "per_duration" || strings.TrimSpace(profile.Unit) != "second" {
		return profile
	}
	if profile.PricePerUnit <= 0 || strings.TrimSpace(profile.DurationSelector) == "" {
		return profile
	}
	label := "按秒"
	if len(profile.Items) > 0 && strings.TrimSpace(profile.Items[0].Label) != "" {
		label = strings.TrimSpace(profile.Items[0].Label)
	}
	profile.BillingMode = billing_setting.BillingModePerCallExpr
	profile.BillingExpr = fmt.Sprintf(
		"tier(%s, max(num(param(%s)), 1) * %s)",
		strconv.Quote(label),
		strconv.Quote(strings.TrimSpace(profile.DurationSelector)),
		strconv.FormatFloat(profile.PricePerUnit, 'f', -1, 64),
	)
	return profile
}

func validateModelPricingProfile(model string, profile ModelPricingProfile) error {
	hasItems := len(profile.Items) > 0
	hasBilling := strings.TrimSpace(profile.BillingMode) != "" && strings.TrimSpace(profile.BillingExpr) != ""
	if hasItems && !hasBilling {
		return fmt.Errorf("model %s pricing profile has display items but no real billing mode/expr", model)
	}
	if strings.TrimSpace(profile.BillingMode) != "" && strings.TrimSpace(profile.BillingExpr) == "" {
		return fmt.Errorf("model %s pricing profile has billing_mode but no billing_expr", model)
	}
	if strings.TrimSpace(profile.BillingMode) == "" && strings.TrimSpace(profile.BillingExpr) != "" {
		return fmt.Errorf("model %s pricing profile has billing_expr but no billing_mode", model)
	}
	for i, item := range profile.Items {
		if strings.TrimSpace(item.Label) == "" {
			return fmt.Errorf("model %s pricing profile item %d has empty label", model, i)
		}
		if item.Price < 0 {
			return fmt.Errorf("model %s pricing profile item %d has negative price", model, i)
		}
	}
	return nil
}

func GetModelPriceItems(name string) ([]ModelPriceItem, bool) {
	name = FormatMatchingModelName(name)
	items, ok := modelPriceItemsMap.Get(name)
	if !ok {
		return nil, false
	}
	return cloneModelPriceItems(items), true
}

func GetModelPriceItemsCopy() map[string][]ModelPriceItem {
	source := modelPriceItemsMap.ReadAll()
	copied := make(map[string][]ModelPriceItem, len(source))
	for model, items := range source {
		copied[model] = cloneModelPriceItems(items)
	}
	return copied
}

func cloneModelPriceItems(items []ModelPriceItem) []ModelPriceItem {
	copied := make([]ModelPriceItem, len(items))
	for i, item := range items {
		copied[i] = item
		if item.Conditions != nil {
			copied[i].Conditions = make(map[string]string, len(item.Conditions))
			for k, v := range item.Conditions {
				copied[i].Conditions[k] = v
			}
		}
	}
	return copied
}
