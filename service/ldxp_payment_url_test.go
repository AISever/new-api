package service

import "testing"

func TestResolveLdxpPaymentURLPrefersPayloadPayURL(t *testing.T) {
	got := ResolveLdxpPaymentURL(
		"https://pay.ldxp.cn",
		"LD260313KK28J3",
		`{"code":1,"msg":"success","data":{"trade_no":"LD260313KK28J3","total_amount":79,"payurl":"https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3"}}`,
	)

	want := "https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3"
	if got != want {
		t.Fatalf("expected pay url %q, got %q", want, got)
	}
}

func TestResolveLdxpPaymentURLFallsBackToTradeNo(t *testing.T) {
	got := ResolveLdxpPaymentURL(
		"https://pay.ldxp.cn/",
		"LD260313KK28J3",
		`{"code":1,"msg":"success","data":{"trade_no":"LD260313KK28J3","status":1}}`,
	)

	want := "https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3"
	if got != want {
		t.Fatalf("expected fallback pay url %q, got %q", want, got)
	}
}

func TestResolveLdxpPaymentURLEmptyWithoutTradeNo(t *testing.T) {
	got := ResolveLdxpPaymentURL("https://pay.ldxp.cn", "", `{"code":1}`)
	if got != "" {
		t.Fatalf("expected empty pay url, got %q", got)
	}
}
