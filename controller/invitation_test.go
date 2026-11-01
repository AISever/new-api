package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"net/http"
	"strings"
	"testing"
)

func TestInvitationEndpointOnlyAuthenticatedOwnerAndNoMoneyByDefault(t *testing.T) {
	db := setupUserControllerTestDB(t)
	if err := db.AutoMigrate(&model.Option{}); err != nil {
		t.Fatal(err)
	}
	for _, u := range []model.User{{Id: 901, Username: "owner901", AffCode: "c901"}, {Id: 902, Username: "friend902", InviterId: 901, AffCode: "c902"}, {Id: 903, Username: "foreign903", InviterId: 999, AffCode: "c903"}} {
		seedUserForUpdateTest(t, db, &u)
	}
	c, w := newAuthenticatedContext(t, http.MethodGet, "/api/user/self/invitees?inviter_id=999", nil, 901)
	GetInvitationUsers(c)
	body := w.Body.String()
	if w.Code != 200 || !strings.Contains(body, "friend902") || strings.Contains(body, "foreign903") || strings.Contains(body, `"amounts"`) {
		t.Fatalf("scope/DTO failed: %s", body)
	}
	if w.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("sensitive response cacheable")
	}
	c, w = newAuthenticatedContext(t, http.MethodGet, "/api/user/self/invitees", nil, 0)
	GetInvitationUsers(c)
	if w.Code != 401 {
		t.Fatalf("anonymous response %d", w.Code)
	}
	c, w = newAuthenticatedContext(t, http.MethodGet, "/api/option/invitation-users", nil, 901)
	GetInvitationUserChoices(c)
	if w.Code != 403 {
		t.Fatal("ordinary user can search site users")
	}
	_ = common.RoleRootUser
}
