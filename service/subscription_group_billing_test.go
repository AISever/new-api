package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedSubscriptionPlan(t *testing.T, id int, effectiveGroups string) {
	t.Helper()
	plan := &model.SubscriptionPlan{
		Id:              id,
		Title:           "test-plan",
		PriceAmount:     9.9,
		Currency:        "USD",
		DurationUnit:    model.SubscriptionDurationMonth,
		DurationValue:   1,
		Enabled:         true,
		TotalAmount:     1000,
		EffectiveGroups: effectiveGroups,
	}
	require.NoError(t, model.DB.Create(plan).Error)
}

func seedSubscriptionWithPlan(t *testing.T, id int, userId int, planId int, amountTotal int64, amountUsed int64) {
	t.Helper()
	sub := &model.UserSubscription{
		Id:          id,
		UserId:      userId,
		PlanId:      planId,
		AmountTotal: amountTotal,
		AmountUsed:  amountUsed,
		Status:      "active",
		StartTime:   time.Now().Unix(),
		EndTime:     time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
	require.NoError(t, model.DB.Create(sub).Error)
}

func TestPreConsumeUserSubscription_FiltersByUsingGroup(t *testing.T) {
	truncate(t)

	const userID = 101
	seedSubscriptionPlan(t, 1, `["vip"]`)
	seedSubscriptionPlan(t, 2, `["default"]`)
	seedSubscriptionWithPlan(t, 11, userID, 1, 100, 0)
	seedSubscriptionWithPlan(t, 12, userID, 2, 100, 0)

	res, err := model.PreConsumeUserSubscription("req-default-group", userID, "default", "test-model", 0, 10)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 12, res.UserSubscriptionId)
	assert.Equal(t, int64(10), getSubscriptionUsed(t, 12))
	assert.Equal(t, int64(0), getSubscriptionUsed(t, 11))
}

func TestHasActiveUserSubscriptionForGroup_RespectsEffectiveGroups(t *testing.T) {
	truncate(t)

	const userID = 102
	seedSubscriptionPlan(t, 3, `["vip"]`)
	seedSubscriptionWithPlan(t, 13, userID, 3, 100, 0)

	ok, err := model.HasActiveUserSubscriptionForGroup(userID, "vip")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = model.HasActiveUserSubscriptionForGroup(userID, "default")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestNewBillingSession_SubscriptionFirstFallsBackToWalletWhenUsingGroupNotEligible(t *testing.T) {
	truncate(t)
	gin.SetMode(gin.TestMode)

	const userID = 103
	const walletQuota = 100

	seedUser(t, userID, walletQuota)
	seedSubscriptionPlan(t, 4, `["vip"]`)
	seedSubscriptionWithPlan(t, 14, userID, 4, 100, 0)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	relayInfo := &relaycommon.RelayInfo{
		UserId:          userID,
		UsingGroup:      "default",
		OriginModelName: "test-model",
		RequestId:       "req-subscription-first-fallback",
		IsPlayground:    true,
		UserSetting: dto.UserSetting{
			BillingPreference: "subscription_first",
		},
	}

	session, apiErr := NewBillingSession(c, relayInfo, 10)
	require.NotNil(t, session)
	require.Nil(t, apiErr)
	assert.Equal(t, BillingSourceWallet, relayInfo.BillingSource)
	assert.Equal(t, walletQuota-10, getUserQuota(t, userID))
	assert.Equal(t, int64(0), getSubscriptionUsed(t, 14))
}

func TestNewBillingSession_SubscriptionOnlyRejectsWhenUsingGroupNotEligible(t *testing.T) {
	truncate(t)
	gin.SetMode(gin.TestMode)

	const userID = 104
	const walletQuota = 100

	seedUser(t, userID, walletQuota)
	seedSubscriptionPlan(t, 5, `["vip"]`)
	seedSubscriptionWithPlan(t, 15, userID, 5, 100, 0)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	relayInfo := &relaycommon.RelayInfo{
		UserId:          userID,
		UsingGroup:      "default",
		OriginModelName: "test-model",
		RequestId:       "req-subscription-only-blocked",
		IsPlayground:    true,
		UserSetting: dto.UserSetting{
			BillingPreference: "subscription_only",
		},
	}

	session, apiErr := NewBillingSession(c, relayInfo, 10)
	require.Nil(t, session)
	require.NotNil(t, apiErr)
	assert.Equal(t, types.ErrorCodeInsufficientUserQuota, apiErr.GetErrorCode())
	assert.Equal(t, walletQuota, getUserQuota(t, userID))
	assert.Equal(t, int64(0), getSubscriptionUsed(t, 15))
}
