package controller

import (
	"strings"

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

func GetGPTTeamPlanStatus(c *gin.Context) {
	cfg := gptteamplan.GetConfig()
	common.ApiSuccess(c, gin.H{
		"enabled": cfg.Enabled,
	})
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
