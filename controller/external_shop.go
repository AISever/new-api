package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	externalshop "github.com/QuantumNous/new-api/service/external_shop"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ExternalShopCreateOrderRequest struct {
	GoodsKey  string `json:"goods_key"`
	Quantity  int    `json:"quantity"`
	Contact   string `json:"contact"`
	ChannelId int    `json:"channel_id"`
}

func GetExternalShopGoods(c *gin.Context) {
	cfg := externalshop.GetConfig()
	if !cfg.IsReady() {
		common.ApiErrorMsg(c, "商城未配置")
		return
	}
	goods, err := model.ListExternalShopGoods(externalshop.ProviderLDXP, cfg.ShopToken, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, goods)
}

func GetExternalShopGood(c *gin.Context) {
	cfg := externalshop.GetConfig()
	if !cfg.IsReady() {
		common.ApiErrorMsg(c, "商城未配置")
		return
	}
	goodsKey := strings.TrimSpace(c.Param("goods_key"))
	if goodsKey == "" {
		common.ApiErrorMsg(c, "商品不存在")
		return
	}
	good, err := model.GetExternalShopGoodByGoodsKey(externalshop.ProviderLDXP, cfg.ShopToken, goodsKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "商品不存在")
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, good)
}

func GetExternalShopChannels(c *gin.Context) {
	cfg := externalshop.GetConfig()
	if !cfg.IsReady() {
		common.ApiErrorMsg(c, "商城未配置")
		return
	}
	client, _, err := externalshop.NewConfiguredClient()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	resp, err := client.ListChannels(c.Request.Context())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if resp.Code != 1 {
		common.ApiErrorMsg(c, strings.TrimSpace(resp.Msg))
		return
	}
	common.ApiSuccess(c, resp.Data)
}

func CreateExternalShopOrder(c *gin.Context) {
	var req ExternalShopCreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	userId := c.GetInt("id")
	order, err := externalshop.CreateLocalOrder(c.Request.Context(), userId, req.GoodsKey, req.Quantity, req.Contact, req.ChannelId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"local_trade_no":    order.LocalTradeNo,
		"upstream_trade_no": order.UpstreamTradeNo,
		"pay_url":           order.PayUrl,
		"status":            order.Status,
		"amount":            order.Amount,
	})
}

func GetExternalShopOrders(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	sortBy := strings.TrimSpace(c.Query("sort_by"))
	sortOrder := strings.TrimSpace(c.Query("sort_order"))
	orders, total, err := model.ListExternalShopOrdersByUser(userId, pageInfo, sortBy, sortOrder)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	now := time.Now()
	for i := range orders {
		if externalshop.ExpireLocalOrderIfTimedOut(&orders[i], now) {
			_ = orders[i].Update()
		}
	}
	pageInfo.Total = int(total)
	pageInfo.Items = orders
	common.ApiSuccess(c, pageInfo)
}

func GetExternalShopOrder(c *gin.Context) {
	userId := c.GetInt("id")
	localTradeNo := strings.TrimSpace(c.Param("local_trade_no"))
	order, err := model.GetExternalShopOrderByLocalTradeNo(localTradeNo)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "订单不存在")
			return
		}
		common.ApiError(c, err)
		return
	}
	if order.UserId != userId {
		common.ApiErrorMsg(c, "无权访问该订单")
		return
	}
	if externalshop.ExpireLocalOrderIfTimedOut(order, time.Now()) {
		_ = order.Update()
	}
	common.ApiSuccess(c, order)
}

func RefreshExternalShopOrder(c *gin.Context) {
	userId := c.GetInt("id")
	localTradeNo := strings.TrimSpace(c.Param("local_trade_no"))
	order, err := model.GetExternalShopOrderByLocalTradeNo(localTradeNo)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "订单不存在")
			return
		}
		common.ApiError(c, err)
		return
	}
	if order.UserId != userId {
		common.ApiErrorMsg(c, "无权访问该订单")
		return
	}
	order, err = externalshop.RefreshLocalOrder(c.Request.Context(), order)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, order)
}

func AdminSyncExternalShopCatalog(c *gin.Context) {
	summary, err := externalshop.SyncCatalog(c.Request.Context())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}

func AdminListExternalShopOrders(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userId, _ := strconv.Atoi(strings.TrimSpace(c.Query("user_id")))
	orders, total, err := model.ListExternalShopOrders(pageInfo, model.ExternalShopOrderQuery{
		Keyword:   c.Query("keyword"),
		Status:    c.Query("status"),
		UserId:    userId,
		SortBy:    strings.TrimSpace(c.Query("sort_by")),
		SortOrder: strings.TrimSpace(c.Query("sort_order")),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	now := time.Now()
	for i := range orders {
		if externalshop.ExpireLocalOrderIfTimedOut(&orders[i], now) {
			_ = orders[i].Update()
		}
	}
	pageInfo.Total = int(total)
	pageInfo.Items = orders
	common.ApiSuccess(c, pageInfo)
}

func AdminRefreshExternalShopOrder(c *gin.Context) {
	localTradeNo := strings.TrimSpace(c.Param("local_trade_no"))
	order, err := model.GetExternalShopOrderByLocalTradeNo(localTradeNo)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "订单不存在")
			return
		}
		common.ApiError(c, err)
		return
	}
	order, err = externalshop.RefreshLocalOrder(c.Request.Context(), order)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, order)
}

func AdminSyncPendingExternalShopOrders(c *gin.Context) {
	summary, err := externalshop.RunExternalShopOrderAutoSyncOnce()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}
