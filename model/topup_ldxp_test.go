package model

import "testing"

func TestResolveAmountValueUsesLdxpProviderPayloadDecimalAmount(t *testing.T) {
	topUp := &TopUp{
		PaymentMethod:   "ldxp",
		Amount:          0,
		Money:           0.1,
		ProviderPayload: `{"topup_meta":{"amount":0.1}}`,
	}

	if got := topUp.ResolveAmountValue(); got != 0.1 {
		t.Fatalf("expected decimal amount value 0.1, got %v", got)
	}
}

func TestResolveAmountValueFallsBackToLegacyIntegerAmount(t *testing.T) {
	topUp := &TopUp{
		PaymentMethod: "ldxp",
		Amount:        5,
		Money:         5,
	}

	if got := topUp.ResolveAmountValue(); got != 5 {
		t.Fatalf("expected legacy integer amount value 5, got %v", got)
	}
}
