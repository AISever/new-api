package operation_setting

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

type LdxpTopupProduct struct {
	Amount    float64 `json:"amount"`
	GoodsKey  string  `json:"goods_key"`
	Label     string  `json:"label"`
	Enabled   bool    `json:"enabled"`
	SortOrder int     `json:"sort_order"`
}

type PaymentSetting struct {
	AmountOptions     []int              `json:"amount_options"`
	AmountDiscount    map[int]float64    `json:"amount_discount"` // 充值金额对应的折扣，例如 100 元 0.9 表示 100 元充值享受 9 折优惠
	LdxpTopupProducts []LdxpTopupProduct `json:"ldxp_topup_products"`
}

// 默认配置
var paymentSetting = PaymentSetting{
	AmountOptions:     []int{10, 20, 50, 100, 200, 500},
	AmountDiscount:    map[int]float64{},
	LdxpTopupProducts: []LdxpTopupProduct{},
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("payment_setting", &paymentSetting)
}

func GetPaymentSetting() *PaymentSetting {
	return &paymentSetting
}

func NormalizeLdxpTopupAmount(amount float64) float64 {
	if amount <= 0 {
		return 0
	}
	normalized := math.Round(amount*100) / 100
	if math.Abs(normalized) < 0.000001 {
		return 0
	}
	return normalized
}

func FormatLdxpTopupAmount(amount float64) string {
	normalized := NormalizeLdxpTopupAmount(amount)
	if normalized <= 0 {
		return "0"
	}
	return strconv.FormatFloat(normalized, 'f', -1, 64)
}

func LdxpTopupAmountKey(amount float64) string {
	normalized := NormalizeLdxpTopupAmount(amount)
	if normalized <= 0 {
		return ""
	}
	return strconv.FormatInt(int64(math.Round(normalized*100)), 10)
}

func NormalizeLdxpTopupProducts(products []LdxpTopupProduct) []LdxpTopupProduct {
	if len(products) == 0 {
		return []LdxpTopupProduct{}
	}
	normalized := make([]LdxpTopupProduct, 0, len(products))
	seenAmounts := make(map[string]struct{}, len(products))
	for _, product := range products {
		product.Amount = NormalizeLdxpTopupAmount(product.Amount)
		product.GoodsKey = strings.TrimSpace(product.GoodsKey)
		product.Label = strings.TrimSpace(product.Label)
		if product.Amount <= 0 || product.GoodsKey == "" {
			continue
		}
		amountKey := LdxpTopupAmountKey(product.Amount)
		if amountKey == "" {
			continue
		}
		if _, ok := seenAmounts[amountKey]; ok {
			continue
		}
		seenAmounts[amountKey] = struct{}{}
		if product.Label == "" {
			product.Label = FormatLdxpTopupAmount(product.Amount)
		}
		normalized = append(normalized, product)
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		if normalized[i].SortOrder != normalized[j].SortOrder {
			return normalized[i].SortOrder < normalized[j].SortOrder
		}
		if normalized[i].Amount != normalized[j].Amount {
			return normalized[i].Amount < normalized[j].Amount
		}
		return normalized[i].GoodsKey < normalized[j].GoodsKey
	})
	return normalized
}

func ValidateLdxpTopupProducts(products []LdxpTopupProduct) ([]LdxpTopupProduct, error) {
	if len(products) == 0 {
		return []LdxpTopupProduct{}, nil
	}
	validated := make([]LdxpTopupProduct, 0, len(products))
	seenAmounts := make(map[string]struct{}, len(products))
	seenGoodsKeys := make(map[string]struct{}, len(products))
	for i, product := range products {
		product.Amount = NormalizeLdxpTopupAmount(product.Amount)
		product.GoodsKey = strings.TrimSpace(product.GoodsKey)
		product.Label = strings.TrimSpace(product.Label)
		if product.Amount <= 0 {
			return nil, fmt.Errorf("第 %d 项 amount 必须大于 0", i+1)
		}
		if product.GoodsKey == "" {
			return nil, fmt.Errorf("第 %d 项 goods_key 不能为空", i+1)
		}
		if product.Label == "" {
			return nil, fmt.Errorf("第 %d 项 label 不能为空", i+1)
		}
		amountKey := LdxpTopupAmountKey(product.Amount)
		if amountKey == "" {
			return nil, fmt.Errorf("第 %d 项 amount 必须大于 0", i+1)
		}
		if _, ok := seenAmounts[amountKey]; ok {
			return nil, fmt.Errorf("第 %d 项 amount 重复", i+1)
		}
		if _, ok := seenGoodsKeys[product.GoodsKey]; ok {
			return nil, fmt.Errorf("第 %d 项 goods_key 重复", i+1)
		}
		seenAmounts[amountKey] = struct{}{}
		seenGoodsKeys[product.GoodsKey] = struct{}{}
		validated = append(validated, product)
	}
	return NormalizeLdxpTopupProducts(validated), nil
}

func ValidateLdxpTopupProductsJSON(jsonString string) ([]LdxpTopupProduct, error) {
	if strings.TrimSpace(jsonString) == "" {
		return []LdxpTopupProduct{}, nil
	}
	var rawProducts []map[string]any
	if err := common.UnmarshalJsonStr(jsonString, &rawProducts); err != nil {
		return nil, err
	}
	requiredKeys := []string{"amount", "goods_key", "label", "enabled", "sort_order"}
	for i, rawProduct := range rawProducts {
		for _, key := range requiredKeys {
			if _, ok := rawProduct[key]; !ok {
				return nil, fmt.Errorf("第 %d 项缺少 %s 字段", i+1, key)
			}
		}
	}
	var products []LdxpTopupProduct
	if err := common.UnmarshalJsonStr(jsonString, &products); err != nil {
		return nil, err
	}
	return ValidateLdxpTopupProducts(products)
}

func (p *PaymentSetting) GetEnabledLdxpTopupProducts() []LdxpTopupProduct {
	if p == nil {
		return []LdxpTopupProduct{}
	}
	normalized := NormalizeLdxpTopupProducts(p.LdxpTopupProducts)
	enabled := make([]LdxpTopupProduct, 0, len(normalized))
	for _, product := range normalized {
		if !product.Enabled {
			continue
		}
		enabled = append(enabled, product)
	}
	return enabled
}
