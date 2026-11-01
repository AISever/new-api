package model

import (
	"github.com/QuantumNous/new-api/common"
	"strings"
	"testing"
)

func TestInvitationPolicyDefaultsSelectedAndRevocation(t *testing.T) {
	db := setupInviteUserTestDB(t)
	if err := db.AutoMigrate(&Option{}); err != nil {
		t.Fatal(err)
	}
	if CanViewInvitationAmounts(1) {
		t.Fatal("missing policy disclosed amounts")
	}
	for _, u := range []User{{Id: 1, Username: "owner", AffCode: "p1"}, {Id: 2, Username: "other", AffCode: "p2"}} {
		if err := db.Create(&u).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{`{"mode":"selected","user_ids":[1,1]}`, `{"mode":"all","user_ids":[]}`, `{"mode":"off","user_ids":[99999]}`} {
		if err := UpdateOption(InvitationVisibilityOption, raw); err != nil {
			t.Fatal(err)
		}
		want := !strings.Contains(raw, `"off"`)
		if CanViewInvitationAmounts(1) != want {
			t.Fatalf("wrong permission for %s", raw)
		}
		if strings.Contains(raw, "selected") && CanViewInvitationAmounts(2) {
			t.Fatal("unselected user granted")
		}
	}
	for _, raw := range []string{`{"mode":"bogus"}`, `{"mode":"selected","user_ids":[0]}`, `{"mode":"selected","user_ids":[2.2]}`, `{"mode":"selected","user_ids":[99999]}`} {
		if err := UpdateOption(InvitationVisibilityOption, raw); err == nil {
			t.Fatalf("accepted malformed policy %s", raw)
		}
	}
	if err := db.Model(&Option{}).Where(&Option{Key: InvitationVisibilityOption}).Update("value", "invalid").Error; err != nil {
		t.Fatal(err)
	}
	if CanViewInvitationAmounts(1) {
		t.Fatal("invalid stored policy disclosed amounts")
	}
}

func TestInvitationPageOwnUsersLiteralSearchAndMinimalFields(t *testing.T) {
	db := setupInviteUserTestDB(t)
	users := []User{{Id: 1, Username: "owner", AffCode: "i1"}, {Id: 2, Username: "alice%_", InviterId: 1, AffCode: "i2", Email: "private@example.test", AccessToken: nil}, {Id: 3, Username: "alice", InviterId: 1, AffCode: "i3"}, {Id: 4, Username: "stranger", InviterId: 8, AffCode: "i4"}}
	for _, u := range users {
		if err := db.Create(&u).Error; err != nil {
			t.Fatal(err)
		}
	}
	page, err := GetInvitationPage(1, 1, 20, "%_", false)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Username != "alice%_" {
		t.Fatalf("escaped search/ownership: %+v", page)
	}
	b, err := common.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{`"amounts":`, "email", "access_token", "inviter_id", "private@example"} {
		if strings.Contains(string(b), s) {
			t.Fatalf("leaked %s: %s", s, b)
		}
	}
	if _, err = GetInvitationPage(1, -1, 20, "", false); err == nil {
		t.Fatal("negative page accepted")
	}
	if _, err = GetInvitationPage(1, 1, 101, "", false); err == nil {
		t.Fatal("unbounded page accepted")
	}
	if err = db.Delete(&User{}, 3).Error; err != nil {
		t.Fatal(err)
	}
	page, err = GetInvitationPage(1, 1, 20, "", false)
	if err != nil || page.Total != 1 {
		t.Fatalf("deleted users not excluded: %+v %v", page, err)
	}
}
