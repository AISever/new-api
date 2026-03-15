package externalshop

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSyncOrderProgression(t *testing.T) {
	t.Parallel()

	order := &OrderSyncState{
		Status: OrderStatusPendingPayment,
	}

	changed := ApplyPayQueryResult(order, PayQueryResponse{
		BaseResponse: BaseResponse{Code: 0, Msg: "not pay"},
	}, time.Unix(1773379354, 0))
	require.False(t, changed)
	require.Equal(t, OrderStatusPendingPayment, order.Status)

	changed = ApplyPayQueryResult(order, PayQueryResponse{
		BaseResponse: BaseResponse{Code: 1, Msg: "success"},
	}, time.Unix(1773379359, 0))
	require.True(t, changed)
	require.Equal(t, OrderStatusPaidWaitingDelivery, order.Status)
	require.NotNil(t, order.PaymentConfirmedAt)

	changed = ApplyOrderInfoResult(order, OrderInfoResponse{
		BaseResponse: BaseResponse{Code: 1, Msg: "success"},
		Data: OrderInfoData{
			TradeNo:       "LD260313KK28J3",
			Status:        1,
			Sendout:       1,
			TransactionID: "2026031322001421151405812850",
			SuccessTime:   1773379357,
			Response: OrderInfoDeliverResponse{
				Cards: []string{"test1"},
			},
		},
	}, time.Unix(1773379361, 0))
	require.True(t, changed)
	require.Equal(t, OrderStatusDelivered, order.Status)
	require.NotNil(t, order.DeliveryConfirmedAt)
	require.Equal(t, []string{"test1"}, order.Cards)
}
