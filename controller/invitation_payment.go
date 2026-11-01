package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"strconv"
)

func normalizeInvitationPayment(amount, currency string, minor bool) (string, string, error) {
	return model.NormalizeInvitationPayment(amount, currency, minor)
}

// Reporting must never change a successful financial transaction's outcome.
func recordInvitationPayment(tradeNo, provider, raw, currency string, minor bool) {
	amount, currency, err := normalizeInvitationPayment(raw, currency, minor)
	if err == nil {
		err = model.RecordInvitationPayment(tradeNo, provider, amount, currency)
	}
	if err != nil {
		common.SysError("invitation amount snapshot unavailable: " + err.Error())
	}
}
func invitationOptionalMinor(amount *int64) string {
	if amount == nil {
		return ""
	}
	return strconv.FormatInt(*amount, 10)
}
