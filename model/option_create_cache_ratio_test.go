package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func TestUpdateOptionMapCreateCacheRatio(t *testing.T) {
	ratio_setting.InitRatioSettings()

	common.OptionMapRWMutex.Lock()
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()

	modelName := "claude-3-sonnet-20240229"

	before, ok := ratio_setting.GetCreateCacheRatio(modelName)
	if !ok {
		t.Fatalf("expected default create cache ratio to exist for %s", modelName)
	}
	if before != 1.25 {
		t.Fatalf("unexpected default create cache ratio for %s: want 1.25, got %v", modelName, before)
	}

	err := updateOptionMap("CreateCacheRatio", `{"claude-3-sonnet-20240229":2.5}`)
	if err != nil {
		t.Fatalf("updateOptionMap returned error: %v", err)
	}

	got, ok := ratio_setting.GetCreateCacheRatio(modelName)
	if !ok {
		t.Fatalf("expected updated create cache ratio to exist for %s", modelName)
	}
	if got != 2.5 {
		t.Fatalf("unexpected updated create cache ratio for %s: want 2.5, got %v", modelName, got)
	}
}
