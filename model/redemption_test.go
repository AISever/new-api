package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ensureRedemptionTestSchema(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Redemption{}, &UserSubscription{}))
	if common.UsingSQLite {
		require.NoError(t, ensureSubscriptionPlanTableSQLite())
	} else {
		require.NoError(t, DB.AutoMigrate(&SubscriptionPlan{}))
	}
	t.Cleanup(func() {
		DB.Exec("DELETE FROM redemptions")
		DB.Exec("DELETE FROM subscription_plans")
		DB.Exec("DELETE FROM user_subscriptions")
		DB.Exec("DELETE FROM users")
		DB.Exec("DELETE FROM logs")
	})
}

func seedRedemptionTestUser(t *testing.T, id int, quota int) {
	t.Helper()
	user := &User{
		Id:       id,
		Username: "redeem_user_" + common.GetRandomString(6),
		Password: "test-password",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		Quota:    quota,
	}
	require.NoError(t, DB.Create(user).Error)
}

func seedSubscriptionPlan(t *testing.T, id int, title string) *SubscriptionPlan {
	t.Helper()
	plan := &SubscriptionPlan{
		Id:            id,
		Title:         title,
		PriceAmount:   9.9,
		Currency:      "USD",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   5000000,
	}
	require.NoError(t, DB.Create(plan).Error)
	InvalidateSubscriptionPlanCache(id)
	return plan
}

func getUserQuotaByID(t *testing.T, id int) int {
	t.Helper()
	var user User
	require.NoError(t, DB.Select("quota").Where("id = ?", id).First(&user).Error)
	return user.Quota
}

func getRedemptionByKey(t *testing.T, key string) *Redemption {
	t.Helper()
	var redemption Redemption
	require.NoError(t, DB.Where("`key` = ?", key).First(&redemption).Error)
	return &redemption
}

func getUserSubscriptionsByUserID(t *testing.T, userId int) []UserSubscription {
	t.Helper()
	var subs []UserSubscription
	require.NoError(t, DB.Where("user_id = ?", userId).Order("id asc").Find(&subs).Error)
	return subs
}

func TestRedeemQuotaRedemptionAddsQuotaAndMarksCodeUsed(t *testing.T) {
	ensureRedemptionTestSchema(t)
	const userID = 101
	const redeemQuota = 200000
	const initialQuota = 300000
	seedRedemptionTestUser(t, userID, initialQuota)

	redemption := &Redemption{
		UserId:      1,
		Key:         "quota-redeem-code",
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "quota card",
		Quota:       redeemQuota,
		CreatedTime: time.Now().Unix(),
		ExpiredTime: 0,
	}
	require.NoError(t, redemption.Insert())

	result, err := Redeem(redemption.Key, userID)
	require.NoError(t, err)

	assert.Equal(t, RedemptionTypeQuota, result.RedeemType)
	assert.Equal(t, redeemQuota, result.Quota)
	assert.Nil(t, result.Subscription)
	assert.Equal(t, initialQuota+redeemQuota, getUserQuotaByID(t, userID))

	reloaded := getRedemptionByKey(t, redemption.Key)
	assert.Equal(t, common.RedemptionCodeStatusUsed, reloaded.Status)
	assert.Equal(t, userID, reloaded.UsedUserId)
	assert.NotZero(t, reloaded.RedeemedTime)
}

func TestRedeemSubscriptionRedemptionCreatesUserSubscription(t *testing.T) {
	ensureRedemptionTestSchema(t)
	const userID = 202
	const planID = 301
	seedRedemptionTestUser(t, userID, 0)
	plan := seedSubscriptionPlan(t, planID, "Enterprise Monthly")

	redemption := &Redemption{
		UserId:             1,
		Key:                "subscription-redeem-code",
		Status:             common.RedemptionCodeStatusEnabled,
		Name:               "subscription card",
		RedeemType:         RedemptionTypeSubscription,
		SubscriptionPlanId: planID,
		CreatedTime:        time.Now().Unix(),
	}
	require.NoError(t, redemption.Insert())

	result, err := Redeem(redemption.Key, userID)
	require.NoError(t, err)

	require.NotNil(t, result.Subscription)
	assert.Equal(t, RedemptionTypeSubscription, result.RedeemType)
	assert.Equal(t, 0, result.Quota)
	assert.Equal(t, planID, result.Subscription.PlanId)
	assert.Equal(t, plan.Title, result.Subscription.PlanTitle)
	assert.NotZero(t, result.Subscription.SubscriptionId)

	subs := getUserSubscriptionsByUserID(t, userID)
	require.Len(t, subs, 1)
	assert.Equal(t, planID, subs[0].PlanId)
	assert.Equal(t, "redemption", subs[0].Source)

	reloaded := getRedemptionByKey(t, redemption.Key)
	assert.Equal(t, common.RedemptionCodeStatusUsed, reloaded.Status)
	assert.Equal(t, userID, reloaded.UsedUserId)
	assert.Equal(t, subs[0].Id, reloaded.RedeemedSubscriptionId)
}

func TestGetAllRedemptionsIncludesSubscriptionPlanTitle(t *testing.T) {
	ensureRedemptionTestSchema(t)
	const planID = 401
	plan := seedSubscriptionPlan(t, planID, "Starter Annual")

	redemption := &Redemption{
		UserId:                 1,
		Key:                    "subscription-title-code",
		Status:                 common.RedemptionCodeStatusEnabled,
		Name:                   "title card",
		RedeemType:             RedemptionTypeSubscription,
		SubscriptionPlanId:     planID,
		CreatedTime:            time.Now().Unix(),
		SubscriptionPlanTitle:  "",
		RedeemedSubscriptionId: 0,
	}
	require.NoError(t, redemption.Insert())

	redemptions, total, err := GetAllRedemptions(0, 10)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, redemptions, 1)
	assert.Equal(t, plan.Title, redemptions[0].SubscriptionPlanTitle)
}

func TestRedeemSubscriptionRedemptionRollsBackWhenPlanMissing(t *testing.T) {
	ensureRedemptionTestSchema(t)
	const userID = 303
	seedRedemptionTestUser(t, userID, 12345)

	redemption := &Redemption{
		UserId:             1,
		Key:                "missing-plan-redeem-code",
		Status:             common.RedemptionCodeStatusEnabled,
		Name:               "broken subscription card",
		RedeemType:         RedemptionTypeSubscription,
		SubscriptionPlanId: 999999,
		CreatedTime:        time.Now().Unix(),
	}
	require.NoError(t, redemption.Insert())

	_, err := Redeem(redemption.Key, userID)
	require.ErrorIs(t, err, ErrRedeemFailed)

	assert.Equal(t, 12345, getUserQuotaByID(t, userID))
	assert.Empty(t, getUserSubscriptionsByUserID(t, userID))

	reloaded := getRedemptionByKey(t, redemption.Key)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, reloaded.Status)
	assert.Equal(t, 0, reloaded.UsedUserId)
	assert.Equal(t, int64(0), reloaded.RedeemedTime)
}
