package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUpdateOption_TaskPricePatchModelsSyncsRuntime(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	originalDB := DB
	originalLogDB := LOG_DB
	originalOptionMap := common.OptionMap
	originalTaskPricePatches := append([]string(nil), constant.TaskPricePatches...)
	originalGeneralSetting := *operation_setting.GetGeneralSetting()
	t.Cleanup(func() {
		DB = originalDB
		LOG_DB = originalLogDB
		common.OptionMap = originalOptionMap
		constant.TaskPricePatches = originalTaskPricePatches
		*operation_setting.GetGeneralSetting() = originalGeneralSetting
	})

	DB = db
	LOG_DB = db
	require.NoError(t, db.AutoMigrate(&Option{}))

	common.OptionMap = map[string]string{}
	constant.TaskPricePatches = nil
	operation_setting.GetGeneralSetting().TaskPricePatchModels = ""

	err = UpdateOption("general_setting.task_price_patch_models", "sora-2,veo-3.1-generate-preview")
	require.NoError(t, err)

	assert.Equal(t, "sora-2,veo-3.1-generate-preview", operation_setting.GetGeneralSetting().TaskPricePatchModels)
	assert.Equal(t, []string{"sora-2", "veo-3.1-generate-preview"}, constant.TaskPricePatches)
	assert.Equal(t, "sora-2,veo-3.1-generate-preview", common.OptionMap["general_setting.task_price_patch_models"])
}

func TestLoadFromDB_TaskPricePatchModelsOverridesEnvSeed(t *testing.T) {
	originalOptionMap := common.OptionMap
	originalTaskPricePatches := append([]string(nil), constant.TaskPricePatches...)
	originalGeneralSetting := *operation_setting.GetGeneralSetting()
	t.Cleanup(func() {
		common.OptionMap = originalOptionMap
		constant.TaskPricePatches = originalTaskPricePatches
		*operation_setting.GetGeneralSetting() = originalGeneralSetting
	})

	common.OptionMap = map[string]string{}
	constant.TaskPricePatches = []string{"env-model"}
	operation_setting.GetGeneralSetting().TaskPricePatchModels = ""

	err := updateOptionMap("general_setting.task_price_patch_models", "db-model-a, db-model-b")
	require.NoError(t, err)

	assert.Equal(t, "db-model-a, db-model-b", operation_setting.GetGeneralSetting().TaskPricePatchModels)
	assert.Equal(t, []string{"db-model-a", "db-model-b"}, constant.TaskPricePatches)
}
