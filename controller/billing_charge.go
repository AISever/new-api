package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

type billingChargeRequest struct {
	Model string `json:"model"`
}

type billingChargeResponse struct {
	Object        string  `json:"object"`
	Model         string  `json:"model"`
	Quota         int     `json:"quota"`
	BillingSource string  `json:"billing_source,omitempty"`
	TokenId       int     `json:"token_id"`
	TokenName     string  `json:"token_name,omitempty"`
	Group         string  `json:"group"`
	UsePrice      bool    `json:"use_price"`
	ModelPrice    float64 `json:"model_price"`
	ModelRatio    float64 `json:"model_ratio,omitempty"`
	GroupRatio    float64 `json:"group_ratio"`
	FreeModel     bool    `json:"free_model,omitempty"`
}

func ChargeBillingByTokenAndModel(c *gin.Context) {
	var req billingChargeRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		writeBillingChargeError(c, types.NewErrorWithStatusCode(err, types.ErrorCodeBadRequestBody, http.StatusBadRequest))
		return
	}

	modelName := strings.TrimSpace(req.Model)
	if modelName == "" {
		writeBillingChargeError(c, types.NewErrorWithStatusCode(errors.New("model is required"), types.ErrorCodeInvalidRequest, http.StatusBadRequest))
		return
	}

	if !ensureBillingChargeTokenModelAllowed(c, modelName) {
		return
	}

	relayInfo := buildBillingChargeRelayInfo(c, modelName)
	priceData, err := helper.ModelPriceHelperPerCall(c, relayInfo)
	if err != nil {
		writeBillingChargeError(c, types.NewErrorWithStatusCode(err, types.ErrorCodeModelPriceError, http.StatusBadRequest))
		return
	}
	relayInfo.PriceData = priceData

	if priceData.Quota > 0 {
		if apiErr := service.PreConsumeBilling(c, priceData.Quota, relayInfo); apiErr != nil {
			writeBillingChargeError(c, apiErr)
			return
		}
		if err := service.SettleBilling(c, relayInfo, priceData.Quota); err != nil {
			writeBillingChargeError(c, types.NewErrorWithStatusCode(err, types.ErrorCodeUpdateDataError, http.StatusInternalServerError))
			return
		}
	}

	recordBillingChargeConsumption(c, relayInfo, priceData)

	c.JSON(http.StatusOK, billingChargeResponse{
		Object:        "billing_charge",
		Model:         modelName,
		Quota:         priceData.Quota,
		BillingSource: relayInfo.BillingSource,
		TokenId:       relayInfo.TokenId,
		TokenName:     c.GetString("token_name"),
		Group:         relayInfo.UsingGroup,
		UsePrice:      priceData.UsePrice,
		ModelPrice:    priceData.ModelPrice,
		ModelRatio:    priceData.ModelRatio,
		GroupRatio:    priceData.GroupRatioInfo.GroupRatio,
		FreeModel:     priceData.FreeModel,
	})
}

func ensureBillingChargeTokenModelAllowed(c *gin.Context, modelName string) bool {
	if !common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled) {
		return true
	}

	value, ok := common.GetContextKey(c, constant.ContextKeyTokenModelLimit)
	if !ok {
		writeBillingChargeError(c, types.NewErrorWithStatusCode(
			fmt.Errorf("token is not allowed to charge model %s", modelName),
			types.ErrorCodeAccessDenied,
			http.StatusForbidden,
		))
		return false
	}

	limits, ok := value.(map[string]bool)
	if !ok {
		limits = map[string]bool{}
	}
	matchName := ratio_setting.FormatMatchingModelName(modelName)
	if !limits[matchName] {
		writeBillingChargeError(c, types.NewErrorWithStatusCode(
			fmt.Errorf("token is not allowed to charge model %s", modelName),
			types.ErrorCodeAccessDenied,
			http.StatusForbidden,
		))
		return false
	}

	return true
}

func buildBillingChargeRelayInfo(c *gin.Context, modelName string) *relaycommon.RelayInfo {
	startTime := time.Now()
	requestId := c.GetString(common.RequestIdKey)
	if requestId == "" {
		requestId = common.GetTimeString() + common.GetRandomString(8)
		c.Set(common.RequestIdKey, requestId)
	}

	relayInfo := &relaycommon.RelayInfo{
		RequestId:         requestId,
		UserId:            common.GetContextKeyInt(c, constant.ContextKeyUserId),
		UsingGroup:        common.GetContextKeyString(c, constant.ContextKeyUsingGroup),
		UserGroup:         common.GetContextKeyString(c, constant.ContextKeyUserGroup),
		UserQuota:         common.GetContextKeyInt(c, constant.ContextKeyUserQuota),
		UserEmail:         common.GetContextKeyString(c, constant.ContextKeyUserEmail),
		OriginModelName:   modelName,
		TokenId:           common.GetContextKeyInt(c, constant.ContextKeyTokenId),
		TokenKey:          common.GetContextKeyString(c, constant.ContextKeyTokenKey),
		TokenUnlimited:    common.GetContextKeyBool(c, constant.ContextKeyTokenUnlimited),
		TokenGroup:        common.GetContextKeyString(c, constant.ContextKeyTokenGroup),
		StartTime:         startTime,
		FirstResponseTime: startTime,
		RequestURLPath:    c.Request.URL.String(),
	}
	if relayInfo.UsingGroup == "" {
		relayInfo.UsingGroup = c.GetString("group")
	}
	if relayInfo.UserGroup == "" {
		relayInfo.UserGroup = relayInfo.UsingGroup
	}

	userSetting, ok := common.GetContextKeyType[dto.UserSetting](c, constant.ContextKeyUserSetting)
	if ok {
		relayInfo.UserSetting = userSetting
	}

	return relayInfo
}

func recordBillingChargeConsumption(c *gin.Context, relayInfo *relaycommon.RelayInfo, priceData types.PriceData) {
	other := map[string]interface{}{
		"manual_charge": true,
		"request_path":  c.Request.URL.Path,
		"model_price":   priceData.ModelPrice,
		"group_ratio":   priceData.GroupRatioInfo.GroupRatio,
	}
	if priceData.ModelRatio > 0 {
		other["model_ratio"] = priceData.ModelRatio
	}
	if priceData.GroupRatioInfo.HasSpecialRatio {
		other["user_group_ratio"] = priceData.GroupRatioInfo.GroupSpecialRatio
	}
	if relayInfo.BillingSource != "" {
		other["billing_source"] = relayInfo.BillingSource
	}
	if relayInfo.UserSetting.BillingPreference != "" {
		other["billing_preference"] = relayInfo.UserSetting.BillingPreference
	}
	if relayInfo.SubscriptionId != 0 {
		other["subscription_id"] = relayInfo.SubscriptionId
	}

	content := fmt.Sprintf("charge by model name, model %s", relayInfo.OriginModelName)
	if priceData.UsePrice {
		content = fmt.Sprintf("%s, model price %.4f, group ratio %.4f", content, priceData.ModelPrice, priceData.GroupRatioInfo.GroupRatio)
	} else {
		content = fmt.Sprintf("%s, model ratio %.4f, group ratio %.4f", content, priceData.ModelRatio, priceData.GroupRatioInfo.GroupRatio)
	}

	model.RecordConsumeLog(c, relayInfo.UserId, model.RecordConsumeLogParams{
		ChannelId:      0,
		ModelName:      relayInfo.OriginModelName,
		TokenName:      c.GetString("token_name"),
		Quota:          priceData.Quota,
		Content:        content,
		TokenId:        relayInfo.TokenId,
		UseTimeSeconds: int(time.Since(relayInfo.StartTime).Seconds()),
		Group:          relayInfo.UsingGroup,
		Other:          other,
	})
	model.UpdateUserUsedQuotaAndRequestCount(relayInfo.UserId, priceData.Quota)
}

func writeBillingChargeError(c *gin.Context, apiErr *types.NewAPIError) {
	if apiErr == nil {
		apiErr = types.NewErrorWithStatusCode(errors.New("unknown billing charge error"), types.ErrorCodeInvalidRequest, http.StatusInternalServerError)
	}
	statusCode := apiErr.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}
	c.JSON(statusCode, gin.H{
		"error": apiErr.ToOpenAIError(),
	})
}
