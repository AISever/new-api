package gptteamplan

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientRedeemSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/redeem/confirm", r.URL.Path)
		_, _ = w.Write([]byte(`{"success":true,"message":"兑换成功","team_info":{"team_name":"GPT Team 01","subscription_plan":"Team","has_warranty":true,"warranty_valid":true,"warranty_expires_at":"2026-04-01 00:00:00","expires_at":"2026-04-01 00:00:00","team_expires_at":"2026-04-10 00:00:00"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	result, err := client.Redeem(context.Background(), "user@example.com", "abc", nil)
	require.NoError(t, err)
	require.Equal(t, "GPT Team 01", result.TeamName)
	require.Equal(t, "Team", result.SubscriptionPlan)
	require.True(t, result.HasWarranty)
	require.True(t, result.WarrantyValid)
}

func TestClientRedeemMapsBusinessError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"detail":"兑换码不存在"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	result, err := client.Redeem(context.Background(), "user@example.com", "bad", nil)
	require.Nil(t, result)
	require.EqualError(t, err, "兑换码不存在")
}

func TestClientCheckWarrantyParsesRecords(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/warranty/check", r.URL.Path)
		_, _ = w.Write([]byte(`{"success":true,"has_warranty":true,"warranty_valid":true,"warranty_expires_at":"2026-04-01","can_reuse":false,"original_code":"ABC-123","records":[{"code":"ABC-123","email":"user@example.com","team_name":"GPT Team 01","team_status":"active","has_warranty":true,"warranty_valid":true,"user_expires_at":"2026-04-01","team_expires_at":"2026-04-10"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	result, err := client.CheckWarranty(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.True(t, result.HasWarranty)
	require.Len(t, result.Records, 1)
	require.Equal(t, "user@example.com", result.Records[0].Email)
}

func TestClientHandlesNonJSONResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	result, err := client.CheckWarranty(context.Background(), "ABC-123")
	require.Nil(t, result)
	require.EqualError(t, err, "上游返回格式错误")
}

func TestClientMapsNon2xxJSONError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"detail":"服务繁忙"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	result, err := client.CheckWarranty(context.Background(), "ABC-123")
	require.Nil(t, result)
	require.EqualError(t, err, "服务繁忙")
}
