package externalshop

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestApplyCreateOrderResponseRejectsUpstreamFailure(t *testing.T) {
	t.Parallel()

	order := &model.ExternalShopOrder{
		Status: OrderStatusCreated,
	}
	err := applyCreateOrderResponse(order, CreateOrderResponse{
		BaseResponse: BaseResponse{Code: 0, Msg: "库存不足"},
		Data: CreateOrderData{
			TradeNo:     "",
			TotalAmount: 0,
			PayURL:      "",
		},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "库存不足")
	require.Equal(t, OrderStatusCreated, order.Status)
	require.Empty(t, order.UpstreamTradeNo)
	require.Empty(t, order.PayUrl)
}

func TestApplyCreateOrderResponseRequiresTradeNoAndPayURL(t *testing.T) {
	t.Parallel()

	order := &model.ExternalShopOrder{
		Status: OrderStatusCreated,
	}
	err := applyCreateOrderResponse(order, CreateOrderResponse{
		BaseResponse: BaseResponse{Code: 1, Msg: "success"},
		Data: CreateOrderData{
			TradeNo:     " ",
			TotalAmount: 0.01,
			PayURL:      "",
		},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid upstream order response")
	require.Equal(t, OrderStatusCreated, order.Status)
}

func TestApplyCreateOrderResponseStoresPayableOrder(t *testing.T) {
	t.Parallel()

	order := &model.ExternalShopOrder{
		Status: OrderStatusCreated,
	}
	err := applyCreateOrderResponse(order, CreateOrderResponse{
		BaseResponse: BaseResponse{Code: 1, Msg: "success"},
		Data: CreateOrderData{
			TradeNo:     "LD260315ABC123",
			TotalAmount: 0.01,
			PayURL:      "https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260315ABC123",
		},
	})

	require.NoError(t, err)
	require.Equal(t, OrderStatusPendingPayment, order.Status)
	require.Equal(t, "LD260315ABC123", order.UpstreamTradeNo)
	require.Equal(t, 0.01, order.Amount)
	require.Equal(t, "https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260315ABC123", order.PayUrl)
}

func TestValidatePurchasableGood(t *testing.T) {
	t.Parallel()

	require.EqualError(t, validatePurchasableGood(nil, 1), "goods not found")
	require.EqualError(t, validatePurchasableGood(&model.ExternalShopGood{StockCount: 0}, 1), "goods is out of stock")
	require.EqualError(t, validatePurchasableGood(&model.ExternalShopGood{StockCount: 2}, 3), "quantity exceeds stock 2")
	require.NoError(t, validatePurchasableGood(&model.ExternalShopGood{StockCount: 2, LimitCount: 1}, 2))
	require.NoError(t, validatePurchasableGood(&model.ExternalShopGood{StockCount: 2, LimitCount: 2}, 2))
}

func TestShouldMarkGoodOutOfStockAfterCreateFailure(t *testing.T) {
	t.Parallel()

	require.True(t, shouldMarkGoodOutOfStockAfterCreateFailure("库存不足"))
	require.True(t, shouldMarkGoodOutOfStockAfterCreateFailure("商品已售罄"))
	require.True(t, shouldMarkGoodOutOfStockAfterCreateFailure("out of stock"))
	require.False(t, shouldMarkGoodOutOfStockAfterCreateFailure("network timeout"))
}
