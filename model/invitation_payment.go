package model

import (
	"errors"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"strings"
)

// Stripe/Creem integer API amounts use currency minor units. Do not apply this
// conversion to Waffo's already formatted display-amount strings.
func NormalizeInvitationPayment(raw, currency string, minor bool) (string, string, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if !invitationCurrencyPattern.MatchString(currency) {
		return "", "", errors.New("invalid currency")
	}
	d, err := invitationDecimal(raw)
	if err != nil {
		return "", "", err
	}
	if minor {
		if !d.Equal(d.Truncate(0)) {
			return "", "", errors.New("minor amount must be an integer")
		}
		digits := int32(2)
		switch currency {
		case "BIF", "CLP", "DJF", "GNF", "JPY", "KMF", "KRW", "MGA", "PYG", "RWF", "UGX", "VND", "VUV", "XAF", "XOF", "XPF":
			digits = 0
		case "BHD", "JOD", "KWD", "OMR", "TND":
			digits = 3
		}
		d = d.Shift(-digits)
	}
	return d.String(), currency, nil
}
func invitationStoredPayment(payload, provider string) (string, string, error) {
	if payload == "" {
		return "", "", errors.New("no payment evidence")
	}
	switch provider {
	case PaymentProviderStripe:
		var p struct {
			Amount   *dto.StringValue `json:"amount_total"`
			Currency string           `json:"currency"`
		}
		if err := common.UnmarshalJsonStr(payload, &p); err != nil || p.Amount == nil {
			return "", "", errors.New("missing paid amount")
		}
		return NormalizeInvitationPayment(string(*p.Amount), p.Currency, true)
	case PaymentProviderCreem:
		var p struct {
			Object struct {
				Order struct {
					Amount   *dto.StringValue `json:"amount_paid"`
					Currency string           `json:"currency"`
					Status   string           `json:"status"`
				} `json:"order"`
			} `json:"object"`
		}
		if err := common.UnmarshalJsonStr(payload, &p); err != nil || p.Object.Order.Amount == nil || p.Object.Order.Status != "paid" {
			return "", "", errors.New("missing paid amount")
		}
		return NormalizeInvitationPayment(string(*p.Object.Order.Amount), p.Object.Order.Currency, true)
	case PaymentProviderEpay:
		// The stored value is the server's successfully verified VerifyRes, not client JSON.
		var p struct {
			Money        string
			VerifyStatus bool
			TradeStatus  string
		}
		if err := common.UnmarshalJsonStr(payload, &p); err != nil || !p.VerifyStatus || p.TradeStatus != "TRADE_SUCCESS" {
			return "", "", errors.New("missing verified payment")
		}
		return NormalizeInvitationPayment(p.Money, "CNY", false)
	}
	return "", "", errors.New("no verified currency evidence")
}
