package externalshop

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestSyncOrdersWithRefresherSkipsIneligibleAndCountsFailures(t *testing.T) {
	t.Parallel()

	now := time.Unix(1773381000, 0)
	orders := []*model.ExternalShopOrder{
		{LocalTradeNo: "A", Status: OrderStatusPendingPayment},
		{LocalTradeNo: "B", Status: OrderStatusDelivered},
		{LocalTradeNo: "C", Status: OrderStatusPaidWaitingDelivery, LastSyncAt: now.Add(-2 * time.Minute).Unix()},
		{LocalTradeNo: "D", Status: OrderStatusPendingPayment, LastSyncAt: now.Add(-10 * time.Second).Unix()},
	}

	var refreshed []string
	summary := syncOrdersWithRefresher(
		context.Background(),
		orders,
		now,
		30*time.Second,
		func(ctx context.Context, order *model.ExternalShopOrder) (*model.ExternalShopOrder, error) {
			refreshed = append(refreshed, order.LocalTradeNo)
			if order.LocalTradeNo == "C" {
				return nil, errors.New("upstream timeout")
			}
			return order, nil
		},
	)

	require.Equal(t, 4, summary.Scanned)
	require.Equal(t, 2, summary.Attempted)
	require.Equal(t, 1, summary.Succeeded)
	require.Equal(t, 1, summary.Failed)
	require.ElementsMatch(t, []string{"A", "C"}, refreshed)
}
