package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupUserControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	model.DB = db
	model.LOG_DB = db

	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate user table: %v", err)
	}

	return db
}

func seedUserForUpdateTest(t *testing.T, db *gorm.DB, user *model.User) {
	t.Helper()

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func TestUpdateUserSparseQuotaUpdatePreservesExistingIdentityFields(t *testing.T) {
	db := setupUserControllerTestDB(t)
	seedUserForUpdateTest(t, db, &model.User{
		Id:          1,
		Username:    "quota-user",
		Password:    "password123",
		DisplayName: "Quota User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		Quota:       100,
		Remark:      "keep-me",
		AffCode:     "quota1",
	})

	ctx, recorder := newAuthenticatedContext(t, http.MethodPut, "/api/user/", map[string]any{
		"id":    1,
		"quota": 250,
	}, 999)
	ctx.Set("role", common.RoleRootUser)

	UpdateUser(ctx)

	response := decodeAPIResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	var reloaded model.User
	if err := db.First(&reloaded, 1).Error; err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}

	if reloaded.Username != "quota-user" {
		t.Fatalf("expected username to stay quota-user, got %q", reloaded.Username)
	}
	if reloaded.DisplayName != "Quota User" {
		t.Fatalf("expected display name to stay Quota User, got %q", reloaded.DisplayName)
	}
	if reloaded.Group != "default" {
		t.Fatalf("expected group to stay default, got %q", reloaded.Group)
	}
	if reloaded.Remark != "keep-me" {
		t.Fatalf("expected remark to stay keep-me, got %q", reloaded.Remark)
	}
	if reloaded.Quota != 250 {
		t.Fatalf("expected quota to update to 250, got %d", reloaded.Quota)
	}
}

func TestUpdateUserEditWithoutQuotaPreservesExistingQuota(t *testing.T) {
	db := setupUserControllerTestDB(t)
	seedUserForUpdateTest(t, db, &model.User{
		Id:          2,
		Username:    "edit-user",
		Password:    "password123",
		DisplayName: "Edit User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		Quota:       500,
		Remark:      "old remark",
		AffCode:     "edit2",
	})

	ctx, recorder := newAuthenticatedContext(t, http.MethodPut, "/api/user/", map[string]any{
		"id":           2,
		"username":     "edit-user",
		"display_name": "Edited User",
		"group":        "vip",
		"remark":       "new remark",
	}, 999)
	ctx.Set("role", common.RoleRootUser)

	UpdateUser(ctx)

	response := decodeAPIResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	var reloaded model.User
	if err := db.First(&reloaded, 2).Error; err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}

	if reloaded.Quota != 500 {
		t.Fatalf("expected quota to stay 500 when omitted, got %d", reloaded.Quota)
	}
	if reloaded.DisplayName != "Edited User" {
		t.Fatalf("expected display name to update, got %q", reloaded.DisplayName)
	}
	if reloaded.Group != "vip" {
		t.Fatalf("expected group to update, got %q", reloaded.Group)
	}
	if reloaded.Remark != "new remark" {
		t.Fatalf("expected remark to update, got %q", reloaded.Remark)
	}
}

func TestUpdateUserExplicitZeroQuotaUpdatesQuota(t *testing.T) {
	db := setupUserControllerTestDB(t)
	seedUserForUpdateTest(t, db, &model.User{
		Id:          3,
		Username:    "zero-quota-user",
		Password:    "password123",
		DisplayName: "Zero Quota User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		Quota:       500,
		Remark:      "keep-me",
		AffCode:     "zero3",
	})

	ctx, recorder := newAuthenticatedContext(t, http.MethodPut, "/api/user/", map[string]any{
		"id":    3,
		"quota": 0,
	}, 999)
	ctx.Set("role", common.RoleRootUser)

	UpdateUser(ctx)

	response := decodeAPIResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	var reloaded model.User
	if err := db.First(&reloaded, 3).Error; err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}

	if reloaded.Quota != 0 {
		t.Fatalf("expected quota to update to explicit zero, got %d", reloaded.Quota)
	}
}

func TestUpdateUserInviteRewardRatioAllowsOverrideAndClear(t *testing.T) {
	db := setupUserControllerTestDB(t)
	seedUserForUpdateTest(t, db, &model.User{
		Id:          4,
		Username:    "ratio-user",
		Password:    "password123",
		DisplayName: "Ratio User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AffCode:     "ratio4",
	})

	ctx, recorder := newAuthenticatedContext(t, http.MethodPut, "/api/user/", map[string]any{
		"id":                  4,
		"username":            "ratio-user",
		"display_name":        "Ratio User",
		"group":               "default",
		"remark":              "",
		"invite_reward_ratio": 0.25,
	}, 999)
	ctx.Set("role", common.RoleRootUser)

	UpdateUser(ctx)

	response := decodeAPIResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	var reloaded model.User
	if err := db.First(&reloaded, 4).Error; err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if reloaded.InviteRewardRatio == nil {
		t.Fatalf("expected invite reward ratio to be set")
	}
	if *reloaded.InviteRewardRatio != 0.25 {
		t.Fatalf("expected invite reward ratio 0.25, got %v", *reloaded.InviteRewardRatio)
	}

	ctx, recorder = newAuthenticatedContext(t, http.MethodPut, "/api/user/", map[string]any{
		"id":                  4,
		"username":            "ratio-user",
		"display_name":        "Ratio User",
		"group":               "default",
		"remark":              "",
		"invite_reward_ratio": nil,
	}, 999)
	ctx.Set("role", common.RoleRootUser)

	UpdateUser(ctx)

	response = decodeAPIResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	if err := db.First(&reloaded, 4).Error; err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if reloaded.InviteRewardRatio != nil {
		t.Fatalf("expected invite reward ratio to be cleared, got %v", *reloaded.InviteRewardRatio)
	}
}

func TestUpdateUserOmittedInviteRewardRatioPreservesExistingValue(t *testing.T) {
	db := setupUserControllerTestDB(t)
	ratio := 0.25
	seedUserForUpdateTest(t, db, &model.User{
		Id:                5,
		Username:          "ratio-keep-user",
		Password:          "password123",
		DisplayName:       "Ratio Keep User",
		Role:              common.RoleCommonUser,
		Status:            common.UserStatusEnabled,
		Group:             "default",
		AffCode:           "ratio5",
		InviteRewardRatio: &ratio,
	})

	ctx, recorder := newAuthenticatedContext(t, http.MethodPut, "/api/user/", map[string]any{
		"id":           5,
		"username":     "ratio-keep-user",
		"display_name": "Ratio Keep Edited",
		"group":        "vip",
		"remark":       "",
	}, 999)
	ctx.Set("role", common.RoleRootUser)

	UpdateUser(ctx)

	response := decodeAPIResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	var reloaded model.User
	if err := db.First(&reloaded, 5).Error; err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if reloaded.InviteRewardRatio == nil {
		t.Fatalf("expected invite reward ratio to be preserved")
	}
	if *reloaded.InviteRewardRatio != 0.25 {
		t.Fatalf("expected invite reward ratio 0.25, got %v", *reloaded.InviteRewardRatio)
	}
}
