package router

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type billingChargeTestResponse struct {
	Object        string  `json:"object"`
	Model         string  `json:"model"`
	Quantity      int     `json:"quantity"`
	Quota         int     `json:"quota"`
	BillingSource string  `json:"billing_source"`
	TokenId       int     `json:"token_id"`
	TokenName     string  `json:"token_name"`
	Group         string  `json:"group"`
	ModelPrice    float64 `json:"model_price"`
	GroupRatio    float64 `json:"group_ratio"`
}

func setupBillingChargeTestDB(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	originalSQLitePath := common.SQLitePath
	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL
	originalRedisEnabled := common.RedisEnabled
	originalBatchUpdateEnabled := common.BatchUpdateEnabled
	originalLogConsumeEnabled := common.LogConsumeEnabled
	originalIsMasterNode := common.IsMasterNode
	t.Cleanup(func() {
		common.SQLitePath = originalSQLitePath
		common.UsingSQLite = originalUsingSQLite
		common.UsingMySQL = originalUsingMySQL
		common.UsingPostgreSQL = originalUsingPostgreSQL
		common.RedisEnabled = originalRedisEnabled
		common.BatchUpdateEnabled = originalBatchUpdateEnabled
		common.LogConsumeEnabled = originalLogConsumeEnabled
		common.IsMasterNode = originalIsMasterNode
	})

	common.SQLitePath = filepath.Join(t.TempDir(), "billing-charge.db") + "?_busy_timeout=30000"
	t.Setenv("SQL_DSN", "local")
	common.UsingSQLite = false
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.LogConsumeEnabled = true
	common.IsMasterNode = true

	require.NoError(t, model.InitDB())
	require.NoError(t, model.InitLogDB())

	t.Cleanup(func() {
		if model.DB != nil {
			if sqlDB, err := model.DB.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
	})
}

func seedBillingChargeUserAndToken(t *testing.T, userQuota int, tokenQuota int) *model.Token {
	t.Helper()

	user := &model.User{
		Id:       1001,
		Username: "charge_user",
		Password: "password",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		Quota:    userQuota,
	}
	require.NoError(t, model.DB.Create(user).Error)

	token := &model.Token{
		Id:             2001,
		UserId:         user.Id,
		Key:            "chargetesttoken",
		Name:           "charge-token",
		Status:         common.TokenStatusEnabled,
		ExpiredTime:    -1,
		RemainQuota:    tokenQuota,
		UnlimitedQuota: false,
		Group:          "default",
	}
	require.NoError(t, model.DB.Create(token).Error)
	return token
}

func performBillingChargeRequest(t *testing.T, body map[string]any, token string) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := common.Marshal(body)
	require.NoError(t, err)

	engine := gin.New()
	SetRelayRouter(engine)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/billing/charge", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer sk-"+token)
	engine.ServeHTTP(recorder, request)
	return recorder
}

func TestBillingChargeByTokenAndModelConsumesWalletAndTokenQuota(t *testing.T) {
	setupBillingChargeTestDB(t)
	token := seedBillingChargeUserAndToken(t, 1_000_000, 1_000_000)

	recorder := performBillingChargeRequest(t, map[string]any{
		"model": "dall-e-3",
	}, token.Key)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var response billingChargeTestResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))

	expectedQuota := int(0.04 * common.QuotaPerUnit)
	require.Equal(t, "billing_charge", response.Object)
	require.Equal(t, "dall-e-3", response.Model)
	require.Equal(t, 1, response.Quantity)
	require.Equal(t, expectedQuota, response.Quota)
	require.Equal(t, "wallet", response.BillingSource)
	require.Equal(t, token.Id, response.TokenId)
	require.Equal(t, token.Name, response.TokenName)
	require.Equal(t, "default", response.Group)
	require.Equal(t, 0.04, response.ModelPrice)
	require.Equal(t, 1.0, response.GroupRatio)

	var updatedUser model.User
	require.NoError(t, model.DB.First(&updatedUser, token.UserId).Error)
	require.Equal(t, 1_000_000-expectedQuota, updatedUser.Quota)
	require.Equal(t, expectedQuota, updatedUser.UsedQuota)
	require.Equal(t, 1, updatedUser.RequestCount)

	var updatedToken model.Token
	require.NoError(t, model.DB.First(&updatedToken, token.Id).Error)
	require.Equal(t, 1_000_000-expectedQuota, updatedToken.RemainQuota)
	require.Equal(t, expectedQuota, updatedToken.UsedQuota)

	var log model.Log
	require.NoError(t, model.LOG_DB.Where("type = ?", model.LogTypeConsume).First(&log).Error)
	require.Equal(t, token.UserId, log.UserId)
	require.Equal(t, token.Id, log.TokenId)
	require.Equal(t, token.Name, log.TokenName)
	require.Equal(t, "dall-e-3", log.ModelName)
	require.Equal(t, expectedQuota, log.Quota)
	require.Equal(t, "default", log.Group)
	require.Equal(t, 0, log.ChannelId)

	var other map[string]interface{}
	require.NoError(t, common.UnmarshalJsonStr(log.Other, &other))
	require.Equal(t, true, other["manual_charge"])
	require.Equal(t, "wallet", other["billing_source"])
	require.Equal(t, "/v1/billing/charge", other["request_path"])
	require.Equal(t, float64(1), other["quantity"])
}

func TestBillingChargeByTokenAndModelAppliesQuantity(t *testing.T) {
	setupBillingChargeTestDB(t)
	token := seedBillingChargeUserAndToken(t, 1_000_000, 1_000_000)

	recorder := performBillingChargeRequest(t, map[string]any{
		"model":    "dall-e-3",
		"quantity": 3,
	}, token.Key)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var response billingChargeTestResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))

	expectedQuota := int(0.04 * common.QuotaPerUnit * 3)
	require.Equal(t, "billing_charge", response.Object)
	require.Equal(t, "dall-e-3", response.Model)
	require.Equal(t, 3, response.Quantity)
	require.Equal(t, expectedQuota, response.Quota)
	require.Equal(t, "wallet", response.BillingSource)
	require.Equal(t, token.Id, response.TokenId)
	require.Equal(t, token.Name, response.TokenName)

	var updatedUser model.User
	require.NoError(t, model.DB.First(&updatedUser, token.UserId).Error)
	require.Equal(t, 1_000_000-expectedQuota, updatedUser.Quota)
	require.Equal(t, expectedQuota, updatedUser.UsedQuota)
	require.Equal(t, 1, updatedUser.RequestCount)

	var updatedToken model.Token
	require.NoError(t, model.DB.First(&updatedToken, token.Id).Error)
	require.Equal(t, 1_000_000-expectedQuota, updatedToken.RemainQuota)
	require.Equal(t, expectedQuota, updatedToken.UsedQuota)

	var log model.Log
	require.NoError(t, model.LOG_DB.Where("type = ?", model.LogTypeConsume).First(&log).Error)
	require.Equal(t, expectedQuota, log.Quota)
	require.Contains(t, log.Content, "quantity 3")

	var other map[string]interface{}
	require.NoError(t, common.UnmarshalJsonStr(log.Other, &other))
	require.Equal(t, float64(3), other["quantity"])
}

func TestBillingChargeByTokenAndModelAcceptsNAsQuantityAlias(t *testing.T) {
	setupBillingChargeTestDB(t)
	token := seedBillingChargeUserAndToken(t, 1_000_000, 1_000_000)

	recorder := performBillingChargeRequest(t, map[string]any{
		"model": "dall-e-3",
		"n":     2,
	}, token.Key)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var response billingChargeTestResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))

	expectedQuota := int(0.04 * common.QuotaPerUnit * 2)
	require.Equal(t, 2, response.Quantity)
	require.Equal(t, expectedQuota, response.Quota)

	var updatedUser model.User
	require.NoError(t, model.DB.First(&updatedUser, token.UserId).Error)
	require.Equal(t, 1_000_000-expectedQuota, updatedUser.Quota)

	var log model.Log
	require.NoError(t, model.LOG_DB.Where("type = ?", model.LogTypeConsume).First(&log).Error)
	require.Equal(t, expectedQuota, log.Quota)

	var other map[string]interface{}
	require.NoError(t, common.UnmarshalJsonStr(log.Other, &other))
	require.Equal(t, float64(2), other["quantity"])
}

func TestBillingChargeByTokenAndModelRejectsMissingModel(t *testing.T) {
	setupBillingChargeTestDB(t)
	token := seedBillingChargeUserAndToken(t, 1_000_000, 1_000_000)

	recorder := performBillingChargeRequest(t, map[string]any{
		"model": strings.Repeat(" ", 3),
	}, token.Key)

	require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), "model is required")
}

func TestBillingChargeByTokenAndModelRejectsInsufficientTokenQuota(t *testing.T) {
	setupBillingChargeTestDB(t)
	token := seedBillingChargeUserAndToken(t, 1_000_000, 10)

	recorder := performBillingChargeRequest(t, map[string]any{
		"model": "dall-e-3",
	}, token.Key)

	require.Equal(t, http.StatusForbidden, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), "token quota is not enough")

	var updatedUser model.User
	require.NoError(t, model.DB.First(&updatedUser, token.UserId).Error)
	require.Equal(t, 1_000_000, updatedUser.Quota)

	var updatedToken model.Token
	require.NoError(t, model.DB.First(&updatedToken, token.Id).Error)
	require.Equal(t, 10, updatedToken.RemainQuota)
	require.Equal(t, 0, updatedToken.UsedQuota)

	var count int64
	require.NoError(t, model.LOG_DB.Model(&model.Log{}).Where("type = ?", model.LogTypeConsume).Count(&count).Error)
	require.Equal(t, int64(0), count)
}

func TestBillingChargeByTokenAndModelRejectsModelOutsideTokenLimits(t *testing.T) {
	setupBillingChargeTestDB(t)
	token := seedBillingChargeUserAndToken(t, 1_000_000, 1_000_000)
	limits := "gpt-4o"
	require.NoError(t, model.DB.Model(&model.Token{}).Where("id = ?", token.Id).Updates(map[string]any{
		"model_limits_enabled": true,
		"model_limits":         limits,
	}).Error)

	recorder := performBillingChargeRequest(t, map[string]any{
		"model": "dall-e-3",
	}, token.Key)

	require.Equal(t, http.StatusForbidden, recorder.Code, fmt.Sprintf("body=%s", recorder.Body.String()))
	require.Contains(t, recorder.Body.String(), "token is not allowed to charge model")
}
