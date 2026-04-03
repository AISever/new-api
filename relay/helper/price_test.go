package helper

import (
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
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
