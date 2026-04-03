package operation_setting

import "testing"

func TestNormalizeLdxpTopupProducts(t *testing.T) {
	products := NormalizeLdxpTopupProducts([]LdxpTopupProduct{
		{Amount: 100, GoodsKey: " goods-a ", Label: " A ", Enabled: true, SortOrder: 20},
		{Amount: 0, GoodsKey: "goods-b", Label: "B", Enabled: true, SortOrder: 10},
		{Amount: 50, GoodsKey: "", Label: "C", Enabled: true, SortOrder: 30},
		{Amount: 100, GoodsKey: "goods-c", Label: "duplicate amount", Enabled: true, SortOrder: 40},
		{Amount: 20, GoodsKey: "goods-d", Label: "", Enabled: false, SortOrder: 5},
		{Amount: 10, GoodsKey: " goods-e ", Label: "", Enabled: true, SortOrder: 1},
	})

	if len(products) != 3 {
		t.Fatalf("expected 3 normalized products, got %d", len(products))
	}

	if products[0].Amount != 10 || products[0].GoodsKey != "goods-e" || products[0].Label != "10" {
		t.Fatalf("unexpected first normalized product: %+v", products[0])
	}
	if products[1].Amount != 20 || products[1].GoodsKey != "goods-d" || products[1].Label != "20" {
		t.Fatalf("unexpected second normalized product: %+v", products[1])
	}
	if products[2].Amount != 100 || products[2].GoodsKey != "goods-a" || products[2].Label != "A" {
		t.Fatalf("unexpected third normalized product: %+v", products[2])
	}
}

func TestEnabledLdxpTopupProductsFiltersDisabledItems(t *testing.T) {
	setting := &PaymentSetting{
		LdxpTopupProducts: []LdxpTopupProduct{
			{Amount: 100, GoodsKey: "goods-a", Label: "100", Enabled: true, SortOrder: 2},
			{Amount: 50, GoodsKey: "goods-b", Label: "50", Enabled: false, SortOrder: 1},
			{Amount: 20, GoodsKey: "goods-c", Label: "20", Enabled: true, SortOrder: 3},
		},
	}

	products := setting.GetEnabledLdxpTopupProducts()
	if len(products) != 2 {
		t.Fatalf("expected 2 enabled products, got %d", len(products))
	}
	if products[0].Amount != 100 || products[1].Amount != 20 {
		t.Fatalf("unexpected enabled products: %+v", products)
	}
}

func TestValidateLdxpTopupProductsJSON(t *testing.T) {
	products, err := ValidateLdxpTopupProductsJSON(`[
		{
			"amount": 10,
			"goods_key": "goods-10",
			"label": "10 元",
			"enabled": true,
			"sort_order": 10
		},
		{
			"amount": 20,
			"goods_key": "goods-20",
			"label": "20 元",
			"enabled": false,
			"sort_order": 20
		}
	]`)
	if err != nil {
		t.Fatalf("expected valid ldxp topup products json, got error: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(products))
	}
}

func TestValidateLdxpTopupProductsJSONSupportsDecimalAmount(t *testing.T) {
	products, err := ValidateLdxpTopupProductsJSON(`[
		{
			"amount": 0.1,
			"goods_key": "goods-01",
			"label": "0.1 元",
			"enabled": true,
			"sort_order": 1
		},
		{
			"amount": 1.5,
			"goods_key": "goods-15",
			"label": "1.5 元",
			"enabled": true,
			"sort_order": 2
		}
	]`)
	if err != nil {
		t.Fatalf("expected decimal ldxp topup products json to be valid, got error: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(products))
	}
	if products[0].Amount != 0.1 || products[1].Amount != 1.5 {
		t.Fatalf("expected decimal amounts to be preserved, got %+v", products)
	}
}

func TestValidateLdxpTopupProductsJSONRequiresFields(t *testing.T) {
	_, err := ValidateLdxpTopupProductsJSON(`[
		{
			"amount": 10,
			"goods_key": "goods-10",
			"enabled": true,
			"sort_order": 10
		}
	]`)
	if err == nil {
		t.Fatal("expected validation error for missing label")
	}
}

func TestValidateLdxpTopupProductsRejectsDuplicateDecimalAmounts(t *testing.T) {
	_, err := ValidateLdxpTopupProducts([]LdxpTopupProduct{
		{Amount: 0.1, GoodsKey: "goods-a", Label: "0.1 A", Enabled: true, SortOrder: 1},
		{Amount: 0.10, GoodsKey: "goods-b", Label: "0.1 B", Enabled: true, SortOrder: 2},
	})
	if err == nil {
		t.Fatal("expected duplicate decimal amount validation error")
	}
}
