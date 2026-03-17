package controller

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/gptteamplan"
	"github.com/gin-gonic/gin"
)

type GPTTeamPlanRedeemRequest struct {
	Email  string `json:"email"`
	Code   string `json:"code"`
	TeamID *int   `json:"team_id"`
}

type GPTTeamPlanWarrantyRequest struct {
	Code string `json:"code"`
}

type gptTeamPlanStatusCache struct {
	mu             sync.RWMutex
	remainingSeats *int
	fetchedAt      time.Time
	lastAttemptAt  time.Time
}

var cachedGPTTeamPlanStatus gptTeamPlanStatusCache
var gptTeamPlanStatusRefreshInFlight atomic.Bool
var gptTeamPlanForceRefreshMinInterval = 15 * time.Second
var fetchGPTTeamPlanRemainingSeats = func(ctx context.Context, baseURL string) (*int, error) {
	client := gptteamplan.NewClient(baseURL, service.GetHttpClient())
	return client.GetRemainingSeats(ctx)
}

func GetGPTTeamPlanStatus(c *gin.Context) {
	forceRefresh := c != nil && c.Query("refresh") == "1"
	common.ApiSuccess(c, buildGPTTeamPlanStatus(getRequestContext(c), forceRefresh))
}

func RedeemGPTTeamPlan(c *gin.Context) {
	cfg := gptteamplan.GetConfig()
	if !cfg.IsReady() {
		common.ApiErrorMsg(c, "GPT Team 兑换未启用")
		return
	}

	var req GPTTeamPlanRedeemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	email := strings.TrimSpace(req.Email)
	code := strings.TrimSpace(req.Code)
	if email == "" || code == "" {
		common.ApiErrorMsg(c, "请输入邮箱和兑换码")
		return
	}

	client := gptteamplan.NewClient(cfg.BaseURL, service.GetHttpClient())
	result, err := client.Redeem(c.Request.Context(), email, code, req.TeamID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func CheckGPTTeamPlanWarranty(c *gin.Context) {
	cfg := gptteamplan.GetConfig()
	if !cfg.IsReady() {
		common.ApiErrorMsg(c, "GPT Team 兑换未启用")
		return
	}

	var req GPTTeamPlanWarrantyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		common.ApiErrorMsg(c, "请输入兑换码")
		return
	}

	client := gptteamplan.NewClient(cfg.BaseURL, service.GetHttpClient())
	result, err := client.CheckWarranty(c.Request.Context(), code)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func buildGPTTeamPlanStatus(ctx context.Context, forceRefresh bool) gin.H {
	cfg := gptteamplan.GetConfig()
	status := gin.H{
		"enabled":         cfg.Enabled,
		"remaining_seats": nil,
	}
	if !cfg.IsReady() {
		return status
	}

	if cachedRemainingSeats, ok := getCachedGPTTeamPlanRemainingSeats(); ok {
		status["remaining_seats"] = cachedRemainingSeats
	}

	if forceRefresh && !isGPTTeamPlanForceRefreshDue() {
		return status
	}

	if status["remaining_seats"] != nil && !forceRefresh {
		return status
	}

	shouldFetch := true
	if forceRefresh {
		shouldFetch = gptTeamPlanStatusRefreshInFlight.CompareAndSwap(false, true)
		if !shouldFetch {
			return status
		}
		markGPTTeamPlanRefreshAttempt()
		defer gptTeamPlanStatusRefreshInFlight.Store(false)
	}

	requestCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	if remainingSeats, err := fetchGPTTeamPlanRemainingSeats(requestCtx, cfg.BaseURL); err == nil {
		setCachedGPTTeamPlanRemainingSeats(remainingSeats)
		status["remaining_seats"] = remainingSeats
	}
	return status
}

func buildGPTTeamPlanStatusSnapshot() gin.H {
	cfg := gptteamplan.GetConfig()
	status := gin.H{
		"enabled":         cfg.Enabled,
		"remaining_seats": nil,
	}
	if !cfg.IsReady() {
		return status
	}
	if cachedRemainingSeats, ok := getCachedGPTTeamPlanRemainingSeats(); ok {
		status["remaining_seats"] = cachedRemainingSeats
	}
	return status
}

func getRequestContext(c *gin.Context) context.Context {
	if c != nil && c.Request != nil {
		return c.Request.Context()
	}
	return context.Background()
}

func getCachedGPTTeamPlanRemainingSeats() (*int, bool) {
	cachedGPTTeamPlanStatus.mu.RLock()
	defer cachedGPTTeamPlanStatus.mu.RUnlock()
	if cachedGPTTeamPlanStatus.remainingSeats == nil {
		return nil, false
	}
	value := *cachedGPTTeamPlanStatus.remainingSeats
	return &value, true
}

func setCachedGPTTeamPlanRemainingSeats(remainingSeats *int) {
	cachedGPTTeamPlanStatus.mu.Lock()
	defer cachedGPTTeamPlanStatus.mu.Unlock()
	if remainingSeats == nil {
		cachedGPTTeamPlanStatus.remainingSeats = nil
		cachedGPTTeamPlanStatus.fetchedAt = time.Time{}
		cachedGPTTeamPlanStatus.lastAttemptAt = time.Time{}
		return
	}
	value := *remainingSeats
	cachedGPTTeamPlanStatus.remainingSeats = &value
	cachedGPTTeamPlanStatus.fetchedAt = time.Now()
	cachedGPTTeamPlanStatus.lastAttemptAt = cachedGPTTeamPlanStatus.fetchedAt
}

func isGPTTeamPlanStatusCacheStale() bool {
	cachedGPTTeamPlanStatus.mu.RLock()
	defer cachedGPTTeamPlanStatus.mu.RUnlock()
	if cachedGPTTeamPlanStatus.remainingSeats == nil || cachedGPTTeamPlanStatus.fetchedAt.IsZero() {
		return true
	}
	return time.Since(cachedGPTTeamPlanStatus.fetchedAt) > time.Minute
}

func isGPTTeamPlanForceRefreshDue() bool {
	cachedGPTTeamPlanStatus.mu.RLock()
	defer cachedGPTTeamPlanStatus.mu.RUnlock()
	if cachedGPTTeamPlanStatus.lastAttemptAt.IsZero() {
		return true
	}
	return time.Since(cachedGPTTeamPlanStatus.lastAttemptAt) >= gptTeamPlanForceRefreshMinInterval
}

func markGPTTeamPlanRefreshAttempt() {
	cachedGPTTeamPlanStatus.mu.Lock()
	defer cachedGPTTeamPlanStatus.mu.Unlock()
	cachedGPTTeamPlanStatus.lastAttemptAt = time.Now()
}
