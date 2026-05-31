package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertUserForPaymentGuardTest(t *testing.T, id int, quota int) {
	t.Helper()
	user := &User{
		Id:       id,
		Username: "payment_guard_user",
		Status:   common.UserStatusEnabled,
		Quota:    quota,
	}
	require.NoError(t, DB.Create(user).Error)
}

func insertSubscriptionPlanForPaymentGuardTest(t *testing.T, id int) *SubscriptionPlan {
	t.Helper()
	plan := &SubscriptionPlan{
		Id:            id,
		Title:         "Guard Plan",
		PriceAmount:   9.99,
		Currency:      "USD",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   1000,
	}
	require.NoError(t, DB.Create(plan).Error)
	return plan
}

func insertSubscriptionOrderForPaymentGuardTest(t *testing.T, tradeNo string, userID int, planID int, paymentProvider string) {
	t.Helper()
	order := &SubscriptionOrder{
		UserId:          userID,
		PlanId:          planID,
		Money:           9.99,
		TradeNo:         tradeNo,
		PaymentMethod:   paymentProvider,
		PaymentProvider: paymentProvider,
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, order.Insert())
}

func insertTopUpForPaymentGuardTest(t *testing.T, tradeNo string, userID int, paymentProvider string) {
	t.Helper()
	topUp := &TopUp{
		UserId:          userID,
		Amount:          2,
		Money:           9.99,
		TradeNo:         tradeNo,
		PaymentMethod:   paymentProvider,
		PaymentProvider: paymentProvider,
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())
}

func getTopUpStatusForPaymentGuardTest(t *testing.T, tradeNo string) string {
	t.Helper()
	topUp := GetTopUpByTradeNo(tradeNo)
	require.NotNil(t, topUp)
	return topUp.Status
}

func countUserSubscriptionsForPaymentGuardTest(t *testing.T, userID int) int64 {
	t.Helper()
	var count int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ?", userID).Count(&count).Error)
	return count
}

func getUserQuotaForPaymentGuardTest(t *testing.T, userID int) int {
	t.Helper()
	var user User
	require.NoError(t, DB.Select("quota").Where("id = ?", userID).First(&user).Error)
	return user.Quota
}

func getUserInviteStateForPaymentGuardTest(t *testing.T, userID int) User {
	t.Helper()
	var user User
	require.NoError(t, DB.Select("id", "quota", "aff_count", "aff_quota", "aff_history", "inviter_id").Where("id = ?", userID).First(&user).Error)
	return user
}

func insertInviterForRewardModeTest(t *testing.T, id int) {
	t.Helper()
	user := &User{
		Id:       id,
		Username: fmt.Sprintf("inviter_%d", id),
		Status:   common.UserStatusEnabled,
		Quota:    0,
		Group:    "default",
		AffCode:  fmt.Sprintf("AFF%d", id),
	}
	require.NoError(t, DB.Create(user).Error)
}

func insertInviteeForRewardModeTest(t *testing.T, id int, inviterID int) {
	t.Helper()
	user := &User{
		Id:        id,
		Username:  fmt.Sprintf("invitee_%d", id),
		Status:    common.UserStatusEnabled,
		Quota:     0,
		Group:     "default",
		AffCode:   fmt.Sprintf("CODE%d", id),
		InviterId: inviterID,
	}
	require.NoError(t, DB.Create(user).Error)
}

func TestInsert_FixedInviteRewardModeGrantsFixedRewardOnRegistration(t *testing.T) {
	truncateTables(t)

	originalMode := common.InviteRewardMode
	originalFixedQuota := common.QuotaForInviter
	originalInviteeQuota := common.QuotaForInvitee
	originalNewUserQuota := common.QuotaForNewUser
	common.InviteRewardMode = common.InviteRewardModeFixed
	common.QuotaForInviter = 2000
	common.QuotaForInvitee = 0
	common.QuotaForNewUser = 0
	t.Cleanup(func() {
		common.InviteRewardMode = originalMode
		common.QuotaForInviter = originalFixedQuota
		common.QuotaForInvitee = originalInviteeQuota
		common.QuotaForNewUser = originalNewUserQuota
	})

	insertInviterForRewardModeTest(t, 701)

	user := &User{
		Username:    "fixed_mode_invitee",
		Password:    "password123",
		DisplayName: "Fixed Invitee",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
	}
	require.NoError(t, user.Insert(701))

	inviter := getUserInviteStateForPaymentGuardTest(t, 701)
	assert.Equal(t, 1, inviter.AffCount)
	assert.Equal(t, 2000, inviter.AffQuota)
	assert.Equal(t, 2000, inviter.AffHistoryQuota)
}

func TestInsert_RatioInviteRewardModeCountsInviteWithoutGrantingFixedReward(t *testing.T) {
	truncateTables(t)

	originalMode := common.InviteRewardMode
	originalFixedQuota := common.QuotaForInviter
	originalInviteeQuota := common.QuotaForInvitee
	originalNewUserQuota := common.QuotaForNewUser
	common.InviteRewardMode = common.InviteRewardModeRatio
	common.QuotaForInviter = 2000
	common.QuotaForInvitee = 0
	common.QuotaForNewUser = 0
	t.Cleanup(func() {
		common.InviteRewardMode = originalMode
		common.QuotaForInviter = originalFixedQuota
		common.QuotaForInvitee = originalInviteeQuota
		common.QuotaForNewUser = originalNewUserQuota
	})

	insertInviterForRewardModeTest(t, 702)

	user := &User{
		Username:    "ratio_mode_invitee",
		Password:    "password123",
		DisplayName: "Ratio Invitee",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
	}
	require.NoError(t, user.Insert(702))

	inviter := getUserInviteStateForPaymentGuardTest(t, 702)
	assert.Equal(t, 1, inviter.AffCount)
	assert.Equal(t, 0, inviter.AffQuota)
	assert.Equal(t, 0, inviter.AffHistoryQuota)
}

func TestRecharge_RatioInviteRewardModeGrantsRewardOnEverySuccessfulTopUp(t *testing.T) {
	truncateTables(t)

	originalMode := common.InviteRewardMode
	originalRatio := common.InviteRewardRatio
	originalQuotaPerUnit := common.QuotaPerUnit
	common.InviteRewardMode = common.InviteRewardModeRatio
	common.InviteRewardRatio = 0.25
	common.QuotaPerUnit = 500000
	t.Cleanup(func() {
		common.InviteRewardMode = originalMode
		common.InviteRewardRatio = originalRatio
		common.QuotaPerUnit = originalQuotaPerUnit
	})

	insertInviterForRewardModeTest(t, 703)
	insertInviteeForRewardModeTest(t, 704, 703)

	firstTopUp := &TopUp{
		UserId:          704,
		Amount:          20,
		Money:           20,
		TradeNo:         "ratio-stripe-first",
		PaymentMethod:   PaymentMethodStripe,
		PaymentProvider: PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, firstTopUp.Insert())
	require.NoError(t, Recharge("ratio-stripe-first", "cus_ratio", "127.0.0.1"))

	inviterAfterFirstTopup := getUserInviteStateForPaymentGuardTest(t, 703)
	expectedFirstReward := int(20 * 0.25 * common.QuotaPerUnit)
	assert.Equal(t, expectedFirstReward, inviterAfterFirstTopup.AffQuota)
	assert.Equal(t, expectedFirstReward, inviterAfterFirstTopup.AffHistoryQuota)

	secondTopUp := &TopUp{
		UserId:          704,
		Amount:          12,
		Money:           12,
		TradeNo:         "ratio-stripe-second",
		PaymentMethod:   PaymentMethodStripe,
		PaymentProvider: PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, secondTopUp.Insert())
	require.NoError(t, Recharge("ratio-stripe-second", "cus_ratio", "127.0.0.1"))

	inviterAfterSecondTopup := getUserInviteStateForPaymentGuardTest(t, 703)
	expectedTotalReward := int((20 + 12) * 0.25 * common.QuotaPerUnit)
	assert.Equal(t, expectedTotalReward, inviterAfterSecondTopup.AffQuota)
	assert.Equal(t, expectedTotalReward, inviterAfterSecondTopup.AffHistoryQuota)
}

func TestAdminCompleteManualTopUp_RatioInviteRewardModeGrantsReward(t *testing.T) {
	truncateTables(t)

	originalMode := common.InviteRewardMode
	originalRatio := common.InviteRewardRatio
	originalQuotaPerUnit := common.QuotaPerUnit
	common.InviteRewardMode = common.InviteRewardModeRatio
	common.InviteRewardRatio = 0.25
	common.QuotaPerUnit = 500000
	t.Cleanup(func() {
		common.InviteRewardMode = originalMode
		common.InviteRewardRatio = originalRatio
		common.QuotaPerUnit = originalQuotaPerUnit
	})

	insertInviterForRewardModeTest(t, 705)
	insertInviteeForRewardModeTest(t, 706, 705)

	topUp := &TopUp{
		UserId:          706,
		Amount:          20,
		Money:           20,
		TradeNo:         "ratio-admin-manual-topup",
		PaymentMethod:   PaymentMethodStripe,
		PaymentProvider: PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, topUp.Insert())

	require.NoError(t, ManualCompleteTopUp("ratio-admin-manual-topup", "127.0.0.1"))

	invitee := getUserInviteStateForPaymentGuardTest(t, 706)
	assert.Equal(t, int(20*common.QuotaPerUnit), invitee.Quota)

	inviter := getUserInviteStateForPaymentGuardTest(t, 705)
	expectedReward := int(20 * 0.25 * common.QuotaPerUnit)
	assert.Equal(t, expectedReward, inviter.AffQuota)
	assert.Equal(t, expectedReward, inviter.AffHistoryQuota)

	completed := GetTopUpByTradeNo("ratio-admin-manual-topup")
	require.NotNil(t, completed)
	assert.Equal(t, common.TopUpStatusSuccess, completed.Status)
}

func TestAdminCreateTopUp_RatioInviteRewardModeGrantsReward(t *testing.T) {
	truncateTables(t)

	originalMode := common.InviteRewardMode
	originalRatio := common.InviteRewardRatio
	originalQuotaPerUnit := common.QuotaPerUnit
	common.InviteRewardMode = common.InviteRewardModeRatio
	common.InviteRewardRatio = 0.25
	common.QuotaPerUnit = 500000
	t.Cleanup(func() {
		common.InviteRewardMode = originalMode
		common.InviteRewardRatio = originalRatio
		common.QuotaPerUnit = originalQuotaPerUnit
	})

	insertInviterForRewardModeTest(t, 707)
	insertInviteeForRewardModeTest(t, 708, 707)

	require.NoError(t, AdminCreateTopUp(708, 12.5, "127.0.0.1"))

	invitee := getUserInviteStateForPaymentGuardTest(t, 708)
	assert.Equal(t, int(12.5*common.QuotaPerUnit), invitee.Quota)

	inviter := getUserInviteStateForPaymentGuardTest(t, 707)
	expectedReward := int(12.5 * 0.25 * common.QuotaPerUnit)
	assert.Equal(t, expectedReward, inviter.AffQuota)
	assert.Equal(t, expectedReward, inviter.AffHistoryQuota)

	var topups []TopUp
	require.NoError(t, DB.Where("user_id = ?", 708).Find(&topups).Error)
	require.Len(t, topups, 1)
	assert.Equal(t, common.TopUpStatusSuccess, topups[0].Status)
	assert.Equal(t, "admin", topups[0].PaymentMethod)
	assert.Equal(t, 12.5, topups[0].Money)
}

func TestRechargeWaffoPancake_RejectsMismatchedPaymentMethod(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 101, 0)
	insertTopUpForPaymentGuardTest(t, "waffo-pancake-guard", 101, PaymentProviderStripe)

	err := RechargeWaffoPancake("waffo-pancake-guard")
	require.Error(t, err)

	topUp := GetTopUpByTradeNo("waffo-pancake-guard")
	require.NotNil(t, topUp)
	assert.Equal(t, common.TopUpStatusPending, topUp.Status)
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, 101))
}

func TestUpdatePendingTopUpStatus_RejectsMismatchedPaymentProvider(t *testing.T) {
	testCases := []struct {
		name                    string
		tradeNo                 string
		storedPaymentProvider   string
		expectedPaymentProvider string
		targetStatus            string
	}{
		{
			name:                    "stripe expire",
			tradeNo:                 "stripe-expire-guard",
			storedPaymentProvider:   PaymentProviderCreem,
			expectedPaymentProvider: PaymentProviderStripe,
			targetStatus:            common.TopUpStatusExpired,
		},
		{
			name:                    "waffo failed",
			tradeNo:                 "waffo-failed-guard",
			storedPaymentProvider:   PaymentProviderStripe,
			expectedPaymentProvider: PaymentProviderWaffo,
			targetStatus:            common.TopUpStatusFailed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			truncateTables(t)
			insertUserForPaymentGuardTest(t, 150, 0)
			insertTopUpForPaymentGuardTest(t, tc.tradeNo, 150, tc.storedPaymentProvider)

			err := UpdatePendingTopUpStatus(tc.tradeNo, tc.expectedPaymentProvider, tc.targetStatus)
			require.ErrorIs(t, err, ErrPaymentMethodMismatch)
			assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, tc.tradeNo))
		})
	}
}

func TestCompleteSubscriptionOrder_RejectsMismatchedPaymentProvider(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 202, 0)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 301)
	insertSubscriptionOrderForPaymentGuardTest(t, "sub-guard-order", 202, plan.Id, PaymentProviderStripe)

	err := CompleteSubscriptionOrder("sub-guard-order", `{"provider":"epay"}`, PaymentProviderEpay, "alipay")
	require.ErrorIs(t, err, ErrPaymentMethodMismatch)

	order := GetSubscriptionOrderByTradeNo("sub-guard-order")
	require.NotNil(t, order)
	assert.Equal(t, common.TopUpStatusPending, order.Status)
	assert.Zero(t, countUserSubscriptionsForPaymentGuardTest(t, 202))

	topUp := GetTopUpByTradeNo("sub-guard-order")
	assert.Nil(t, topUp)
}

func TestExpireSubscriptionOrder_RejectsMismatchedPaymentProvider(t *testing.T) {
	truncateTables(t)

	insertUserForPaymentGuardTest(t, 303, 0)
	plan := insertSubscriptionPlanForPaymentGuardTest(t, 401)
	insertSubscriptionOrderForPaymentGuardTest(t, "sub-expire-guard", 303, plan.Id, PaymentProviderStripe)

	err := ExpireSubscriptionOrder("sub-expire-guard", PaymentProviderCreem)
	require.ErrorIs(t, err, ErrPaymentMethodMismatch)

	order := GetSubscriptionOrderByTradeNo("sub-expire-guard")
	require.NotNil(t, order)
	assert.Equal(t, common.TopUpStatusPending, order.Status)
}
