package controller

import (
	"testing"
	"time"
)

func TestResolveLdxpTopupContact(t *testing.T) {
	now := time.Unix(1774851710, 0)
	tests := []struct {
		name           string
		username       string
		requestContact string
		expected       string
	}{
		{
			name:           "fallback to generated username contact",
			username:       "demo_user",
			requestContact: "",
			expected:       "demo_user-1774851710",
		},
		{
			name:           "prefer explicit contact",
			username:       "demo_user",
			requestContact: " wechat-001 ",
			expected:       "wechat-001",
		},
		{
			name:           "fallback to generic user when username missing",
			username:       " ",
			requestContact: " ",
			expected:       "user-1774851710",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveLdxpTopupContact(tt.username, tt.requestContact, now)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
