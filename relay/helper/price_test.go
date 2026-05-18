package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelPriceHelperTreatsZeroGroupRatioAsFreeModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldQuotaSetting := *operation_setting.GetQuotaSetting()
	oldGroupRatios := ratio_setting.GroupRatio2JSONString()
	oldModelPrices := ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() {
		*operation_setting.GetQuotaSetting() = oldQuotaSetting
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldGroupRatios))
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(oldModelPrices))
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1,"svip":1,"free-group":0}`))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"test-free-group-model":0.1}`))
	operation_setting.GetQuotaSetting().EnableFreeModelPreConsume = true

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{
		UsingGroup:      "free-group",
		UserGroup:       "default",
		OriginModelName: "test-free-group-model",
	}

	priceData, err := ModelPriceHelper(ctx, info, 128, &types.TokenCountMeta{})
	require.NoError(t, err)
	require.True(t, priceData.UsePrice)
	require.True(t, priceData.FreeModel)
	require.Equal(t, 0, priceData.QuotaToPreConsume)
}

func TestModelPriceHelperTieredUsesPreloadedRequestInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{"tiered-test-model":"tiered_expr"}`,
		"billing_setting.billing_expr": `{"tiered-test-model":"param(\"stream\") == true ? tier(\"stream\", p * 3) : tier(\"base\", p * 2)"}`,
	}))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/channel/test/1", nil)
	req.Body = nil
	req.ContentLength = 0
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	ctx.Set("group", "default")

	info := &relaycommon.RelayInfo{
		OriginModelName: "tiered-test-model",
		UserGroup:       "default",
		UsingGroup:      "default",
		RequestHeaders:  map[string]string{"Content-Type": "application/json"},
		BillingRequestInput: &billingexpr.RequestInput{
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    []byte(`{"stream":true}`),
		},
	}

	priceData, err := ModelPriceHelper(ctx, info, 1000, &types.TokenCountMeta{})
	require.NoError(t, err)
	require.Equal(t, 1500, priceData.QuotaToPreConsume)
	require.NotNil(t, info.TieredBillingSnapshot)
	require.Equal(t, "stream", info.TieredBillingSnapshot.EstimatedTier)
	require.Equal(t, billing_setting.BillingModeTieredExpr, info.TieredBillingSnapshot.BillingMode)
	require.Equal(t, common.QuotaPerUnit, info.TieredBillingSnapshot.QuotaPerUnit)
}

func TestModelPriceHelperPerCallUsesExpressionBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	oldGroupRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldGroupRatios))
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{"kling-audio":"per_call_expr"}`,
		"billing_setting.billing_expr": `{"kling-audio":"param(\"metadata.scenario\") == \"speech\" ? tier(\"语音合成\", 0.085) : tier(\"音效\", 0.425)"}`,
	}))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":2}`))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/v1/video/generations", nil)
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	info := &relaycommon.RelayInfo{
		OriginModelName: "kling-audio",
		UserGroup:       "default",
		UsingGroup:      "default",
		RequestHeaders:  map[string]string{"Content-Type": "application/json"},
		BillingRequestInput: &billingexpr.RequestInput{
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    []byte(`{"model":"kling-audio","metadata":{"scenario":"speech"}}`),
		},
	}

	priceData, err := ModelPriceHelperPerCall(ctx, info)
	require.NoError(t, err)
	require.True(t, priceData.UsePrice)
	require.Equal(t, 0.085, priceData.ModelPrice)
	require.Equal(t, 85000, priceData.Quota)
	require.NotNil(t, info.TieredBillingSnapshot)
	require.Equal(t, billing_setting.BillingModePerCallExpr, info.TieredBillingSnapshot.BillingMode)
	require.Equal(t, "语音合成", info.TieredBillingSnapshot.EstimatedTier)
}

func TestModelPriceHelperPerCallExpressionDefaultsToHigherPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	oldGroupRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldGroupRatios))
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{"kling-audio":"per_call_expr"}`,
		"billing_setting.billing_expr": `{"kling-audio":"param(\"metadata.scenario\") == \"speech\" ? tier(\"语音合成\", 0.085) : tier(\"音效\", 0.425)"}`,
	}))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1}`))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/v1/video/generations", nil)
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	info := &relaycommon.RelayInfo{
		OriginModelName: "kling-audio",
		UserGroup:       "default",
		UsingGroup:      "default",
		RequestHeaders:  map[string]string{"Content-Type": "application/json"},
		BillingRequestInput: &billingexpr.RequestInput{
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    []byte(`{"model":"kling-audio","metadata":{"scenario":"unknown"}}`),
		},
	}

	priceData, err := ModelPriceHelperPerCall(ctx, info)
	require.NoError(t, err)
	require.Equal(t, 0.425, priceData.ModelPrice)
	require.Equal(t, 212500, priceData.Quota)
	require.NotNil(t, info.TieredBillingSnapshot)
	require.Equal(t, "音效", info.TieredBillingSnapshot.EstimatedTier)
}

func TestModelPriceHelperPerCallUsesPricingProfileBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	oldGroupRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldGroupRatios))
	})

	_, err := ratio_setting.ApplyModelPricingProfilesByJSONString(`{
		"kling-audio": {
			"billing_mode": "per_call_expr",
			"billing_expr": "param(\"metadata.scenario\") == \"speech\" ? tier(\"语音合成\", 0.085) : tier(\"音效\", 0.425)",
			"items": [
				{"label":"文生音效","price":0.425,"unit":"次","conditions":{"metadata.scenario":"effect"}},
				{"label":"语音合成","price":0.085,"unit":"次","conditions":{"metadata.scenario":"speech"}}
			]
		}
	}`)
	require.NoError(t, err)
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1}`))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/v1/video/generations", nil)
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	info := &relaycommon.RelayInfo{
		OriginModelName: "kling-audio",
		UserGroup:       "default",
		UsingGroup:      "default",
		RequestHeaders:  map[string]string{"Content-Type": "application/json"},
		BillingRequestInput: &billingexpr.RequestInput{
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    []byte(`{"model":"kling-audio","metadata":{"scenario":"speech"}}`),
		},
	}

	priceData, err := ModelPriceHelperPerCall(ctx, info)
	require.NoError(t, err)
	require.Equal(t, 0.085, priceData.ModelPrice)
	require.Equal(t, 42500, priceData.Quota)
	require.Equal(t, "语音合成", info.TieredBillingSnapshot.EstimatedTier)
}

func TestModelPriceHelperPerCallUsesPerDurationPricingProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	oldGroupRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldGroupRatios))
	})

	_, err := ratio_setting.ApplyModelPricingProfilesByJSONString(`{
		"viduq2": {
			"shape": "per_duration",
			"unit": "second",
			"price_per_unit": 0.12,
			"duration_selector": "duration",
			"items": [
				{"label":"按秒","price":0.12,"unit":"秒"}
			]
		}
	}`)
	require.NoError(t, err)
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1}`))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/v1/video/generations", nil)
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req

	info := &relaycommon.RelayInfo{
		OriginModelName: "viduq2",
		UserGroup:       "default",
		UsingGroup:      "default",
		RequestHeaders:  map[string]string{"Content-Type": "application/json"},
		BillingRequestInput: &billingexpr.RequestInput{
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    []byte(`{"model":"viduq2","duration":8}`),
		},
	}

	priceData, err := ModelPriceHelperPerCall(ctx, info)
	require.NoError(t, err)
	require.Equal(t, 0.96, priceData.ModelPrice)
	require.Equal(t, 480000, priceData.Quota)
	require.Equal(t, "按秒", info.TieredBillingSnapshot.EstimatedTier)
}
