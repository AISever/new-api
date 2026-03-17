package gptteamplan

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClientRedeemSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/redeem/confirm", r.URL.Path)
		_, _ = w.Write([]byte(`{"success":true,"message":"兑换成功","team_info":{"team_name":"GPT Team 01","subscription_plan":"Team","remaining_seats":2,"seat_count":5,"used_seats":3,"has_warranty":true,"warranty_valid":true,"warranty_expires_at":"2026-04-01 00:00:00","expires_at":"2026-04-01 00:00:00","team_expires_at":"2026-04-10 00:00:00"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	result, err := client.Redeem(context.Background(), "user@example.com", "abc", nil)
	require.NoError(t, err)
	require.Equal(t, "GPT Team 01", result.TeamName)
	require.Equal(t, "Team", result.SubscriptionPlan)
	require.Equal(t, 2, result.RemainingSeats)
	require.Equal(t, 5, result.TotalSeats)
	require.Equal(t, 3, result.UsedSeats)
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
		_, _ = w.Write([]byte(`{"success":true,"has_warranty":true,"warranty_valid":true,"warranty_expires_at":"2026-04-01","can_reuse":false,"original_code":"ABC-123","records":[{"code":"ABC-123","status":"active","used_at":"2026-03-01","email":"user@example.com","team_name":"GPT Team 01","team_status":"active","has_warranty":true,"warranty_valid":true,"warranty_expires_at":"2026-04-01","user_expires_at":"2026-04-01","team_expires_at":"2026-04-10"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	result, err := client.CheckWarranty(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.True(t, result.HasWarranty)
	require.Len(t, result.Records, 1)
	require.Equal(t, "user@example.com", result.Records[0].Email)
	require.Equal(t, "active", result.Records[0].Status)
	require.Equal(t, "2026-03-01", result.Records[0].UsedAt)
	require.Equal(t, "2026-04-01", result.Records[0].WarrantyExpiresAt)
}

func TestClientCheckWarrantyKeepsPendingWarrantyShapeFromUpstream(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/warranty/check", r.URL.Path)
		_, _ = w.Write([]byte(`{"success":true,"has_warranty":true,"warranty_valid":true,"warranty_expires_at":null,"can_reuse":false,"original_code":"ABC-123","records":[{"code":"ABC-123","status":"unused","used_at":null,"team_name":null,"team_status":null,"has_warranty":true,"warranty_valid":true,"warranty_expires_at":null,"user_expires_at":null,"team_expires_at":null,"email":null}],"message":"兑换码尚未被使用","error":null}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	result, err := client.CheckWarranty(context.Background(), "ABC-123")
	require.NoError(t, err)
	require.Equal(t, "兑换码尚未被使用", result.Message)
	require.Len(t, result.Records, 1)
	require.Equal(t, "unused", result.Records[0].Status)
	require.Equal(t, "", result.Records[0].TeamName)
	require.Equal(t, "", result.Records[0].TeamStatus)
	require.Equal(t, "", result.Records[0].UsedAt)
}

func TestClientGetRemainingSeatsFromLandingPage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/", r.URL.Path)
		_, _ = w.Write([]byte(`<html><body><span id="spotsCount" data-target="5">0</span></body></html>`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	result, err := client.GetRemainingSeats(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 5, *result)
}

func TestClientGetRemainingSeatsReturnsBeforeReadingWholeLandingPage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/", r.URL.Path)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, `<html><body><span id="spotsCount" data-target="8">0</span>`)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		time.Sleep(250 * time.Millisecond)
		_, _ = fmt.Fprint(w, strings.Repeat("x", 2048)+`</body></html>`)
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	start := time.Now()
	result, err := client.GetRemainingSeats(context.Background())
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 8, *result)
	require.Less(t, elapsed, 200*time.Millisecond)
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
