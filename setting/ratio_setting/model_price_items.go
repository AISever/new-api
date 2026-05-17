package ratio_setting

import (
	"github.com/QuantumNous/new-api/types"
)

type ModelPriceItem struct {
	Label string  `json:"label"`
	Price float64 `json:"price"`
	Unit  string  `json:"unit,omitempty"`
}

var modelPriceItemsMap = types.NewRWMap[string, []ModelPriceItem]()

func ModelPriceItems2JSONString() string {
	return modelPriceItemsMap.MarshalJSONString()
}

func UpdateModelPriceItemsByJSONString(jsonStr string) error {
	return types.LoadFromJsonStringWithCallback(modelPriceItemsMap, jsonStr, InvalidateExposedDataCache)
}

func GetModelPriceItems(name string) ([]ModelPriceItem, bool) {
	name = FormatMatchingModelName(name)
	items, ok := modelPriceItemsMap.Get(name)
	if !ok {
		return nil, false
	}
	copied := make([]ModelPriceItem, len(items))
	copy(copied, items)
	return copied, true
}

func GetModelPriceItemsCopy() map[string][]ModelPriceItem {
	source := modelPriceItemsMap.ReadAll()
	copied := make(map[string][]ModelPriceItem, len(source))
	for model, items := range source {
		modelItems := make([]ModelPriceItem, len(items))
		copy(modelItems, items)
		copied[model] = modelItems
	}
	return copied
}
