package externalshop

import (
	"time"

	"github.com/QuantumNous/new-api/model"
)

const pendingPaymentExpireAfter = 30 * time.Minute

func ShouldExpireLocalOrder(order *model.ExternalShopOrder, now time.Time) bool {
	if order == nil {
		return false
	}
	switch order.Status {
	case OrderStatusCreated, OrderStatusPendingPayment:
	default:
		return false
	}
	if order.PaymentConfirmedAt > 0 || order.CreatedAt <= 0 {
		return false
	}
	return now.Unix()-order.CreatedAt >= int64(pendingPaymentExpireAfter/time.Second)
}

func ExpireLocalOrderIfTimedOut(order *model.ExternalShopOrder, now time.Time) bool {
	if !ShouldExpireLocalOrder(order, now) {
		return false
	}
	order.Status = OrderStatusExpired
	order.LastSyncAt = now.Unix()
	if order.LastError == "" {
		order.LastError = "payment timeout"
	}
	return true
}
