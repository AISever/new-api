package controller

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupRedemptionControllerTestDB(t *testing.T) *gorm.DB {
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

	model.DB = db
	model.LOG_DB = db

	if err := db.AutoMigrate(&model.Redemption{}); err != nil {
		t.Fatalf("failed to migrate redemption table: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func seedRedemptionForControllerTest(t *testing.T, db *gorm.DB, name string) *model.Redemption {
	t.Helper()

	redemption := &model.Redemption{
		UserId:      1,
		Name:        name,
		Key:         fmt.Sprintf("%s-%s", name, common.GetRandomString(6)),
		Status:      common.RedemptionCodeStatusEnabled,
		CreatedTime: common.GetTimestamp(),
		Quota:       1000,
	}
	if err := db.Create(redemption).Error; err != nil {
		t.Fatalf("failed to seed redemption: %v", err)
	}
	return redemption
}

func TestDeleteRedemptionBatchDeletesOnlySelectedCodes(t *testing.T) {
	db := setupRedemptionControllerTestDB(t)
	first := seedRedemptionForControllerTest(t, db, "first")
	second := seedRedemptionForControllerTest(t, db, "second")
	third := seedRedemptionForControllerTest(t, db, "third")

	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/redemption/batch", map[string]any{
		"ids": []int{first.Id, third.Id},
	}, 1)
	DeleteRedemptionBatch(ctx)

	response := decodeAPIResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	var deletedCount int
	if err := common.Unmarshal(response.Data, &deletedCount); err != nil {
		t.Fatalf("failed to decode deleted count: %v", err)
	}
	if deletedCount != 2 {
		t.Fatalf("expected deleted count 2, got %d", deletedCount)
	}

	var remaining []model.Redemption
	if err := db.Order("id asc").Find(&remaining).Error; err != nil {
		t.Fatalf("failed to load remaining redemptions: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected one redemption remaining, got %d", len(remaining))
	}
	if remaining[0].Id != second.Id {
		t.Fatalf("expected remaining redemption %d, got %d", second.Id, remaining[0].Id)
	}
}

func TestDeleteRedemptionBatchRejectsEmptySelection(t *testing.T) {
	setupRedemptionControllerTestDB(t)

	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/redemption/batch", map[string]any{
		"ids": []int{},
	}, 1)
	DeleteRedemptionBatch(ctx)

	response := decodeAPIResponse(t, recorder)
	if response.Success {
		t.Fatalf("expected failure response for empty selection")
	}
	if response.Message == "" {
		t.Fatalf("expected failure message for empty selection")
	}
}
