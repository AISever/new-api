package service

import (
	"strings"
	"testing"
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
