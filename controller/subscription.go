package controller

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---- Shared types ----

type SubscriptionPlanDTO struct {
	Plan SubscriptionPlanPayload `json:"plan"`
}

type SubscriptionPlanPayload struct {
	Id int `json:"id"`

	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`

	PriceAmount float64 `json:"price_amount"`
	Currency    string  `json:"currency"`

	DurationUnit  string `json:"duration_unit"`
	DurationValue int    `json:"duration_value"`
	CustomSeconds int64  `json:"custom_seconds"`

	Enabled   bool `json:"enabled"`
	SortOrder int  `json:"sort_order"`

	StripePriceId  string `json:"stripe_price_id"`
	CreemProductId string `json:"creem_product_id"`

	MaxPurchasePerUser int      `json:"max_purchase_per_user"`
	UpgradeGroup       string   `json:"upgrade_group"`
	EffectiveGroups    []string `json:"effective_groups"`
	TotalAmount        int64    `json:"total_amount"`

	QuotaResetPeriod        string `json:"quota_reset_period"`
	QuotaResetCustomSeconds int64  `json:"quota_reset_custom_seconds"`

	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

type BillingPreferenceRequest struct {
	BillingPreference string `json:"billing_preference"`
}

func buildSubscriptionPlanPayload(plan model.SubscriptionPlan) SubscriptionPlanPayload {
	effectiveGroups, err := plan.GetEffectiveGroups()
	if err != nil {
		common.SysLog("failed to parse subscription plan effective groups: " + err.Error())
		effectiveGroups = nil
	}
	return SubscriptionPlanPayload{
		Id:                      plan.Id,
		Title:                   plan.Title,
		Subtitle:                plan.Subtitle,
		PriceAmount:             plan.PriceAmount,
		Currency:                plan.Currency,
		DurationUnit:            plan.DurationUnit,
		DurationValue:           plan.DurationValue,
		CustomSeconds:           plan.CustomSeconds,
		Enabled:                 plan.Enabled,
		SortOrder:               plan.SortOrder,
		StripePriceId:           plan.StripePriceId,
		CreemProductId:          plan.CreemProductId,
		MaxPurchasePerUser:      plan.MaxPurchasePerUser,
		UpgradeGroup:            plan.UpgradeGroup,
		EffectiveGroups:         effectiveGroups,
		TotalAmount:             plan.TotalAmount,
		QuotaResetPeriod:        plan.QuotaResetPeriod,
		QuotaResetCustomSeconds: plan.QuotaResetCustomSeconds,
		CreatedAt:               plan.CreatedAt,
		UpdatedAt:               plan.UpdatedAt,
	}
}

func normalizeAndValidateSubscriptionEffectiveGroups(groups []string) ([]string, bool) {
	normalized := model.NormalizeSubscriptionEffectiveGroups(groups)
	if len(normalized) == 0 {
		return nil, true
	}
	groupRatio := ratio_setting.GetGroupRatioCopy()
	for _, group := range normalized {
		if _, ok := groupRatio[group]; !ok {
			return nil, false
		}
	}
	return normalized, true
}

func buildSubscriptionPlanModel(payload SubscriptionPlanPayload) (model.SubscriptionPlan, bool) {
	effectiveGroups, ok := normalizeAndValidateSubscriptionEffectiveGroups(payload.EffectiveGroups)
	if !ok {
		return model.SubscriptionPlan{}, false
	}
	effectiveGroupsRaw, err := model.SerializeSubscriptionEffectiveGroups(effectiveGroups)
	if err != nil {
		return model.SubscriptionPlan{}, false
	}
	return model.SubscriptionPlan{
		Id:                      payload.Id,
		Title:                   payload.Title,
		Subtitle:                payload.Subtitle,
		PriceAmount:             payload.PriceAmount,
		Currency:                payload.Currency,
		DurationUnit:            payload.DurationUnit,
		DurationValue:           payload.DurationValue,
		CustomSeconds:           payload.CustomSeconds,
		Enabled:                 payload.Enabled,
		SortOrder:               payload.SortOrder,
		StripePriceId:           payload.StripePriceId,
		CreemProductId:          payload.CreemProductId,
		MaxPurchasePerUser:      payload.MaxPurchasePerUser,
		UpgradeGroup:            payload.UpgradeGroup,
		EffectiveGroups:         effectiveGroupsRaw,
		TotalAmount:             payload.TotalAmount,
		QuotaResetPeriod:        payload.QuotaResetPeriod,
		QuotaResetCustomSeconds: payload.QuotaResetCustomSeconds,
		CreatedAt:               payload.CreatedAt,
		UpdatedAt:               payload.UpdatedAt,
	}, true
}

// ---- User APIs ----

func GetSubscriptionPlans(c *gin.Context) {
	var plans []model.SubscriptionPlan
	if err := model.DB.Where("enabled = ?", true).Order("sort_order desc, id desc").Find(&plans).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	result := make([]SubscriptionPlanDTO, 0, len(plans))
	for _, p := range plans {
		result = append(result, SubscriptionPlanDTO{
			Plan: buildSubscriptionPlanPayload(p),
		})
	}
	common.ApiSuccess(c, result)
}

func GetSubscriptionSelf(c *gin.Context) {
	userId := c.GetInt("id")
	settingMap, _ := model.GetUserSetting(userId, false)
	pref := common.NormalizeBillingPreference(settingMap.BillingPreference)

	// Get all subscriptions (including expired)
	allSubscriptions, err := model.GetAllUserSubscriptions(userId)
	if err != nil {
		allSubscriptions = []model.SubscriptionSummary{}
	}

	// Get active subscriptions for backward compatibility
	activeSubscriptions, err := model.GetAllActiveUserSubscriptions(userId)
	if err != nil {
		activeSubscriptions = []model.SubscriptionSummary{}
	}

	common.ApiSuccess(c, gin.H{
		"billing_preference": pref,
		"subscriptions":      activeSubscriptions, // all active subscriptions
		"all_subscriptions":  allSubscriptions,    // all subscriptions including expired
	})
}

func UpdateSubscriptionPreference(c *gin.Context) {
	userId := c.GetInt("id")
	var req BillingPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	pref := common.NormalizeBillingPreference(req.BillingPreference)

	user, err := model.GetUserById(userId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	current := user.GetSetting()
	current.BillingPreference = pref
	user.SetSetting(current)
	if err := user.Update(false); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"billing_preference": pref})
}

// ---- Admin APIs ----

func AdminListSubscriptionPlans(c *gin.Context) {
	var plans []model.SubscriptionPlan
	if err := model.DB.Order("sort_order desc, id desc").Find(&plans).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	result := make([]SubscriptionPlanDTO, 0, len(plans))
	for _, p := range plans {
		result = append(result, SubscriptionPlanDTO{
			Plan: buildSubscriptionPlanPayload(p),
		})
	}
	common.ApiSuccess(c, result)
}

type AdminUpsertSubscriptionPlanRequest struct {
	Plan SubscriptionPlanPayload `json:"plan"`
}

func AdminCreateSubscriptionPlan(c *gin.Context) {
	var req AdminUpsertSubscriptionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	plan, ok := buildSubscriptionPlanModel(req.Plan)
	if !ok {
		common.ApiErrorMsg(c, "生效分组不存在或配置无效")
		return
	}
	plan.Id = 0
	if strings.TrimSpace(plan.Title) == "" {
		common.ApiErrorMsg(c, "套餐标题不能为空")
		return
	}
	if plan.PriceAmount < 0 {
		common.ApiErrorMsg(c, "价格不能为负数")
		return
	}
	if plan.PriceAmount > 9999 {
		common.ApiErrorMsg(c, "价格不能超过9999")
		return
	}
	if plan.Currency == "" {
		plan.Currency = "USD"
	}
	plan.Currency = "USD"
	if plan.DurationUnit == "" {
		plan.DurationUnit = model.SubscriptionDurationMonth
	}
	if plan.DurationValue <= 0 && plan.DurationUnit != model.SubscriptionDurationCustom {
		plan.DurationValue = 1
	}
	if plan.MaxPurchasePerUser < 0 {
		common.ApiErrorMsg(c, "购买上限不能为负数")
		return
	}
	if plan.TotalAmount < 0 {
		common.ApiErrorMsg(c, "总额度不能为负数")
		return
	}
	plan.UpgradeGroup = strings.TrimSpace(plan.UpgradeGroup)
	if plan.UpgradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[plan.UpgradeGroup]; !ok {
			common.ApiErrorMsg(c, "升级分组不存在")
			return
		}
	}
	plan.QuotaResetPeriod = model.NormalizeResetPeriod(plan.QuotaResetPeriod)
	if plan.QuotaResetPeriod == model.SubscriptionResetCustom && plan.QuotaResetCustomSeconds <= 0 {
		common.ApiErrorMsg(c, "自定义重置周期需大于0秒")
		return
	}
	err := model.DB.Create(&plan).Error
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(plan.Id)
	common.ApiSuccess(c, buildSubscriptionPlanPayload(plan))
}

func AdminUpdateSubscriptionPlan(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req AdminUpsertSubscriptionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	plan, ok := buildSubscriptionPlanModel(req.Plan)
	if !ok {
		common.ApiErrorMsg(c, "生效分组不存在或配置无效")
		return
	}
	if strings.TrimSpace(plan.Title) == "" {
		common.ApiErrorMsg(c, "套餐标题不能为空")
		return
	}
	if plan.PriceAmount < 0 {
		common.ApiErrorMsg(c, "价格不能为负数")
		return
	}
	if plan.PriceAmount > 9999 {
		common.ApiErrorMsg(c, "价格不能超过9999")
		return
	}
	plan.Id = id
	if plan.Currency == "" {
		plan.Currency = "USD"
	}
	plan.Currency = "USD"
	if plan.DurationUnit == "" {
		plan.DurationUnit = model.SubscriptionDurationMonth
	}
	if plan.DurationValue <= 0 && plan.DurationUnit != model.SubscriptionDurationCustom {
		plan.DurationValue = 1
	}
	if plan.MaxPurchasePerUser < 0 {
		common.ApiErrorMsg(c, "购买上限不能为负数")
		return
	}
	if plan.TotalAmount < 0 {
		common.ApiErrorMsg(c, "总额度不能为负数")
		return
	}
	plan.UpgradeGroup = strings.TrimSpace(plan.UpgradeGroup)
	if plan.UpgradeGroup != "" {
		if _, ok := ratio_setting.GetGroupRatioCopy()[plan.UpgradeGroup]; !ok {
			common.ApiErrorMsg(c, "升级分组不存在")
			return
		}
	}
	plan.QuotaResetPeriod = model.NormalizeResetPeriod(plan.QuotaResetPeriod)
	if plan.QuotaResetPeriod == model.SubscriptionResetCustom && plan.QuotaResetCustomSeconds <= 0 {
		common.ApiErrorMsg(c, "自定义重置周期需大于0秒")
		return
	}

	err := model.DB.Transaction(func(tx *gorm.DB) error {
		// update plan (allow zero values updates with map)
		updateMap := map[string]interface{}{
			"title":                      plan.Title,
			"subtitle":                   plan.Subtitle,
			"price_amount":               plan.PriceAmount,
			"currency":                   plan.Currency,
			"duration_unit":              plan.DurationUnit,
			"duration_value":             plan.DurationValue,
			"custom_seconds":             plan.CustomSeconds,
			"enabled":                    plan.Enabled,
			"sort_order":                 plan.SortOrder,
			"stripe_price_id":            plan.StripePriceId,
			"creem_product_id":           plan.CreemProductId,
			"max_purchase_per_user":      plan.MaxPurchasePerUser,
			"total_amount":               plan.TotalAmount,
			"upgrade_group":              plan.UpgradeGroup,
			"effective_groups":           plan.EffectiveGroups,
			"quota_reset_period":         plan.QuotaResetPeriod,
			"quota_reset_custom_seconds": plan.QuotaResetCustomSeconds,
			"updated_at":                 common.GetTimestamp(),
		}
		if err := tx.Model(&model.SubscriptionPlan{}).Where("id = ?", id).Updates(updateMap).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(id)
	common.ApiSuccess(c, nil)
}

type AdminUpdateSubscriptionPlanStatusRequest struct {
	Enabled *bool `json:"enabled"`
}

func AdminUpdateSubscriptionPlanStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "无效的ID")
		return
	}
	var req AdminUpdateSubscriptionPlanStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.DB.Model(&model.SubscriptionPlan{}).Where("id = ?", id).Update("enabled", *req.Enabled).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateSubscriptionPlanCache(id)
	common.ApiSuccess(c, nil)
}

type AdminBindSubscriptionRequest struct {
	UserId int `json:"user_id"`
	PlanId int `json:"plan_id"`
}

func AdminBindSubscription(c *gin.Context) {
	var req AdminBindSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.UserId <= 0 || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	msg, err := model.AdminBindSubscription(req.UserId, req.PlanId, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

// ---- Admin: user subscription management ----

func AdminListUserSubscriptions(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户ID")
		return
	}
	subs, err := model.GetAllUserSubscriptions(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, subs)
}

type AdminCreateUserSubscriptionRequest struct {
	PlanId int `json:"plan_id"`
}

// AdminCreateUserSubscription creates a new user subscription from a plan (no payment).
func AdminCreateUserSubscription(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户ID")
		return
	}
	var req AdminCreateUserSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	msg, err := model.AdminBindSubscription(userId, req.PlanId, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

// AdminInvalidateUserSubscription cancels a user subscription immediately.
func AdminInvalidateUserSubscription(c *gin.Context) {
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.ApiErrorMsg(c, "无效的订阅ID")
		return
	}
	msg, err := model.AdminInvalidateUserSubscription(subId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}

// AdminDeleteUserSubscription hard-deletes a user subscription.
func AdminDeleteUserSubscription(c *gin.Context) {
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.ApiErrorMsg(c, "无效的订阅ID")
		return
	}
	msg, err := model.AdminDeleteUserSubscription(subId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if msg != "" {
		common.ApiSuccess(c, gin.H{"message": msg})
		return
	}
	common.ApiSuccess(c, nil)
}
