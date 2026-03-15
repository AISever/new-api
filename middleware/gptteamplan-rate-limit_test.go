package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

func TestGPTTeamPlanRateLimitUsesUserScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	defer func() {
		common.RedisEnabled = oldRedisEnabled
		inMemoryRateLimiter = common.InMemoryRateLimiter{}
	}()

	inMemoryRateLimiter = common.InMemoryRateLimiter{}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", 1)
		c.Next()
	})
	router.POST("/redeem", GPTTeamPlanRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for attempt := 1; attempt <= gptTeamPlanRateLimitNum; attempt++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/redeem", nil)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("attempt %d should pass, got %d", attempt, recorder.Code)
		}
	}

	blocked := httptest.NewRecorder()
	blockedReq := httptest.NewRequest(http.MethodPost, "/redeem", nil)
	router.ServeHTTP(blocked, blockedReq)
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("expected rate limit on extra attempt, got %d", blocked.Code)
	}

	secondUserRouter := gin.New()
	secondUserRouter.Use(func(c *gin.Context) {
		c.Set("id", 2)
		c.Next()
	})
	secondUserRouter.POST("/redeem", GPTTeamPlanRateLimit(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	secondUser := httptest.NewRecorder()
	secondUserReq := httptest.NewRequest(http.MethodPost, "/redeem", nil)
	secondUserRouter.ServeHTTP(secondUser, secondUserReq)
	if secondUser.Code != http.StatusNoContent {
		t.Fatalf("expected other user to keep separate limit, got %d", secondUser.Code)
	}
}
