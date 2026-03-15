package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetGPTTeamPlanStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	setGPTTeamPlanOptionForTest("gptteamplan.enabled", "true")
	defer setGPTTeamPlanOptionForTest("gptteamplan.enabled", "false")

	GetGPTTeamPlanStatus(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"enabled":true`)
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
