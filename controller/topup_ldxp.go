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

type LdxpTopupPayRequest struct {
	Amount  float64 `json:"amount"`
	Contact string  `json:"contact"`
}

func resolveLdxpTopupContact(username string, requestContact string, now time.Time) string {
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

func RequestLdxpPay(c *gin.Context) {
	if !service.IsLdxpTopupEnabled() {
		c.JSON(200, gin.H{"message": "error", "data": "当前支付方式暂未开启"})
		return
	}

	var req LdxpTopupPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	product, ok := service.FindLdxpTopupProductByAmount(req.Amount)
	if !ok {
		c.JSON(200, gin.H{"message": "error", "data": "当前充值档位无效"})
		return
	}

	userId := c.GetInt("id")
	user, err := model.GetUserById(userId, false)
	if err != nil || user == nil {
		c.JSON(200, gin.H{"message": "error", "data": "用户不存在"})
		return
	}
	contact := resolveLdxpTopupContact(user.Username, req.Contact, time.Now())

	expectedAmount := service.ResolveLdxpTopupSettlementAmount(*product)
	if expectedAmount < 0.01 {
		c.JSON(200, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	cfg := service.GetLdxpConfig()
	client := service.NewLDXPClient(cfg)
	quoteResp, err := client.GetGoodsPrice(c.Request.Context(), product.GoodsKey, 1, cfg.DefaultChannelId)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "获取支付信息失败"})
		return
	}
	if quoteResp.Code != 1 {
		message := strings.TrimSpace(quoteResp.Msg)
		if message == "" {
			message = "获取支付信息失败"
		}
		c.JSON(200, gin.H{"message": "error", "data": message})
		return
	}
	if !service.AmountMatchesLdxpQuote(expectedAmount, quoteResp.Data.TotalAmount) {
		c.JSON(200, gin.H{"message": "error", "data": "支付商品配置错误"})
		return
	}

	amount := service.ResolveLdxpTopupGrantedAmount(*product)

	tradeNo := fmt.Sprintf("LDXPUSR%dNO%s%d", userId, common.GetRandomString(6), time.Now().Unix())
	topUp := &model.TopUp{
		UserId:          userId,
		Amount:          int64(amount),
		Money:           quoteResp.Data.TotalAmount,
		TradeNo:         tradeNo,
		PaymentMethod:   "ldxp",
		ProviderPayload: service.MergeLdxpTopupAmountIntoProviderPayload(common.GetJsonString(quoteResp), amount),
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	createResp, err := client.CreateOrder(c.Request.Context(), service.LdxpCreateOrderRequest{
		GoodsKey:       product.GoodsKey,
		Quantity:       1,
		ChannelID:      cfg.DefaultChannelId,
		Contact:        contact,
		CouponCode:     "",
		QueryPassword:  "",
		SelectCardsIDs: []int{},
	})
	if err != nil {
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(200, gin.H{"message": "error", "data": "拉起支付失败"})
		return
	}
	if createResp.Code != 1 || strings.TrimSpace(createResp.Data.TradeNo) == "" || strings.TrimSpace(createResp.Data.PayURL) == "" {
		topUp.Status = common.TopUpStatusFailed
		topUp.ProviderPayload = service.MergeLdxpTopupAmountIntoProviderPayload(common.GetJsonString(createResp), amount)
		_ = topUp.Update()
		message := strings.TrimSpace(createResp.Msg)
		if message == "" {
			message = "拉起支付失败"
		}
		c.JSON(200, gin.H{"message": "error", "data": message})
		return
	}

	topUp.ProviderTradeNo = strings.TrimSpace(createResp.Data.TradeNo)
	topUp.ProviderPayload = service.MergeLdxpTopupAmountIntoProviderPayload(common.GetJsonString(createResp), amount)
	topUp.Money = createResp.Data.TotalAmount
	if err := topUp.Update(); err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "更新订单失败"})
		return
	}

	c.JSON(200, gin.H{
		"message": "success",
		"data": gin.H{
			"payment_url": createResp.Data.PayURL,
			"order_id":    tradeNo,
			"status":      topUp.Status,
		},
	})
}

func GetLdxpTopupOrder(c *gin.Context) {
	tradeNo := strings.TrimSpace(c.Param("trade_no"))
	if tradeNo == "" {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		common.ApiErrorMsg(c, "订单不存在")
		return
	}
	if topUp.UserId != c.GetInt("id") {
		common.ApiErrorMsg(c, "无权访问该订单")
		return
	}

	topUp = syncPendingLdxpTopUp(c.Request.Context(), topUp)

	payload := buildLdxpTopupOrderPayload(topUp)
	if providerPayload, updated := enrichLdxpOrderPayloadWithCheckoutData(c.Request.Context(), &payload, topUp.ProviderPayload); updated {
		topUp.ProviderPayload = providerPayload
		_ = topUp.Update()
	}
	common.ApiSuccess(c, payload)
}
