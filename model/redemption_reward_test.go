package model

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRedemptionRewardTest(t *testing.T) {
	t.Helper()
	setupInvitationAmountTest(t)
	require.NoError(t, DB.AutoMigrate(&Log{}))
	require.NoError(t, ensureSubscriptionPlanTableSQLite())
	oldMode, oldRatio := common.InviteRewardMode, common.InviteRewardRatio
	common.InviteRewardMode, common.InviteRewardRatio = common.InviteRewardModeRatio, 0.25
	t.Cleanup(func() { common.InviteRewardMode, common.InviteRewardRatio = oldMode, oldRatio })
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 1).Updates(map[string]any{"aff_quota": 7, "aff_history": 11, "quota": 9876}).Error)
}
func assertRedemptionReward(t *testing.T, want int) {
	t.Helper()
	var owner User
	require.NoError(t, DB.First(&owner, 1).Error)
	require.Equal(t, 7+want, owner.AffQuota)
	require.Equal(t, 11+want, owner.AffHistoryQuota)
	require.Equal(t, 9876, owner.Quota, "reward must not silently become wallet balance")
}
func TestRedemptionInviteRewardQuotaUsesRatioOnce(t *testing.T) {
	setupRedemptionRewardTest(t)
	r := &Redemption{Key: "ratio-quota", Quota: 500001, Status: common.RedemptionCodeStatusEnabled}
	require.NoError(t, DB.Create(r).Error)
	result, err := Redeem(r.Key, 2)
	require.NoError(t, err)
	require.Equal(t, 500001, result.Quota)
	assertRedemptionReward(t, 125000)
	require.Equal(t, 509877, getUserQuotaByID(t, 2))
	_, err = Redeem(r.Key, 2)
	require.Error(t, err)
	assertRedemptionReward(t, 125000)
	require.Equal(t, 509877, getUserQuotaByID(t, 2))
}
func TestRedemptionInviteRewardSubscriptionUsesGrantedQuota(t *testing.T) {
	for _, total := range []int64{5000000, 0} {
		t.Run(fmt.Sprintf("quota_%d", total), func(t *testing.T) {
			setupRedemptionRewardTest(t)
			plan := seedSubscriptionPlan(t, 301, "Redeemed reward plan")
			require.NoError(t, DB.Model(plan).Update("total_amount", total).Error)
			InvalidateSubscriptionPlanCache(plan.Id)
			r := &Redemption{Key: "ratio-sub", RedeemType: RedemptionTypeSubscription, SubscriptionPlanId: plan.Id, Quota: 999, Status: common.RedemptionCodeStatusEnabled}
			require.NoError(t, DB.Create(r).Error)
			result, err := Redeem(r.Key, 2)
			require.NoError(t, err)
			require.NotNil(t, result.Subscription)
			if total == 0 {
				assertRedemptionReward(t, 0)
			} else {
				assertRedemptionReward(t, 1250000)
			}
			require.Equal(t, 9876, getUserQuotaByID(t, 2))
			var sub UserSubscription
			require.NoError(t, DB.First(&sub, result.Subscription.SubscriptionId).Error)
			require.Equal(t, total, sub.AmountTotal)
			_, err = Redeem(r.Key, 2)
			require.Error(t, err)
			var count int64
			require.NoError(t, DB.Model(&UserSubscription{}).Count(&count).Error)
			require.EqualValues(t, 1, count)
		})
	}
}
func TestRedemptionInviteRewardHonorsExistingSettings(t *testing.T) {
	for _, tc := range []struct {
		name, mode string
		personal   *float64
		inviter    int
		disabled   bool
		want       int
	}{
		{name: "global", mode: "ratio", inviter: 1, want: 25},
		{name: "personal", mode: "ratio", personal: float64Ptr(0.1), inviter: 1, want: 10},
		{name: "personal_zero", mode: "ratio", personal: float64Ptr(0), inviter: 1},
		{name: "fixed", mode: "fixed", inviter: 1},
		{name: "no_inviter", mode: "ratio"},
		{name: "self_invitation", mode: "ratio", inviter: 2},
		{name: "disabled_inviter", mode: "ratio", inviter: 1, disabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupRedemptionRewardTest(t)
			common.InviteRewardMode = tc.mode
			require.NoError(t, DB.Model(&User{}).Where("id = ?", 2).Update("inviter_id", tc.inviter).Error)
			require.NoError(t, DB.Model(&User{}).Where("id = ?", 1).Update("invite_reward_ratio", tc.personal).Error)
			if tc.disabled {
				require.NoError(t, DB.Model(&User{}).Where("id = ?", 1).Update("status", common.UserStatusDisabled).Error)
			}
			r := &Redemption{Key: "settings-code", Quota: 100, Status: common.RedemptionCodeStatusEnabled}
			require.NoError(t, DB.Create(r).Error)
			_, err := Redeem(r.Key, 2)
			require.NoError(t, err)
			assertRedemptionReward(t, tc.want)
			require.Equal(t, 9976, getUserQuotaByID(t, 2))
		})
	}
}
func TestRedemptionInviteRewardFailureRollsBackAndRetryIsSafe(t *testing.T) {
	for _, kind := range []string{"quota", "subscription"} {
		t.Run(kind, func(t *testing.T) {
			setupRedemptionRewardTest(t)
			r := &Redemption{Key: "atomic-reward", Quota: 100, RedeemType: kind, Status: common.RedemptionCodeStatusEnabled}
			want := 25
			if kind == "subscription" {
				plan := seedSubscriptionPlan(t, 301, "Atomic reward plan")
				r.SubscriptionPlanId = plan.Id
				want = 1250000
			}
			require.NoError(t, DB.Create(r).Error)
			const callback = "test:redemption_reward_failure"
			require.NoError(t, DB.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
				if updates, ok := tx.Statement.Dest.(map[string]interface{}); ok && updates["aff_quota"] != nil {
					tx.AddError(errors.New("injected reward write failure"))
				}
			}))
			t.Cleanup(func() { DB.Callback().Update().Remove(callback) })
			_, err := Redeem(r.Key, 2)
			require.Error(t, err)
			assertRedemptionReward(t, 0)
			require.Equal(t, 9876, getUserQuotaByID(t, 2))
			var current Redemption
			require.NoError(t, DB.First(&current, r.Id).Error)
			require.Equal(t, common.RedemptionCodeStatusEnabled, current.Status)
			require.Zero(t, current.RedeemedSubscriptionId)
			var count int64
			require.NoError(t, DB.Model(&UserSubscription{}).Count(&count).Error)
			require.Zero(t, count)
			require.NoError(t, DB.Callback().Update().Remove(callback))
			_, err = Redeem(r.Key, 2)
			require.NoError(t, err)
			assertRedemptionReward(t, want)
		})
	}
}
func float64Ptr(v float64) *float64 { return &v }

func TestRedemptionInviteRewardRejectedCodesNeverCredit(t *testing.T) {
	for _, tc := range []struct {
		name string
		code Redemption
	}{
		{"expired", Redemption{Key: "expired-reward", Quota: 100, Status: common.RedemptionCodeStatusEnabled, ExpiredTime: 1}},
		{"used", Redemption{Key: "used-reward", Quota: 100, Status: common.RedemptionCodeStatusUsed}},
		{"missing_plan", Redemption{Key: "missing-reward-plan", RedeemType: RedemptionTypeSubscription, SubscriptionPlanId: 999999, Status: common.RedemptionCodeStatusEnabled}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupRedemptionRewardTest(t)
			require.NoError(t, DB.Create(&tc.code).Error)
			_, err := Redeem(tc.code.Key, 2)
			require.Error(t, err)
			assertRedemptionReward(t, 0)
			require.Equal(t, 9876, getUserQuotaByID(t, 2))
		})
	}
}
func TestRedemptionInviteRewardOverflowDoesNotLoseRedemption(t *testing.T) {
	for _, field := range []string{"aff_quota", "aff_history"} {
		t.Run(field, func(t *testing.T) {
			setupRedemptionRewardTest(t)
			max := int(^uint(0) >> 1)
			require.NoError(t, DB.Model(&User{}).Where("id = ?", 1).Update(field, max).Error)
			r := &Redemption{Key: "overflow-reward", Quota: 100, Status: common.RedemptionCodeStatusEnabled}
			require.NoError(t, DB.Create(r).Error)
			_, err := Redeem(r.Key, 2)
			require.Error(t, err)
			require.Equal(t, 9876, getUserQuotaByID(t, 2))
			var current Redemption
			require.NoError(t, DB.First(&current, r.Id).Error)
			require.Equal(t, common.RedemptionCodeStatusEnabled, current.Status)
			var owner User
			require.NoError(t, DB.First(&owner, 1).Error)
			if field == "aff_quota" {
				require.Equal(t, max, owner.AffQuota)
				require.Equal(t, 11, owner.AffHistoryQuota)
			} else {
				require.Equal(t, 7, owner.AffQuota)
				require.Equal(t, max, owner.AffHistoryQuota)
			}
		})
	}
}
func TestRedemptionInviteRewardSQLiteContentionDoesNotDuplicate(t *testing.T) {
	setupRedemptionRewardTest(t)
	r := &Redemption{Key: "sqlite-reward-race", Quota: 100, Status: common.RedemptionCodeStatusEnabled}
	require.NoError(t, DB.Create(r).Error)
	var wins atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := Redeem(r.Key, 2); err == nil {
				wins.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	require.LessOrEqual(t, wins.Load(), int32(1))
	if wins.Load() == 0 {
		_, err := Redeem(r.Key, 2)
		require.NoError(t, err)
	}
	assertRedemptionReward(t, 25)
	require.Equal(t, 9976, getUserQuotaByID(t, 2))
}
