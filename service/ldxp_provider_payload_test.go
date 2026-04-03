package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestBuildLdxpProviderPayloadIncludesOrderInfoCards(t *testing.T) {
	payload := BuildLdxpProviderPayload(
		LdxpPayQueryResponse{
			LdxpBaseResponse: LdxpBaseResponse{Code: 1, Msg: "success"},
			Data: map[string]interface{}{
				"trade_no": "LD260313KK28J3",
				"payurl":   "https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3",
			},
		},
		&LdxpOrderInfoResponse{
			LdxpBaseResponse: LdxpBaseResponse{Code: 1, Msg: "success"},
			Data: LdxpOrderInfoData{
				TradeNo: "LD260313KK28J3",
				Response: LdxpOrderInfoDeliverResponse{
					Cards:          []string{"card-001", "card-002"},
					ExportCardsURL: "https://pay.ldxp.cn/shopApi/Order/exportCards?trade_no=LD260313KK28J3",
				},
			},
		},
	)

	if !strings.Contains(payload, `"cards":["card-001","card-002"]`) {
		t.Fatalf("expected payload to include cards, got %s", payload)
	}
	if !strings.Contains(payload, `"payurl":"https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3"`) {
		t.Fatalf("expected payload to preserve payurl, got %s", payload)
	}
}

func TestResolveLdxpPaymentURLSupportsMergedProviderPayload(t *testing.T) {
	payload := `{"query":{"code":1,"msg":"success","data":{"trade_no":"LD260313KK28J3","payurl":"https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3"}},"order_info":{"code":1,"msg":"success","data":{"trade_no":"LD260313KK28J3","response":{"cards":["card-001"]}}}}`

	got := ResolveLdxpPaymentURL("https://pay.ldxp.cn", "LD260313KK28J3", payload)
	want := "https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3"
	if got != want {
		t.Fatalf("expected pay url %q, got %q", want, got)
	}
}

func TestMergeLdxpCheckoutIntoProviderPayloadAddsCheckoutToPlainCreatePayload(t *testing.T) {
	payload := `{"code":1,"msg":"success","data":{"trade_no":"LD260313KK28J3","payurl":"https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3"}}`

	merged := MergeLdxpCheckoutIntoProviderPayload(payload, LdxpCheckoutData{
		QRCode:     "https://qr.alipay.com/demo-code",
		QRImageURL: "https://mobilecodec.alipay.com/show.htm?code=demo-code",
	})

	if !strings.Contains(merged, `"checkout":{"qr_code":"https://qr.alipay.com/demo-code","qr_img_url":"https://mobilecodec.alipay.com/show.htm?code=demo-code"}`) {
		t.Fatalf("expected merged payload to include checkout, got %s", merged)
	}
	if !strings.Contains(merged, `"payurl":"https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3"`) {
		t.Fatalf("expected merged payload to preserve original payurl, got %s", merged)
	}

	var decoded struct {
		Checkout LdxpCheckoutData `json:"checkout"`
		Data     struct {
			PayURL string `json:"payurl"`
		} `json:"data"`
	}
	if err := common.UnmarshalJsonStr(merged, &decoded); err != nil {
		t.Fatalf("expected merged payload to remain valid json, got error %v", err)
	}
	if decoded.Checkout.QRCode != "https://qr.alipay.com/demo-code" {
		t.Fatalf("expected qr code to be merged, got %q", decoded.Checkout.QRCode)
	}
	if decoded.Data.PayURL != "https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3" {
		t.Fatalf("expected pay url to stay intact, got %q", decoded.Data.PayURL)
	}
}

func TestMergeLdxpCheckoutIntoProviderPayloadKeepsExistingMergedSections(t *testing.T) {
	payload := `{"query":{"code":1,"msg":"success","data":{"trade_no":"LD260313KK28J3","payurl":"https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3"}},"order_info":{"code":1,"msg":"success","data":{"trade_no":"LD260313KK28J3","response":{"cards":["card-001"]}}}}`

	merged := MergeLdxpCheckoutIntoProviderPayload(payload, LdxpCheckoutData{
		QRCode:     "https://qr.alipay.com/demo-code",
		QRImageURL: "https://mobilecodec.alipay.com/show.htm?code=demo-code",
	})

	var decoded struct {
		Checkout  LdxpCheckoutData `json:"checkout"`
		OrderInfo struct {
			Data struct {
				Response struct {
					Cards []string `json:"cards"`
				} `json:"response"`
			} `json:"data"`
		} `json:"order_info"`
		Query struct {
			Data struct {
				PayURL string `json:"payurl"`
			} `json:"data"`
		} `json:"query"`
	}
	if err := common.UnmarshalJsonStr(merged, &decoded); err != nil {
		t.Fatalf("expected merged payload to remain valid json, got error %v", err)
	}
	if len(decoded.OrderInfo.Data.Response.Cards) != 1 || decoded.OrderInfo.Data.Response.Cards[0] != "card-001" {
		t.Fatalf("expected merged payload to keep order info cards, got %#v", decoded.OrderInfo.Data.Response.Cards)
	}
	if decoded.Query.Data.PayURL != "https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3" {
		t.Fatalf("expected merged payload to keep query pay url, got %q", decoded.Query.Data.PayURL)
	}
	if decoded.Checkout.QRImageURL != "https://mobilecodec.alipay.com/show.htm?code=demo-code" {
		t.Fatalf("expected merged payload to include checkout qr image url, got %q", decoded.Checkout.QRImageURL)
	}
}

func TestPreserveLdxpCheckoutInProviderPayloadCarriesForwardCachedCheckout(t *testing.T) {
	existingPayload := `{"query":{"code":1,"msg":"success","data":{"trade_no":"LD260313KK28J3","payurl":"https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3"}},"checkout":{"qr_code":"https://qr.alipay.com/demo-code","qr_img_url":"https://mobilecodec.alipay.com/show.htm?code=demo-code"}}`
	nextPayload := `{"query":{"code":1,"msg":"success","data":{"trade_no":"LD260313KK28J3","payurl":"https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3&refresh=1"}}}`

	preserved := PreserveLdxpCheckoutInProviderPayload(nextPayload, existingPayload)

	var decoded struct {
		Checkout LdxpCheckoutData `json:"checkout"`
		Query    struct {
			Data struct {
				PayURL string `json:"payurl"`
			} `json:"data"`
		} `json:"query"`
	}
	if err := common.UnmarshalJsonStr(preserved, &decoded); err != nil {
		t.Fatalf("expected preserved payload to remain valid json, got error %v", err)
	}
	if decoded.Query.Data.PayURL != "https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3&refresh=1" {
		t.Fatalf("expected latest pay url to be kept, got %q", decoded.Query.Data.PayURL)
	}
	if decoded.Checkout.QRImageURL != "https://mobilecodec.alipay.com/show.htm?code=demo-code" {
		t.Fatalf("expected cached checkout to survive payload rebuild, got %q", decoded.Checkout.QRImageURL)
	}
}
