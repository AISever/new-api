package operation_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
)

func TestTaskPricePatchModelsFromString(t *testing.T) {
	original := append([]string(nil), constant.TaskPricePatches...)
	t.Cleanup(func() {
		constant.TaskPricePatches = original
	})

	TaskPricePatchModelsFromString(" sora-2, veo-3.1-fast-generate-preview ,, sora-2 ")

	assert.Equal(t, []string{
		"sora-2",
		"veo-3.1-fast-generate-preview",
		"sora-2",
	}, constant.TaskPricePatches)
}

func TestTaskPricePatchModelsFromString_EmptyClears(t *testing.T) {
	original := append([]string(nil), constant.TaskPricePatches...)
	t.Cleanup(func() {
		constant.TaskPricePatches = original
	})

	constant.TaskPricePatches = []string{"sora-2"}

	TaskPricePatchModelsFromString("   ")

	assert.Empty(t, constant.TaskPricePatches)
}
