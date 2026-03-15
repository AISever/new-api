package externalshop

import "time"

func ApplyPayQueryResult(order *OrderSyncState, resp PayQueryResponse, now time.Time) bool {
	if order == nil || !resp.IsPaid() {
		return false
	}
	changed := false
	if order.Status == "" || order.Status == OrderStatusCreated || order.Status == OrderStatusPendingPayment {
		order.Status = OrderStatusPaidWaitingDelivery
		changed = true
	}
	if order.PaymentConfirmedAt == nil {
		ts := now
		order.PaymentConfirmedAt = &ts
		changed = true
	}
	return changed
}

func ApplyOrderInfoResult(order *OrderSyncState, resp OrderInfoResponse, now time.Time) bool {
	if order == nil || resp.Code != 1 || resp.Data.Sendout != 1 {
		return false
	}
	changed := false
	if order.Status != OrderStatusDelivered {
		order.Status = OrderStatusDelivered
		changed = true
	}
	if order.DeliveryConfirmedAt == nil {
		ts := now
		order.DeliveryConfirmedAt = &ts
		changed = true
	}
	if order.TransactionID != resp.Data.TransactionID {
		order.TransactionID = resp.Data.TransactionID
		changed = true
	}
	if len(resp.Data.Response.Cards) > 0 {
		if !sameCards(order.Cards, resp.Data.Response.Cards) {
			order.Cards = append([]string(nil), resp.Data.Response.Cards...)
			changed = true
		}
	}
	return changed
}

func sameCards(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
