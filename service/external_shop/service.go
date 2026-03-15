package externalshop

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

type catalogClient interface {
	GetShopInfo(ctx context.Context) (ShopInfoResponse, error)
	ListCategories(ctx context.Context, goodsType string) (CategoryListResponse, error)
	ListGoods(ctx context.Context, categoryID int, goodsType string, current int, pageSize int, keywords string) (GoodsListResponse, error)
}

type SyncSummary struct {
	ShopName      string `json:"shop_name"`
	Categories    int    `json:"categories"`
	GoodsSynced   int    `json:"goods_synced"`
	GoodsDisabled int    `json:"goods_disabled"`
}

func NewConfiguredClient() (*LDXPClient, Config, error) {
	cfg := GetConfig()
	if !cfg.IsReady() {
		return nil, cfg, errors.New("external shop is not configured")
	}
	return NewLDXPClient(cfg.BaseURL, cfg.ShopToken, service.GetHttpClient()), cfg, nil
}

func SyncCatalog(ctx context.Context) (*SyncSummary, error) {
	client, cfg, err := NewConfiguredClient()
	if err != nil {
		return nil, err
	}
	return syncCatalogWithClient(
		ctx,
		client,
		cfg,
		100,
		model.UpsertExternalShopGood,
		model.DisableMissingExternalShopGoods,
		time.Now,
	)
}

func syncCatalogWithClient(
	ctx context.Context,
	client catalogClient,
	cfg Config,
	pageSize int,
	upsert func(*model.ExternalShopGood) error,
	disableMissing func(provider string, shopToken string, keepGoodsKeys []string) (int64, error),
	now func() time.Time,
) (*SyncSummary, error) {
	shopInfo, err := client.GetShopInfo(ctx)
	if err != nil {
		return nil, err
	}
	shopName := strings.TrimSpace(shopInfo.Data.Nickname)
	if cfg.ShopName == "" && shopName != "" {
		cfg.ShopName = shopName
	}

	categoryResp, err := client.ListCategories(ctx, "card")
	if err != nil {
		return nil, err
	}

	keepGoodsKeys := make([]string, 0)
	totalSynced := 0
	nowUnix := now().Unix()
	for _, category := range categoryResp.Data {
		if len(cfg.AllowedCategoryIDs) > 0 {
			if _, ok := cfg.AllowedCategoryIDs[category.Id]; !ok {
				continue
			}
		}
		items, err := listAllCategoryGoods(ctx, client, category.Id, pageSize)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if len(cfg.AllowedGoodsKeys) > 0 {
				if _, ok := cfg.AllowedGoodsKeys[item.GoodsKey]; !ok {
					continue
				}
			}
			rawPayload, err := common.Marshal(item)
			if err != nil {
				return nil, err
			}
			good := &model.ExternalShopGood{
				Provider:            ProviderLDXP,
				ShopToken:           cfg.ShopToken,
				GoodsKey:            item.GoodsKey,
				GoodsType:           item.GoodsType,
				Name:                item.Name,
				Description:         item.Description,
				Image:               item.Image,
				CategoryId:          item.Category.Id,
				CategoryName:        item.Category.Name,
				Price:               item.Price,
				MarketPrice:         item.MarketPrice,
				CouponStatus:        item.CouponStatus,
				StockCount:          item.Extend.StockCount,
				ShowStockType:       item.Extend.ShowStockType,
				LimitCount:          item.Extend.LimitCount,
				SendOrder:           item.Extend.SendOrder,
				QueryPasswordStatus: item.Extend.QueryPasswordStatus,
				Enabled:             true,
				RawPayload:          string(rawPayload),
				SyncedAt:            nowUnix,
			}
			if err := upsert(good); err != nil {
				return nil, err
			}
			keepGoodsKeys = append(keepGoodsKeys, item.GoodsKey)
			totalSynced++
		}
	}
	disabledCount, err := disableMissing(ProviderLDXP, cfg.ShopToken, keepGoodsKeys)
	if err != nil {
		return nil, err
	}
	return &SyncSummary{
		ShopName:      cfg.ShopName,
		Categories:    len(categoryResp.Data),
		GoodsSynced:   totalSynced,
		GoodsDisabled: int(disabledCount),
	}, nil
}

func listAllCategoryGoods(ctx context.Context, client catalogClient, categoryID int, pageSize int) ([]GoodsItem, error) {
	if pageSize <= 0 {
		pageSize = 100
	}
	allItems := make([]GoodsItem, 0)
	for current := 1; ; current++ {
		goodsResp, err := client.ListGoods(ctx, categoryID, "card", current, pageSize, "")
		if err != nil {
			return nil, err
		}
		allItems = append(allItems, goodsResp.Data.List...)
		if len(goodsResp.Data.List) == 0 || len(allItems) >= goodsResp.Data.Total {
			break
		}
	}
	return allItems, nil
}

func CreateLocalOrder(ctx context.Context, userId int, goodsKey string, quantity int, contact string, channelID int) (*model.ExternalShopOrder, error) {
	client, cfg, err := NewConfiguredClient()
	if err != nil {
		return nil, err
	}
	good, err := model.GetExternalShopGoodByGoodsKey(ProviderLDXP, cfg.ShopToken, goodsKey)
	if err != nil {
		return nil, err
	}
	if !good.Enabled {
		return nil, errors.New("goods is disabled")
	}
	if quantity <= 0 {
		quantity = 1
	}
	if err := validatePurchasableGood(good, quantity); err != nil {
		return nil, err
	}
	if strings.TrimSpace(contact) == "" {
		return nil, errors.New("contact is required")
	}

	channelsResp, err := client.ListChannels(ctx)
	if err != nil {
		return nil, err
	}
	channel, err := pickChannel(channelsResp.Data, channelID)
	if err != nil {
		return nil, err
	}

	priceResp, err := client.GetGoodsPrice(ctx, goodsKey, quantity, "", channel.Id)
	if err != nil {
		return nil, err
	}
	if priceResp.Code != 1 {
		return nil, fmt.Errorf("price quote failed: %s", priceResp.Msg)
	}

	localTradeNo := fmt.Sprintf("ESHOP%s%d", common.GetRandomString(6), time.Now().Unix())
	order := &model.ExternalShopOrder{
		LocalTradeNo: localTradeNo,
		UserId:       userId,
		Provider:     ProviderLDXP,
		ShopToken:    cfg.ShopToken,
		GoodsKey:     goodsKey,
		GoodsName:    good.Name,
		Quantity:     quantity,
		Contact:      strings.TrimSpace(contact),
		ChannelId:    channel.Id,
		ChannelCode:  channel.Code,
		ChannelName:  channel.ShowName,
		Amount:       priceResp.Data.TotalAmount,
		Status:       OrderStatusCreated,
	}
	if err := order.Insert(); err != nil {
		return nil, err
	}

	createResp, err := client.CreateOrder(ctx, CreateOrderRequest{
		GoodsKey:       goodsKey,
		Quantity:       quantity,
		ChannelID:      channel.Id,
		Contact:        order.Contact,
		CouponCode:     "",
		QueryPassword:  "",
		SelectCardsIDs: []int{},
	})
	if err != nil {
		order.Status = OrderStatusFailed
		order.LastError = err.Error()
		_ = order.Update()
		return nil, err
	}
	rawPayload, err := common.Marshal(createResp)
	if err != nil {
		return nil, err
	}
	order.ProviderPayload = string(rawPayload)
	if err := applyCreateOrderResponse(order, createResp); err != nil {
		if shouldMarkGoodOutOfStockAfterCreateFailure(createResp.Msg) {
			_ = model.MarkExternalShopGoodOutOfStock(ProviderLDXP, cfg.ShopToken, goodsKey)
		}
		_ = order.Delete()
		order.Status = OrderStatusFailed
		order.LastError = err.Error()
		return nil, err
	}
	if err := order.Update(); err != nil {
		return nil, err
	}
	return order, nil
}

func applyCreateOrderResponse(order *model.ExternalShopOrder, createResp CreateOrderResponse) error {
	if order == nil {
		return errors.New("order is nil")
	}
	if createResp.Code != 1 {
		return fmt.Errorf("create order failed: %s", strings.TrimSpace(createResp.Msg))
	}
	tradeNo := strings.TrimSpace(createResp.Data.TradeNo)
	payURL := strings.TrimSpace(createResp.Data.PayURL)
	if tradeNo == "" || payURL == "" {
		return errors.New("create order failed: invalid upstream order response")
	}
	order.UpstreamTradeNo = tradeNo
	order.Amount = createResp.Data.TotalAmount
	order.PayUrl = payURL
	order.Status = OrderStatusPendingPayment
	order.LastError = ""
	return nil
}

func validatePurchasableGood(good *model.ExternalShopGood, quantity int) error {
	if good == nil {
		return errors.New("goods not found")
	}
	if good.StockCount <= 0 {
		return errors.New("goods is out of stock")
	}
	if quantity > 0 && good.LimitCount > 0 && quantity > good.LimitCount {
		return fmt.Errorf("quantity exceeds limit %d", good.LimitCount)
	}
	return nil
}

func shouldMarkGoodOutOfStockAfterCreateFailure(message string) bool {
	normalized := strings.TrimSpace(strings.ToLower(message))
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "库存不足") ||
		strings.Contains(normalized, "售罄") ||
		strings.Contains(normalized, "out of stock")
}

func RefreshLocalOrder(ctx context.Context, order *model.ExternalShopOrder) (*model.ExternalShopOrder, error) {
	if order == nil {
		return nil, errors.New("order is nil")
	}
	now := time.Now()
	if ExpireLocalOrderIfTimedOut(order, now) {
		if err := order.Update(); err != nil {
			return nil, err
		}
		return order, nil
	}
	client, cfg, err := NewConfiguredClient()
	if err != nil {
		return nil, err
	}
	if order.ShopToken == "" {
		order.ShopToken = cfg.ShopToken
	}
	if order.UpstreamTradeNo == "" {
		return order, nil
	}

	state := &OrderSyncState{
		Status:        order.Status,
		TransactionID: order.TransactionId,
	}
	if order.PaymentConfirmedAt > 0 {
		ts := time.Unix(order.PaymentConfirmedAt, 0)
		state.PaymentConfirmedAt = &ts
	}
	if order.DeliveryConfirmedAt > 0 {
		ts := time.Unix(order.DeliveryConfirmedAt, 0)
		state.DeliveryConfirmedAt = &ts
	}

	payResp, err := client.QueryOrder(ctx, order.UpstreamTradeNo)
	if err != nil {
		order.LastError = err.Error()
		order.LastSyncAt = now.Unix()
		_ = order.Update()
		return nil, err
	}
	ApplyPayQueryResult(state, payResp, now)

	var orderInfoResp OrderInfoResponse
	if state.Status == OrderStatusPaidWaitingDelivery || state.Status == OrderStatusDelivered {
		orderInfoResp, err = client.GetOrderInfo(ctx, order.UpstreamTradeNo)
		if err == nil {
			ApplyOrderInfoResult(state, orderInfoResp, now)
		}
	}

	order.Status = state.Status
	order.TransactionId = state.TransactionID
	order.LastSyncAt = now.Unix()
	if state.PaymentConfirmedAt != nil {
		order.PaymentConfirmedAt = state.PaymentConfirmedAt.Unix()
	}
	if state.DeliveryConfirmedAt != nil {
		order.DeliveryConfirmedAt = state.DeliveryConfirmedAt.Unix()
	}
	if len(state.Cards) > 0 {
		payload, marshalErr := common.Marshal(map[string]interface{}{
			"cards": state.Cards,
		})
		if marshalErr == nil {
			order.DeliveryPayload = string(payload)
		}
	}
	if orderInfoResp.Code == 1 {
		payload, marshalErr := common.Marshal(orderInfoResp)
		if marshalErr == nil {
			order.ProviderPayload = string(payload)
		}
	}
	if err := order.Update(); err != nil {
		return nil, err
	}
	return order, nil
}

func pickChannel(channels []PaymentChannel, requestedID int) (*PaymentChannel, error) {
	if requestedID > 0 {
		for i := range channels {
			if channels[i].Id == requestedID {
				return &channels[i], nil
			}
		}
		return nil, errors.New("payment channel not found")
	}
	for i := range channels {
		if channels[i].Status == 1 && channels[i].CustomStatus == 1 {
			return &channels[i], nil
		}
	}
	return nil, errors.New("no available payment channel")
}
