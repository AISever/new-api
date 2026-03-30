package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
)

func TestBuildLdxpTopupOrderPayloadIncludesPaymentURL(t *testing.T) {
	topUp := &model.TopUp{
		TradeNo:         "LDXPLOCAL001",
		ProviderTradeNo: "LD260313KK28J3",
		PaymentMethod:   "ldxp",
		Status:          "pending",
		Amount:          500000,
		Money:           5,
		ProviderPayload: `{"code":1,"data":{"payurl":"https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3"}}`,
	}

	payload := buildLdxpTopupOrderPayload(topUp)
	if payload.PaymentURL != "https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3" {
		t.Fatalf("expected payment_url to be preserved, got %q", payload.PaymentURL)
	}
	if payload.Amount != 500000 {
		t.Fatalf("expected amount to be included, got %d", payload.Amount)
	}
}

func TestBuildLdxpSubscriptionOrderPayloadIncludesPlanTitleAndFallbackURL(t *testing.T) {
	order := &model.SubscriptionOrder{
		TradeNo:         "SUBLDXP001",
		ProviderTradeNo: "LD260313KK28J3",
		PaymentMethod:   "ldxp",
		Status:          "pending",
		Money:           79,
		PlanId:          2,
		ProviderPayload: `{"code":1,"data":{"trade_no":"LD260313KK28J3","status":1}}`,
	}

	payload := buildLdxpSubscriptionOrderPayload(order, "LDXP 包月 Codex Pro")
	if payload.PaymentURL != "https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3" {
		t.Fatalf("expected fallback payment_url, got %q", payload.PaymentURL)
	}
	if payload.PlanTitle != "LDXP 包月 Codex Pro" {
		t.Fatalf("expected plan title to be included, got %q", payload.PlanTitle)
	}
}
