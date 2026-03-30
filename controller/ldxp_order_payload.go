package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

type ldxpOrderPayload struct {
	TradeNo         string  `json:"trade_no"`
	ProviderTradeNo string  `json:"provider_trade_no"`
	Status          string  `json:"status"`
	PaymentMethod   string  `json:"payment_method"`
	PaymentURL      string  `json:"payment_url"`
	Amount          int64   `json:"amount,omitempty"`
	Money           float64 `json:"money"`
	PlanId          int     `json:"plan_id,omitempty"`
	PlanTitle       string  `json:"plan_title,omitempty"`
}

func buildLdxpTopupOrderPayload(topUp *model.TopUp) ldxpOrderPayload {
	if topUp == nil {
		return ldxpOrderPayload{}
	}
	cfg := service.GetLdxpConfig()
	return ldxpOrderPayload{
		TradeNo:         topUp.TradeNo,
		ProviderTradeNo: topUp.ProviderTradeNo,
		Status:          topUp.Status,
		PaymentMethod:   topUp.PaymentMethod,
		PaymentURL: service.ResolveLdxpPaymentURL(
			cfg.BaseURL,
			topUp.ProviderTradeNo,
			topUp.ProviderPayload,
		),
		Amount: topUp.Amount,
		Money:  topUp.Money,
	}
}

func buildLdxpSubscriptionOrderPayload(order *model.SubscriptionOrder, planTitle string) ldxpOrderPayload {
	if order == nil {
		return ldxpOrderPayload{}
	}
	cfg := service.GetLdxpConfig()
	return ldxpOrderPayload{
		TradeNo:         order.TradeNo,
		ProviderTradeNo: order.ProviderTradeNo,
		Status:          order.Status,
		PaymentMethod:   order.PaymentMethod,
		PaymentURL: service.ResolveLdxpPaymentURL(
			cfg.BaseURL,
			order.ProviderTradeNo,
			order.ProviderPayload,
		),
		Money:     order.Money,
		PlanId:    order.PlanId,
		PlanTitle: strings.TrimSpace(planTitle),
	}
}
