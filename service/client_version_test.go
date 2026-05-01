package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestIsClientVersionSupported(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    bool
	}{
		{name: "empty allowed for backward compatibility", version: "", want: true},
		{name: "minimum supported", version: dto.ClientMinVersion, want: true},
		{name: "newer supported", version: "3.14.0", want: true},
		{name: "older unsupported", version: "3.12.9", want: false},
		{name: "invalid unsupported", version: "dev", want: false},
		{name: "prefixed supported", version: "v3.13.1", want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsClientVersionSupported(tc.version); got != tc.want {
				t.Fatalf("IsClientVersionSupported(%q) = %v, want %v", tc.version, got, tc.want)
			}
		})
	}
}
