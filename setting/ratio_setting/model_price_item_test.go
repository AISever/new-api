package ratio_setting

import "testing"

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
