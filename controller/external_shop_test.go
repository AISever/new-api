package controller

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	externalshop "github.com/QuantumNous/new-api/service/external_shop"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to open test db: " + err.Error())
	}
	model.DB = db
	model.LOG_DB = db

	common.UsingSQLite = true
	common.RedisEnabled = false

	if err := db.AutoMigrate(&model.ExternalShopOrder{}); err != nil {
		panic("failed to migrate external shop orders: " + err.Error())
	}
	if err := db.AutoMigrate(&model.ExternalShopChannelSnapshot{}); err != nil {
		panic("failed to migrate external shop channel snapshots: " + err.Error())
	}
	if err := db.AutoMigrate(&model.ExternalShopGood{}); err != nil {
		panic("failed to migrate external shop goods: " + err.Error())
	}
	if err := db.AutoMigrate(&model.ExternalShopCategory{}); err != nil {
		panic("failed to migrate external shop categories: " + err.Error())
	}

	os.Exit(m.Run())
}

func TestGetExternalShopStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	setExternalShopOptionForTest("external_shop.ldxp.enabled", "false")
	setExternalShopOptionForTest("external_shop.ldxp.shop_token", "")
	setExternalShopOptionForTest("gptteamplan.enabled", "true")
	defer setExternalShopOptionForTest("gptteamplan.enabled", "false")
	cachedSeats := 4
	setCachedGPTTeamPlanRemainingSeats(&cachedSeats)

	GetExternalShopStatus(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"external_shop"`)
	require.Contains(t, recorder.Body.String(), `"ready":false`)
	require.Contains(t, recorder.Body.String(), `"gptteamplan"`)
	require.Contains(t, recorder.Body.String(), `"enabled":true`)
	require.Contains(t, recorder.Body.String(), `"remaining_seats":4`)
}

func TestAdminSyncPendingExternalShopOrdersRequiresConfiguredShop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	setExternalShopOptionForTest("external_shop.ldxp.enabled", "true")
	setExternalShopOptionForTest("external_shop.ldxp.shop_token", "")

	AdminSyncPendingExternalShopOrders(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":false`)
	require.Contains(t, recorder.Body.String(), `商城未配置`)
}

func TestAdminSyncExternalShopCatalogRequiresConfiguredShop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/external-shop/admin/sync", nil)

	setExternalShopOptionForTest("external_shop.ldxp.enabled", "true")
	setExternalShopOptionForTest("external_shop.ldxp.shop_token", "")

	AdminSyncExternalShopCatalog(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":false`)
	require.Contains(t, recorder.Body.String(), `商城未配置`)
}

func TestGetExternalShopGoodsReturnsCachedDataAndTriggersBackgroundRefresh(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/api/external-shop/goods", nil)

	setExternalShopOptionForTest("external_shop.ldxp.enabled", "true")
	setExternalShopOptionForTest("external_shop.ldxp.shop_token", "shop-token")
	setExternalShopOptionForTest("external_shop.ldxp.shop_name", "测试商城")

	originalRefreshCatalogAsync := refreshExternalShopCatalogAsync
	defer func() {
		refreshExternalShopCatalogAsync = originalRefreshCatalogAsync
	}()

	var refreshCalls atomic.Int32
	refreshExternalShopCatalogAsync = func() {
		refreshCalls.Add(1)
	}

	require.NoError(t, model.UpsertExternalShopGood(&model.ExternalShopGood{
		Provider:     "ldxp",
		ShopToken:    "shop-token",
		GoodsKey:     "goods-1",
		Name:         "商品1",
		CategoryId:   20746,
		CategoryName: "ChatGPT",
		Enabled:      true,
		SyncedAt:     1773664560,
	}))

	GetExternalShopGoods(ginContext)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.EqualValues(t, 1, refreshCalls.Load())
	require.Contains(t, recorder.Body.String(), `"goods_key":"goods-1"`)
}

func TestGetExternalShopChannelsReturnsCachedDataAndTriggersBackgroundRefresh(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/api/external-shop/channels", nil)

	setExternalShopOptionForTest("external_shop.ldxp.enabled", "true")
	setExternalShopOptionForTest("external_shop.ldxp.shop_token", "shop-token")

	originalListCachedExternalShopChannels := listCachedExternalShopChannels
	originalRefreshExternalShopChannelsAsync := refreshExternalShopChannelsAsync
	defer func() {
		listCachedExternalShopChannels = originalListCachedExternalShopChannels
		refreshExternalShopChannelsAsync = originalRefreshExternalShopChannelsAsync
	}()

	listCachedExternalShopChannels = func() []externalshop.PaymentChannel {
		return []externalshop.PaymentChannel{
			{Id: 1, ShowName: "支付宝", Code: "alipay"},
		}
	}

	var refreshCalls atomic.Int32
	refreshExternalShopChannelsAsync = func() {
		refreshCalls.Add(1)
	}

	GetExternalShopChannels(ginContext)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.EqualValues(t, 1, refreshCalls.Load())
	require.Contains(t, recorder.Body.String(), `"show_name":"支付宝"`)
}

func TestGetExternalShopOrdersReturnsCachedDataAndTriggersBackgroundRefresh(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/api/external-shop/orders?p=1&page_size=10&sort_by=created_at&sort_order=desc", nil)
	ginContext.Set("id", 9)

	originalRefreshExternalShopOrdersAsync := refreshExternalShopOrdersAsync
	defer func() {
		refreshExternalShopOrdersAsync = originalRefreshExternalShopOrdersAsync
	}()

	var refreshCalls atomic.Int32
	refreshExternalShopOrdersAsync = func(userId int, localTradeNos []string) {
		refreshCalls.Add(1)
		require.Equal(t, 9, userId)
		require.Equal(t, []string{"ESHOPTEST-CACHED"}, localTradeNos)
	}

	require.NoError(t, (&model.ExternalShopOrder{
		LocalTradeNo: "ESHOPTEST-CACHED",
		UserId:       9,
		Provider:     "ldxp",
		ShopToken:    "shop-token",
		GoodsKey:     "goods-key",
		GoodsName:    "goods",
		Status:       "pending_payment",
	}).Insert())

	GetExternalShopOrders(ginContext)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.EqualValues(t, 1, refreshCalls.Load())
	require.Contains(t, recorder.Body.String(), `"local_trade_no":"ESHOPTEST-CACHED"`)
}

func TestDeleteExternalShopOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "local_trade_no", Value: "ESHOPTEST001"}}
	context.Set("id", 7)

	order := &model.ExternalShopOrder{
		LocalTradeNo: "ESHOPTEST001",
		UserId:       7,
		Provider:     "ldxp",
		ShopToken:    "shop-token",
		GoodsKey:     "goods-key",
		GoodsName:    "goods",
		Status:       "created",
	}
	require.NoError(t, order.Insert())

	DeleteExternalShopOrder(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)
	_, err := model.GetExternalShopOrderByLocalTradeNo("ESHOPTEST001")
	require.Error(t, err)
}

func TestDeleteExternalShopOrderRejectsOtherUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "local_trade_no", Value: "ESHOPTEST002"}}
	context.Set("id", 7)

	order := &model.ExternalShopOrder{
		LocalTradeNo: "ESHOPTEST002",
		UserId:       8,
		Provider:     "ldxp",
		ShopToken:    "shop-token",
		GoodsKey:     "goods-key",
		GoodsName:    "goods",
		Status:       "created",
	}
	require.NoError(t, order.Insert())

	DeleteExternalShopOrder(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":false`)
	require.Contains(t, recorder.Body.String(), `无权删除该订单`)
	_, err := model.GetExternalShopOrderByLocalTradeNo("ESHOPTEST002")
	require.NoError(t, err)
}

func setExternalShopOptionForTest(key string, value string) {
	common.OptionMapRWMutex.Lock()
	defer common.OptionMapRWMutex.Unlock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMap[key] = value
	if common.OptionMap["external_shop.ldxp.base_url"] == "" {
		common.OptionMap["external_shop.ldxp.base_url"] = "https://pay.ldxp.cn"
	}
}
