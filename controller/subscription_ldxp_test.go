package controller

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
)

func TestBuildSubscriptionPlanPayloadIncludesLdxpGoodsKey(t *testing.T) {
	payload := buildSubscriptionPlanPayload(model.SubscriptionPlan{
		Id:           1,
		Title:        "LDXP Plan",
		LdxpGoodsKey: " goods-001 ",
	})

	if payload.LdxpGoodsKey != "goods-001" {
		t.Fatalf("expected ldxp_goods_key to be trimmed, got %q", payload.LdxpGoodsKey)
	}
	if !payload.LdxpAvailable {
		t.Fatal("expected ldxp_available to be true when ldxp_goods_key is configured")
	}
}

func TestBuildSubscriptionPlanModelTrimsLdxpGoodsKey(t *testing.T) {
	plan, ok := buildSubscriptionPlanModel(SubscriptionPlanPayload{
		Title:        "LDXP Plan",
		LdxpGoodsKey: " goods-002 ",
	})
	if !ok {
		t.Fatal("expected payload to be valid")
	}

	if plan.LdxpGoodsKey != "goods-002" {
		t.Fatalf("expected trimmed ldxp_goods_key, got %q", plan.LdxpGoodsKey)
	}
}

func TestBuildPublicSubscriptionPlanPayloadHidesLdxpGoodsKey(t *testing.T) {
	payload := buildPublicSubscriptionPlanPayload(model.SubscriptionPlan{
		Id:           1,
		Title:        "LDXP Plan",
		LdxpGoodsKey: "goods-003",
	}, true)

	if payload.LdxpGoodsKey != "" {
		t.Fatalf("expected public payload to hide ldxp_goods_key, got %q", payload.LdxpGoodsKey)
	}
	if !payload.LdxpAvailable {
		t.Fatal("expected public payload to keep ldxp_available when ldxp is ready")
	}
}

func TestResolveLdxpSubscriptionContact(t *testing.T) {
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
			got := resolveLdxpSubscriptionContact(tt.username, tt.requestContact, now)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
