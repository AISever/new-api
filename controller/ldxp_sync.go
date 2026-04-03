package controller

import (
	"context"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

func syncPendingLdxpTopUp(ctx context.Context, topUp *model.TopUp) *model.TopUp {
	if topUp == nil || topUp.PaymentMethod != "ldxp" || topUp.Status != common.TopUpStatusPending || strings.TrimSpace(topUp.ProviderTradeNo) == "" {
		return topUp
	}

	cfg := service.GetLdxpConfig()
	if !cfg.IsReady() {
		return topUp
	}

	client := service.NewLDXPClient(cfg)
	queryResp, err := client.QueryOrder(ctx, topUp.ProviderTradeNo)
	if err != nil {
		return topUp
	}

	var orderInfo *service.LdxpOrderInfoResponse
	if queryResp.IsPaid() {
		if infoResp, infoErr := client.GetOrderInfo(ctx, topUp.ProviderTradeNo); infoErr == nil && infoResp.Code == 1 {
			orderInfo = &infoResp
		}
	}

	payload := service.BuildLdxpProviderPayload(queryResp, orderInfo)
	payload = service.PreserveLdxpCheckoutInProviderPayload(payload, topUp.ProviderPayload)
	payload = service.PreserveLdxpTopupAmountInProviderPayload(payload, topUp.ProviderPayload)
	topUp.ProviderPayload = payload
	if queryResp.IsPaid() {
		LockOrder(topUp.TradeNo)
		rechargeErr := model.RechargeLdxp(topUp.TradeNo, payload)
		UnlockOrder(topUp.TradeNo)
		if rechargeErr == nil {
			return model.GetTopUpByTradeNo(topUp.TradeNo)
		}
		return topUp
	}

	_ = topUp.Update()
	return topUp
}

func syncPendingLdxpSubscriptionOrder(ctx context.Context, order *model.SubscriptionOrder) *model.SubscriptionOrder {
	if order == nil || order.PaymentMethod != "ldxp" || order.Status != common.TopUpStatusPending || strings.TrimSpace(order.ProviderTradeNo) == "" {
		return order
	}

	cfg := service.GetLdxpConfig()
	if !cfg.IsReady() {
		return order
	}

	client := service.NewLDXPClient(cfg)
	queryResp, err := client.QueryOrder(ctx, order.ProviderTradeNo)
	if err != nil {
		return order
	}

	var orderInfo *service.LdxpOrderInfoResponse
	if queryResp.IsPaid() {
		if infoResp, infoErr := client.GetOrderInfo(ctx, order.ProviderTradeNo); infoErr == nil && infoResp.Code == 1 {
			orderInfo = &infoResp
		}
	}

	payload := service.BuildLdxpProviderPayload(queryResp, orderInfo)
	payload = service.PreserveLdxpCheckoutInProviderPayload(payload, order.ProviderPayload)
	order.ProviderPayload = payload
	if queryResp.IsPaid() {
		LockOrder(order.TradeNo)
		completeErr := model.CompleteSubscriptionOrder(order.TradeNo, payload)
		UnlockOrder(order.TradeNo)
		if completeErr == nil {
			return model.GetSubscriptionOrderByTradeNo(order.TradeNo)
		}
		return order
	}

	_ = order.Update()
	return order
}
