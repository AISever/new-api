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

var ensureExternalShopCatalogFresh = externalshop.EnsureCatalogFresh
var refreshExternalShopCatalogAsync = externalshop.RefreshCatalogAsync
var listCachedExternalShopChannels = externalshop.ListCachedChannels
var fetchExternalShopChannels = externalshop.FetchChannels
var refreshExternalShopChannelsAsync = externalshop.RefreshChannelsAsync
var refreshExternalShopOrders = externalshop.RefreshUserOrders
var refreshExternalShopOrdersAsync = externalshop.RefreshUserOrdersAsync

type ExternalShopCreateOrderRequest struct {
	GoodsKey  string `json:"goods_key"`
	Quantity  int    `json:"quantity"`
	Contact   string `json:"contact"`
	ChannelId int    `json:"channel_id"`
}

func GetExternalShopStatus(c *gin.Context) {
	externalShopCfg := externalshop.GetConfig()

	common.ApiSuccess(c, gin.H{
		"external_shop": gin.H{
			"enabled":   externalShopCfg.Enabled,
			"ready":     externalShopCfg.IsReady(),
			"shop_name": externalShopCfg.ShopName,
		},
		"gptteamplan": buildGPTTeamPlanStatusSnapshot(),
	})
}

func GetExternalShopGoods(c *gin.Context) {
	cfg := externalshop.GetConfig()
	if !cfg.IsReady() {
		common.ApiErrorMsg(c, "商城未配置")
		return
	}
	if c.Query("refresh") == "1" {
		if err := ensureExternalShopCatalogFresh(c.Request.Context()); err != nil {
			common.ApiError(c, err)
			return
		}
	} else {
		refreshExternalShopCatalogAsync()
	}
	goods, err := model.ListExternalShopGoods(externalshop.ProviderLDXP, cfg.ShopToken, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, goods)
}

func GetExternalShopCategories(c *gin.Context) {
	cfg := externalshop.GetConfig()
	if !cfg.IsReady() {
		common.ApiErrorMsg(c, "商城未配置")
		return
	}
	if c.Query("refresh") == "1" {
		if err := ensureExternalShopCatalogFresh(c.Request.Context()); err != nil {
			common.ApiError(c, err)
			return
		}
	} else {
		refreshExternalShopCatalogAsync()
	}
	categories, err := externalshop.ListCatalogCategories(c.Request.Context())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, categories)
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
	if c.Query("refresh") == "1" {
		channels, err := fetchExternalShopChannels(c.Request.Context())
		if err != nil {
			common.ApiError(c, err)
			return
		}
		common.ApiSuccess(c, channels)
		return
	}
	refreshExternalShopChannelsAsync()
	common.ApiSuccess(c, listCachedExternalShopChannels())
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
	loadOrders := func() ([]model.ExternalShopOrder, int64, error) {
		orders, total, err := model.ListExternalShopOrdersByUser(userId, pageInfo, sortBy, sortOrder)
		if err != nil {
			return nil, 0, err
		}
		now := time.Now()
		for i := range orders {
			if externalshop.ExpireLocalOrderIfTimedOut(&orders[i], now) {
				_ = orders[i].Update()
			}
		}
		return orders, total, nil
	}

	orders, total, err := loadOrders()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	localTradeNos := make([]string, 0, len(orders))
	for _, order := range orders {
		if strings.TrimSpace(order.LocalTradeNo) == "" {
			continue
		}
		localTradeNos = append(localTradeNos, order.LocalTradeNo)
	}
	if c.Query("refresh") == "1" {
		refreshExternalShopOrders(c.Request.Context(), userId, localTradeNos)
		orders, total, err = loadOrders()
		if err != nil {
			common.ApiError(c, err)
			return
		}
	} else {
		refreshExternalShopOrdersAsync(userId, localTradeNos)
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

func DeleteExternalShopOrder(c *gin.Context) {
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
		common.ApiErrorMsg(c, "无权删除该订单")
		return
	}
	if err := order.Delete(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"local_trade_no": localTradeNo,
	})
}

func AdminSyncExternalShopCatalog(c *gin.Context) {
	cfg := externalshop.GetConfig()
	if !cfg.IsReady() {
		common.ApiErrorMsg(c, "商城未配置")
		return
	}
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
	cfg := externalshop.GetConfig()
	if !cfg.IsReady() {
		common.ApiErrorMsg(c, "商城未配置")
		return
	}
	summary, err := externalshop.RunExternalShopOrderAutoSyncOnce()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}
