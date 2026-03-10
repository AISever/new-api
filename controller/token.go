package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func GetAllTokens(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	tokens, err := model.GetAllUserTokens(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	total, _ := model.CountUserTokens(userId)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tokens)
	common.ApiSuccess(c, pageInfo)
	return
}

func SearchTokens(c *gin.Context) {
	userId := c.GetInt("id")
	keyword := c.Query("keyword")
	token := c.Query("token")

	pageInfo := common.GetPageQuery(c)

	tokens, total, err := model.SearchUserTokens(userId, keyword, token, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tokens)
	common.ApiSuccess(c, pageInfo)
	return
}

func GetToken(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	token, err := model.GetTokenByIds(id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    token,
	})
	return
}

func GetTokenStatus(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	userId := c.GetInt("id")
	token, err := model.GetTokenByIds(tokenId, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	expiredAt := token.ExpiredTime
	if expiredAt == -1 {
		expiredAt = 0
	}
	c.JSON(http.StatusOK, gin.H{
		"object":          "credit_summary",
		"total_granted":   token.RemainQuota,
		"total_used":      0, // not supported currently
		"total_available": token.RemainQuota,
		"expires_at":      expiredAt * 1000,
	})
}

func GetTokenUsage(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "No Authorization header",
		})
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Invalid Bearer token",
		})
		return
	}
	tokenKey := parts[1]

	token, err := model.GetTokenByKey(strings.TrimPrefix(tokenKey, "sk-"), false)
	if err != nil {
		common.SysError("failed to get token by key: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgTokenGetInfoFailed)
		return
	}

	expiredAt := token.ExpiredTime
	if expiredAt == -1 {
		expiredAt = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    true,
		"message": "ok",
		"data": gin.H{
			"object":               "token_usage",
			"name":                 token.Name,
			"total_granted":        token.RemainQuota + token.UsedQuota,
			"total_used":           token.UsedQuota,
			"total_available":      token.RemainQuota,
			"unlimited_quota":      token.UnlimitedQuota,
			"model_limits":         token.GetModelLimitsMap(),
			"model_limits_enabled": token.ModelLimitsEnabled,
			"expires_at":           expiredAt,
		},
	})
}

func AddToken(c *gin.Context) {
	token := model.Token{}
	err := c.ShouldBindJSON(&token)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(token.Name) > 50 {
		common.ApiErrorI18n(c, i18n.MsgTokenNameTooLong)
		return
	}
	// 非无限额度时，检查额度值是否超出有效范围
	if !token.UnlimitedQuota {
		if token.RemainQuota < 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaNegative)
			return
		}
		maxQuotaValue := int((1000000000 * common.QuotaPerUnit))
		if token.RemainQuota > maxQuotaValue {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaExceedMax, map[string]any{"Max": maxQuotaValue})
			return
		}
	}
	// 检查用户令牌数量是否已达上限
	maxTokens := operation_setting.GetMaxUserTokens()
	count, err := model.CountUserTokens(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if int(count) >= maxTokens {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("已达到最大令牌数量限制 (%d)", maxTokens),
		})
		return
	}
	key, err := common.GenerateKey()
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgTokenGenerateFailed)
		common.SysLog("failed to generate token key: " + err.Error())
		return
	}
	cleanToken := model.Token{
		UserId:             c.GetInt("id"),
		Name:               token.Name,
		Key:                key,
		CreatedTime:        common.GetTimestamp(),
		AccessedTime:       common.GetTimestamp(),
		ExpiredTime:        token.ExpiredTime,
		RemainQuota:        token.RemainQuota,
		UnlimitedQuota:     token.UnlimitedQuota,
		ModelLimitsEnabled: token.ModelLimitsEnabled,
		ModelLimits:        token.ModelLimits,
		AllowIps:           token.AllowIps,
		Group:              token.Group,
		CrossGroupRetry:    token.CrossGroupRetry,
	}
	err = cleanToken.Insert()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func DeleteToken(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	err := model.DeleteTokenById(id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func UpdateToken(c *gin.Context) {
	userId := c.GetInt("id")
	statusOnly := c.Query("status_only")
	token := model.Token{}
	err := c.ShouldBindJSON(&token)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(token.Name) > 50 {
		common.ApiErrorI18n(c, i18n.MsgTokenNameTooLong)
		return
	}
	if !token.UnlimitedQuota {
		if token.RemainQuota < 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaNegative)
			return
		}
		maxQuotaValue := int((1000000000 * common.QuotaPerUnit))
		if token.RemainQuota > maxQuotaValue {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaExceedMax, map[string]any{"Max": maxQuotaValue})
			return
		}
	}
	cleanToken, err := model.GetTokenByIds(token.Id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if token.Status == common.TokenStatusEnabled {
		if cleanToken.Status == common.TokenStatusExpired && cleanToken.ExpiredTime <= common.GetTimestamp() && cleanToken.ExpiredTime != -1 {
			common.ApiErrorI18n(c, i18n.MsgTokenExpiredCannotEnable)
			return
		}
		if cleanToken.Status == common.TokenStatusExhausted && cleanToken.RemainQuota <= 0 && !cleanToken.UnlimitedQuota {
			common.ApiErrorI18n(c, i18n.MsgTokenExhaustedCannotEable)
			return
		}
	}
	if statusOnly != "" {
		cleanToken.Status = token.Status
	} else {
		// If you add more fields, please also update token.Update()
		cleanToken.Name = token.Name
		cleanToken.ExpiredTime = token.ExpiredTime
		cleanToken.RemainQuota = token.RemainQuota
		cleanToken.UnlimitedQuota = token.UnlimitedQuota
		cleanToken.ModelLimitsEnabled = token.ModelLimitsEnabled
		cleanToken.ModelLimits = token.ModelLimits
		cleanToken.AllowIps = token.AllowIps
		cleanToken.Group = token.Group
		cleanToken.CrossGroupRetry = token.CrossGroupRetry
	}
	err = cleanToken.Update()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanToken,
	})
}

type TokenBatch struct {
	Ids []int `json:"ids"`
}

func DeleteTokenBatch(c *gin.Context) {
	tokenBatch := TokenBatch{}
	if err := c.ShouldBindJSON(&tokenBatch); err != nil || len(tokenBatch.Ids) == 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	userId := c.GetInt("id")
	count, err := model.BatchDeleteTokens(tokenBatch.Ids, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    count,
	})
}

// QueryToken 公开的令牌查询接口，无需登录即可查询令牌使用情况
func QueryToken(c *gin.Context) {
	var req struct {
		Key  string `json:"key"`
		Page int    `json:"page"`
		Size int    `json:"size"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.Key == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "请提供令牌",
		})
		return
	}

	// 处理 sk- 前缀，支持带或不带前缀
	key := strings.TrimPrefix(req.Key, "sk-")

	token, err := model.GetTokenByKey(key, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的令牌",
		})
		return
	}

	// 计算过期时间显示
	expiredAt := token.ExpiredTime
	if expiredAt == -1 {
		expiredAt = 0 // 0 表示永不过期
	}

	// 分页参数处理
	page := req.Page
	size := req.Size
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	// 获取调用日志（分页）
	logs, total, _ := model.GetLogByKeyPaginated(key, (page-1)*size, size)

	// 获取令牌分组信息
	// 如果令牌没有设置分组，则使用用户的分组
	group := token.Group
	if group == "" {
		// 获取用户信息以获取用户的默认分组
		user, err := model.GetUserById(token.UserId, false)
		if err == nil && user != nil && user.Group != "" {
			group = user.Group
		} else {
			group = "default"
		}
	}

	// 获取分组可用模型列表
	availableModels := getAvailableModelsForToken(token, group)

	// 返回令牌信息（不包含敏感数据如完整 key、user_id、分组等）
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_granted":    token.RemainQuota + token.UsedQuota,
			"total_used":       token.UsedQuota,
			"total_available":  token.RemainQuota,
			"expires_at":       expiredAt,
			"status":           token.Status,
			"logs":             logs,
			"logs_total":       total,
			"available_models": availableModels,
		},
	})
}

// ModelInfo 模型信息结构体，用于令牌查询返回
type ModelInfo struct {
	ModelName       string  `json:"model_name"`
	QuotaType       int     `json:"quota_type"`       // 0: 按量计费, 1: 按次计费
	ModelRatio      float64 `json:"model_ratio"`      // 模型倍率（按量计费时使用）
	ModelPrice      float64 `json:"model_price"`      // 模型价格（按次计费时使用）
	CompletionRatio float64 `json:"completion_ratio"` // 补全倍率
	InputPrice      float64 `json:"input_price"`      // 输入价格（$/M tokens）
	OutputPrice     float64 `json:"output_price"`     // 输出价格（$/M tokens）
}

// getAvailableModelsForToken 获取令牌可用的模型列表
func getAvailableModelsForToken(token *model.Token, group string) []ModelInfo {
	// 获取该分组可用的模型列表
	groupModels := model.GetGroupEnabledModels(group)
	if len(groupModels) == 0 {
		return []ModelInfo{}
	}

	// 创建分组模型的 Set 用于快速查找
	groupModelSet := make(map[string]bool)
	for _, m := range groupModels {
		groupModelSet[m] = true
	}

	// 获取所有定价数据
	allPricing := model.GetPricing()

	// 创建定价数据的 Map 用于快速查找
	pricingMap := make(map[string]*model.Pricing)
	for i := range allPricing {
		pricingMap[allPricing[i].ModelName] = &allPricing[i]
	}

	// 获取令牌模型限制
	tokenLimits := token.GetModelLimitsMap()

	// 获取分组倍率
	groupRatio := ratio_setting.GetGroupRatio(group)

	result := make([]ModelInfo, 0)

	for _, modelName := range groupModels {
		// 如果启用了令牌模型限制，检查交集
		if token.ModelLimitsEnabled && len(tokenLimits) > 0 {
			if _, ok := tokenLimits[modelName]; !ok {
				continue
			}
		}

		// 获取定价信息
		p, hasPricing := pricingMap[modelName]

		var modelRatio, completionRatio, modelPrice float64
		var quotaType int

		if hasPricing {
			modelRatio = p.ModelRatio
			completionRatio = p.CompletionRatio
			modelPrice = p.ModelPrice
			quotaType = p.QuotaType
		} else {
			// 如果没有定价信息，使用默认倍率
			modelRatio, _, _ = ratio_setting.GetModelRatio(modelName)
			completionRatio = ratio_setting.GetCompletionRatio(modelName)
			quotaType = 0
		}

		// 计算输入/输出价格（$/M tokens）
		// 基础价格：$0.002 / 1K tokens = $2 / 1M tokens
		// 实际价格 = 基础价格 * 模型倍率 * 分组倍率
		basePrice := 2.0 // $2 per 1M tokens
		inputPrice := basePrice * modelRatio * groupRatio
		outputPrice := basePrice * modelRatio * completionRatio * groupRatio

		result = append(result, ModelInfo{
			ModelName:       modelName,
			QuotaType:       quotaType,
			ModelRatio:      modelRatio,
			ModelPrice:      modelPrice,
			CompletionRatio: completionRatio,
			InputPrice:      inputPrice,
			OutputPrice:     outputPrice,
		})
	}

	return result
}

// containsGroup 检查分组列表中是否包含指定分组
func containsGroup(groups []string, target string) bool {
	for _, g := range groups {
		if g == target {
			return true
		}
	}
	return false
}
