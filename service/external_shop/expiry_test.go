package externalshop

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestShouldExpireLocalOrder(t *testing.T) {
	now := time.Unix(1_773_500_000, 0)

	require.True(t, ShouldExpireLocalOrder(&model.ExternalShopOrder{
		Status:    OrderStatusPendingPayment,
		CreatedAt: now.Add(-31 * time.Minute).Unix(),
	}, now))

	require.False(t, ShouldExpireLocalOrder(&model.ExternalShopOrder{
		Status:    OrderStatusPendingPayment,
		CreatedAt: now.Add(-10 * time.Minute).Unix(),
	}, now))

	require.False(t, ShouldExpireLocalOrder(&model.ExternalShopOrder{
		Status:             OrderStatusPendingPayment,
		CreatedAt:          now.Add(-31 * time.Minute).Unix(),
		PaymentConfirmedAt: now.Unix(),
	}, now))
}

func TestExpireLocalOrderIfTimedOut(t *testing.T) {
	now := time.Unix(1_773_500_000, 0)
	order := &model.ExternalShopOrder{
		Status:    OrderStatusCreated,
		CreatedAt: now.Add(-31 * time.Minute).Unix(),
	}

	changed := ExpireLocalOrderIfTimedOut(order, now)

	require.True(t, changed)
	require.Equal(t, OrderStatusExpired, order.Status)
	require.Equal(t, now.Unix(), order.LastSyncAt)
	require.Equal(t, "payment timeout", order.LastError)
}
