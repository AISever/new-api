package service

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

var ldxpJUUIDPattern = regexp.MustCompile(`juuid\s*=\s*'([^']+)'`)
var ldxpTopupAmountPattern = regexp.MustCompile(`(?i)充值\s*(\d+(?:\.\d+)?)\s*r`)
var ldxpFormTagPattern = regexp.MustCompile(`(?is)<form\b[^>]*>`)
var ldxpInputTagPattern = regexp.MustCompile(`(?is)<input\b[^>]*>`)
var ldxpAttrDoubleQuotePattern = regexp.MustCompile(`(?is)([a-zA-Z_:][a-zA-Z0-9_:\-]*)\s*=\s*"([^"]*)"`)
var ldxpAttrSingleQuotePattern = regexp.MustCompile(`(?is)([a-zA-Z_:][a-zA-Z0-9_:\-]*)\s*=\s*'([^']*)'`)

const ldxpDesktopCheckoutUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"

type LdxpConfig struct {
	Enabled          bool
	BaseURL          string
	ShopToken        string
	DefaultChannelId int
}

type LdxpCheckoutData struct {
	QRCode     string `json:"qr_code,omitempty"`
	QRImageURL string `json:"qr_img_url,omitempty"`
}

type ldxpAutoSubmitForm struct {
	Action string
	Method string
	Fields url.Values
}

type ldxpTopupMeta struct {
	Amount float64 `json:"amount,omitempty"`
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

func FindLdxpTopupProductByAmount(amount float64) (*operation_setting.LdxpTopupProduct, bool) {
	normalizedAmount := operation_setting.NormalizeLdxpTopupAmount(amount)
	if normalizedAmount <= 0 {
		return nil, false
	}
	for _, product := range GetEnabledLdxpTopupProducts() {
		if operation_setting.LdxpTopupAmountKey(product.Amount) != operation_setting.LdxpTopupAmountKey(normalizedAmount) {
			continue
		}
		productCopy := product
		return &productCopy, true
	}
	return nil, false
}

func ResolveLdxpTopupSettlementAmount(product operation_setting.LdxpTopupProduct) float64 {
	return operation_setting.NormalizeLdxpTopupAmount(product.Amount)
}

func ResolveLdxpTopupGrantedAmount(product operation_setting.LdxpTopupProduct) float64 {
	return operation_setting.NormalizeLdxpTopupAmount(product.Amount)
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

func ResolveLdxpCheckoutData(providerPayload string) LdxpCheckoutData {
	providerPayload = strings.TrimSpace(providerPayload)
	if providerPayload == "" {
		return LdxpCheckoutData{}
	}

	var payload struct {
		Checkout LdxpCheckoutData `json:"checkout"`
	}
	if err := common.UnmarshalJsonStr(providerPayload, &payload); err == nil {
		if payload.Checkout.QRCode != "" || payload.Checkout.QRImageURL != "" {
			return payload.Checkout
		}
	}
	return LdxpCheckoutData{}
}

func ResolveLdxpTopupAmountFromProviderPayload(providerPayload string) float64 {
	providerPayload = strings.TrimSpace(providerPayload)
	if providerPayload == "" {
		return 0
	}

	var payload struct {
		TopupMeta ldxpTopupMeta `json:"topup_meta"`
	}
	if err := common.UnmarshalJsonStr(providerPayload, &payload); err != nil {
		return 0
	}
	return operation_setting.NormalizeLdxpTopupAmount(payload.TopupMeta.Amount)
}

func MergeLdxpCheckoutIntoProviderPayload(providerPayload string, checkout LdxpCheckoutData) string {
	providerPayload = strings.TrimSpace(providerPayload)
	checkout.QRCode = strings.TrimSpace(checkout.QRCode)
	checkout.QRImageURL = strings.TrimSpace(checkout.QRImageURL)

	if checkout.QRCode == "" && checkout.QRImageURL == "" {
		return providerPayload
	}

	if providerPayload == "" {
		return common.GetJsonString(map[string]interface{}{
			"checkout": checkout,
		})
	}

	var payload map[string]interface{}
	if err := common.UnmarshalJsonStr(providerPayload, &payload); err != nil || payload == nil {
		return providerPayload
	}

	payload["checkout"] = checkout
	return common.GetJsonString(payload)
}

func MergeLdxpTopupAmountIntoProviderPayload(providerPayload string, amount float64) string {
	providerPayload = strings.TrimSpace(providerPayload)
	normalizedAmount := operation_setting.NormalizeLdxpTopupAmount(amount)
	if normalizedAmount <= 0 {
		return providerPayload
	}

	if providerPayload == "" {
		return common.GetJsonString(map[string]interface{}{
			"topup_meta": ldxpTopupMeta{Amount: normalizedAmount},
		})
	}

	var payload map[string]interface{}
	if err := common.UnmarshalJsonStr(providerPayload, &payload); err != nil || payload == nil {
		return providerPayload
	}

	payload["topup_meta"] = ldxpTopupMeta{Amount: normalizedAmount}
	return common.GetJsonString(payload)
}

func PreserveLdxpCheckoutInProviderPayload(providerPayload string, existingProviderPayload string) string {
	return MergeLdxpCheckoutIntoProviderPayload(
		providerPayload,
		ResolveLdxpCheckoutData(existingProviderPayload),
	)
}

func PreserveLdxpTopupAmountInProviderPayload(providerPayload string, existingProviderPayload string) string {
	return MergeLdxpTopupAmountIntoProviderPayload(
		providerPayload,
		ResolveLdxpTopupAmountFromProviderPayload(existingProviderPayload),
	)
}

func FetchLdxpCheckoutData(ctx context.Context, paymentURL string) (LdxpCheckoutData, error) {
	return fetchLdxpCheckoutData(ctx, paymentURL, nil, ldxpDesktopCheckoutUserAgent)
}

func fetchLdxpCheckoutData(ctx context.Context, paymentURL string, client *http.Client, userAgent string) (LdxpCheckoutData, error) {
	paymentURL = strings.TrimSpace(paymentURL)
	if paymentURL == "" {
		return LdxpCheckoutData{}, fmt.Errorf("payment url is empty")
	}
	if client == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return LdxpCheckoutData{}, err
		}
		client = &http.Client{
			Timeout: 15 * time.Second,
			Jar:     jar,
		}
	}

	formHTML, err := fetchLdxpCheckoutDocument(ctx, client, userAgent, http.MethodGet, paymentURL, "")
	if err != nil {
		return LdxpCheckoutData{}, err
	}
	form, ok := extractLdxpAutoSubmitForm(formHTML)
	if !ok || form.Action == "" {
		return LdxpCheckoutData{}, fmt.Errorf("unable to parse ldxp checkout form")
	}

	checkoutHTML, err := fetchLdxpCheckoutDocument(
		ctx,
		client,
		userAgent,
		form.Method,
		form.Action,
		form.Fields.Encode(),
	)
	if err != nil {
		return LdxpCheckoutData{}, err
	}

	data := extractAlipayCheckoutData(checkoutHTML)
	if data.QRCode == "" && data.QRImageURL == "" {
		return LdxpCheckoutData{}, fmt.Errorf("alipay checkout qr data missing")
	}
	return data, nil
}

func fetchLdxpCheckoutDocument(ctx context.Context, client *http.Client, userAgent string, method string, target string, body string) (string, error) {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected checkout status: %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func extractLdxpAutoSubmitForm(doc string) (ldxpAutoSubmitForm, bool) {
	tag := ldxpFormTagPattern.FindString(doc)
	if tag == "" {
		return ldxpAutoSubmitForm{}, false
	}
	attrs := extractLdxpTagAttributes(tag)
	action := strings.TrimSpace(attrs["action"])
	if action == "" {
		return ldxpAutoSubmitForm{}, false
	}
	method := strings.ToUpper(strings.TrimSpace(attrs["method"]))
	if method == "" {
		method = http.MethodPost
	}
	fields := url.Values{}
	for _, inputTag := range ldxpInputTagPattern.FindAllString(doc, -1) {
		inputAttrs := extractLdxpTagAttributes(inputTag)
		if !strings.EqualFold(strings.TrimSpace(inputAttrs["type"]), "hidden") {
			continue
		}
		name := strings.TrimSpace(inputAttrs["name"])
		if name == "" {
			continue
		}
		fields.Set(name, inputAttrs["value"])
	}
	return ldxpAutoSubmitForm{
		Action: action,
		Method: method,
		Fields: fields,
	}, true
}

func extractAlipayCheckoutData(doc string) LdxpCheckoutData {
	fields := make(map[string]string)
	for _, inputTag := range ldxpInputTagPattern.FindAllString(doc, -1) {
		inputAttrs := extractLdxpTagAttributes(inputTag)
		name := strings.TrimSpace(inputAttrs["name"])
		if name == "" {
			continue
		}
		fields[name] = inputAttrs["value"]
	}
	return LdxpCheckoutData{
		QRCode:     strings.TrimSpace(fields["qrCode"]),
		QRImageURL: strings.TrimSpace(fields["qrImgUrl"]),
	}
}

func extractLdxpTagAttributes(tag string) map[string]string {
	attrs := make(map[string]string)
	for _, match := range ldxpAttrDoubleQuotePattern.FindAllStringSubmatch(tag, -1) {
		if len(match) < 3 {
			continue
		}
		attrs[strings.ToLower(strings.TrimSpace(match[1]))] = html.UnescapeString(match[2])
	}
	for _, match := range ldxpAttrSingleQuotePattern.FindAllStringSubmatch(tag, -1) {
		if len(match) < 3 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(match[1]))
		if _, exists := attrs[key]; exists {
			continue
		}
		attrs[key] = html.UnescapeString(match[2])
	}
	return attrs
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

	productByAmount := make(map[string]operation_setting.LdxpTopupProduct)
	for _, good := range goods {
		amount, ok := detectLdxpTopupAmount(good)
		if !ok || amount <= 0 {
			continue
		}
		amountKey := operation_setting.LdxpTopupAmountKey(amount)
		if amountKey == "" {
			continue
		}
		existing, exists := productByAmount[amountKey]
		if exists && strings.TrimSpace(existing.GoodsKey) != "" {
			continue
		}
		productByAmount[amountKey] = operation_setting.LdxpTopupProduct{
			Amount:    amount,
			GoodsKey:  strings.TrimSpace(good.GoodsKey),
			Label:     strings.TrimSpace(good.Name),
			Enabled:   true,
			SortOrder: int(math.Round(amount * 100)),
		}
	}

	if len(productByAmount) == 0 {
		return []operation_setting.LdxpTopupProduct{}
	}

	amounts := make([]float64, 0, len(productByAmount))
	for _, product := range productByAmount {
		amounts = append(amounts, product.Amount)
	}
	sort.Float64s(amounts)

	products := make([]operation_setting.LdxpTopupProduct, 0, len(amounts))
	for _, amount := range amounts {
		products = append(products, productByAmount[operation_setting.LdxpTopupAmountKey(amount)])
	}
	return operation_setting.NormalizeLdxpTopupProducts(products)
}

func detectLdxpTopupAmount(good LdxpGoodsItem) (float64, bool) {
	name := strings.TrimSpace(good.Name)
	if name == "" {
		return 0, false
	}
	matches := ldxpTopupAmountPattern.FindStringSubmatch(name)
	if len(matches) != 2 {
		return 0, false
	}
	amount, err := strconv.ParseFloat(matches[1], 64)
	if err != nil || amount <= 0 {
		return 0, false
	}
	return operation_setting.NormalizeLdxpTopupAmount(amount), true
}
