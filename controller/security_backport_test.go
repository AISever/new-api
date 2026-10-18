package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func securityControllerFixture(t *testing.T, targetRole int) {
	t.Helper()
	oldDB, oldLog, oldRedis := model.DB, model.LOG_DB, common.RedisEnabled
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "security.db")), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB, common.RedisEnabled = db, db, false
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.PasskeyCredential{}, &model.TwoFA{}, &model.Log{}))
	require.NoError(t, db.Create(&model.User{Id: 1, Username: "security-target", Password: "test", Role: targetRole, Status: common.UserStatusEnabled}).Error)
	require.NoError(t, db.Create(&model.PasskeyCredential{UserID: 1, CredentialID: "test-credential", PublicKey: "test-key"}).Error)
	t.Cleanup(func() {
		model.DB, model.LOG_DB, common.RedisEnabled = oldDB, oldLog, oldRedis
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
}

func TestAdminResetPasskeyEnforcesRoleBoundary(t *testing.T) {
	for _, tc := range []struct {
		actor, target int
		allowed       bool
	}{
		{common.RoleAdminUser, common.RoleRootUser, false},
		{common.RoleAdminUser, common.RoleAdminUser, false},
		{common.RoleCommonUser, common.RoleCommonUser, false},
		{200, common.RoleRootUser, false},
		{common.RoleRootUser, common.RoleAdminUser, true},
		{common.RoleAdminUser, common.RoleCommonUser, true},
	} {
		t.Run(fmt.Sprintf("%d-to-%d", tc.actor, tc.target), func(t *testing.T) {
			securityControllerFixture(t, tc.target)
			r := gin.New()
			r.DELETE("/users/:id/passkey", func(c *gin.Context) { c.Set("role", tc.actor) }, AdminResetPasskey)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/users/1/passkey", nil))
			var got struct {
				Success bool `json:"success"`
			}
			require.NoError(t, common.Unmarshal(w.Body.Bytes(), &got))
			require.Equal(t, tc.allowed, got.Success)
			_, err := model.GetPasskeyByUserID(1)
			if tc.allowed {
				require.ErrorIs(t, err, model.ErrPasskeyNotFound)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPasskeyVerificationRequiresFreshSingleUseAssertionMarker(t *testing.T) {
	securityControllerFixture(t, common.RoleRootUser)
	for _, tc := range []struct {
		name            string
		offset          int64
		marker, allowed bool
	}{
		{"registered-without-assertion", 0, false, false},
		{"expired", -61, true, false},
		{"future", 60, true, false},
		{"completed-assertion", 0, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(sessions.Sessions("security-test", cookie.NewStore([]byte("security-test-session-key-32bytes"))))
			r.Use(func(c *gin.Context) { c.Set("id", 1) })
			r.GET("/seed", func(c *gin.Context) {
				s := sessions.Default(c)
				if tc.marker {
					s.Set(PasskeyReadySessionKey, time.Now().Unix()+tc.offset)
				}
				require.NoError(t, s.Save())
				c.Status(204)
			})
			r.POST("/verify", UniversalVerify)
			seed := httptest.NewRecorder()
			r.ServeHTTP(seed, httptest.NewRequest(http.MethodGet, "/seed", nil))
			request := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewBufferString(`{"method":"passkey"}`))
			request.Header.Set("Content-Type", "application/json")
			for _, cookie := range seed.Result().Cookies() {
				request.AddCookie(cookie)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)
			var got struct {
				Success bool `json:"success"`
			}
			require.NoError(t, common.Unmarshal(w.Body.Bytes(), &got))
			require.Equal(t, tc.allowed, got.Success)
			if tc.allowed {
				replay := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewBufferString(`{"method":"passkey"}`))
				replay.Header.Set("Content-Type", "application/json")
				for _, cookie := range w.Result().Cookies() {
					replay.AddCookie(cookie)
				}
				rw := httptest.NewRecorder()
				r.ServeHTTP(rw, replay)
				require.NoError(t, common.Unmarshal(rw.Body.Bytes(), &got))
				require.False(t, got.Success)
			}
		})
	}
}

func TestPrivateBillingChargeRejectsExtremeQuantity(t *testing.T) {
	for _, quantity := range []int{2147483647, int(^uint(0) >> 1)} {
		price := &types.PriceData{Quota: 5000}
		require.Error(t, applyBillingChargeQuantity(price, quantity))
		require.Equal(t, 5000, price.Quota)
	}
	price := &types.PriceData{Quota: 5000}
	require.NoError(t, applyBillingChargeQuantity(price, 3))
	require.Equal(t, 15000, price.Quota)
}

func TestRelayOverflowInputsReturnBadRequestBeforeBilling(t *testing.T) {
	for _, tc := range []struct{ path, body string }{
		{"/v1/images/generations", `{"model":"security-image","prompt":"test","n":18446744073686646784}`},
		{"/v1/chat/completions", `{"model":"security-chat","messages":[{"role":"user","content":"test"}],"max_tokens":2147483647}`},
	} {
		t.Run(tc.path, func(t *testing.T) {
			r := gin.New()
			r.POST(tc.path, func(c *gin.Context) { Relay(c, types.RelayFormatOpenAI) })
			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		})
	}
}
