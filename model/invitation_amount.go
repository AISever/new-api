package model

import (
	"errors"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Independent reporting facts: never overwrite wallet, reward or order financial fields.
// Amount is a decimal string so SQLite and SQL aggregate floating point cannot change it.
type InvitationAmountSnapshot struct {
	ID             uint   `gorm:"primaryKey"`
	SourceKind     string `gorm:"type:varchar(16);not null;uniqueIndex:idx_invitation_source,priority:1"`
	SourceID       int    `gorm:"not null;uniqueIndex:idx_invitation_source,priority:2"`
	UserID         int    `gorm:"not null;index"`
	Amount         string `gorm:"type:varchar(64);not null"`
	Currency       string `gorm:"type:varchar(8);not null"`
	CurrencySymbol string `gorm:"type:text"`
	Basis          string `gorm:"type:varchar(24);not null"`
	Quota          int64
	QuotaPerUnit   string `gorm:"type:varchar(64)"`
	ExchangeRate   string `gorm:"type:varchar(64)"`
	CreatedAt      int64
}
type InvitationLegacyAmount struct {
	Category string `json:"category"`
	Amount   string `json:"amount"`
	Count    int    `json:"count"`
}

var invitationCurrencyPattern = regexp.MustCompile(`^[A-Z]{3,8}$`)

func invitationDecimal(value string) (decimal.Decimal, error) {
	if len(value) == 0 || len(value) > 64 {
		return decimal.Zero, errors.New("invalid payment amount")
	}
	d, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, err
	}
	if d.Exponent() < -30 || d.Exponent() > 12 || d.IsNegative() || d.GreaterThan(decimal.New(1, 18)) {
		return decimal.Zero, errors.New("payment amount outside reporting bounds")
	}
	return d, nil
}
func saveInvitationSnapshot(s *InvitationAmountSnapshot) error {
	if s.SourceID <= 0 || s.UserID <= 0 || !invitationCurrencyPattern.MatchString(s.Currency) {
		return errors.New("invalid invitation amount source")
	}
	if s.Basis != "unpriced" {
		d, err := invitationDecimal(s.Amount)
		if err != nil {
			return err
		}
		s.Amount = d.String()
	}
	s.CreatedAt = common.GetTimestamp()
	r := DB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "source_kind"}, {Name: "source_id"}}, DoNothing: true}).Create(s)
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		var old InvitationAmountSnapshot
		if err := DB.Where("source_kind = ? AND source_id = ?", s.SourceKind, s.SourceID).Take(&old).Error; err != nil {
			return err
		}
		if old.UserID != s.UserID || old.Amount != s.Amount || old.Currency != s.Currency || old.CurrencySymbol != s.CurrencySymbol || old.Basis != s.Basis {
			return errors.New("conflicting immutable invitation amount")
		}
	}
	return nil
}

// Call only after a verified successful gateway result. This is idempotent and can
// be retried independently of payment fulfillment; source status gates visibility.
func RecordInvitationPayment(tradeNo, provider, amount, currency string) error {
	d, err := invitationDecimal(amount)
	if err != nil {
		return err
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if !invitationCurrencyPattern.MatchString(currency) {
		return errors.New("invalid currency")
	}
	s := InvitationAmountSnapshot{Amount: d.String(), Currency: currency, Basis: "paid"}
	var order SubscriptionOrder
	if err = DB.Where("trade_no = ?", tradeNo).Take(&order).Error; err == nil {
		if order.PaymentProvider != provider {
			return ErrPaymentMethodMismatch
		}
		s.SourceKind = "subscription"
		s.SourceID = order.Id
		s.UserID = order.UserId
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		var topup TopUp
		if err = DB.Where("trade_no = ?", tradeNo).Take(&topup).Error; err != nil {
			return err
		}
		if topup.PaymentProvider != provider {
			return ErrPaymentMethodMismatch
		}
		s.SourceKind = "topup"
		s.SourceID = topup.Id
		s.UserID = topup.UserId
	} else {
		return err
	}
	return saveInvitationSnapshot(&s)
}
func invitationRedemptionValue(quota int64) (string, string, string, string, string, error) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	unit := common.QuotaPerUnit
	rate := operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
	if quota <= 0 || math.IsNaN(unit) || math.IsInf(unit, 0) || unit <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) || rate <= 0 {
		return "", "", "", "", "", errors.New("redemption value unavailable")
	}
	currency := operation_setting.GetQuotaDisplayType()
	if currency == operation_setting.QuotaDisplayTypeTokens {
		currency = "USD"
		rate = 1
	}
	if !invitationCurrencyPattern.MatchString(currency) {
		return "", "", "", "", "", errors.New("invalid display currency")
	}
	symbol := ""
	if currency == operation_setting.QuotaDisplayTypeCustom {
		symbol = operation_setting.GetCurrencySymbol()
	}
	u, r := decimal.NewFromFloat(unit), decimal.NewFromFloat(rate)
	return decimal.NewFromInt(quota).Mul(r).DivRound(u, 30).String(), currency, symbol, u.String(), r.String(), nil
}
func RecordInvitationRedemption(redemption *Redemption) error {
	if redemption == nil || redemption.Status != common.RedemptionCodeStatusUsed || redemption.UsedUserId <= 0 {
		return errors.New("redemption not used")
	}
	quota := int64(redemption.Quota)
	if NormalizeRedemptionType(redemption.RedeemType) == RedemptionTypeSubscription {
		var sub UserSubscription
		if err := DB.Where("id = ? AND user_id = ?", redemption.RedeemedSubscriptionId, redemption.UsedUserId).Take(&sub).Error; err != nil {
			return err
		}
		quota = sub.AmountTotal
	}
	s := InvitationAmountSnapshot{SourceKind: "redemption", SourceID: redemption.Id, UserID: redemption.UsedUserId, Quota: quota, Basis: "redemption", Currency: "USD"}
	if quota <= 0 {
		s.Basis = "unpriced"
		return saveInvitationSnapshot(&s)
	}
	var err error
	s.Amount, s.Currency, s.CurrencySymbol, s.QuotaPerUnit, s.ExchangeRate, err = invitationRedemptionValue(quota)
	if err != nil {
		return err
	}
	return saveInvitationSnapshot(&s)
}
func RecordInvitationAdminTopUp(tradeNo string) error {
	var topup TopUp
	if err := DB.Where("trade_no = ? AND payment_method = ? AND status = ?", tradeNo, "admin", common.TopUpStatusSuccess).Take(&topup).Error; err != nil {
		return err
	}
	if math.IsNaN(topup.Money) || math.IsInf(topup.Money, 0) {
		return errors.New("invalid admin amount")
	}
	return saveInvitationSnapshot(&InvitationAmountSnapshot{SourceKind: "topup", SourceID: topup.Id, UserID: topup.UserId, Amount: decimal.NewFromFloat(topup.Money).String(), Currency: "USD", Basis: "admin"})
}

type invitationAmountRow struct {
	SnapshotUnit   *string
	SnapshotRate   *string
	Payload        string
	Provider       string
	UserID         int
	Money          float64
	SnapshotID     *uint
	ReportAmount   *string
	Currency       *string
	CurrencySymbol *string
	Basis          *string
	Quota          int64
	RedeemType     string
	GrantedQuota   *int64
}
type invitationAccumulator struct {
	conversions          map[string]map[string]InvitationConversion
	conversionIncomplete map[string]bool
	result               *InvitationAmounts
	totals               map[string]*[3]decimal.Decimal
	estimated            map[string]bool
	admin                map[string]decimal.Decimal
	legacy               map[string]*InvitationLegacyAmount
}

func invitationAssetKey(currency, symbol string) string {
	if currency == operation_setting.QuotaDisplayTypeCustom {
		return currency + "\x00" + symbol
	}
	return currency
}

func (a *invitationAccumulator) add(kind, amount, currency, symbol string, estimated bool) error {
	d, err := invitationDecimal(amount)
	if err != nil {
		return err
	}
	if !invitationCurrencyPattern.MatchString(currency) {
		return errors.New("invalid report currency")
	}
	currency = invitationAssetKey(currency, symbol)
	v := a.totals[currency]
	if v == nil {
		v = &[3]decimal.Decimal{}
		a.totals[currency] = v
	}
	index := 0
	if kind == "subscription" {
		index = 1
	}
	if kind == "redemption" {
		index = 2
	}
	v[index] = v[index].Add(d)
	a.estimated[currency] = a.estimated[currency] || estimated
	return nil
}

func (a *invitationAccumulator) conversion(currency, unit, rate string, estimated bool) {
	u, e1 := invitationDecimal(unit)
	r, e2 := invitationDecimal(rate)
	if e1 != nil || e2 != nil || u.IsZero() || r.IsZero() {
		a.conversionIncomplete[currency] = true
		return
	}
	key := unit + "|" + rate
	if estimated {
		key += "|estimated"
	}
	if a.conversions[currency] == nil {
		a.conversions[currency] = map[string]InvitationConversion{}
	}
	if len(a.conversions[currency]) >= 50 {
		if _, ok := a.conversions[currency][key]; !ok {
			a.conversionIncomplete[currency] = true
		}
		return
	}
	a.conversions[currency][key] = InvitationConversion{QuotaPerUnit: unit, ExchangeRate: rate, Estimated: estimated}
}

// Stream only the current page's sources with indexed joins. Memory is bounded by
// page users/currency buckets (plus distinct legacy recorded-amount groups).
func AttachInvitationAmounts(items []InvitationUser) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]int, 0, len(items))
	acc := map[int]*invitationAccumulator{}
	for i := range items {
		items[i].Amounts = &InvitationAmounts{Currencies: []InvitationCurrencyAmount{}, LegacyRecorded: []InvitationLegacyAmount{}}
		ids = append(ids, items[i].ID)
		acc[items[i].ID] = &invitationAccumulator{result: items[i].Amounts, conversions: map[string]map[string]InvitationConversion{}, conversionIncomplete: map[string]bool{}, totals: map[string]*[3]decimal.Decimal{}, estimated: map[string]bool{}, admin: map[string]decimal.Decimal{}, legacy: map[string]*InvitationLegacyAmount{}}
	}
	consume := func(q *gorm.DB, kind string) error {
		rows, err := q.Rows()
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r invitationAmountRow
			if err = DB.ScanRows(rows, &r); err != nil {
				return err
			}
			a := acc[r.UserID]
			if a == nil {
				continue
			}
			if r.SnapshotID != nil {
				if r.Basis != nil && *r.Basis == "unpriced" {
					a.result.UnpricedCount++
					continue
				}
				if r.ReportAmount != nil && r.Currency != nil {
					symbol := ""
					if r.CurrencySymbol != nil {
						symbol = *r.CurrencySymbol
					}
					asset := invitationAssetKey(*r.Currency, symbol)
					if err = a.add(kind, *r.ReportAmount, *r.Currency, symbol, false); err != nil {
						a.result.UnknownCount++
					} else if kind == "topup" && r.Basis != nil && *r.Basis == "admin" {
						d, _ := invitationDecimal(*r.ReportAmount)
						a.admin[asset] = a.admin[asset].Add(d)
					}
					if err == nil && kind == "redemption" {
						unit, rate := "", ""
						if r.SnapshotUnit != nil {
							unit = *r.SnapshotUnit
						}
						if r.SnapshotRate != nil {
							rate = *r.SnapshotRate
						}
						a.conversion(asset, unit, rate, false)
					}
					continue
				}
			}
			if kind == "redemption" {
				quota := r.Quota
				if NormalizeRedemptionType(r.RedeemType) == RedemptionTypeSubscription {
					if r.GrantedQuota == nil {
						a.result.UnpricedCount++
						continue
					}
					quota = *r.GrantedQuota
				}
				value, currency, symbol, unit, rate, e := invitationRedemptionValue(quota)
				if e != nil {
					a.result.UnpricedCount++
					continue
				}
				if err = a.add(kind, value, currency, symbol, true); err != nil {
					a.result.UnpricedCount++
					continue
				}
				a.conversion(invitationAssetKey(currency, symbol), unit, rate, true)
				continue
			}
			if kind == "subscription" {
				amount, currency, e := invitationStoredPayment(r.Payload, r.Provider)
				if e == nil {
					if e = a.add(kind, amount, currency, "", false); e != nil {
						return e
					}
					continue
				}
			}
			a.result.UnknownCount++
			if !math.IsNaN(r.Money) && !math.IsInf(r.Money, 0) && r.Money >= 0 {
				amount := decimal.NewFromFloat(r.Money).String()
				key := kind + "|" + amount
				if a.legacy[key] == nil {
					if len(a.legacy) >= 50 {
						a.result.LegacyTruncated = true
						continue
					}
					a.legacy[key] = &InvitationLegacyAmount{Category: kind, Amount: amount}
				}
				a.legacy[key].Count++
			}
		}
		return rows.Err()
	}
	join := func(table, kind string) string {
		return "LEFT JOIN invitation_amount_snapshots AS s ON s.source_kind = '" + kind + "' AND s.source_id = " + table + ".id AND s.user_id = " + table + ".user_id"
	}
	selectPayment := func(table string) string {
		return table + ".user_id, " + table + ".money, s.id AS snapshot_id, s.amount AS report_amount, s.currency, s.currency_symbol, s.basis, s.quota_per_unit AS snapshot_unit, s.exchange_rate AS snapshot_rate"
	}
	mirrored := DB.Model(&SubscriptionOrder{}).Select("1").Where("subscription_orders.trade_no = top_ups.trade_no AND subscription_orders.user_id = top_ups.user_id AND subscription_orders.status = ?", common.TopUpStatusSuccess)
	if err := consume(DB.Model(&TopUp{}).Select(selectPayment("top_ups")).Joins(join("top_ups", "topup")).Where("top_ups.user_id IN ? AND top_ups.status = ?", ids, common.TopUpStatusSuccess).Where("NOT EXISTS (?)", mirrored), "topup"); err != nil {
		return err
	}
	if err := consume(DB.Model(&SubscriptionOrder{}).Select(selectPayment("subscription_orders")+", subscription_orders.provider_payload AS payload, subscription_orders.payment_provider AS provider").Joins(join("subscription_orders", "subscription")).Where("subscription_orders.user_id IN ? AND subscription_orders.status = ?", ids, common.TopUpStatusSuccess), "subscription"); err != nil {
		return err
	}
	q := DB.Unscoped().Model(&Redemption{}).Select("redemptions.used_user_id AS user_id, redemptions.quota, redemptions.redeem_type, us.amount_total AS granted_quota, s.id AS snapshot_id, s.amount AS report_amount, s.currency, s.currency_symbol, s.basis, s.quota_per_unit AS snapshot_unit, s.exchange_rate AS snapshot_rate").Joins("LEFT JOIN user_subscriptions AS us ON us.id = redemptions.redeemed_subscription_id AND us.user_id = redemptions.used_user_id").Joins("LEFT JOIN invitation_amount_snapshots AS s ON s.source_kind = 'redemption' AND s.source_id = redemptions.id AND s.user_id = redemptions.used_user_id").Where("redemptions.used_user_id IN ? AND redemptions.status = ?", ids, common.RedemptionCodeStatusUsed)
	if err := consume(q, "redemption"); err != nil {
		return err
	}
	// A redeemed snapshot is written only after successful redemption. It remains
	// authoritative if an administrator later physically cleans the code record.
	existing := DB.Unscoped().Model(&Redemption{}).Select("1").Where("redemptions.id = s.source_id")
	orphan := DB.Table("invitation_amount_snapshots AS s").Select("s.user_id, s.id AS snapshot_id, s.amount AS report_amount, s.currency, s.currency_symbol, s.basis, s.quota_per_unit AS snapshot_unit, s.exchange_rate AS snapshot_rate").Where("s.source_kind = ? AND s.user_id IN ?", "redemption", ids).Where("NOT EXISTS (?)", existing)
	if err := consume(orphan, "redemption"); err != nil {
		return err
	}
	for _, a := range acc {
		currencies := make([]string, 0, len(a.totals))
		for c := range a.totals {
			currencies = append(currencies, c)
		}
		sort.Strings(currencies)
		for _, c := range currencies {
			v := a.totals[c]
			currency, symbol, _ := strings.Cut(c, "\x00")
			factors := []InvitationConversion{}
			factorKeys := make([]string, 0, len(a.conversions[c]))
			for k := range a.conversions[c] {
				factorKeys = append(factorKeys, k)
			}
			sort.Strings(factorKeys)
			for _, k := range factorKeys {
				factors = append(factors, a.conversions[c][k])
			}
			a.result.Currencies = append(a.result.Currencies, InvitationCurrencyAmount{Conversions: factors, ConversionIncomplete: a.conversionIncomplete[c], Currency: currency, CurrencySymbol: symbol, AdminTopUp: a.admin[c].String(), TopUp: v[0].String(), Subscription: v[1].String(), Redemption: v[2].String(), Total: v[0].Add(v[1]).Add(v[2]).String(), Estimated: a.estimated[c]})
		}
		keys := make([]string, 0, len(a.legacy))
		for k := range a.legacy {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			a.result.LegacyRecorded = append(a.result.LegacyRecorded, *a.legacy[k])
		}
	}
	return nil
}
