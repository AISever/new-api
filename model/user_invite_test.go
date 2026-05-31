package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupInviteUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	DB = db
	LOG_DB = db

	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("failed to migrate user table: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func seedInviterForInviteTest(t *testing.T, db *gorm.DB, inviterID int, affCode string) {
	t.Helper()

	inviter := &User{
		Id:          inviterID,
		Username:    fmt.Sprintf("inviter-%d", inviterID),
		Password:    "password123",
		DisplayName: "Inviter",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AffCode:     affCode,
	}
	if err := db.Create(inviter).Error; err != nil {
		t.Fatalf("failed to seed inviter: %v", err)
	}
}

func TestInsertCountsInviteesEvenWithoutInviterQuotaReward(t *testing.T) {
	db := setupInviteUserTestDB(t)
	seedInviterForInviteTest(t, db, 1, "AFF1")

	originalQuotaForNewUser := common.QuotaForNewUser
	originalQuotaForInvitee := common.QuotaForInvitee
	originalQuotaForInviter := common.QuotaForInviter
	common.QuotaForNewUser = 0
	common.QuotaForInvitee = 0
	common.QuotaForInviter = 0
	t.Cleanup(func() {
		common.QuotaForNewUser = originalQuotaForNewUser
		common.QuotaForInvitee = originalQuotaForInvitee
		common.QuotaForInviter = originalQuotaForInviter
	})

	user := &User{
		Username:    "invitee-password",
		Password:    "password123",
		DisplayName: "Invitee Password",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
	}
	if err := user.Insert(1); err != nil {
		t.Fatalf("failed to insert invitee: %v", err)
	}

	var inviter User
	if err := db.First(&inviter, 1).Error; err != nil {
		t.Fatalf("failed to reload inviter: %v", err)
	}
	if inviter.AffCount != 1 {
		t.Fatalf("expected inviter aff_count to increment to 1, got %d", inviter.AffCount)
	}
}

func TestFinalizeOAuthUserCreationCountsInviteesEvenWithoutInviterQuotaReward(t *testing.T) {
	db := setupInviteUserTestDB(t)
	seedInviterForInviteTest(t, db, 2, "AFF2")

	originalQuotaForNewUser := common.QuotaForNewUser
	originalQuotaForInvitee := common.QuotaForInvitee
	originalQuotaForInviter := common.QuotaForInviter
	common.QuotaForNewUser = 0
	common.QuotaForInvitee = 0
	common.QuotaForInviter = 0
	t.Cleanup(func() {
		common.QuotaForNewUser = originalQuotaForNewUser
		common.QuotaForInvitee = originalQuotaForInvitee
		common.QuotaForInviter = originalQuotaForInviter
	})

	user := &User{
		Username:    "invitee-oauth",
		DisplayName: "Invitee OAuth",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
	}
	if err := user.InsertWithTx(db, 2); err != nil {
		t.Fatalf("failed to insert oauth invitee: %v", err)
	}

	user.FinalizeOAuthUserCreation(2)

	var inviter User
	if err := db.First(&inviter, 2).Error; err != nil {
		t.Fatalf("failed to reload inviter: %v", err)
	}
	if inviter.AffCount != 1 {
		t.Fatalf("expected inviter aff_count to increment to 1 after oauth registration, got %d", inviter.AffCount)
	}
}
