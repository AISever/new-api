package common

import (
	"github.com/shopspring/decimal"
	"math"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSecurityReservedAddressesRemainBlocked(t *testing.T) {
	for _, address := range []string{"0.0.0.0", "0.1.2.3", "127.0.0.1", "100.64.0.1", "169.254.169.254", "198.18.0.1", "::", "::1", "::ffff:127.0.0.1", "fe80::1"} {
		t.Run(address, func(t *testing.T) {
			require.True(t, isPrivateIP(net.ParseIP(address)))
		})
	}
	require.False(t, isPrivateIP(net.ParseIP("8.8.8.8")))
	require.Error(t, ValidateURLWithFetchSetting("http://0.0.0.0/probe", true, false, false, false, nil, nil, []string{"80"}, true))
}

func TestQuotaAuditSupportsNonFiniteInputs(t *testing.T) {
	for _, value := range []float64{math.Inf(1), math.Inf(-1), math.NaN()} {
		_, clamp := QuotaFromFloatChecked(value)
		require.NotNil(t, clamp)
		_, err := Marshal(clamp.AuditMap())
		require.NoError(t, err, "security log serialization must survive non-finite arithmetic")
	}
}

func TestWalletQuotaRetainsBigintAndTruncation(t *testing.T) {
	for _, text := range []string{"5000000000.9", "-5000000000.9", "0.99", "-0.99"} {
		value := decimal.RequireFromString(text)
		got, err := TruncateWalletQuotaStrict(value)
		require.NoError(t, err)
		require.EqualValues(t, value.IntPart(), got)
	}
	for _, text := range []string{"9223372036854775808", "-9223372036854775809"} {
		_, err := TruncateWalletQuotaStrict(decimal.RequireFromString(text))
		require.Error(t, err)
	}
}
