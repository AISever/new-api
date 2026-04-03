package model

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestSubscriptionPlanEffectiveGroupsTagAvoidsTextDefault(t *testing.T) {
	field, ok := reflect.TypeOf(SubscriptionPlan{}).FieldByName("EffectiveGroups")
	if !ok {
		t.Fatal("EffectiveGroups field not found")
	}
	tag := field.Tag.Get("gorm")
	if !strings.Contains(tag, "type:text") {
		t.Fatalf("EffectiveGroups gorm tag should keep text storage, got %q", tag)
	}
	if strings.Contains(tag, "default:") {
		t.Fatalf("EffectiveGroups gorm tag must not define a default for text columns, got %q", tag)
	}
}

func TestSubscriptionPlanSQLiteMigrationAvoidsTextDefault(t *testing.T) {
	content, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, "effective_groups") && strings.Contains(line, "DEFAULT ''") {
			t.Fatalf("effective_groups sqlite migration must not define TEXT DEFAULT '': %s", strings.TrimSpace(line))
		}
	}
}
