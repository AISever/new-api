package controller

import (
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type SubscriptionLdxpPayRequest struct {
	PlanId  int    `json:"plan_id"`
	Contact string `json:"contact"`
}

func resolveLdxpSubscriptionContact(username string, requestContact string, now time.Time) string {
	contact := strings.TrimSpace(requestContact)
	if contact != "" {
		return contact
	}
	username = strings.TrimSpace(username)
	if username == "" {
		username = "user"
	}
	return fmt.Sprintf("%s-%d", username, now.Unix())
}

func SubscriptionRequestLdxpPay(c *gin.Context) {
	cfg := service.GetLdxpConfig()
	if !cfg.IsReady() {
		common.ApiErrorMsg(c, "当前支付方式暂未开启")
		return
	}

	var req SubscriptionLdxpPayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	plan, err := model.GetSubscriptionPlanById(req.PlanId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !plan.Enabled {
		common.ApiErrorMsg(c, "套餐未启用")
		return
	}
	ldxpGoodsKey := strings.TrimSpace(plan.LdxpGoodsKey)
	if ldxpGoodsKey == "" {
		common.ApiErrorMsg(c, "该套餐暂不支持当前支付方式")
		return
	}

	userId := c.GetInt("id")
	user, err := model.GetUserById(userId, false)
	if err != nil || user == nil {
		common.ApiErrorMsg(c, "用户不存在")
		return
	}
	contact := resolveLdxpSubscriptionContact(user.Username, req.Contact, time.Now())

	if plan.MaxPurchasePerUser > 0 {
		count, err := model.CountUserSubscriptionsByPlan(userId, plan.Id)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if count >= int64(plan.MaxPurchasePerUser) {
			common.ApiErrorMsg(c, "已达到该套餐购买上限")
			return
		}
	}

	client := service.NewLDXPClient(cfg)
	quoteResp, err := client.GetGoodsPrice(c.Request.Context(), ldxpGoodsKey, 1, cfg.DefaultChannelId)
	if err != nil {
		common.ApiErrorMsg(c, "获取支付信息失败")
		return
	}
	if quoteResp.Code != 1 {
		message := strings.TrimSpace(quoteResp.Msg)
		if message == "" {
			message = "获取支付信息失败"
		}
		common.ApiErrorMsg(c, message)
		return
	}
	if !service.AmountMatchesLdxpQuote(plan.PriceAmount, quoteResp.Data.TotalAmount) {
		common.ApiErrorMsg(c, "支付商品配置错误")
		return
	}

	tradeNo := fmt.Sprintf("SUBLDXP%s%d", common.GetRandomString(6), time.Now().Unix())
	order := &model.SubscriptionOrder{
		UserId:          userId,
		PlanId:          plan.Id,
		Money:           quoteResp.Data.TotalAmount,
		TradeNo:         tradeNo,
		PaymentMethod:   "ldxp",
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
		ProviderPayload: common.GetJsonString(quoteResp),
	}
	if err := order.Insert(); err != nil {
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}

	createResp, err := client.CreateOrder(c.Request.Context(), service.LdxpCreateOrderRequest{
		GoodsKey:       ldxpGoodsKey,
		Quantity:       1,
		ChannelID:      cfg.DefaultChannelId,
		Contact:        contact,
		CouponCode:     "",
		QueryPassword:  "",
		SelectCardsIDs: []int{},
	})
	if err != nil {
		order.Status = common.TopUpStatusFailed
		_ = order.Update()
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}
	if createResp.Code != 1 || strings.TrimSpace(createResp.Data.TradeNo) == "" || strings.TrimSpace(createResp.Data.PayURL) == "" {
		order.Status = common.TopUpStatusFailed
		order.ProviderPayload = common.GetJsonString(createResp)
		_ = order.Update()
		message := strings.TrimSpace(createResp.Msg)
		if message == "" {
			message = "拉起支付失败"
		}
		common.ApiErrorMsg(c, message)
		return
	}

	order.ProviderTradeNo = strings.TrimSpace(createResp.Data.TradeNo)
	order.ProviderPayload = common.GetJsonString(createResp)
	order.Money = createResp.Data.TotalAmount
	if err := order.Update(); err != nil {
		common.ApiErrorMsg(c, "更新订单失败")
		return
	}

	common.ApiSuccess(c, gin.H{
		"payment_url": createResp.Data.PayURL,
		"order_id":    tradeNo,
		"status":      order.Status,
	})
}

func GetSubscriptionLdxpOrder(c *gin.Context) {
	tradeNo := strings.TrimSpace(c.Param("trade_no"))
	if tradeNo == "" {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}
	order := model.GetSubscriptionOrderByTradeNo(tradeNo)
	if order == nil {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}
	if order.UserId != c.GetInt("id") {
		common.ApiErrorMsg(c, "无权访问该订单")
		return
	}

	order = syncPendingLdxpSubscriptionOrder(c.Request.Context(), order)

	planTitle := ""
	if plan, err := model.GetSubscriptionPlanById(order.PlanId); err == nil && plan != nil {
		planTitle = plan.Title
	}

	common.ApiSuccess(c, buildLdxpSubscriptionOrderPayload(order, planTitle))
}
