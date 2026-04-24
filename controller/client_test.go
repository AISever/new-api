package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestClientBootstrapAdvertisesDeviceLoginWhenWebPageExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/client/bootstrap", nil)

	ClientBootstrap(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", recorder.Code)
	}
	var envelope struct {
		Success bool                        `json:"success"`
		Data    dto.ClientBootstrapResponse `json:"data"`
	}
	if err := common.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !envelope.Data.Capabilities.DeviceLogin {
		t.Fatalf("device login should be enabled when /console/client/device is implemented")
	}
}

func TestClientProfileRouteUsesClientAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := model.DB
	oldRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	db, err := gorm.Open(sqlite.Open("file:client_profile_route?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	model.DB = db
	t.Cleanup(func() {
		model.DB = oldDB
		common.RedisEnabled = oldRedisEnabled
	})
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate user: %v", err)
	}
	accessToken := "client-profile-token"
	user := model.User{Username: "client", DisplayName: "Client User", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, Group: "default", Quota: int(5 * common.QuotaPerUnit), RequestCount: 3, AccessToken: &accessToken}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	engine := gin.New()
	clientRoute := engine.Group("/api/client")
	clientRoute.Use(middleware.ClientVersionCheck(), middleware.ClientAuth())
	clientRoute.GET("/profile", ClientProfile)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/client/profile", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", recorder.Code, recorder.Body.String())
	}
	var envelope struct {
		Success bool                      `json:"success"`
		Data    dto.ClientProfileResponse `json:"data"`
	}
	if err := common.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !envelope.Success || envelope.Data.ID != user.Id || envelope.Data.Username != "client" || envelope.Data.DisplayName != "Client User" || envelope.Data.Quota != 5 || envelope.Data.RequestCount != 3 {
		t.Fatalf("unexpected profile response: %+v", envelope)
	}
}
