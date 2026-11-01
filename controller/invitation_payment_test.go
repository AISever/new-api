package controller

import (
	"github.com/stretchr/testify/require"
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
