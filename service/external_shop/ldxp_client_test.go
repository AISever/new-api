package externalshop

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestClientFetchBuyerJUUID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/shopApi/common/buyerBlackIframe", r.URL.Path)
		_, _ = w.Write([]byte(`<!DOCTYPE html><script> const juuid = 'PLKyId1mtgihGCoY';window.parent.postMessage({type: 'CloudBuyerBlack', juuid: juuid}, '*');</script>`))
	}))
	defer server.Close()

	client := NewLDXPClient(server.URL, "itools", server.Client())

	juuid, err := client.FetchBuyerJUUID(context.Background())
	require.NoError(t, err)
	require.Equal(t, "PLKyId1mtgihGCoY", juuid)
}

func TestClientQueryAndOrderInfo(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/shopApi/Pay/query":
			_, _ = w.Write([]byte(`{"code":1,"msg":"success","data":null}`))
		case "/shopApi/Order/info":
			_, _ = w.Write([]byte(`{"code":1,"msg":"success","data":{"trade_no":"LD260313KK28J3","status":1,"sendout":1,"transaction_id":"2026031322001421151405812850","success_time":1773379357,"response":{"cards":["test1"],"export_cards_url":"https://pay.ldxp.cn/shopApi/Order/exportCards?trade_no=LD260313KK28J3"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewLDXPClient(server.URL, "itools", server.Client())

	payResp, err := client.QueryOrder(context.Background(), "LD260313KK28J3")
	require.NoError(t, err)
	require.True(t, payResp.IsPaid())

	orderInfo, err := client.GetOrderInfo(context.Background(), "LD260313KK28J3")
	require.NoError(t, err)
	require.Equal(t, "LD260313KK28J3", orderInfo.Data.TradeNo)
	require.Equal(t, 1, orderInfo.Data.Status)
	require.Equal(t, 1, orderInfo.Data.Sendout)
	require.Equal(t, []string{"test1"}, orderInfo.Data.Response.Cards)
}

func TestClientCreateOrderInjectsJUUID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/shopApi/common/buyerBlackIframe":
			_, _ = w.Write([]byte(`<!DOCTYPE html><script> const juuid = 'buyer-black-123';</script>`))
		case "/shopApi/Pay/order":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var payload CreateOrderRequest
			require.NoError(t, common.Unmarshal(body, &payload))
			require.Equal(t, "qnx5gk", payload.GoodsKey)
			require.Equal(t, "13800138000", payload.Contact)
			require.Equal(t, "buyer-black-123", payload.Extend.JUUID)
			_, _ = w.Write([]byte(`{"code":1,"msg":"success","data":{"trade_no":"LD260313KK28J3","total_amount":0.01,"payurl":"https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewLDXPClient(server.URL, "itools", server.Client())
	result, err := client.CreateOrder(context.Background(), CreateOrderRequest{
		GoodsKey:  "qnx5gk",
		Quantity:  1,
		ChannelID: 1,
		Contact:   "13800138000",
	})
	require.NoError(t, err)
	require.Equal(t, "LD260313KK28J3", result.Data.TradeNo)
	require.Equal(t, "https://pay.ldxp.cn/shopApi/Pay/payment?trade_no=LD260313KK28J3", result.Data.PayURL)
}
