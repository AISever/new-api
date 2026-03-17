package externalshop

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
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

var (
	catalogFreshTTL        = time.Minute
	catalogSyncMu          sync.Mutex
	catalogRefreshInFlight atomic.Bool
)

type channelCacheState struct {
	mu        sync.RWMutex
	channels  []PaymentChannel
	fetchedAt time.Time
}

var (
	channelCache           channelCacheState
	channelCacheTTL        = time.Minute
	channelRefreshInFlight atomic.Bool
)

var triggerChannelRefreshAsync = func() {
	RefreshChannelsAsync()
}

type persistedChannelCache struct {
	Channels  []PaymentChannel `json:"channels"`
	FetchedAt int64            `json:"fetched_at"`
}

func RefreshUserOrders(ctx context.Context, userId int, localTradeNos []string) {
	now := time.Now()
	seen := make(map[string]struct{}, len(localTradeNos))
	for _, localTradeNo := range localTradeNos {
		localTradeNo = strings.TrimSpace(localTradeNo)
		if localTradeNo == "" {
			continue
		}
		if _, ok := seen[localTradeNo]; ok {
			continue
		}
		seen[localTradeNo] = struct{}{}

		order, err := model.GetExternalShopOrderByLocalTradeNo(localTradeNo)
		if err != nil || order == nil || order.UserId != userId {
			continue
		}
		if ExpireLocalOrderIfTimedOut(order, now) {
			_ = order.Update()
			continue
		}
		if !shouldSyncOrder(order, now, autoSyncMinInterval) {
			continue
		}
		_, _ = RefreshLocalOrder(ctx, order)
	}
}

func RefreshUserOrdersAsync(userId int, localTradeNos []string) {
	if userId <= 0 || len(localTradeNos) == 0 {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		RefreshUserOrders(ctx, userId, localTradeNos)
	}()
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
		model.UpsertExternalShopCategory,
		model.UpsertExternalShopGood,
		model.DisableMissingExternalShopCategories,
		model.DisableMissingExternalShopGoods,
		time.Now,
	)
}

func EnsureCatalogFresh(ctx context.Context) error {
	cfg := GetConfig()
	if !cfg.IsReady() {
		return errors.New("external shop is not configured")
	}
	if catalogIsFresh(cfg, time.Now()) {
		return nil
	}
	catalogSyncMu.Lock()
	defer catalogSyncMu.Unlock()
	if catalogIsFresh(cfg, time.Now()) {
		return nil
	}
	_, err := SyncCatalog(ctx)
	return err
}

func RefreshCatalogAsync() {
	if !catalogRefreshInFlight.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer catalogRefreshInFlight.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = EnsureCatalogFresh(ctx)
	}()
}

func ListCachedChannels() []PaymentChannel {
	cfg := GetConfig()
	if !cfg.IsReady() {
		return nil
	}
	channelCache.mu.RLock()
	if len(channelCache.channels) > 0 {
		channels := make([]PaymentChannel, len(channelCache.channels))
		copy(channels, channelCache.channels)
		channelCache.mu.RUnlock()
		return channels
	}
	channelCache.mu.RUnlock()

	channels, fetchedAt, err := loadPersistedChannels(ProviderLDXP, cfg.ShopToken)
	if err != nil || len(channels) == 0 {
		return nil
	}
	storeChannelsInMemory(channels, fetchedAt)
	cloned := make([]PaymentChannel, len(channels))
	copy(cloned, channels)
	return cloned
}

func FetchChannels(ctx context.Context) ([]PaymentChannel, error) {
	client, cfg, err := NewConfiguredClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListChannels(ctx)
	if err != nil {
		return nil, err
	}
	if resp.Code != 1 {
		return nil, fmt.Errorf("list channels failed: %s", strings.TrimSpace(resp.Msg))
	}
	fetchedAt := time.Now()
	if err := persistChannelsSnapshot(ProviderLDXP, cfg.ShopToken, resp.Data, fetchedAt); err != nil {
		return nil, err
	}
	storeChannelsInMemory(resp.Data, fetchedAt)
	return resp.Data, nil
}

func RefreshChannelsAsync() {
	if !channelRefreshInFlight.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer channelRefreshInFlight.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, _ = FetchChannels(ctx)
	}()
}

func PrewarmChannelCache() {
	cfg := GetConfig()
	if !cfg.IsReady() {
		return
	}
	channels, fetchedAt, err := loadPersistedChannels(ProviderLDXP, cfg.ShopToken)
	if err == nil && len(channels) > 0 {
		storeChannelsInMemory(channels, fetchedAt)
	}
	triggerChannelRefreshAsync()
}

func storeChannelsInMemory(channels []PaymentChannel, fetchedAt time.Time) {
	channelCache.mu.Lock()
	channelCache.channels = append([]PaymentChannel(nil), channels...)
	channelCache.fetchedAt = fetchedAt
	channelCache.mu.Unlock()
}

func loadPersistedChannels(provider string, shopToken string) ([]PaymentChannel, time.Time, error) {
	if strings.TrimSpace(provider) == "" || strings.TrimSpace(shopToken) == "" {
		return nil, time.Time{}, nil
	}
	if channels, fetchedAt, err := loadChannelsFromRedis(provider, shopToken); err == nil && len(channels) > 0 {
		return channels, fetchedAt, nil
	}
	snapshots, err := model.ListExternalShopChannelSnapshots(provider, shopToken, true)
	if err != nil || len(snapshots) == 0 {
		return nil, time.Time{}, err
	}
	channels := make([]PaymentChannel, 0, len(snapshots))
	var latestSyncedAt int64
	for _, snapshot := range snapshots {
		channels = append(channels, PaymentChannel{
			Id:           snapshot.ChannelId,
			Name:         snapshot.Name,
			Code:         snapshot.Code,
			ShowName:     snapshot.ShowName,
			Status:       snapshot.Status,
			CustomStatus: snapshot.CustomStatus,
			Rate:         snapshot.Rate,
			PayType: PaymentChannelPayType{
				Name: snapshot.PayTypeName,
				Icon: snapshot.PayTypeIcon,
			},
		})
		if snapshot.SyncedAt > latestSyncedAt {
			latestSyncedAt = snapshot.SyncedAt
		}
	}
	fetchedAt := time.Unix(latestSyncedAt, 0)
	persistChannelsToRedis(provider, shopToken, channels, fetchedAt)
	return channels, fetchedAt, nil
}

func persistChannelsSnapshot(provider string, shopToken string, channels []PaymentChannel, fetchedAt time.Time) error {
	keepChannelIDs := make([]int, 0, len(channels))
	for _, channel := range channels {
		rawPayload, err := common.Marshal(channel)
		if err != nil {
			return err
		}
		if err := model.UpsertExternalShopChannelSnapshot(&model.ExternalShopChannelSnapshot{
			Provider:     provider,
			ShopToken:    shopToken,
			ChannelId:    channel.Id,
			Name:         channel.Name,
			Code:         channel.Code,
			ShowName:     channel.ShowName,
			Status:       channel.Status,
			CustomStatus: channel.CustomStatus,
			Rate:         channel.Rate,
			PayTypeName:  channel.PayType.Name,
			PayTypeIcon:  channel.PayType.Icon,
			Enabled:      true,
			RawPayload:   string(rawPayload),
			SyncedAt:     fetchedAt.Unix(),
		}); err != nil {
			return err
		}
		keepChannelIDs = append(keepChannelIDs, channel.Id)
	}
	if _, err := model.DisableMissingExternalShopChannelSnapshots(provider, shopToken, keepChannelIDs); err != nil {
		return err
	}
	persistChannelsToRedis(provider, shopToken, channels, fetchedAt)
	return nil
}

func loadChannelsFromRedis(provider string, shopToken string) ([]PaymentChannel, time.Time, error) {
	if !common.RedisEnabled || common.RDB == nil {
		return nil, time.Time{}, errors.New("redis is not enabled")
	}
	payload, err := common.RedisGet(buildChannelSnapshotRedisKey(provider, shopToken))
	if err != nil || strings.TrimSpace(payload) == "" {
		return nil, time.Time{}, err
	}
	var snapshot persistedChannelCache
	if err := common.UnmarshalJsonStr(payload, &snapshot); err != nil {
		return nil, time.Time{}, err
	}
	return snapshot.Channels, time.Unix(snapshot.FetchedAt, 0), nil
}

func persistChannelsToRedis(provider string, shopToken string, channels []PaymentChannel, fetchedAt time.Time) {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	payload, err := common.Marshal(persistedChannelCache{
		Channels:  channels,
		FetchedAt: fetchedAt.Unix(),
	})
	if err != nil {
		return
	}
	_ = common.RedisSet(buildChannelSnapshotRedisKey(provider, shopToken), string(payload), 24*time.Hour)
}

func buildChannelSnapshotRedisKey(provider string, shopToken string) string {
	return fmt.Sprintf("external_shop:channels:%s:%s", strings.TrimSpace(provider), strings.TrimSpace(shopToken))
}

func catalogIsFresh(cfg Config, now time.Time) bool {
	count, latestSyncedAt, err := model.GetExternalShopCatalogSyncState(ProviderLDXP, cfg.ShopToken)
	if err != nil || count == 0 || latestSyncedAt <= 0 {
		return false
	}
	return now.Unix()-latestSyncedAt < int64(catalogFreshTTL/time.Second)
}

func ListCatalogCategories(ctx context.Context) ([]Category, error) {
	_, cfg, err := NewConfiguredClient()
	if err != nil {
		return nil, err
	}
	cachedCategories, err := model.ListExternalShopCategories(ProviderLDXP, cfg.ShopToken, true)
	if err != nil {
		return nil, err
	}
	categories := make([]Category, 0, len(cachedCategories))
	for _, category := range cachedCategories {
		if len(cfg.AllowedCategoryIDs) > 0 {
			if _, ok := cfg.AllowedCategoryIDs[category.CategoryId]; !ok {
				continue
			}
		}
		categories = append(categories, Category{
			Id:         category.CategoryId,
			Name:       category.Name,
			Image:      category.Image,
			GoodsCount: category.GoodsCount,
		})
	}
	return categories, nil
}

func listCategoriesFromSource(ctx context.Context, client catalogClient, cfg Config) ([]Category, error) {
	resp, err := client.ListCategories(ctx, "card")
	if err != nil {
		return nil, err
	}
	if resp.Code != 1 {
		return nil, fmt.Errorf("list categories failed: %s", strings.TrimSpace(resp.Msg))
	}
	categories := make([]Category, 0, len(resp.Data))
	for _, category := range resp.Data {
		if len(cfg.AllowedCategoryIDs) > 0 {
			if _, ok := cfg.AllowedCategoryIDs[category.Id]; !ok {
				continue
			}
		}
		categories = append(categories, category)
	}
	return categories, nil
}

func syncCatalogWithClient(
	ctx context.Context,
	client catalogClient,
	cfg Config,
	pageSize int,
	upsertCategory func(*model.ExternalShopCategory) error,
	upsert func(*model.ExternalShopGood) error,
	disableMissingCategories func(provider string, shopToken string, keepCategoryIDs []int) (int64, error),
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

	categories, err := listCategoriesFromSource(ctx, client, cfg)
	if err != nil {
		return nil, err
	}

	keepGoodsKeys := make([]string, 0)
	keepCategoryIDs := make([]int, 0)
	totalSynced := 0
	nowUnix := now().Unix()
	for categoryIndex, category := range categories {
		if err := upsertCategory(&model.ExternalShopCategory{
			Provider:   ProviderLDXP,
			ShopToken:  cfg.ShopToken,
			CategoryId: category.Id,
			Name:       category.Name,
			Image:      category.Image,
			GoodsCount: category.GoodsCount,
			SortIndex:  categoryIndex,
			Enabled:    true,
			SyncedAt:   nowUnix,
		}); err != nil {
			return nil, err
		}
		keepCategoryIDs = append(keepCategoryIDs, category.Id)
		items, err := listAllCategoryGoods(ctx, client, category.Id, pageSize)
		if err != nil {
			return nil, err
		}
		for goodsIndex, item := range items {
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
				CategorySort:        categoryIndex,
				SortIndex:           goodsIndex,
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
	if _, err := disableMissingCategories(ProviderLDXP, cfg.ShopToken, keepCategoryIDs); err != nil {
		return nil, err
	}
	disabledCount, err := disableMissing(ProviderLDXP, cfg.ShopToken, keepGoodsKeys)
	if err != nil {
		return nil, err
	}
	return &SyncSummary{
		ShopName:      cfg.ShopName,
		Categories:    len(categories),
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
	if err := EnsureCatalogFresh(ctx); err != nil {
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
	if quantity > int(good.StockCount) {
		return fmt.Errorf("quantity exceeds stock %d", good.StockCount)
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
