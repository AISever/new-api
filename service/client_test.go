package service

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupClientServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldUsingSQLite := common.UsingSQLite
	oldUsingMySQL := common.UsingMySQL
	oldUsingPostgreSQL := common.UsingPostgreSQL
	oldRedisEnabled := common.RedisEnabled
	oldBatchUpdateEnabled := common.BatchUpdateEnabled
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db
	requireNoError(t, model.DB.Exec("SELECT 1").Error)
	if err := db.AutoMigrate(&model.User{}, &model.Token{}, &model.Ability{}, &model.QuotaData{}, &model.ClientDeviceAuth{}, &model.ClientTokenBinding{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}
	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.UsingSQLite = oldUsingSQLite
		common.UsingMySQL = oldUsingMySQL
		common.UsingPostgreSQL = oldUsingPostgreSQL
		common.RedisEnabled = oldRedisEnabled
		common.BatchUpdateEnabled = oldBatchUpdateEnabled
	})
	return db
}

func TestGetClientProfileReturnsCurrentUser(t *testing.T) {
	db := setupClientServiceTestDB(t)
	user := model.User{Username: "alice", DisplayName: "Alice A", Password: "password", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, Group: "default", Quota: int(12 * common.QuotaPerUnit), RequestCount: 7}
	requireNoError(t, db.Create(&user).Error)

	profile, err := GetClientProfile(user.Id)
	requireNoError(t, err)
	if profile.ID != user.Id || profile.Username != "alice" || profile.DisplayName != "Alice A" || profile.Group != "default" || profile.Quota != 12 || profile.RequestCount != 7 || profile.Role != common.RoleCommonUser {
		t.Fatalf("unexpected profile: %+v", profile)
	}
}

func TestEnsureClientTokenCreatesAndReusesStableToken(t *testing.T) {
	db := setupClientServiceTestDB(t)
	requireNoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"codex-01":1}`))
	requireNoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"codex-01":"Codex"}`))
	user := model.User{Username: "alice", Password: "password", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, Group: "codex-01"}
	requireNoError(t, db.Create(&user).Error)

	req := dto.ClientTokenEnsureRequest{App: "codex", Group: "codex-01", DeviceID: "MacBook Pro"}
	first, err := EnsureClientToken(user.Id, req)
	requireNoError(t, err)
	if !first.Created {
		t.Fatalf("expected first ensure to create token")
	}
	if first.TokenName != "cc-switch-codex-codex-01-macbook-pro" {
		t.Fatalf("unexpected token name: %s", first.TokenName)
	}
	if first.Key == "" || first.Status != "active" || first.Group != "codex-01" {
		t.Fatalf("unexpected first response: %+v", first)
	}

	second, err := EnsureClientToken(user.Id, req)
	requireNoError(t, err)
	if second.Created {
		t.Fatalf("expected second ensure to reuse token")
	}
	if second.TokenID != first.TokenID || second.Key != first.Key || second.TokenName != first.TokenName {
		t.Fatalf("expected same token, first=%+v second=%+v", first, second)
	}
	var count int64
	requireNoError(t, db.Model(&model.Token{}).Where("user_id = ?", user.Id).Count(&count).Error)
	if count != 1 {
		t.Fatalf("expected one token, got %d", count)
	}
	var binding model.ClientTokenBinding
	requireNoError(t, db.Where(&model.ClientTokenBinding{UserID: user.Id, DeviceID: "macbook-pro", App: "codex", Group: "codex-01"}).First(&binding).Error)
	if binding.TokenID != first.TokenID {
		t.Fatalf("expected binding token id %d, got %d", first.TokenID, binding.TokenID)
	}
}

func TestEnsureClientTokenUsesExistingBindingOverTokenName(t *testing.T) {
	db := setupClientServiceTestDB(t)
	requireNoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"codex-01":1}`))
	requireNoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"codex-01":"Codex"}`))
	user := model.User{Username: "bound", Password: "password", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, Group: "codex-01"}
	requireNoError(t, db.Create(&user).Error)
	token := model.Token{UserId: user.Id, Name: "custom-bound-token", Key: "sk-bound", CreatedTime: common.GetTimestamp(), AccessedTime: common.GetTimestamp(), ExpiredTime: -1, UnlimitedQuota: true, Group: "codex-01", Status: common.TokenStatusEnabled}
	requireNoError(t, db.Create(&token).Error)
	binding := model.ClientTokenBinding{UserID: user.Id, DeviceID: "macbook-pro", App: "codex", Group: "codex-01", TokenID: token.Id, TokenName: token.Name}
	requireNoError(t, db.Create(&binding).Error)

	ensured, err := EnsureClientToken(user.Id, dto.ClientTokenEnsureRequest{App: "codex", Group: "codex-01", DeviceID: "MacBook Pro"})
	requireNoError(t, err)
	if ensured.Created || ensured.TokenID != token.Id || ensured.TokenName != token.Name || ensured.Key != token.Key {
		t.Fatalf("expected bound token reuse, got %+v", ensured)
	}
}

func TestEnsureClientTokenRejectsUnsupportedAppAndGroup(t *testing.T) {
	db := setupClientServiceTestDB(t)
	requireNoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"codex-01":1}`))
	requireNoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"codex-01":"Codex"}`))
	user := model.User{Username: "bob", Password: "password", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, Group: "codex-01"}
	requireNoError(t, db.Create(&user).Error)

	_, err := EnsureClientToken(user.Id, dto.ClientTokenEnsureRequest{App: "unknown", Group: "codex-01", DeviceID: "device"})
	if err != ErrClientAppNotSupported {
		t.Fatalf("expected unsupported app error, got %v", err)
	}
	_, err = EnsureClientToken(user.Id, dto.ClientTokenEnsureRequest{App: "codex", Group: "other", DeviceID: "device"})
	if err != ErrClientGroupNotAllowed {
		t.Fatalf("expected group not allowed error, got %v", err)
	}
}

func TestGetClientTokenStatusFindsStableToken(t *testing.T) {
	db := setupClientServiceTestDB(t)
	requireNoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"codex-01":1}`))
	requireNoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"codex-01":"Codex"}`))
	user := model.User{Username: "carol", Password: "password", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, Group: "codex-01", Quota: int(common.QuotaPerUnit)}
	requireNoError(t, db.Create(&user).Error)

	ensure, err := EnsureClientToken(user.Id, dto.ClientTokenEnsureRequest{App: "codex", Group: "codex-01", DeviceID: "MacBook Pro"})
	requireNoError(t, err)

	status, err := GetClientTokenStatus(user.Id, dto.ClientTokenStatusRequest{App: "codex", Group: "codex-01", DeviceID: "MacBook Pro"})
	requireNoError(t, err)
	if status.TokenID != ensure.TokenID || status.Status != "active" || status.Group != "codex-01" || status.QuotaStatus != "ok" {
		t.Fatalf("unexpected token status: %+v", status)
	}
}

func TestGetClientUsageSummaryAggregatesQuotaData(t *testing.T) {
	db := setupClientServiceTestDB(t)
	requireNoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"codex-01":1}`))
	requireNoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"codex-01":"Codex"}`))
	user := model.User{Username: "dave", Password: "password", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, Group: "codex-01", Quota: int(42 * common.QuotaPerUnit)}
	requireNoError(t, db.Create(&user).Error)
	now := time.Now().Unix()
	todayHour := now - (now % 3600)
	lastMonth := time.Now().AddDate(0, -1, 0).Unix()
	requireNoError(t, db.Create(&model.QuotaData{UserID: user.Id, Username: user.Username, ModelName: "gpt-5.4", CreatedAt: todayHour, Count: 2, TokenUsed: 300, Quota: int(3 * common.QuotaPerUnit)}).Error)
	requireNoError(t, db.Create(&model.QuotaData{UserID: user.Id, Username: user.Username, ModelName: "old", CreatedAt: lastMonth, Count: 1, TokenUsed: 100, Quota: int(common.QuotaPerUnit)}).Error)

	summary, err := GetClientUsageSummary(user.Id, "codex", "codex-01")
	requireNoError(t, err)
	if summary.TodayRequests != 2 || summary.TodayTokens != 300 || summary.MonthRequests != 2 || summary.MonthTokens != 300 {
		t.Fatalf("unexpected usage counters: %+v", summary)
	}
	if summary.RemainingQuota != 42 || summary.Currency != "USD" || summary.QuotaStatus != "ok" {
		t.Fatalf("unexpected quota fields: %+v", summary)
	}
}

func TestClientDeviceAuthStartAuthorizeAndPoll(t *testing.T) {
	db := setupClientServiceTestDB(t)
	user := model.User{Username: "erin", Password: "password", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, Group: "default"}
	requireNoError(t, db.Create(&user).Error)

	started, err := StartClientDeviceAuth("https://example.com")
	requireNoError(t, err)
	if started.DeviceCode == "" || started.UserCode == "" || started.VerificationURL != "https://example.com/console/client/device" || started.ExpiresIn <= 0 || started.Interval <= 0 {
		t.Fatalf("unexpected start response: %+v", started)
	}

	pending, err := PollClientDeviceAuth(started.DeviceCode)
	requireNoError(t, err)
	if pending.Status != "pending" || pending.AccessToken != "" {
		t.Fatalf("expected pending poll, got %+v", pending)
	}

	requireNoError(t, AuthorizeClientDeviceAuth(started.UserCode, user.Id))
	authorized, err := PollClientDeviceAuth(started.DeviceCode)
	requireNoError(t, err)
	if authorized.Status != "authorized" || authorized.UserID != user.Id || authorized.Username != user.Username || authorized.AccessToken == "" || authorized.ExpiresAt == "" {
		t.Fatalf("unexpected authorized poll: %+v", authorized)
	}

	consumed, err := PollClientDeviceAuth(started.DeviceCode)
	requireNoError(t, err)
	if consumed.Status != "consumed" || consumed.AccessToken != "" || consumed.UserID != 0 {
		t.Fatalf("expected consumed poll without credentials, got %+v", consumed)
	}

	var refreshed model.User
	requireNoError(t, db.First(&refreshed, user.Id).Error)
	if refreshed.GetAccessToken() != authorized.AccessToken {
		t.Fatalf("expected persisted access token, got %q want %q", refreshed.GetAccessToken(), authorized.AccessToken)
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
