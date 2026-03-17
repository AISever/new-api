package controller

import (
	stdctx "context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetGPTTeamPlanStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/gptteamplan/status", nil)
	setGPTTeamPlanOptionForTest("gptteamplan.enabled", "true")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><span id="spotsCount" data-target="4">0</span></body></html>`))
	}))
	defer server.Close()
	setGPTTeamPlanOptionForTest("gptteamplan.base_url", server.URL)
	defer setGPTTeamPlanOptionForTest("gptteamplan.enabled", "false")

	GetGPTTeamPlanStatus(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"enabled":true`)
	require.Contains(t, recorder.Body.String(), `"remaining_seats":4`)
}

func TestGetGPTTeamPlanStatusForceRefreshBypassesCache(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/gptteamplan/status?refresh=1", nil)
	setGPTTeamPlanOptionForTest("gptteamplan.enabled", "true")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><span id="spotsCount" data-target="9">0</span></body></html>`))
	}))
	defer server.Close()
	setGPTTeamPlanOptionForTest("gptteamplan.base_url", server.URL)
	defer setGPTTeamPlanOptionForTest("gptteamplan.enabled", "false")

	cachedSeats := 3
	setCachedGPTTeamPlanRemainingSeats(&cachedSeats)
	cachedGPTTeamPlanStatus.mu.Lock()
	cachedGPTTeamPlanStatus.fetchedAt = time.Now().Add(-time.Minute)
	cachedGPTTeamPlanStatus.lastAttemptAt = time.Now().Add(-time.Minute)
	cachedGPTTeamPlanStatus.mu.Unlock()

	GetGPTTeamPlanStatus(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"remaining_seats":9`)
}

func TestGetGPTTeamPlanStatusForceRefreshSkipsUpstreamWhenCacheIsFresh(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/gptteamplan/status?refresh=1", nil)
	setGPTTeamPlanOptionForTest("gptteamplan.enabled", "true")
	setGPTTeamPlanOptionForTest("gptteamplan.base_url", "http://gptteamplan.tech")
	defer setGPTTeamPlanOptionForTest("gptteamplan.enabled", "false")

	cachedSeats := 6
	setCachedGPTTeamPlanRemainingSeats(&cachedSeats)
	cachedGPTTeamPlanStatus.mu.Lock()
	cachedGPTTeamPlanStatus.fetchedAt = time.Now()
	cachedGPTTeamPlanStatus.mu.Unlock()

	originalFetch := fetchGPTTeamPlanRemainingSeats
	defer func() {
		fetchGPTTeamPlanRemainingSeats = originalFetch
	}()

	var fetchCalls atomic.Int32
	fetchGPTTeamPlanRemainingSeats = func(ctx stdctx.Context, baseURL string) (*int, error) {
		fetchCalls.Add(1)
		value := 9
		return &value, nil
	}

	GetGPTTeamPlanStatus(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"remaining_seats":6`)
	require.EqualValues(t, 0, fetchCalls.Load())
}

func TestGetGPTTeamPlanStatusForceRefreshSkipsUpstreamWhenRecentRefreshAttemptExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/gptteamplan/status?refresh=1", nil)
	setGPTTeamPlanOptionForTest("gptteamplan.enabled", "true")
	setGPTTeamPlanOptionForTest("gptteamplan.base_url", "http://gptteamplan.tech")
	defer setGPTTeamPlanOptionForTest("gptteamplan.enabled", "false")

	cachedSeats := 6
	setCachedGPTTeamPlanRemainingSeats(&cachedSeats)
	cachedGPTTeamPlanStatus.mu.Lock()
	cachedGPTTeamPlanStatus.lastAttemptAt = time.Now()
	cachedGPTTeamPlanStatus.fetchedAt = time.Now().Add(-time.Minute)
	cachedGPTTeamPlanStatus.mu.Unlock()

	originalFetch := fetchGPTTeamPlanRemainingSeats
	defer func() {
		fetchGPTTeamPlanRemainingSeats = originalFetch
	}()

	var fetchCalls atomic.Int32
	fetchGPTTeamPlanRemainingSeats = func(ctx stdctx.Context, baseURL string) (*int, error) {
		fetchCalls.Add(1)
		value := 9
		return &value, nil
	}

	GetGPTTeamPlanStatus(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"remaining_seats":6`)
	require.EqualValues(t, 0, fetchCalls.Load())
}

func TestGetGPTTeamPlanStatusReturnsCachedSeatsWithoutBlockingRefresh(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/gptteamplan/status", nil)
	setGPTTeamPlanOptionForTest("gptteamplan.enabled", "true")
	setGPTTeamPlanOptionForTest("gptteamplan.base_url", "http://gptteamplan.tech")
	defer setGPTTeamPlanOptionForTest("gptteamplan.enabled", "false")

	cachedSeats := 7
	setCachedGPTTeamPlanRemainingSeats(&cachedSeats)
	cachedGPTTeamPlanStatus.mu.Lock()
	cachedGPTTeamPlanStatus.fetchedAt = time.Now().Add(-2 * time.Minute)
	cachedGPTTeamPlanStatus.mu.Unlock()

	originalFetch := fetchGPTTeamPlanRemainingSeats
	defer func() {
		fetchGPTTeamPlanRemainingSeats = originalFetch
	}()

	var fetchCalls atomic.Int32
	fetchGPTTeamPlanRemainingSeats = func(ctx stdctx.Context, baseURL string) (*int, error) {
		fetchCalls.Add(1)
		value := 9
		return &value, nil
	}

	GetGPTTeamPlanStatus(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"remaining_seats":7`)
	require.EqualValues(t, 0, fetchCalls.Load())
}

func TestRedeemGPTTeamPlanRequiresParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/gptteamplan/redeem", strings.NewReader(`{"email":"","code":""}`))
	context.Request.Header.Set("Content-Type", "application/json")
	setGPTTeamPlanOptionForTest("gptteamplan.enabled", "true")
	defer setGPTTeamPlanOptionForTest("gptteamplan.enabled", "false")

	RedeemGPTTeamPlan(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `请输入邮箱和兑换码`)
}

func TestRedeemGPTTeamPlanDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/gptteamplan/redeem", strings.NewReader(`{"email":"user@example.com","code":"code"}`))
	context.Request.Header.Set("Content-Type", "application/json")
	setGPTTeamPlanOptionForTest("gptteamplan.enabled", "false")

	RedeemGPTTeamPlan(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `GPT Team 兑换未启用`)
}

func setGPTTeamPlanOptionForTest(key string, value string) {
	common.OptionMapRWMutex.Lock()
	defer common.OptionMapRWMutex.Unlock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMap[key] = value
	if common.OptionMap["gptteamplan.base_url"] == "" {
		common.OptionMap["gptteamplan.base_url"] = "http://gptteamplan.tech"
	}
}
