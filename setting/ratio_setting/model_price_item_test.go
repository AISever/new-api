package ratio_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/billing_setting"
)

func TestModelPriceItemsRoundTrip(t *testing.T) {
	raw := `{"kling-audio":[{"label":"文生音效","price":0.425},{"label":"语音合成","price":0.085}]}`

	if err := UpdateModelPriceItemsByJSONString(raw); err != nil {
		t.Fatalf("UpdateModelPriceItemsByJSONString returned error: %v", err)
	}

	items, ok := GetModelPriceItems("kling-audio")
	if !ok {
		t.Fatal("expected kling-audio price items")
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 price items, got %d", len(items))
	}
	if items[0].Label != "文生音效" || items[0].Price != 0.425 {
		t.Fatalf("unexpected first price item: %#v", items[0])
	}
	if items[1].Label != "语音合成" || items[1].Price != 0.085 {
		t.Fatalf("unexpected second price item: %#v", items[1])
	}
}

func TestModelPricingProfilesApplyPerCallItemsAndBillingExpr(t *testing.T) {
	raw := `{
		"kling-audio": {
			"billing_mode": "per_call_expr",
			"billing_expr": "param(\"metadata.scenario\") == \"speech\" ? tier(\"语音合成\", 0.085) : tier(\"音效\", 0.425)",
			"items": [
				{"label": "文生音效", "price": 0.425, "unit": "次", "conditions": {"metadata.scenario": "effect"}},
				{"label": "语音合成", "price": 0.085, "unit": "次", "conditions": {"metadata.scenario": "speech"}}
			]
		}
	}`

	result, err := ApplyModelPricingProfilesByJSONString(raw)
	if err != nil {
		t.Fatalf("ApplyModelPricingProfilesByJSONString returned error: %v", err)
	}

	items, ok := GetModelPriceItems("kling-audio")
	if !ok {
		t.Fatal("expected kling-audio price items")
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 price items, got %d", len(items))
	}
	if items[0].Label != "文生音效" || items[0].Price != 0.425 || items[0].Unit != "次" {
		t.Fatalf("unexpected first price item: %#v", items[0])
	}
	if items[0].Conditions["metadata.scenario"] != "effect" {
		t.Fatalf("expected first item condition to be preserved, got %#v", items[0].Conditions)
	}
	if result.BillingMode["kling-audio"] != "per_call_expr" {
		t.Fatalf("unexpected billing mode output: %#v", result.BillingMode)
	}
	if result.BillingExpr["kling-audio"] == "" {
		t.Fatalf("expected billing expr output")
	}
	if billing_setting.GetBillingMode("kling-audio") != "per_call_expr" {
		t.Fatalf("expected billing setting to be updated")
	}
	if expr, ok := billing_setting.GetBillingExpr("kling-audio"); !ok || expr == "" {
		t.Fatalf("expected billing expression to be stored")
	}
}

func TestModelPricingProfilesRejectDisplayOnlyComplexProfile(t *testing.T) {
	raw := `{
		"kling-motion-control": {
			"shape": "per_duration_matrix",
			"items": [
				{"label": "V2.6 std 720P", "price": 0.85, "unit": "秒", "conditions": {"version": "2.6", "mode": "std", "resolution": "720P"}}
			]
		}
	}`

	if _, err := ApplyModelPricingProfilesByJSONString(raw); err == nil {
		t.Fatal("expected complex display-only profile to be rejected")
	}
}

func TestModelPricingProfilesGeneratePerDurationExpr(t *testing.T) {
	raw := `{
		"viduq2": {
			"shape": "per_duration",
			"unit": "second",
			"price_per_unit": 0.12,
			"duration_selector": "duration",
			"items": [
				{"label": "按秒", "price": 0.12, "unit": "秒"}
			]
		}
	}`

	result, err := ApplyModelPricingProfilesByJSONString(raw)
	if err != nil {
		t.Fatalf("ApplyModelPricingProfilesByJSONString returned error: %v", err)
	}

	if result.BillingMode["viduq2"] != "per_call_expr" {
		t.Fatalf("expected per_call_expr mode, got %#v", result.BillingMode)
	}
	want := `tier("按秒", max(num(param("duration")), 1) * 0.12)`
	if result.BillingExpr["viduq2"] != want {
		t.Fatalf("unexpected generated expr:\n got: %s\nwant: %s", result.BillingExpr["viduq2"], want)
	}
}
