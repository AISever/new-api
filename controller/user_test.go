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
