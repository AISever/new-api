package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type InspectLdxpShopRequest struct {
	BaseURL   string `json:"base_url"`
	ShopInput string `json:"shop_input"`
}

type ldxpShopCatalogItem struct {
	Name         string  `json:"name"`
	GoodsKey     string  `json:"goods_key"`
	Price        float64 `json:"price"`
	CategoryID   int     `json:"category_id"`
	CategoryName string  `json:"category_name"`
}

func InspectLdxpShop(c *gin.Context) {
	var req InspectLdxpShopRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}

	baseURL, shopToken, err := service.ParseLdxpShopInput(req.BaseURL, req.ShopInput)
	if err != nil {
		common.ApiErrorMsg(c, "请输入有效的 LDXP 店铺地址或 Token")
		return
	}

	client := service.NewLDXPClient(service.LdxpConfig{
		BaseURL:   baseURL,
		ShopToken: shopToken,
	})

	shopInfoResp, err := client.GetShopInfo(c.Request.Context())
	if err != nil {
		common.ApiErrorMsg(c, "读取 LDXP 店铺信息失败")
		return
	}
	if shopInfoResp.Code != 1 {
		message := strings.TrimSpace(shopInfoResp.Msg)
		if message == "" {
			message = "读取 LDXP 店铺信息失败"
		}
		common.ApiErrorMsg(c, message)
		return
	}

	channelsResp, err := client.ListChannels(c.Request.Context())
	if err != nil {
		common.ApiErrorMsg(c, "读取 LDXP 支付渠道失败")
		return
	}
	if channelsResp.Code != 1 {
		message := strings.TrimSpace(channelsResp.Msg)
		if message == "" {
			message = "读取 LDXP 支付渠道失败"
		}
		common.ApiErrorMsg(c, message)
		return
	}

	goodsResp, err := client.ListGoods(c.Request.Context(), 0, "card", 1, 100, "")
	if err != nil {
		common.ApiErrorMsg(c, "读取 LDXP 商品失败")
		return
	}
	if goodsResp.Code != 1 {
		message := strings.TrimSpace(goodsResp.Msg)
		if message == "" {
			message = "读取 LDXP 商品失败"
		}
		common.ApiErrorMsg(c, message)
		return
	}

	topupProducts := service.BuildSuggestedLdxpTopupProducts(goodsResp.Data.List)
	topupKeys := make(map[string]struct{}, len(topupProducts))
	for _, product := range topupProducts {
		key := strings.TrimSpace(product.GoodsKey)
		if key != "" {
			topupKeys[key] = struct{}{}
		}
	}

	subscriptionGoods := make([]ldxpShopCatalogItem, 0)
	for _, good := range goodsResp.Data.List {
		if _, ok := topupKeys[strings.TrimSpace(good.GoodsKey)]; ok {
			continue
		}
		subscriptionGoods = append(subscriptionGoods, ldxpShopCatalogItem{
			Name:         strings.TrimSpace(good.Name),
			GoodsKey:     strings.TrimSpace(good.GoodsKey),
			Price:        good.Price,
			CategoryID:   good.Category.ID,
			CategoryName: strings.TrimSpace(good.Category.Name),
		})
	}

	common.ApiSuccess(c, gin.H{
		"base_url":           baseURL,
		"shop_token":         strings.TrimSpace(shopInfoResp.Data.Token),
		"shop_name":          strings.TrimSpace(shopInfoResp.Data.Nickname),
		"shop_link":          strings.TrimSpace(shopInfoResp.Data.Link),
		"default_channel_id": pickRecommendedLdxpChannelID(channelsResp.Data),
		"channels":           channelsResp.Data,
		"topup_products":     topupProducts,
		"subscription_goods": subscriptionGoods,
	})
}

func pickRecommendedLdxpChannelID(channels []service.LdxpChannelItem) int {
	for _, channel := range channels {
		if channel.Status == 1 && channel.CustomStatus == 1 && channel.ID > 0 {
			return channel.ID
		}
	}
	for _, channel := range channels {
		if channel.ID > 0 {
			return channel.ID
		}
	}
	return 0
}
