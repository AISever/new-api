package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"gorm.io/gorm"
)

var (
	ErrClientAppNotSupported    = errors.New("app not supported")
	ErrClientGroupNotAllowed    = errors.New("group not allowed")
	ErrClientTokenLimitExceeded = errors.New("token limit exceeded")
	ErrClientTokenNotFound      = errors.New("token not found")
	ErrClientDevicePending      = errors.New("authorization pending")
	ErrClientDeviceExpired      = errors.New("device code expired")
	ErrClientDeviceInvalid      = errors.New("device code invalid")
)

const (
	clientDeviceStatusPending    = "pending"
	clientDeviceStatusAuthorized = "authorized"
	clientDeviceStatusConsumed   = "consumed"
	clientDeviceExpiresIn        = 600
	clientDevicePollInterval     = 5
)

func IsClientAppSupported(app string) bool {
	for _, supportedApp := range dto.ClientSupportedApps {
		if app == supportedApp {
			return true
		}
	}
	return false
}

func IsClientVersionSupported(version string) bool {
	version = strings.TrimSpace(version)
	if version == "" {
		return true
	}
	return compareSemanticVersion(version, dto.ClientMinVersion) >= 0
}

func compareSemanticVersion(left string, right string) int {
	leftParts, leftOK := parseSemanticVersion(left)
	rightParts, rightOK := parseSemanticVersion(right)
	if !leftOK || !rightOK {
		return -1
	}
	for i := 0; i < 3; i++ {
		if leftParts[i] > rightParts[i] {
			return 1
		}
		if leftParts[i] < rightParts[i] {
			return -1
		}
	}
	return 0
}

func parseSemanticVersion(version string) ([3]int, bool) {
	var result [3]int
	version = strings.TrimPrefix(strings.TrimSpace(strings.ToLower(version)), "v")
	mainVersion := strings.SplitN(version, "-", 2)[0]
	parts := strings.Split(mainVersion, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return result, false
	}
	for i, part := range parts {
		if part == "" {
			return result, false
		}
		value := 0
		for _, r := range part {
			if r < '0' || r > '9' {
				return result, false
			}
			value = value*10 + int(r-'0')
		}
		result[i] = value
	}
	return result, true
}

func StartClientDeviceAuth(baseURL string) (dto.ClientDeviceStartResponse, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "http://localhost"
	}
	now := common.GetTimestamp()
	for i := 0; i < 5; i++ {
		deviceCode, err := common.GenerateRandomCharsKey(48)
		if err != nil {
			return dto.ClientDeviceStartResponse{}, err
		}
		userCode := generateClientUserCode()
		record := model.ClientDeviceAuth{
			DeviceCode:  deviceCode,
			UserCode:    userCode,
			Status:      clientDeviceStatusPending,
			CreatedTime: now,
			ExpiresTime: now + clientDeviceExpiresIn,
		}
		if err := model.DB.Create(&record).Error; err != nil {
			if i < 4 {
				continue
			}
			return dto.ClientDeviceStartResponse{}, err
		}
		return dto.ClientDeviceStartResponse{
			DeviceCode:      deviceCode,
			UserCode:        userCode,
			VerificationURL: baseURL + "/console/client/device",
			ExpiresIn:       clientDeviceExpiresIn,
			Interval:        clientDevicePollInterval,
		}, nil
	}
	return dto.ClientDeviceStartResponse{}, errors.New("failed to create device auth")
}

func AuthorizeClientDeviceAuth(userCode string, userID int) error {
	userCode = normalizeClientUserCode(userCode)
	if userCode == "" || userID == 0 {
		return ErrClientDeviceInvalid
	}
	user, err := model.GetUserById(userID, true)
	if err != nil {
		return err
	}
	if user.Status == common.UserStatusDisabled {
		return ErrClientDeviceInvalid
	}
	var record model.ClientDeviceAuth
	err = model.DB.Where("user_code = ?", userCode).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrClientDeviceInvalid
	}
	if err != nil {
		return err
	}
	now := common.GetTimestamp()
	if record.ExpiresTime < now {
		return ErrClientDeviceExpired
	}
	if record.Status == clientDeviceStatusAuthorized && record.UserId != userID {
		return ErrClientDeviceInvalid
	}
	return model.DB.Model(&record).Updates(map[string]any{
		"user_id":         userID,
		"status":          clientDeviceStatusAuthorized,
		"authorized_time": now,
	}).Error
}

func PollClientDeviceAuth(deviceCode string) (dto.ClientDevicePollResponse, error) {
	deviceCode = strings.TrimSpace(deviceCode)
	if deviceCode == "" {
		return dto.ClientDevicePollResponse{}, ErrClientDeviceInvalid
	}
	var record model.ClientDeviceAuth
	err := model.DB.Where("device_code = ?", deviceCode).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.ClientDevicePollResponse{}, ErrClientDeviceInvalid
	}
	if err != nil {
		return dto.ClientDevicePollResponse{}, err
	}
	now := common.GetTimestamp()
	_ = model.DB.Model(&record).Update("polled_time", now).Error
	if record.ExpiresTime < now {
		return dto.ClientDevicePollResponse{}, ErrClientDeviceExpired
	}
	if record.Status == clientDeviceStatusConsumed {
		return dto.ClientDevicePollResponse{Status: clientDeviceStatusConsumed}, nil
	}
	if record.Status != clientDeviceStatusAuthorized || record.UserId == 0 {
		return dto.ClientDevicePollResponse{Status: clientDeviceStatusPending}, nil
	}
	user, err := model.GetUserById(record.UserId, true)
	if err != nil {
		return dto.ClientDevicePollResponse{}, err
	}
	if user.GetAccessToken() == "" {
		accessToken, err := ensureClientUserAccessToken(user)
		if err != nil {
			return dto.ClientDevicePollResponse{}, err
		}
		user.SetAccessToken(accessToken)
	}
	response := dto.ClientDevicePollResponse{
		Status:      clientDeviceStatusAuthorized,
		UserID:      user.Id,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		AccessToken: user.GetAccessToken(),
		ExpiresAt:   time.Unix(record.ExpiresTime, 0).UTC().Format(time.RFC3339),
	}
	if err := model.DB.Model(&record).Updates(map[string]any{
		"status":      clientDeviceStatusConsumed,
		"polled_time": now,
	}).Error; err != nil {
		return dto.ClientDevicePollResponse{}, err
	}
	return response, nil
}

func GetClientProfile(userID int) (dto.ClientProfileResponse, error) {
	user, err := model.GetUserById(userID, true)
	if err != nil {
		return dto.ClientProfileResponse{}, err
	}
	return dto.ClientProfileResponse{
		ID:           user.Id,
		Username:     user.Username,
		DisplayName:  user.DisplayName,
		Group:        user.Group,
		Quota:        float64(user.Quota) / common.QuotaPerUnit,
		RequestCount: user.RequestCount,
		Role:         user.Role,
		Status:       user.Status,
	}, nil
}

func GetClientGroups(userID int) ([]dto.ClientGroup, error) {
	user, err := model.GetUserCache(userID)
	if err != nil {
		return nil, err
	}
	userGroup := user.Group
	userUsableGroups := GetUserUsableGroups(userGroup)
	groupRatios := ratio_setting.GetGroupRatioCopy()
	groups := make([]dto.ClientGroup, 0, len(userUsableGroups))
	for groupName, desc := range userUsableGroups {
		if groupName != "auto" {
			if _, ok := groupRatios[groupName]; !ok {
				continue
			}
		}
		groups = append(groups, dto.ClientGroup{
			ID:                 groupName,
			Name:               groupName,
			Description:        desc,
			Ratio:              GetUserGroupRatio(userGroup, groupName),
			SupportedApps:      supportedAppsForGroup(groupName),
			DefaultModelsByApp: defaultModelsForGroup(groupName),
			QuotaStatus:        "ok",
		})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].ID < groups[j].ID })
	return groups, nil
}

func GetClientModels(userID int, app string, group string) ([]dto.ClientModel, error) {
	if app != "" && !IsClientAppSupported(app) {
		return nil, ErrClientAppNotSupported
	}
	user, err := model.GetUserCache(userID)
	if err != nil {
		return nil, err
	}
	userGroup := user.Group
	usableGroups := GetUserUsableGroups(userGroup)
	groups := make([]string, 0, len(usableGroups))
	if group != "" {
		if _, ok := usableGroups[group]; !ok {
			return nil, ErrClientGroupNotAllowed
		}
		groups = append(groups, group)
	} else {
		for groupName := range usableGroups {
			groups = append(groups, groupName)
		}
	}
	modelSet := make(map[string]struct{})
	for _, groupName := range groups {
		if groupName == "auto" {
			for _, autoGroup := range GetUserAutoGroup(userGroup) {
				addGroupModels(modelSet, autoGroup)
			}
			continue
		}
		addGroupModels(modelSet, groupName)
	}
	modelNames := make([]string, 0, len(modelSet))
	for modelName := range modelSet {
		modelNames = append(modelNames, modelName)
	}
	sort.Strings(modelNames)
	models := make([]dto.ClientModel, 0, len(modelNames))
	preferredModel := ""
	if app != "" {
		preferredModel = dto.ClientDefaultModelsByApp[app]
	}
	for _, modelName := range modelNames {
		models = append(models, dto.ClientModel{
			ID:            modelName,
			Name:          modelName,
			Protocol:      protocolForClientApp(app),
			Recommended:   preferredModel != "" && modelName == preferredModel,
			Enabled:       true,
			ContextWindow: 0,
			InputPrice:    0,
			OutputPrice:   0,
		})
	}
	return models, nil
}

func EnsureClientToken(userID int, req dto.ClientTokenEnsureRequest) (dto.ClientTokenEnsureResponse, error) {
	req.App = strings.TrimSpace(strings.ToLower(req.App))
	req.Group = strings.TrimSpace(req.Group)
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	req.Name = strings.TrimSpace(req.Name)
	if req.App == "" || req.Group == "" || req.DeviceID == "" {
		return dto.ClientTokenEnsureResponse{}, fmt.Errorf("app, group and deviceId are required")
	}
	if !IsClientAppSupported(req.App) {
		return dto.ClientTokenEnsureResponse{}, ErrClientAppNotSupported
	}
	user, err := model.GetUserCache(userID)
	if err != nil {
		return dto.ClientTokenEnsureResponse{}, err
	}
	userGroup := user.Group
	if _, ok := GetUserUsableGroups(userGroup)[req.Group]; !ok {
		return dto.ClientTokenEnsureResponse{}, ErrClientGroupNotAllowed
	}
	tokenName := StableClientTokenName(req)
	var response dto.ClientTokenEnsureResponse
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		var token model.Token
		var binding model.ClientTokenBinding
		bindingErr := tx.Where(&model.ClientTokenBinding{UserID: userID, DeviceID: stableDeviceID(req.DeviceID), App: req.App, Group: req.Group}).First(&binding).Error
		if bindingErr == nil {
			if err := tx.Where("id = ? AND user_id = ?", binding.TokenID, userID).First(&token).Error; err != nil {
				return err
			}
			response = buildEnsureTokenResponse(&token, false)
			return nil
		}
		if !errors.Is(bindingErr, gorm.ErrRecordNotFound) {
			return bindingErr
		}
		err = tx.Where("user_id = ? AND name = ?", userID, tokenName).First(&token).Error
		if err == nil {
			if err := createClientTokenBinding(tx, userID, req, &token); err != nil {
				return err
			}
			response = buildEnsureTokenResponse(&token, false)
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var count int64
		if err := tx.Model(&model.Token{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
			return err
		}
		if int(count) >= operation_setting.GetMaxUserTokens() {
			return ErrClientTokenLimitExceeded
		}
		key, err := common.GenerateKey()
		if err != nil {
			return err
		}
		token = model.Token{
			UserId:         userID,
			Name:           tokenName,
			Key:            key,
			CreatedTime:    common.GetTimestamp(),
			AccessedTime:   common.GetTimestamp(),
			ExpiredTime:    -1,
			RemainQuota:    0,
			UnlimitedQuota: true,
			Group:          req.Group,
			Status:         common.TokenStatusEnabled,
		}
		if err := tx.Create(&token).Error; err != nil {
			return err
		}
		if err := createClientTokenBinding(tx, userID, req, &token); err != nil {
			return err
		}
		response = buildEnsureTokenResponse(&token, true)
		return nil
	})
	if err != nil {
		return dto.ClientTokenEnsureResponse{}, err
	}
	return response, nil
}

func GetClientTokenStatus(userID int, req dto.ClientTokenStatusRequest) (dto.ClientTokenStatusResponse, error) {
	ensureReq := dto.ClientTokenEnsureRequest{
		App:      strings.TrimSpace(strings.ToLower(req.App)),
		Group:    strings.TrimSpace(req.Group),
		DeviceID: strings.TrimSpace(req.DeviceID),
		Name:     strings.TrimSpace(req.Name),
	}
	if ensureReq.App == "" || ensureReq.Group == "" || ensureReq.DeviceID == "" {
		return dto.ClientTokenStatusResponse{}, fmt.Errorf("app, group and deviceId are required")
	}
	if !IsClientAppSupported(ensureReq.App) {
		return dto.ClientTokenStatusResponse{}, ErrClientAppNotSupported
	}
	user, err := model.GetUserCache(userID)
	if err != nil {
		return dto.ClientTokenStatusResponse{}, err
	}
	if _, ok := GetUserUsableGroups(user.Group)[ensureReq.Group]; !ok {
		return dto.ClientTokenStatusResponse{}, ErrClientGroupNotAllowed
	}
	var token model.Token
	var binding model.ClientTokenBinding
	err = model.DB.Where(&model.ClientTokenBinding{UserID: userID, DeviceID: stableDeviceID(ensureReq.DeviceID), App: ensureReq.App, Group: ensureReq.Group}).First(&binding).Error
	if err == nil {
		err = model.DB.Where("id = ? AND user_id = ?", binding.TokenID, userID).First(&token).Error
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		tokenName := StableClientTokenName(ensureReq)
		err = model.DB.Where("user_id = ? AND name = ?", userID, tokenName).First(&token).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.ClientTokenStatusResponse{}, ErrClientTokenNotFound
	}
	if err != nil {
		return dto.ClientTokenStatusResponse{}, err
	}
	return dto.ClientTokenStatusResponse{
		TokenID:     token.Id,
		Status:      clientTokenStatus(token.Status),
		Group:       token.Group,
		QuotaStatus: quotaStatusForUser(user),
	}, nil
}

func GetClientUsageSummary(userID int, app string, group string) (dto.ClientUsageSummaryResponse, error) {
	app = strings.TrimSpace(strings.ToLower(app))
	group = strings.TrimSpace(group)
	if app != "" && !IsClientAppSupported(app) {
		return dto.ClientUsageSummaryResponse{}, ErrClientAppNotSupported
	}
	user, err := model.GetUserCache(userID)
	if err != nil {
		return dto.ClientUsageSummaryResponse{}, err
	}
	if group != "" {
		if _, ok := GetUserUsableGroups(user.Group)[group]; !ok {
			return dto.ClientUsageSummaryResponse{}, ErrClientGroupNotAllowed
		}
	}
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Unix()
	tomorrowStart := todayStart + int64(24*time.Hour/time.Second)
	nextMonthStart := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).Unix()
	today, err := sumClientQuotaData(userID, todayStart, tomorrowStart-1)
	if err != nil {
		return dto.ClientUsageSummaryResponse{}, err
	}
	month, err := sumClientQuotaData(userID, monthStart, nextMonthStart-1)
	if err != nil {
		return dto.ClientUsageSummaryResponse{}, err
	}
	return dto.ClientUsageSummaryResponse{
		TodayRequests:  today.Count,
		TodayTokens:    today.TokenUsed,
		MonthRequests:  month.Count,
		MonthTokens:    month.TokenUsed,
		QuotaStatus:    quotaStatusForUser(user),
		RemainingQuota: quotaToUSD(user.Quota),
		Currency:       "USD",
	}, nil
}

func StableClientTokenName(req dto.ClientTokenEnsureRequest) string {
	if req.Name != "" {
		return truncateTokenName(req.Name)
	}
	deviceID := sanitizeTokenNamePart(req.DeviceID)
	return truncateTokenName(fmt.Sprintf("cc-switch-%s-%s-%s", req.App, req.Group, deviceID))
}

func stableDeviceID(deviceID string) string {
	return sanitizeTokenNamePart(deviceID)
}

func createClientTokenBinding(tx *gorm.DB, userID int, req dto.ClientTokenEnsureRequest, token *model.Token) error {
	now := common.GetTimestamp()
	binding := model.ClientTokenBinding{
		UserID:    userID,
		DeviceID:  stableDeviceID(req.DeviceID),
		App:       strings.TrimSpace(strings.ToLower(req.App)),
		Group:     strings.TrimSpace(req.Group),
		TokenID:   token.Id,
		TokenName: token.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return tx.Where(&model.ClientTokenBinding{UserID: binding.UserID, DeviceID: binding.DeviceID, App: binding.App, Group: binding.Group}).FirstOrCreate(&binding).Error
}

func addGroupModels(modelSet map[string]struct{}, groupName string) {
	for _, modelName := range model.GetGroupEnabledModels(groupName) {
		modelSet[modelName] = struct{}{}
	}
}

func supportedAppsForGroup(groupName string) []string {
	apps := make([]string, 0, len(dto.ClientSupportedApps))
	for _, app := range dto.ClientSupportedApps {
		preferredGroup := dto.ClientPreferredGroupByApp[app]
		if preferredGroup == "" || preferredGroup == groupName || groupName == "auto" {
			apps = append(apps, app)
		}
	}
	if len(apps) == 0 {
		return append([]string(nil), dto.ClientSupportedApps...)
	}
	return apps
}

func defaultModelsForGroup(groupName string) map[string]string {
	defaults := make(map[string]string)
	for _, app := range supportedAppsForGroup(groupName) {
		if modelName := dto.ClientDefaultModelsByApp[app]; modelName != "" {
			defaults[app] = modelName
		}
	}
	return defaults
}

func protocolForClientApp(app string) string {
	switch app {
	case "claude":
		return "anthropic-messages"
	case "gemini":
		return "google-generative-ai"
	case "codex", "opencode", "openclaw":
		return "openai-responses"
	default:
		return "openai-compatible"
	}
}

func sanitizeTokenNamePart(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
		} else {
			builder.WriteByte('-')
		}
	}
	cleaned := strings.Trim(builder.String(), "-_")
	if cleaned == "" {
		return "device"
	}
	return cleaned
}

func truncateTokenName(name string) string {
	name = strings.TrimSpace(name)
	if len(name) <= 50 {
		return name
	}
	return name[:50]
}

func buildEnsureTokenResponse(token *model.Token, created bool) dto.ClientTokenEnsureResponse {
	var expiresAt *int64
	if token.ExpiredTime >= 0 {
		expiresAt = &token.ExpiredTime
	}
	return dto.ClientTokenEnsureResponse{
		TokenID:   token.Id,
		TokenName: token.Name,
		Group:     token.Group,
		Key:       token.GetFullKey(),
		Created:   created,
		ExpiresAt: expiresAt,
		Status:    clientTokenStatus(token.Status),
	}
}

type clientQuotaDataSum struct {
	Count     int
	TokenUsed int
	Quota     int
}

func sumClientQuotaData(userID int, startTime int64, endTime int64) (clientQuotaDataSum, error) {
	var result clientQuotaDataSum
	err := model.DB.Table("quota_data").
		Select("COALESCE(sum(count), 0) as count, COALESCE(sum(token_used), 0) as token_used, COALESCE(sum(quota), 0) as quota").
		Where("user_id = ? and created_at >= ? and created_at <= ?", userID, startTime, endTime).
		Scan(&result).Error
	return result, err
}

func quotaStatusForUserQuota(quota int) string {
	if quota <= 0 {
		return "exhausted"
	}
	return "ok"
}

func quotaStatusForUser(user *model.UserBase) string {
	if user == nil {
		return "unknown"
	}
	return quotaStatusForUserQuota(user.Quota)
}

func quotaToUSD(quota int) float64 {
	return float64(quota) / common.QuotaPerUnit
}

func ensureClientUserAccessToken(user *model.User) (string, error) {
	for i := 0; i < 5; i++ {
		keyLength := 29 + common.GetRandomInt(4)
		key, err := common.GenerateRandomKey(keyLength)
		if err != nil {
			return "", err
		}
		var count int64
		if err := model.DB.Model(&model.User{}).Where("access_token = ?", key).Count(&count).Error; err != nil {
			return "", err
		}
		if count != 0 {
			continue
		}
		user.SetAccessToken(key)
		if err := user.Update(false); err != nil {
			return "", err
		}
		return key, nil
	}
	return "", errors.New("failed to generate unique access token")
}

func generateClientUserCode() string {
	code := strings.ToUpper(common.GetRandomString(8))
	return code[:4] + "-" + code[4:]
}

func normalizeClientUserCode(userCode string) string {
	userCode = strings.TrimSpace(strings.ToUpper(userCode))
	userCode = strings.ReplaceAll(userCode, " ", "")
	if len(userCode) == 8 && !strings.Contains(userCode, "-") {
		userCode = userCode[:4] + "-" + userCode[4:]
	}
	return userCode
}

func clientTokenStatus(status int) string {
	switch status {
	case common.TokenStatusEnabled:
		return "active"
	case common.TokenStatusDisabled:
		return "disabled"
	case common.TokenStatusExpired:
		return "expired"
	case common.TokenStatusExhausted:
		return "exhausted"
	default:
		return "unknown"
	}
}

func GetClientGroupDescription(groupName string) string {
	return setting.GetUsableGroupDescription(groupName)
}
