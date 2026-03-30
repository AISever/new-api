package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

var ldxpJUUIDPattern = regexp.MustCompile(`juuid\s*=\s*'([^']+)'`)
var ldxpTopupAmountPattern = regexp.MustCompile(`(?i)充值\s*(\d+)\s*r`)

type LdxpConfig struct {
	Enabled          bool
	BaseURL          string
	ShopToken        string
	DefaultChannelId int
}

func GetLdxpConfig() LdxpConfig {
	baseURL := strings.TrimRight(strings.TrimSpace(setting.LdxpBaseURL), "/")
	if baseURL == "" {
		baseURL = "https://pay.ldxp.cn"
	}
	return LdxpConfig{
		Enabled:          setting.LdxpEnabled,
		BaseURL:          baseURL,
		ShopToken:        strings.TrimSpace(setting.LdxpShopToken),
		DefaultChannelId: setting.LdxpDefaultChannelId,
	}
}

func (c LdxpConfig) IsReady() bool {
	return c.Enabled && c.BaseURL != "" && c.ShopToken != "" && c.DefaultChannelId > 0
}

func GetEnabledLdxpTopupProducts() []operation_setting.LdxpTopupProduct {
	return operation_setting.GetPaymentSetting().GetEnabledLdxpTopupProducts()
}

func FindLdxpTopupProductByAmount(amount int64) (*operation_setting.LdxpTopupProduct, bool) {
	for _, product := range GetEnabledLdxpTopupProducts() {
		if int64(product.Amount) != amount {
			continue
		}
		productCopy := product
		return &productCopy, true
	}
	return nil, false
}

func ResolveLdxpTopupSettlementAmount(product operation_setting.LdxpTopupProduct) float64 {
	return float64(product.Amount)
}

func ResolveLdxpTopupGrantedAmount(product operation_setting.LdxpTopupProduct) int64 {
	return int64(product.Amount)
}

func IsLdxpTopupEnabled() bool {
	return GetLdxpConfig().IsReady() && len(GetEnabledLdxpTopupProducts()) > 0
}

func HasAnyLdxpSubscriptionPlan() bool {
	if !GetLdxpConfig().IsReady() {
		return false
	}
	var count int64
	if err := model.DB.Model(&model.SubscriptionPlan{}).
		Where("enabled = ? AND ldxp_goods_key <> ''", true).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func AmountMatchesLdxpQuote(expected float64, quoted float64) bool {
	return math.Abs(expected-quoted) < 0.01
}

func ResolveLdxpPaymentURL(baseURL string, providerTradeNo string, providerPayload string) string {
	providerTradeNo = strings.TrimSpace(providerTradeNo)
	providerPayload = strings.TrimSpace(providerPayload)

	if providerPayload != "" {
		var payload struct {
			Data struct {
				PayURL string `json:"payurl"`
			} `json:"data"`
		}
		if err := common.UnmarshalJsonStr(providerPayload, &payload); err == nil {
			payURL := strings.TrimSpace(payload.Data.PayURL)
			if payURL != "" {
				return payURL
			}
		}
		var mergedPayload struct {
			Query struct {
				Data struct {
					PayURL string `json:"payurl"`
				} `json:"data"`
			} `json:"query"`
		}
		if err := common.UnmarshalJsonStr(providerPayload, &mergedPayload); err == nil {
			payURL := strings.TrimSpace(mergedPayload.Query.Data.PayURL)
			if payURL != "" {
				return payURL
			}
		}
	}

	if providerTradeNo == "" {
		return ""
	}

	normalizedBaseURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalizedBaseURL == "" {
		normalizedBaseURL = "https://pay.ldxp.cn"
	}

	return fmt.Sprintf(
		"%s/shopApi/Pay/payment?trade_no=%s",
		normalizedBaseURL,
		url.QueryEscape(providerTradeNo),
	)
}

func ParseLdxpShopInput(baseURL string, shopInput string) (string, string, error) {
	normalizedBaseURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalizedBaseURL == "" {
		normalizedBaseURL = "https://pay.ldxp.cn"
	}

	raw := strings.TrimSpace(shopInput)
	if raw == "" {
		return normalizedBaseURL, "", fmt.Errorf("shop input is empty")
	}

	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		parsedURL, err := url.Parse(raw)
		if err != nil {
			return "", "", err
		}
		if parsedURL.Scheme != "" && parsedURL.Host != "" {
			normalizedBaseURL = strings.TrimRight(parsedURL.Scheme+"://"+parsedURL.Host, "/")
		}
		pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		if len(pathParts) >= 2 && strings.EqualFold(pathParts[0], "shop") && strings.TrimSpace(pathParts[1]) != "" {
			return normalizedBaseURL, strings.TrimSpace(pathParts[1]), nil
		}
		if len(pathParts) == 1 && strings.TrimSpace(pathParts[0]) != "" {
			return normalizedBaseURL, strings.TrimSpace(pathParts[0]), nil
		}
		return "", "", fmt.Errorf("invalid ldxp shop url")
	}

	return normalizedBaseURL, raw, nil
}

type LDXPClient struct {
	baseURL    string
	shopToken  string
	httpClient *http.Client
}

func NewLDXPClient(config LdxpConfig) *LDXPClient {
	return &LDXPClient{
		baseURL:    config.BaseURL,
		shopToken:  config.ShopToken,
		httpClient: GetHttpClient(),
	}
}

type LdxpBaseResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type LdxpGoodsPriceData struct {
	OriginalAmount float64 `json:"original_amount"`
	TotalAmount    float64 `json:"total_amount"`
}

type LdxpGoodsPriceResponse struct {
	LdxpBaseResponse
	Data LdxpGoodsPriceData `json:"data"`
}

type LdxpShopInfoData struct {
	Link        string `json:"link"`
	Nickname    string `json:"nickname"`
	Description string `json:"description"`
	Token       string `json:"token"`
}

type LdxpShopInfoResponse struct {
	LdxpBaseResponse
	Data LdxpShopInfoData `json:"data"`
}

type LdxpCategoryItem struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	GoodsCount int    `json:"goods_count"`
}

type LdxpCategoryListResponse struct {
	LdxpBaseResponse
	Data []LdxpCategoryItem `json:"data"`
}

type LdxpGoodsCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type LdxpGoodsUser struct {
	Link     string `json:"link"`
	Nickname string `json:"nickname"`
	Token    string `json:"token"`
}

type LdxpGoodsItem struct {
	Link      string            `json:"link"`
	GoodsType string            `json:"goods_type"`
	GoodsKey  string            `json:"goods_key"`
	Name      string            `json:"name"`
	Price     float64           `json:"price"`
	Category  LdxpGoodsCategory `json:"category"`
	User      LdxpGoodsUser     `json:"user"`
}

type LdxpGoodsListData struct {
	Total int             `json:"total"`
	List  []LdxpGoodsItem `json:"list"`
}

type LdxpGoodsListResponse struct {
	LdxpBaseResponse
	Data LdxpGoodsListData `json:"data"`
}

type LdxpChannelPayType struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type LdxpChannelItem struct {
	ID           int                `json:"id"`
	Name         string             `json:"name"`
	ShowName     string             `json:"show_name"`
	Status       int                `json:"status"`
	CustomStatus int                `json:"custom_status"`
	Rate         float64            `json:"rate"`
	PayType      LdxpChannelPayType `json:"paytype"`
}

type LdxpChannelListResponse struct {
	LdxpBaseResponse
	Data []LdxpChannelItem `json:"data"`
}

type LdxpCreateOrderExtend struct {
	JUUID string `json:"juuid"`
}

type LdxpCreateOrderRequest struct {
	GoodsKey       string                `json:"goods_key"`
	Quantity       int                   `json:"quantity"`
	CouponCode     string                `json:"coupon_code"`
	ChannelID      int                   `json:"channel_id"`
	Contact        string                `json:"contact"`
	QueryPassword  string                `json:"query_password"`
	SelectCardsIDs []int                 `json:"select_cards_ids"`
	Extend         LdxpCreateOrderExtend `json:"extend"`
}

type LdxpCreateOrderData struct {
	TradeNo     string  `json:"trade_no"`
	TotalAmount float64 `json:"total_amount"`
	PayURL      string  `json:"payurl"`
}

type LdxpCreateOrderResponse struct {
	LdxpBaseResponse
	Data LdxpCreateOrderData `json:"data"`
}

type LdxpPayQueryResponse struct {
	LdxpBaseResponse
	Data interface{} `json:"data"`
}

func (r LdxpPayQueryResponse) IsPaid() bool {
	return r.Code == 1
}

type LdxpOrderInfoDeliverResponse struct {
	Cards          []string `json:"cards"`
	ExportCardsURL string   `json:"export_cards_url"`
}

type LdxpOrderInfoData struct {
	TradeNo       string                       `json:"trade_no"`
	Status        int                          `json:"status"`
	Sendout       int                          `json:"sendout"`
	TransactionID string                       `json:"transaction_id"`
	SuccessTime   int64                        `json:"success_time"`
	Response      LdxpOrderInfoDeliverResponse `json:"response"`
}

type LdxpOrderInfoResponse struct {
	LdxpBaseResponse
	Data LdxpOrderInfoData `json:"data"`
}

func (c *LDXPClient) FetchBuyerJUUID(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/shopApi/common/buyerBlackIframe", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	matches := ldxpJUUIDPattern.FindSubmatch(body)
	if len(matches) != 2 {
		return "", fmt.Errorf("juuid not found")
	}
	return string(matches[1]), nil
}

func (c *LDXPClient) GetGoodsPrice(ctx context.Context, goodsKey string, quantity int, channelID int) (LdxpGoodsPriceResponse, error) {
	var out LdxpGoodsPriceResponse
	if quantity <= 0 {
		quantity = 1
	}
	err := c.postJSON(ctx, "/shopApi/Shop/getGoodsPrice", map[string]interface{}{
		"goods_key":  strings.TrimSpace(goodsKey),
		"quantity":   quantity,
		"channel_id": channelID,
	}, &out)
	return out, err
}

func (c *LDXPClient) GetShopInfo(ctx context.Context) (LdxpShopInfoResponse, error) {
	var out LdxpShopInfoResponse
	err := c.postJSON(ctx, "/shopApi/Shop/info", map[string]interface{}{
		"token":        strings.TrimSpace(c.shopToken),
		"category_key": "",
	}, &out)
	return out, err
}

func (c *LDXPClient) ListCategories(ctx context.Context, goodsType string) (LdxpCategoryListResponse, error) {
	var out LdxpCategoryListResponse
	if goodsType == "" {
		goodsType = "card"
	}
	err := c.postJSON(ctx, "/shopApi/Shop/categoryList", map[string]interface{}{
		"token":        strings.TrimSpace(c.shopToken),
		"goods_type":   goodsType,
		"category_key": "",
	}, &out)
	return out, err
}

func (c *LDXPClient) ListGoods(ctx context.Context, categoryID int, goodsType string, current int, pageSize int, keywords string) (LdxpGoodsListResponse, error) {
	var out LdxpGoodsListResponse
	if goodsType == "" {
		goodsType = "card"
	}
	if current <= 0 {
		current = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	err := c.postJSON(ctx, "/shopApi/Shop/goodsList", map[string]interface{}{
		"token":       strings.TrimSpace(c.shopToken),
		"keywords":    strings.TrimSpace(keywords),
		"category_id": categoryID,
		"goods_type":  goodsType,
		"current":     current,
		"pageSize":    pageSize,
	}, &out)
	return out, err
}

func (c *LDXPClient) ListChannels(ctx context.Context) (LdxpChannelListResponse, error) {
	var out LdxpChannelListResponse
	err := c.postJSON(ctx, "/shopApi/Shop/getUserChannel", map[string]interface{}{
		"token": strings.TrimSpace(c.shopToken),
	}, &out)
	return out, err
}

func (c *LDXPClient) CreateOrder(ctx context.Context, req LdxpCreateOrderRequest) (LdxpCreateOrderResponse, error) {
	var out LdxpCreateOrderResponse
	if req.Quantity <= 0 {
		req.Quantity = 1
	}
	if req.Extend.JUUID == "" {
		juuid, err := c.FetchBuyerJUUID(ctx)
		if err != nil {
			return out, err
		}
		req.Extend.JUUID = juuid
	}
	if req.SelectCardsIDs == nil {
		req.SelectCardsIDs = []int{}
	}
	err := c.postJSON(ctx, "/shopApi/Pay/order", req, &out)
	return out, err
}

func (c *LDXPClient) QueryOrder(ctx context.Context, tradeNo string) (LdxpPayQueryResponse, error) {
	var out LdxpPayQueryResponse
	err := c.postJSON(ctx, "/shopApi/Pay/query", map[string]string{
		"trade_no": strings.TrimSpace(tradeNo),
	}, &out)
	return out, err
}

func (c *LDXPClient) GetOrderInfo(ctx context.Context, tradeNo string) (LdxpOrderInfoResponse, error) {
	var out LdxpOrderInfoResponse
	err := c.postJSON(ctx, "/shopApi/Order/info", map[string]interface{}{
		"trade_no": strings.TrimSpace(tradeNo),
		"dump":     1,
	}, &out)
	return out, err
}

func BuildLdxpProviderPayload(queryResp LdxpPayQueryResponse, orderInfo *LdxpOrderInfoResponse) string {
	if orderInfo == nil {
		return common.GetJsonString(queryResp)
	}
	return common.GetJsonString(map[string]interface{}{
		"query":      queryResp,
		"order_info": orderInfo,
	})
}

func (c *LDXPClient) postJSON(ctx context.Context, path string, payload interface{}, out interface{}) error {
	body, err := common.Marshal(payload)
	if err != nil {
		return err
	}
	target, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return common.DecodeJson(resp.Body, out)
}

func BuildSuggestedLdxpTopupProducts(goods []LdxpGoodsItem) []operation_setting.LdxpTopupProduct {
	if len(goods) == 0 {
		return []operation_setting.LdxpTopupProduct{}
	}

	type keyedProduct struct {
		product operation_setting.LdxpTopupProduct
	}

	productByAmount := make(map[int]operation_setting.LdxpTopupProduct)
	for _, good := range goods {
		amount, ok := detectLdxpTopupAmount(good)
		if !ok || amount <= 0 {
			continue
		}
		existing, exists := productByAmount[amount]
		if exists && strings.TrimSpace(existing.GoodsKey) != "" {
			continue
		}
		productByAmount[amount] = operation_setting.LdxpTopupProduct{
			Amount:    amount,
			GoodsKey:  strings.TrimSpace(good.GoodsKey),
			Label:     strings.TrimSpace(good.Name),
			Enabled:   true,
			SortOrder: amount,
		}
	}

	if len(productByAmount) == 0 {
		return []operation_setting.LdxpTopupProduct{}
	}

	amounts := make([]int, 0, len(productByAmount))
	for amount := range productByAmount {
		amounts = append(amounts, amount)
	}
	sort.Ints(amounts)

	products := make([]operation_setting.LdxpTopupProduct, 0, len(amounts))
	for _, amount := range amounts {
		products = append(products, productByAmount[amount])
	}
	return operation_setting.NormalizeLdxpTopupProducts(products)
}

func detectLdxpTopupAmount(good LdxpGoodsItem) (int, bool) {
	name := strings.TrimSpace(good.Name)
	if name == "" {
		return 0, false
	}
	matches := ldxpTopupAmountPattern.FindStringSubmatch(name)
	if len(matches) != 2 {
		return 0, false
	}
	amount, err := strconv.Atoi(matches[1])
	if err != nil || amount <= 0 {
		return 0, false
	}
	return amount, true
}
