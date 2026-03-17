package externalshop

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestFetchChannelsPersistsSnapshotAndListCachedChannelsRestoresIt(t *testing.T) {
	restore := setupExternalShopChannelCacheTest(t)
	defer restore()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/shopApi/Shop/getUserChannel", r.URL.Path)
		_, _ = w.Write([]byte(`{"code":1,"msg":"success","data":[{"id":1,"name":"支付宝","code":"alipay","show_name":"支付宝","status":1,"custom_status":1,"rate":0,"paytype":{"name":"支付宝","icon":"alipay.svg"}},{"id":2,"name":"微信","code":"wechat","show_name":"微信支付","status":1,"custom_status":1,"rate":0,"paytype":{"name":"微信","icon":"wechat.svg"}}]}`))
	}))
	defer server.Close()

	setExternalShopOptionForServiceTest("external_shop.ldxp.enabled", "true")
	setExternalShopOptionForServiceTest("external_shop.ldxp.shop_token", "shop-token")
	setExternalShopOptionForServiceTest("external_shop.ldxp.base_url", server.URL)

	channels, err := FetchChannels(context.Background())
	require.NoError(t, err)
	require.Len(t, channels, 2)

	snapshots, err := model.ListExternalShopChannelSnapshots(ProviderLDXP, "shop-token", true)
	require.NoError(t, err)
	require.Len(t, snapshots, 2)
	require.Equal(t, "alipay", snapshots[0].Code)
	require.Equal(t, "wechat", snapshots[1].Code)

	channelCache = channelCacheState{}

	restored := ListCachedChannels()
	require.Len(t, restored, 2)
	require.Equal(t, "支付宝", restored[0].ShowName)
	require.Equal(t, "微信支付", restored[1].ShowName)
}

func TestPrewarmChannelCacheLoadsPersistentSnapshot(t *testing.T) {
	restore := setupExternalShopChannelCacheTest(t)
	defer restore()

	originalTriggerChannelRefreshAsync := triggerChannelRefreshAsync
	triggerChannelRefreshAsync = func() {}
	defer func() {
		triggerChannelRefreshAsync = originalTriggerChannelRefreshAsync
	}()

	setExternalShopOptionForServiceTest("external_shop.ldxp.enabled", "true")
	setExternalShopOptionForServiceTest("external_shop.ldxp.shop_token", "shop-token")
	setExternalShopOptionForServiceTest("external_shop.ldxp.base_url", "https://pay.ldxp.cn")

	require.NoError(t, model.UpsertExternalShopChannelSnapshot(&model.ExternalShopChannelSnapshot{
		Provider:     ProviderLDXP,
		ShopToken:    "shop-token",
		ChannelId:    9,
		Name:         "支付宝",
		Code:         "alipay",
		ShowName:     "支付宝",
		Status:       1,
		CustomStatus: 1,
		Enabled:      true,
		SyncedAt:     1773700000,
	}))

	channelCache = channelCacheState{}

	PrewarmChannelCache()

	require.Len(t, ListCachedChannels(), 1)
	require.Equal(t, "支付宝", ListCachedChannels()[0].ShowName)
}

func setupExternalShopChannelCacheTest(t *testing.T) func() {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.ExternalShopChannelSnapshot{}))

	oldDB := model.DB
	oldLOGDB := model.LOG_DB
	oldUsingSQLite := common.UsingSQLite
	oldRedisEnabled := common.RedisEnabled

	common.OptionMapRWMutex.Lock()
	oldOptionMap := common.OptionMap
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()

	model.DB = db
	model.LOG_DB = db
	common.UsingSQLite = true
	common.RedisEnabled = false
	channelCache = channelCacheState{}

	return func() {
		model.DB = oldDB
		model.LOG_DB = oldLOGDB
		common.UsingSQLite = oldUsingSQLite
		common.RedisEnabled = oldRedisEnabled
		channelCache = channelCacheState{}

		common.OptionMapRWMutex.Lock()
		common.OptionMap = oldOptionMap
		common.OptionMapRWMutex.Unlock()
	}
}

func setExternalShopOptionForServiceTest(key string, value string) {
	common.OptionMapRWMutex.Lock()
	defer common.OptionMapRWMutex.Unlock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMap[key] = value
}
