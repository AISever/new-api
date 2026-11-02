// Copyright (C) 2026 QuantumNous
package main

import (
	"errors"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Real source tables with unrelated fields: replacing complete users would lose them.
type fixtureUser struct {
	ID                int
	Username          string
	Status            int
	InviterID         int
	InviteRewardRatio *float64
	AffQuota          int64
	AffHistory        int64
	Quota             int64
	DeletedAt         gorm.DeletedAt
}

func (fixtureUser) TableName() string { return "users" }

type fixtureCode struct {
	ID                     int
	Key                    string
	Status                 int
	Quota                  int64
	RedeemType             string
	UsedUserID             int
	RedeemedTime           int64
	RedeemedSubscriptionID int
	DeletedAt              gorm.DeletedAt
}

func (fixtureCode) TableName() string { return "redemptions" }

type fixtureSub struct {
	ID            int
	UserID        int
	AmountTotal   int64
	AmountUsed    int64
	LastResetTime int64
	NextResetTime int64
}

func (fixtureSub) TableName() string { return "user_subscriptions" }

type fixtureOption struct {
	Key   string `gorm:"primaryKey;size:100"`
	Value string
}

func (fixtureOption) TableName() string { return "options" }

type fixtureLog struct {
	ID        int
	UserID    int
	Username  string
	CreatedAt int64
	Type      int
	Content   string
	Quota     int64
	RequestID string `gorm:"size:64;index"`
	Other     string
}

func (fixtureLog) TableName() string { return "logs" }
func seedDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "backfill.db")+"?_pragma=busy_timeout(5000)"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	raw, _ := db.DB()
	raw.SetMaxOpenConns(1)
	t.Cleanup(func() { raw.Close() })
	seed(t, db)
	return db
}
func seed(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(&fixtureUser{}, &fixtureCode{}, &fixtureSub{}, &fixtureOption{}, &fixtureLog{}))
	require.NoError(t, db.Create(&[]fixtureOption{{Key: "InviteRewardMode", Value: "ratio"}, {Key: "InviteRewardRatio", Value: "0.05"}, {Key: "QuotaPerUnit", Value: "500000"}}).Error)
	require.NoError(t, db.Create(&[]fixtureUser{{ID: 1, Username: "owner", Status: 1, AffQuota: 7, AffHistory: 11, Quota: 9876}, {ID: 2, Username: "friend", Status: 1, InviterID: 1, Quota: 1234}}).Error)
	require.NoError(t, db.Create(&fixtureCode{ID: 1, Key: "SECRET-DO-NOT-EXPORT", Status: 3, Quota: 500001, RedeemType: "quota", UsedUserID: 2, RedeemedTime: 100}).Error)
}
func TestPreviewExactAndReadOnly(t *testing.T) {
	db := seedDB(t)
	p, err := Preview(db, 200)
	require.NoError(t, err)
	require.Equal(t, "25000", p.TotalQuota)
	require.Len(t, p.Entries, 1)
	require.EqualValues(t, 25000, p.Entries[0].Reward)
	require.Equal(t, "0.05", p.Entries[0].Ratio)
	require.False(t, db.Migrator().HasTable("redemption_reward_backfills"))
	var u fixtureUser
	require.NoError(t, db.First(&u, 1).Error)
	require.EqualValues(t, 7, u.AffQuota)
	encoded, _ := Digest(p)
	require.Len(t, encoded, 64)
}
func TestPreviewSubscriptionUsesGrantedTotal(t *testing.T) {
	db := seedDB(t)
	require.NoError(t, db.Create(&fixtureSub{ID: 131, UserID: 2, AmountTotal: 5000000, AmountUsed: 4999999, LastResetTime: 88, NextResetTime: 86488}).Error)
	require.NoError(t, db.Model(&fixtureCode{}).Where("id = ?", 1).Updates(map[string]any{"redeem_type": "subscription", "redeemed_subscription_id": 131, "quota": 999999999}).Error)
	p, err := Preview(db, 200)
	require.NoError(t, err)
	require.Equal(t, "250000", p.TotalQuota)
	require.EqualValues(t, 5000000, p.Entries[0].Basis)
}
func TestPreviewPersonalRatioAndSkips(t *testing.T) {
	for _, tc := range []struct {
		name          string
		updates       map[string]any
		reason, total string
	}{
		{"personal", map[string]any{"invite_reward_ratio": 0.1}, "", "50000"},
		{"zero", map[string]any{"invite_reward_ratio": 0}, "zero_ratio", "0"},
		{"disabled", map[string]any{"status": 2}, "inviter_unavailable", "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := seedDB(t)
			require.NoError(t, db.Model(&fixtureUser{}).Where("id = ?", 1).Updates(tc.updates).Error)
			p, err := Preview(db, 200)
			require.NoError(t, err)
			require.Equal(t, tc.total, p.TotalQuota)
			require.Equal(t, tc.reason, p.Entries[0].Reason)
		})
	}
}
func TestPreviewCutoffAndMissingBasis(t *testing.T) {
	db := seedDB(t)
	p, err := Preview(db, 100)
	require.NoError(t, err)
	require.Empty(t, p.Entries)
	require.NoError(t, db.Model(&fixtureCode{}).Where("id = ?", 1).Updates(map[string]any{"redeem_type": "subscription", "redeemed_subscription_id": 999}).Error)
	p, err = Preview(db, 200)
	require.NoError(t, err)
	require.Equal(t, "missing_subscription_basis", p.Entries[0].Reason)
	require.Equal(t, "0", p.TotalQuota)
}
func plan(t *testing.T, db *gorm.DB) (Plan, string) {
	t.Helper()
	p, err := Preview(db, 200)
	require.NoError(t, err)
	h, err := Digest(p)
	require.NoError(t, err)
	return p, h
}
func assertCredit(t *testing.T, db *gorm.DB, quota int64, logs int64) {
	t.Helper()
	var owner, friend fixtureUser
	require.NoError(t, db.First(&owner, 1).Error)
	require.NoError(t, db.First(&friend, 2).Error)
	require.EqualValues(t, 7+quota, owner.AffQuota)
	require.EqualValues(t, 11+quota, owner.AffHistory)
	require.EqualValues(t, 9876, owner.Quota)
	require.EqualValues(t, 1234, friend.Quota)
	var n int64
	require.NoError(t, db.Model(&fixtureLog{}).Count(&n).Error)
	require.Equal(t, logs, n)
}
func TestApplyOnceAndAfterLogCleanup(t *testing.T) {
	db := seedDB(t)
	p, h := plan(t, db)
	result, err := Apply(db, p, h)
	require.NoError(t, err)
	require.Equal(t, 1, result.Credited)
	assertCredit(t, db, 25000, 1)
	var log fixtureLog
	require.NoError(t, db.First(&log).Error)
	require.Equal(t, 1, log.UserID)
	require.Equal(t, 4, log.Type)
	require.Contains(t, log.Content, "兑换码ID 1")
	require.Contains(t, log.Content, "friend")
	require.Contains(t, log.Content, "$0.050000")
	require.NotContains(t, log.Content, "SECRET")
	result, err = Apply(db, p, h)
	require.NoError(t, err)
	require.Equal(t, 1, result.AlreadyCompleted)
	assertCredit(t, db, 25000, 1)
	require.NoError(t, db.Where("id = ?", log.ID).Delete(&fixtureLog{}).Error)
	_, err = Apply(db, p, h)
	require.NoError(t, err)
	assertCredit(t, db, 25000, 0)
	next, hash := plan(t, db)
	require.Equal(t, "0", next.TotalQuota)
	require.True(t, next.Entries[0].Completed)
	_, err = Apply(db, next, hash)
	require.NoError(t, err)
	assertCredit(t, db, 25000, 0)
}
func TestApplyLogFailureRollsBackCreditAndLedger(t *testing.T) {
	db := seedDB(t)
	p, h := plan(t, db)
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("fail-log", func(tx *gorm.DB) {
		if tx.Statement.Table == "logs" {
			tx.AddError(errors.New("injected log failure"))
		}
	}))
	_, err := Apply(db, p, h)
	require.Error(t, err)
	assertCredit(t, db, 0, 0)
	var n int64
	require.NoError(t, db.Table(ledgerTable).Count(&n).Error)
	require.Zero(t, n)
	require.NoError(t, db.Callback().Create().Remove("fail-log"))
	_, err = Apply(db, p, h)
	require.NoError(t, err)
	assertCredit(t, db, 25000, 1)
}
func TestApplyRefusesDriftAndTamperingBeforeWrites(t *testing.T) {
	for _, kind := range []string{"ratio", "inviter", "basis", "status", "hash", "entry", "duplicate", "total"} {
		t.Run(kind, func(t *testing.T) {
			db := seedDB(t)
			p, h := plan(t, db)
			switch kind {
			case "ratio":
				require.NoError(t, db.Model(&fixtureOption{}).Where(map[string]any{"key": "InviteRewardRatio"}).Update("value", "0.1").Error)
			case "inviter":
				require.NoError(t, db.Model(&fixtureUser{}).Where("id = ?", 2).Update("inviter_id", 0).Error)
			case "basis":
				require.NoError(t, db.Model(&fixtureCode{}).Where("id = ?", 1).Update("quota", 100).Error)
			case "status":
				require.NoError(t, db.Model(&fixtureCode{}).Where("id = ?", 1).Update("status", 1).Error)
			case "hash":
				h = strings.Repeat("0", 64)
			case "entry":
				p.Entries[0].Reward = 999999
				h, _ = Digest(p)
			case "duplicate":
				p.Entries = append(p.Entries, p.Entries[0])
				h, _ = Digest(p)
			case "total":
				p.TotalQuota = "1"
				h, _ = Digest(p)
			}
			_, err := Apply(db, p, h)
			require.Error(t, err)
			assertCredit(t, db, 0, 0)
			require.False(t, db.Migrator().HasTable(ledgerTable))
		})
	}
}
func TestApplyOverflowAndNativeRewardExclusion(t *testing.T) {
	t.Run("overflow", func(t *testing.T) {
		db := seedDB(t)
		require.NoError(t, db.Model(&fixtureUser{}).Where("id = ?", 1).Update("aff_quota", int64(math.MaxInt64-1)).Error)
		p, h := plan(t, db)
		_, err := Apply(db, p, h)
		require.Error(t, err)
		var n int64
		require.NoError(t, db.Model(&fixtureLog{}).Count(&n).Error)
		require.Zero(t, n)
		var u fixtureUser
		require.NoError(t, db.First(&u, 1).Error)
		require.EqualValues(t, math.MaxInt64-1, u.AffQuota)
		require.EqualValues(t, 11, u.AffHistory)
	})
	t.Run("native log", func(t *testing.T) {
		db := seedDB(t)
		require.NoError(t, db.Create(&fixtureLog{UserID: 1, Type: 4, Content: "邀请兑换返奖 $0.05，兑换码ID 1，返奖比例 0.05"}).Error)
		p, h := plan(t, db)
		require.Equal(t, "existing_reward", p.Entries[0].Reason)
		require.Equal(t, "0", p.TotalQuota)
		_, err := Apply(db, p, h)
		require.NoError(t, err)
		assertCredit(t, db, 0, 1)
	})
}
func TestApplyConcurrentOnce(t *testing.T) {
	db := seedDB(t)
	p, h := plan(t, db)
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := Apply(db, p, h); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	assertCredit(t, db, 25000, 1)
}

func TestApplyRejectsDifferentDatabaseWithIdenticalRows(t *testing.T) {
	db := seedDB(t)
	p, h := plan(t, db)
	other := seedDB(t)
	_, err := Apply(other, p, h)
	require.Error(t, err)
	assertCredit(t, other, 0, 0)
	require.False(t, other.Migrator().HasTable(ledgerTable))
}
func TestApplyProcessCrashRecovery(t *testing.T) {
	if path := os.Getenv("BACKFILL_CRASH_FIXTURE"); path != "" {
		if !strings.Contains(path, "TestApplyProcessCrashRecovery") {
			os.Exit(74)
		}
		db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		if err != nil {
			os.Exit(75)
		}
		p, h := plan(t, db)
		db.Callback().Create().Before("gorm:create").Register("crash-before-log", func(tx *gorm.DB) {
			if tx.Statement.Table == "logs" {
				os.Exit(73)
			}
		})
		Apply(db, p, h)
		os.Exit(76)
	}
	db := seedDB(t)
	var path string
	require.NoError(t, db.Raw("PRAGMA database_list").Row().Scan(new(int), new(string), &path))
	child := exec.Command(os.Args[0], "-test.run=^TestApplyProcessCrashRecovery$")
	child.Env = append(os.Environ(), "BACKFILL_CRASH_FIXTURE="+path)
	err := child.Run()
	var exit *exec.ExitError
	require.ErrorAs(t, err, &exit)
	require.Equal(t, 73, exit.ExitCode())
	assertCredit(t, db, 0, 0)
	p, h := plan(t, db)
	_, err = Apply(db, p, h)
	require.NoError(t, err)
	assertCredit(t, db, 25000, 1)
}

func TestHistoricalEligibilityBoundaries(t *testing.T) {
	for _, kind := range []string{"no_inviter", "self_invitation", "missing_inviter", "deleted_inviter", "deleted_invitee", "unknown_type", "unlimited_subscription", "rounds_to_zero", "fixed_mode"} {
		t.Run(kind, func(t *testing.T) {
			db := seedDB(t)
			reason := kind
			switch kind {
			case "no_inviter":
				require.NoError(t, db.Model(&fixtureUser{}).Where("id = ?", 2).Update("inviter_id", 0).Error)
			case "self_invitation":
				require.NoError(t, db.Model(&fixtureUser{}).Where("id = ?", 2).Update("inviter_id", 2).Error)
			case "missing_inviter":
				require.NoError(t, db.Model(&fixtureUser{}).Where("id = ?", 2).Update("inviter_id", 999).Error)
				reason = "inviter_unavailable"
			case "deleted_inviter":
				require.NoError(t, db.Delete(&fixtureUser{}, 1).Error)
				reason = "inviter_unavailable"
			case "deleted_invitee":
				require.NoError(t, db.Delete(&fixtureUser{}, 2).Error)
				reason = "invitee_unavailable"
			case "unknown_type":
				require.NoError(t, db.Model(&fixtureCode{}).Where("id = ?", 1).Update("redeem_type", "other").Error)
				reason = "unknown_redemption_type"
			case "unlimited_subscription":
				require.NoError(t, db.Create(&fixtureSub{ID: 1, UserID: 2, AmountTotal: 0}).Error)
				require.NoError(t, db.Model(&fixtureCode{}).Where("id = ?", 1).Updates(map[string]any{"redeem_type": "subscription", "redeemed_subscription_id": 1}).Error)
				reason = "nonpositive_basis"
			case "rounds_to_zero":
				require.NoError(t, db.Model(&fixtureCode{}).Where("id = ?", 1).Update("quota", 1).Error)
			case "fixed_mode":
				require.NoError(t, db.Model(&fixtureOption{}).Where(map[string]any{"key": "InviteRewardMode"}).Update("value", "fixed").Error)
			}
			p, h := plan(t, db)
			require.Equal(t, reason, p.Entries[0].Reason)
			require.Equal(t, "0", p.TotalQuota)
			r, err := Apply(db, p, h)
			require.NoError(t, err)
			require.Zero(t, r.Credited)
			require.False(t, db.Migrator().HasTable(ledgerTable))
		})
	}
}
func TestSoftDeletedUsedCodeRetainsFulfillmentEntitlement(t *testing.T) {
	db := seedDB(t)
	require.NoError(t, db.Delete(&fixtureCode{}, 1).Error)
	p, h := plan(t, db)
	require.Equal(t, "25000", p.TotalQuota)
	_, err := Apply(db, p, h)
	require.NoError(t, err)
	assertCredit(t, db, 25000, 1)
}
func TestNativeLogIDBoundaryAndBadRatios(t *testing.T) {
	t.Run("ID 10 cannot suppress ID 1", func(t *testing.T) {
		db := seedDB(t)
		require.NoError(t, db.Create(&fixtureLog{UserID: 1, Type: 4, Content: "邀请兑换返奖 $1，兑换码ID 10，返奖比例 0.05"}).Error)
		p, err := Preview(db, 200)
		require.NoError(t, err)
		require.Equal(t, "25000", p.TotalQuota)
	})
	for _, ratio := range []float64{-0.1, 1.1} {
		t.Run("invalid personal ratio", func(t *testing.T) {
			db := seedDB(t)
			require.NoError(t, db.Model(&fixtureUser{}).Where("id = ?", 1).Update("invite_reward_ratio", ratio).Error)
			_, err := Preview(db, 200)
			require.Error(t, err)
			assertCredit(t, db, 0, 0)
			require.False(t, db.Migrator().HasTable(ledgerTable))
		})
	}
}
