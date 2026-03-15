package externalshop

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

var juuidPattern = regexp.MustCompile(`juuid\s*=\s*'([^']+)'`)

type LDXPClient struct {
	baseURL    string
	shopToken  string
	httpClient *http.Client
}

func NewLDXPClient(baseURL string, shopToken string, httpClient *http.Client) *LDXPClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &LDXPClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		shopToken:  strings.TrimSpace(shopToken),
		httpClient: httpClient,
	}
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
	matches := juuidPattern.FindSubmatch(body)
	if len(matches) != 2 {
		return "", fmt.Errorf("juuid not found")
	}
	return string(matches[1]), nil
}

func (c *LDXPClient) GetShopInfo(ctx context.Context) (ShopInfoResponse, error) {
	var out ShopInfoResponse
	err := c.postJSON(ctx, "/shopApi/Shop/info", map[string]interface{}{
		"token":        c.shopToken,
		"category_key": "",
	}, &out)
	return out, err
}

func (c *LDXPClient) ListCategories(ctx context.Context, goodsType string) (CategoryListResponse, error) {
	var out CategoryListResponse
	if goodsType == "" {
		goodsType = "card"
	}
	err := c.postJSON(ctx, "/shopApi/Shop/categoryList", map[string]interface{}{
		"token":        c.shopToken,
		"goods_type":   goodsType,
		"category_key": "",
	}, &out)
	return out, err
}

func (c *LDXPClient) ListGoods(ctx context.Context, categoryID int, goodsType string, current int, pageSize int, keywords string) (GoodsListResponse, error) {
	var out GoodsListResponse
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
		"token":       c.shopToken,
		"keywords":    keywords,
		"category_id": categoryID,
		"goods_type":  goodsType,
		"current":     current,
		"pageSize":    pageSize,
	}, &out)
	return out, err
}

func (c *LDXPClient) ListChannels(ctx context.Context) (ChannelListResponse, error) {
	var out ChannelListResponse
	err := c.postJSON(ctx, "/shopApi/Shop/getUserChannel", map[string]interface{}{
		"token": c.shopToken,
	}, &out)
	return out, err
}

func (c *LDXPClient) GetGoodsPrice(ctx context.Context, goodsKey string, quantity int, couponCode string, channelID int) (GoodsPriceResponse, error) {
	var out GoodsPriceResponse
	if quantity <= 0 {
		quantity = 1
	}
	err := c.postJSON(ctx, "/shopApi/Shop/getGoodsPrice", map[string]interface{}{
		"goods_key":   goodsKey,
		"quantity":    quantity,
		"coupon_code": couponCode,
		"channel_id":  channelID,
	}, &out)
	return out, err
}

func (c *LDXPClient) QueryOrder(ctx context.Context, tradeNo string) (PayQueryResponse, error) {
	var out PayQueryResponse
	err := c.postJSON(ctx, "/shopApi/Pay/query", map[string]string{
		"trade_no": tradeNo,
	}, &out)
	return out, err
}

func (c *LDXPClient) GetOrderInfo(ctx context.Context, tradeNo string) (OrderInfoResponse, error) {
	var out OrderInfoResponse
	err := c.postJSON(ctx, "/shopApi/Order/info", map[string]interface{}{
		"trade_no": tradeNo,
		"dump":     1,
	}, &out)
	return out, err
}

func (c *LDXPClient) CreateOrder(ctx context.Context, req CreateOrderRequest) (CreateOrderResponse, error) {
	var out CreateOrderResponse
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
