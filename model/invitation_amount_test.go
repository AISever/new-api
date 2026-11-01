package model

import (
	"fmt"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/require"
	"testing"
)

func setupInvitationAmountTest(t *testing.T) {
	db := setupInvitationTestDB(t)
	require.NoError(t, db.AutoMigrate(&Option{}, &TopUp{}, &SubscriptionOrder{}, &Redemption{}, &UserSubscription{}, &InvitationAmountSnapshot{}))
	require.NoError(t, db.Create(&User{Id: 1, Username: "owner", AffCode: "am1"}).Error)
	require.NoError(t, db.Create(&User{Id: 2, Username: "friend", AffCode: "am2", InviterId: 1, Quota: 9876}).Error)
	oldUnit, oldRate, oldDisplay := common.QuotaPerUnit, operation_setting.USDExchangeRate, operation_setting.GetGeneralSetting().QuotaDisplayType
	common.QuotaPerUnit = 500000
	operation_setting.USDExchangeRate = 7
	operation_setting.GetGeneralSetting().QuotaDisplayType = "USD"
	t.Cleanup(func() {
		common.QuotaPerUnit = oldUnit
		operation_setting.USDExchangeRate = oldRate
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldDisplay
	})
}
func TestInvitationAmountsDeduplicateSubscriptionAndPreserveWallet(t *testing.T) {
	setupInvitationAmountTest(t)
	require.NoError(t, DB.Create(&TopUp{Id: 1, UserId: 2, Money: 100, TradeNo: "paid", PaymentProvider: PaymentProviderStripe, Status: common.TopUpStatusSuccess}).Error)
	require.NoError(t, RecordInvitationPayment("paid", PaymentProviderStripe, "2.10", "USD"))
	require.NoError(t, DB.Create(&SubscriptionOrder{Id: 1, UserId: 2, Money: 3.2, TradeNo: "subpaid", PaymentProvider: PaymentProviderEpay, Status: common.TopUpStatusSuccess}).Error)
	require.NoError(t, DB.Create(&TopUp{Id: 2, UserId: 2, Money: 3.2, TradeNo: "subpaid", Status: common.TopUpStatusSuccess}).Error)
	require.NoError(t, RecordInvitationPayment("subpaid", PaymentProviderEpay, "3.20", "USD"))
	r := &Redemption{Id: 1, UsedUserId: 2, Key: "used", Status: common.RedemptionCodeStatusUsed, Quota: 500000, RedeemType: RedemptionTypeQuota}
	require.NoError(t, DB.Create(r).Error)
	require.NoError(t, recordInvitationTestRedemption(r))
	// Retries do not duplicate a payment or change its initial immutable snapshot.
	require.NoError(t, RecordInvitationPayment("paid", PaymentProviderStripe, "2.10", "USD"))
	require.Error(t, RecordInvitationPayment("paid", PaymentProviderStripe, "999", "USD"))
	require.Error(t, RecordInvitationPayment("paid", PaymentProviderEpay, "2.10", "USD"))
	require.NoError(t, DB.Delete(r).Error)
	page, err := GetInvitationPage(1, 1, 20, "", true)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	a := page.Items[0].Amounts
	require.NotNil(t, a)
	require.Len(t, a.Currencies, 1)
	require.Equal(t, "6.3", a.Currencies[0].Total)
	require.Equal(t, "2.1", a.Currencies[0].TopUp)
	require.Equal(t, "3.2", a.Currencies[0].Subscription)
	require.Equal(t, "1", a.Currencies[0].Redemption)
	require.Zero(t, a.UnknownCount)
	require.False(t, a.Currencies[0].Estimated)
	var user User
	require.NoError(t, DB.First(&user, 2).Error)
	require.Equal(t, 9876, user.Quota)
}
func TestInvitationAmountsZeroUnknownCurrencyAndConversionSnapshot(t *testing.T) {
	setupInvitationAmountTest(t)
	for _, topup := range []TopUp{{Id: 1, UserId: 2, TradeNo: "free", Status: common.TopUpStatusSuccess, PaymentProvider: PaymentProviderStripe}, {Id: 2, UserId: 2, TradeNo: "old", Money: 9.99, Status: common.TopUpStatusSuccess}, {Id: 3, UserId: 2, TradeNo: "pending", Money: 100, Status: common.TopUpStatusPending}} {
		require.NoError(t, DB.Create(&topup).Error)
	}
	require.NoError(t, RecordInvitationPayment("free", PaymentProviderStripe, "0", "JPY"))
	r := &Redemption{Id: 1, UsedUserId: 2, Key: "unit", Status: common.RedemptionCodeStatusUsed, Quota: 500000, RedeemType: RedemptionTypeQuota}
	require.NoError(t, DB.Create(r).Error)
	require.NoError(t, recordInvitationTestRedemption(r))
	common.QuotaPerUnit = 1000000
	// Historical snapshots remain USD 1; only a missing old snapshot uses the current estimate.
	old := &Redemption{Id: 2, UsedUserId: 2, Key: "legacy", Status: common.RedemptionCodeStatusUsed, Quota: 500000}
	require.NoError(t, DB.Create(old).Error)
	require.NoError(t, DB.Create(&UserSubscription{Id: 4, UserId: 2, AmountTotal: 0}).Error)
	require.NoError(t, DB.Create(&Redemption{Id: 3, UsedUserId: 2, Key: "unlimited", Status: common.RedemptionCodeStatusUsed, RedeemType: RedemptionTypeSubscription, RedeemedSubscriptionId: 4}).Error)
	page, err := GetInvitationPage(1, 1, 20, "", true)
	require.NoError(t, err)
	a := page.Items[0].Amounts
	require.Equal(t, 1, a.UnknownCount)
	require.Equal(t, 1, a.UnpricedCount)
	require.Len(t, a.Currencies, 2)
	require.Equal(t, "JPY", a.Currencies[0].Currency)
	require.Equal(t, "0", a.Currencies[0].Total)
	require.Equal(t, "USD", a.Currencies[1].Currency)
	require.Equal(t, "1.5", a.Currencies[1].Total)
	require.True(t, a.Currencies[1].Estimated)
	require.Len(t, a.Currencies[1].Conversions, 2)
	require.Equal(t, "1000000", a.Currencies[1].Conversions[0].QuotaPerUnit)
	require.Equal(t, "9.99", a.LegacyRecorded[0].Amount)
}
func TestInvitationPaymentValidationRejectsUnboundedOrNegativeMoney(t *testing.T) {
	setupInvitationAmountTest(t)
	require.NoError(t, DB.Create(&TopUp{Id: 1, UserId: 2, TradeNo: "x", PaymentProvider: PaymentProviderStripe, Status: common.TopUpStatusSuccess}).Error)
	for _, value := range []string{"-1", "NaN", "1e99999999", "", "2.1234567891234567891234567891234"} {
		require.Error(t, RecordInvitationPayment("x", PaymentProviderStripe, value, "USD"))
	}
	require.Error(t, RecordInvitationPayment("x", PaymentProviderStripe, "1", "not-a-currency"))
}
func TestInvitationAmountsReadVerifiedHistoricalSubscriptionPayload(t *testing.T) {
	setupInvitationAmountTest(t)
	require.NoError(t, DB.Create(&SubscriptionOrder{Id: 9, UserId: 2, TradeNo: "old-sub", Money: 99, Status: common.TopUpStatusSuccess, PaymentProvider: PaymentProviderStripe, ProviderPayload: `{"amount_total":"400","currency":"USD"}`}).Error)
	page, err := GetInvitationPage(1, 1, 20, "", true)
	require.NoError(t, err)
	require.Len(t, page.Items[0].Amounts.Currencies, 1)
	require.Equal(t, "4", page.Items[0].Amounts.Currencies[0].Subscription)
	require.Zero(t, page.Items[0].Amounts.UnknownCount)
}
func TestInvitationSnapshotFailureDoesNotUndoRedemption(t *testing.T) {
	setupInvitationAmountTest(t)
	require.NoError(t, DB.Migrator().DropTable(&InvitationAmountSnapshot{}))
	require.NoError(t, DB.Create(&Redemption{Id: 7, Key: "snapshot-failure", Status: common.RedemptionCodeStatusEnabled, Quota: 500000}).Error)
	result, err := Redeem("snapshot-failure", 2)
	require.NoError(t, err)
	require.Equal(t, 500000, result.Quota)
	var user User
	require.NoError(t, DB.First(&user, 2).Error)
	require.Equal(t, 509876, user.Quota)
}
func TestInvitationRetainsRedeemedSnapshotAfterPhysicalCleanupAndLabelsAdminCredit(t *testing.T) {
	setupInvitationAmountTest(t)
	r := &Redemption{Id: 11, UsedUserId: 2, Key: "cleaned", Status: common.RedemptionCodeStatusUsed, Quota: 500000}
	require.NoError(t, DB.Create(r).Error)
	require.NoError(t, recordInvitationTestRedemption(r))
	require.NoError(t, DB.Unscoped().Delete(r).Error)
	require.NoError(t, DB.Create(&TopUp{Id: 1, UserId: 2, TradeNo: "admin-credit", PaymentMethod: "admin", Money: 2.5, Status: common.TopUpStatusSuccess}).Error)
	require.NoError(t, RecordInvitationAdminTopUp("admin-credit"))
	page, err := GetInvitationPage(1, 1, 20, "", true)
	require.NoError(t, err)
	require.Len(t, page.Items[0].Amounts.Currencies, 1)
	require.Equal(t, "3.5", page.Items[0].Amounts.Currencies[0].Total)
	require.Equal(t, "2.5", page.Items[0].Amounts.Currencies[0].AdminTopUp)
}
func TestInvitationLegacyDetailsHaveBoundedMemoryButCountsRemainComplete(t *testing.T) {
	setupInvitationAmountTest(t)
	for i := 1; i <= 55; i++ {
		require.NoError(t, DB.Create(&TopUp{Id: i, UserId: 2, TradeNo: fmt.Sprintf("old-%d", i), Money: float64(i), Status: common.TopUpStatusSuccess}).Error)
	}
	page, err := GetInvitationPage(1, 1, 20, "", true)
	require.NoError(t, err)
	a := page.Items[0].Amounts
	require.Equal(t, 55, a.UnknownCount)
	require.Len(t, a.LegacyRecorded, 50)
	require.True(t, a.LegacyTruncated)
}
func TestInvitationPaymentsPreserveSmallVerifiedAmounts(t *testing.T) {
	setupInvitationAmountTest(t)
	require.NoError(t, DB.Create(&TopUp{Id: 1, UserId: 2, TradeNo: "small", PaymentProvider: PaymentProviderWaffo, Status: common.TopUpStatusSuccess}).Error)
	require.NoError(t, RecordInvitationPayment("small", PaymentProviderWaffo, "0.000000000000000001", "ETH"))
	page, err := GetInvitationPage(1, 1, 20, "", true)
	require.NoError(t, err)
	require.Equal(t, "0.000000000000000001", page.Items[0].Amounts.Currencies[0].Total)
}

func TestInvitationCustomRedemptionCurrenciesRemainSeparate(t *testing.T) {
	setupInvitationAmountTest(t)
	settings := operation_setting.GetGeneralSetting()
	oldSymbol, oldRate := settings.CustomCurrencySymbol, settings.CustomCurrencyExchangeRate
	t.Cleanup(func() { settings.CustomCurrencySymbol, settings.CustomCurrencyExchangeRate = oldSymbol, oldRate })
	settings.QuotaDisplayType = "CUSTOM"
	for i, symbol := range []string{"积分", "金币"} {
		settings.CustomCurrencySymbol = symbol
		settings.CustomCurrencyExchangeRate = float64(7 + i*3)
		r := &Redemption{Id: 30 + i, UsedUserId: 2, Key: fmt.Sprintf("custom-%d", i), Status: common.RedemptionCodeStatusUsed, Quota: 500000}
		require.NoError(t, DB.Create(r).Error)
		require.NoError(t, recordInvitationTestRedemption(r))
	}
	page, err := GetInvitationPage(1, 1, 20, "", true)
	require.NoError(t, err)
	require.Len(t, page.Items[0].Amounts.Currencies, 2)
	raw, err := common.Marshal(page.Items[0].Amounts)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"currency_symbol":"积分"`)
	require.Contains(t, string(raw), `"currency_symbol":"金币"`)
	require.Contains(t, string(raw), `"total":"7"`)
	require.Contains(t, string(raw), `"total":"10"`)
}

// Simulate the live completion hook with a conversion captured at fulfillment.
func recordInvitationTestRedemption(r *Redemption) error {
	rate, err := captureInvitationRedemptionRate()
	if err != nil {
		return err
	}
	return recordInvitationRedemption(r, rate)
}
func TestInvitationRedemptionBackfillRetainsEstimatedBasisAfterSnapshotFailure(t *testing.T) {
	setupInvitationAmountTest(t)
	require.NoError(t, DB.Migrator().DropTable(&InvitationAmountSnapshot{}))
	r := &Redemption{Id: 80, Key: "failed-snapshot-backfill", Status: common.RedemptionCodeStatusEnabled, Quota: 500000}
	require.NoError(t, DB.Create(r).Error)
	_, err := Redeem(r.Key, 2)
	require.NoError(t, err)
	common.QuotaPerUnit = 1000000
	require.NoError(t, DB.AutoMigrate(&InvitationAmountSnapshot{}))
	require.NoError(t, DB.First(r, 80).Error)
	require.NoError(t, RecordInvitationRedemption(r))
	page, err := GetInvitationPage(1, 1, 20, "", true)
	require.NoError(t, err)
	a := page.Items[0].Amounts.Currencies[0]
	require.Equal(t, "0.5", a.Redemption)
	require.True(t, a.Estimated, "backfill cannot recover the original conversion basis")
	require.True(t, a.Conversions[0].Estimated)
	common.QuotaPerUnit = 2000000
	require.NoError(t, RecordInvitationRedemption(r), "retry must retain the existing immutable estimate")
	page, err = GetInvitationPage(1, 1, 20, "", true)
	require.NoError(t, err)
	require.Equal(t, "0.5", page.Items[0].Amounts.Currencies[0].Redemption)
	var user User
	require.NoError(t, DB.First(&user, 2).Error)
	require.Equal(t, 509876, user.Quota)
}
