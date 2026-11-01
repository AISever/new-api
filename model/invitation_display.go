package model

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/shopspring/decimal"
	"math"
)

// Reporting follows the site's existing currency settings. It never changes
// source payment facts, redemption snapshots, or wallet accounting.
type invitationDisplayCurrency struct {
	currency, symbol  string
	perUSD, cnyPerUSD decimal.Decimal
	valid, cnyValid   bool
}

func invitationPositiveRate(v float64) (decimal.Decimal, bool) {
	if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return decimal.Zero, false
	}
	return decimal.NewFromFloat(v), true
}
func captureInvitationDisplayCurrency() *invitationDisplayCurrency {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c := &invitationDisplayCurrency{currency: "USD", symbol: "$", perUSD: decimal.NewFromInt(1), valid: true}
	c.cnyPerUSD, c.cnyValid = invitationPositiveRate(operation_setting.USDExchangeRate)
	switch operation_setting.GetQuotaDisplayType() {
	case operation_setting.QuotaDisplayTypeCNY:
		c.currency, c.symbol, c.perUSD, c.valid = "CNY", "¥", c.cnyPerUSD, c.cnyValid
	case operation_setting.QuotaDisplayTypeCustom:
		c.currency, c.symbol = "CUSTOM", operation_setting.GetCurrencySymbol()
		c.perUSD, c.valid = invitationPositiveRate(operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate))
	}
	return c
}
func (c *invitationDisplayCurrency) convert(amount decimal.Decimal, currency, symbol, sourceRate string) (decimal.Decimal, bool, bool) {
	if amount.IsZero() {
		return decimal.Zero, false, true
	}
	var usd decimal.Decimal
	converted := currency != c.currency
	if currency == "USD" {
		usd = amount
	} else if sourceRate != "" {
		// Redemption's frozen rate recovers its original USD basis, even if the
		// display currency or its exchange rate has subsequently changed.
		rate, err := invitationDecimal(sourceRate)
		if err != nil || rate.IsZero() {
			return decimal.Zero, false, false
		}
		usd = amount.DivRound(rate, 30)
		converted = converted || !rate.Equal(c.perUSD) || (currency == "CUSTOM" && symbol != c.symbol)
	} else if currency == c.currency && currency != "CUSTOM" {
		return amount, false, true
	} else if currency == "CNY" && c.cnyValid {
		usd = amount.DivRound(c.cnyPerUSD, 30)
	} else {
		return decimal.Zero, false, false
	}
	if !c.valid {
		return decimal.Zero, false, false
	}
	return usd.Mul(c.perUSD), converted, true
}
