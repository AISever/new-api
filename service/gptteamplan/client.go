package gptteamplan

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

var (
	landingSeatsElementPattern = regexp.MustCompile(`(?is)<span\b[^>]*\bid=(?:"spotsCount"|'spotsCount')[^>]*>.*?</span>`)
	landingSeatsTargetPattern  = regexp.MustCompile(`(?i)\bdata-target=(?:"(-?\d+)"|'(-?\d+)')`)
	landingSeatsTextPattern    = regexp.MustCompile(`(?is)>([^<]*)<`)
)

func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		httpClient: httpClient,
	}
}

func (c *Client) Redeem(ctx context.Context, email string, code string, teamID *int) (*RedeemResult, error) {
	payload := redeemRequestPayload{
		Email:  strings.TrimSpace(email),
		Code:   strings.TrimSpace(code),
		TeamID: teamID,
	}
	responseBody, err := c.postJSON(ctx, "/redeem/confirm", payload)
	if err != nil {
		return nil, err
	}
	return parseRedeemResult(responseBody)
}

func (c *Client) CheckWarranty(ctx context.Context, code string) (*WarrantyResult, error) {
	responseBody, err := c.postJSON(ctx, "/warranty/check", map[string]string{
		"code": strings.TrimSpace(code),
	})
	if err != nil {
		return nil, err
	}
	return parseWarrantyResult(responseBody)
}

func (c *Client) GetRemainingSeats(ctx context.Context) (*int, error) {
	if c == nil || c.httpClient == nil {
		return nil, fmt.Errorf("上游服务未初始化")
	}
	if c.baseURL == "" {
		return nil, fmt.Errorf("上游地址未配置")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/", nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求上游失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("上游服务暂不可用")
	}
	return parseRemainingSeatsFromReader(resp.Body)
}

func (c *Client) postJSON(ctx context.Context, path string, payload interface{}) (map[string]interface{}, error) {
	if c == nil || c.httpClient == nil {
		return nil, fmt.Errorf("上游服务未初始化")
	}
	if c.baseURL == "" {
		return nil, fmt.Errorf("上游地址未配置")
	}

	body, err := common.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求上游失败: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取上游响应失败: %w", err)
	}

	var result map[string]interface{}
	if err := common.Unmarshal(rawBody, &result); err != nil {
		return nil, fmt.Errorf("上游返回格式错误")
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%s", extractMessage(result, "上游服务暂不可用"))
	}
	return result, nil
}

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	if c == nil || c.httpClient == nil {
		return nil, fmt.Errorf("上游服务未初始化")
	}
	if c.baseURL == "" {
		return nil, fmt.Errorf("上游地址未配置")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求上游失败: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取上游响应失败: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("上游服务暂不可用")
	}
	return rawBody, nil
}

func parseRedeemResult(payload map[string]interface{}) (*RedeemResult, error) {
	if payload == nil {
		return nil, fmt.Errorf("兑换响应为空")
	}
	if !extractSuccess(payload) {
		return nil, fmt.Errorf("%s", extractMessage(payload, "兑换失败"))
	}

	teamInfo, _ := payload["team_info"].(map[string]interface{})
	if teamInfo == nil {
		return nil, fmt.Errorf("%s", extractMessage(payload, "兑换结果缺少团队信息"))
	}

	return &RedeemResult{
		Message:           extractMessage(payload, ""),
		TeamName:          stringValue(teamInfo["team_name"]),
		SubscriptionPlan:  stringValue(teamInfo["subscription_plan"]),
		RemainingSeats:    firstIntValue(teamInfo, "remaining_seats", "remaining_seat_count", "available_seats", "left_seats", "empty_seats"),
		TotalSeats:        firstIntValue(teamInfo, "total_seats", "total_seat_count", "seat_count", "seats"),
		UsedSeats:         firstIntValue(teamInfo, "used_seats", "used_seat_count", "occupied_seats"),
		ExpiresAt:         stringValue(teamInfo["expires_at"]),
		TeamExpiresAt:     stringValue(teamInfo["team_expires_at"]),
		HasWarranty:       boolValue(teamInfo["has_warranty"]),
		WarrantyValid:     boolValue(teamInfo["warranty_valid"]),
		WarrantyExpiresAt: stringValue(teamInfo["warranty_expires_at"]),
	}, nil
}

func parseWarrantyResult(payload map[string]interface{}) (*WarrantyResult, error) {
	if payload == nil {
		return nil, fmt.Errorf("质保响应为空")
	}
	if success, hasSuccess := payload["success"].(bool); hasSuccess && !success {
		return nil, fmt.Errorf("%s", extractMessage(payload, "质保查询失败"))
	}

	result := &WarrantyResult{
		Message:           extractMessage(payload, ""),
		HasWarranty:       boolValue(payload["has_warranty"]),
		WarrantyValid:     boolValue(payload["warranty_valid"]),
		WarrantyExpiresAt: stringValue(payload["warranty_expires_at"]),
		CanReuse:          boolValue(payload["can_reuse"]),
		OriginalCode:      stringValue(payload["original_code"]),
		Records:           make([]WarrantyRecord, 0),
	}

	if rawRecords, ok := payload["records"].([]interface{}); ok {
		result.Records = make([]WarrantyRecord, 0, len(rawRecords))
		for _, rawRecord := range rawRecords {
			record, ok := rawRecord.(map[string]interface{})
			if !ok {
				continue
			}
			result.Records = append(result.Records, WarrantyRecord{
				Code:              stringValue(record["code"]),
				Status:            stringValue(record["status"]),
				UsedAt:            stringValue(record["used_at"]),
				Email:             stringValue(record["email"]),
				TeamName:          stringValue(record["team_name"]),
				TeamStatus:        stringValue(record["team_status"]),
				HasWarranty:       boolValue(record["has_warranty"]),
				WarrantyValid:     boolValue(record["warranty_valid"]),
				WarrantyExpiresAt: stringValue(record["warranty_expires_at"]),
				UserExpiresAt:     stringValue(record["user_expires_at"]),
				TeamExpiresAt:     stringValue(record["team_expires_at"]),
			})
		}
	}

	return result, nil
}

func extractSuccess(payload map[string]interface{}) bool {
	if success, ok := payload["success"].(bool); ok {
		return success
	}
	if detail := stringValue(payload["detail"]); detail != "" {
		return false
	}
	if errMessage := stringValue(payload["error"]); errMessage != "" {
		return false
	}
	return true
}

func extractMessage(payload map[string]interface{}, fallback string) string {
	for _, key := range []string{"detail", "message", "msg", "error"} {
		if value := stringValue(payload[key]); value != "" {
			return value
		}
	}
	return fallback
}

func stringValue(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func boolValue(value interface{}) bool {
	typed, ok := value.(bool)
	if ok {
		return typed
	}
	return false
}

func firstIntValue(payload map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			return intValue(value)
		}
	}
	return 0
}

func intValue(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func parseRemainingSeatsFromLandingPage(rawBody []byte) (*int, error) {
	element := landingSeatsElementPattern.Find(rawBody)
	if len(element) == 0 {
		return nil, fmt.Errorf("上游页面缺少车位信息")
	}
	return parseRemainingSeatsFromElement(element)
}

func parseRemainingSeatsFromReader(reader io.Reader) (*int, error) {
	const (
		readBufferSize    = 2048
		maxWindowSize     = 64 * 1024
		trimmedWindowSize = 16 * 1024
	)

	buffer := make([]byte, readBufferSize)
	window := make([]byte, 0, maxWindowSize)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			window = append(window, buffer[:n]...)
			if element := landingSeatsElementPattern.Find(window); len(element) > 0 {
				return parseRemainingSeatsFromElement(element)
			}
			if len(window) > maxWindowSize {
				window = append([]byte(nil), window[len(window)-trimmedWindowSize:]...)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("读取上游响应失败: %w", err)
		}
	}
	return parseRemainingSeatsFromLandingPage(window)
}

func parseRemainingSeatsFromElement(element []byte) (*int, error) {
	if matches := landingSeatsTargetPattern.FindSubmatch(element); len(matches) >= 3 {
		for _, raw := range matches[1:] {
			if len(raw) == 0 {
				continue
			}
			seats, err := strconv.Atoi(string(raw))
			if err == nil {
				return &seats, nil
			}
		}
	}

	if matches := landingSeatsTextPattern.FindSubmatch(element); len(matches) >= 2 {
		seats, err := strconv.Atoi(strings.TrimSpace(string(matches[1])))
		if err == nil {
			return &seats, nil
		}
	}

	return nil, fmt.Errorf("上游页面车位信息格式错误")
}
