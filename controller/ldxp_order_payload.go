package controller

import (
	"context"
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
	QRCode          string  `json:"qr_code,omitempty"`
	QRImageURL      string  `json:"qr_img_url,omitempty"`
	Amount          float64 `json:"amount,omitempty"`
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
		Amount: topUp.ResolveAmountValue(),
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

func enrichLdxpOrderPayloadWithCheckoutData(ctx context.Context, payload *ldxpOrderPayload, providerPayload string) (string, bool) {
	if payload == nil || payload.Status != "pending" || strings.TrimSpace(payload.PaymentURL) == "" {
		return providerPayload, false
	}
	checkout := service.ResolveLdxpCheckoutData(providerPayload)
	updatedProviderPayload := providerPayload
	updated := false
	if checkout.QRCode == "" && checkout.QRImageURL == "" {
		resolved, err := service.FetchLdxpCheckoutData(ctx, payload.PaymentURL)
		if err == nil {
			checkout = resolved
			mergedProviderPayload := service.MergeLdxpCheckoutIntoProviderPayload(providerPayload, checkout)
			if strings.TrimSpace(mergedProviderPayload) != "" && mergedProviderPayload != providerPayload {
				updatedProviderPayload = mergedProviderPayload
				updated = true
			}
		}
	}
	payload.QRCode = strings.TrimSpace(checkout.QRCode)
	payload.QRImageURL = strings.TrimSpace(checkout.QRImageURL)
	return updatedProviderPayload, updated
}
