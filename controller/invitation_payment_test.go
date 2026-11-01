package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
	"github.com/waffo-com/waffo-go/core"
	"testing"
)

func TestInvitationPaidAmountsRespectGatewayUnitsAndMissingValues(t *testing.T) {
	for _, tc := range []struct {
		raw, currency, want string
		minor               bool
	}{{"1234", "USD", "12.34", true}, {"1234", "JPY", "1234", true}, {"1234", "BHD", "1.234", true}, {"0", "USD", "0", true}, {"12.34", "CNY", "12.34", false}} {
		got, _, err := normalizeInvitationPayment(tc.raw, tc.currency, tc.minor)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
	for _, raw := range []string{"", "-1", "NaN", "1e10000000", "0.5"} {
		_, _, err := normalizeInvitationPayment(raw, "USD", true)
		require.Error(t, err)
	}
}
func TestInvitationWaffoDoesNotGuessActualChargeFromListPrice(t *testing.T) {
	db := setupUserControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.TopUp{}, &model.SubscriptionOrder{}, &model.InvitationAmountSnapshot{}))
	require.NoError(t, db.Create(&model.TopUp{Id: 701, UserId: 2, TradeNo: "waffo-actual-fixture", Money: 20, PaymentProvider: model.PaymentProviderWaffo, Status: common.TopUpStatusSuccess}).Error)
	result := &core.PaymentNotificationResult{MerchantOrderID: "waffo-actual-fixture", OrderAmount: "20", OrderCurrency: "USD"}
	recordWaffoInvitationPayment(result)
	var count int64
	require.NoError(t, db.Model(&model.InvitationAmountSnapshot{}).Where("source_id = ?", 701).Count(&count).Error)
	require.Zero(t, count)
	result.FinalDealAmount = "0"
	result.UserCurrency = "USD"
	recordWaffoInvitationPayment(result)
	var snapshot model.InvitationAmountSnapshot
	require.NoError(t, db.Where("source_id = ?", 701).Take(&snapshot).Error)
	require.Equal(t, "0", snapshot.Amount)
}
