package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func TestFetchBuyerJUUIDParsesIframeResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/shopApi/common/buyerBlackIframe" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`<script>const juuid = 'buyer-juuid-001';</script>`))
	}))
	defer server.Close()

	client := &LDXPClient{
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	juuid, err := client.FetchBuyerJUUID(context.Background())
	if err != nil {
		t.Fatalf("expected juuid to be parsed, got error: %v", err)
	}
	if juuid != "buyer-juuid-001" {
		t.Fatalf("unexpected juuid: %s", juuid)
	}
}

func TestParseLdxpShopInputSupportsFullShopURL(t *testing.T) {
	baseURL, token, err := ParseLdxpShopInput("https://pay.ldxp.cn", "https://pay.ldxp.cn/shop/itools")
	if err != nil {
		t.Fatalf("expected full shop url to parse, got error: %v", err)
	}
	if baseURL != "https://pay.ldxp.cn" {
		t.Fatalf("unexpected base url: %s", baseURL)
	}
	if token != "itools" {
		t.Fatalf("unexpected token: %s", token)
	}
}

func TestParseLdxpShopInputSupportsRawToken(t *testing.T) {
	baseURL, token, err := ParseLdxpShopInput("https://pay.ldxp.cn", "itools")
	if err != nil {
		t.Fatalf("expected raw token to parse, got error: %v", err)
	}
	if baseURL != "https://pay.ldxp.cn" {
		t.Fatalf("unexpected base url: %s", baseURL)
	}
	if token != "itools" {
		t.Fatalf("unexpected token: %s", token)
	}
}

func TestBuildSuggestedLdxpTopupProductsFromCatalog(t *testing.T) {
	products := BuildSuggestedLdxpTopupProducts([]LdxpGoodsItem{
		{
			GoodsKey: "krqmsq",
			Name:     "充值10r",
			Price:    10,
			Category: LdxpGoodsCategory{Name: "API充值"},
		},
		{
			GoodsKey: "o534i3",
			Name:     "充值20r",
			Price:    20,
			Category: LdxpGoodsCategory{Name: "API充值"},
		},
		{
			GoodsKey: "qpyvi2",
			Name:     "套餐24h卡",
			Price:    3.9,
			Category: LdxpGoodsCategory{Name: "套餐购买"},
		},
	})

	if len(products) != 2 {
		t.Fatalf("expected 2 topup products, got %d", len(products))
	}
	if products[0].Amount != 10 || products[0].GoodsKey != "krqmsq" {
		t.Fatalf("unexpected first product: %+v", products[0])
	}
	if products[1].Amount != 20 || products[1].GoodsKey != "o534i3" {
		t.Fatalf("unexpected second product: %+v", products[1])
	}
	if !products[0].Enabled || !products[1].Enabled {
		t.Fatal("expected imported topup products to default to enabled")
	}
}

func TestResolveLdxpTopupSettlementAmountUsesProductFaceValue(t *testing.T) {
	product := operation_setting.LdxpTopupProduct{
		Amount:   5,
		GoodsKey: "827eyl",
		Label:    "充值5r",
		Enabled:  true,
	}

	got := ResolveLdxpTopupSettlementAmount(product)
	if got != 5 {
		t.Fatalf("expected settlement amount 5, got %v", got)
	}
}

func TestResolveLdxpTopupGrantedAmountUsesConfiguredAmount(t *testing.T) {
	product := operation_setting.LdxpTopupProduct{
		Amount:   20,
		GoodsKey: "o534i3",
		Label:    "充值20r",
		Enabled:  true,
	}

	got := ResolveLdxpTopupGrantedAmount(product)
	if got != 20 {
		t.Fatalf("expected granted amount 20, got %d", got)
	}
}
