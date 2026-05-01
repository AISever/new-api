package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

func ClientError(c *gin.Context, httpStatus int, code string, message string) {
	c.JSON(httpStatus, dto.ClientErrorResponse{
		Success:   false,
		Code:      code,
		Message:   message,
		RequestID: c.GetHeader("X-Request-Id"),
	})
}

func ClientBootstrap(c *gin.Context) {
	baseURL := strings.TrimRight(system_setting.ServerAddress, "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(getRequestBaseURL(c), "/")
	}
	common.ApiSuccess(c, dto.ClientBootstrapResponse{
		ProtocolVersion:          dto.ClientProtocolVersion,
		MinClientVersion:         dto.ClientMinVersion,
		RecommendedClientVersion: dto.ClientRecommendedVersion,
		BrandName:                common.SystemName,
		HomepageURL:              baseURL,
		ConsoleURL:               baseURL + "/console",
		GatewayBaseURL:           baseURL,
		TokenURL:                 baseURL + "/console/token",
		SupportedApps:            append([]string(nil), dto.ClientSupportedApps...),
		DefaultModelsByApp:       copyStringMap(dto.ClientDefaultModelsByApp),
		PreferredGroupByApp:      copyStringMap(dto.ClientPreferredGroupByApp),
		Capabilities: dto.ClientCapabilities{
			DeviceLogin:       true,
			TokenEnsure:       true,
			Usage:             true,
			ModelCatalog:      true,
			GroupCapabilities: true,
		},
	})
}

func ClientDeviceStart(c *gin.Context) {
	baseURL := strings.TrimRight(system_setting.ServerAddress, "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(getRequestBaseURL(c), "/")
	}
	response, err := service.StartClientDeviceAuth(baseURL)
	if err != nil {
		ClientError(c, http.StatusOK, dto.ClientErrorInternal, err.Error())
		return
	}
	common.ApiSuccess(c, response)
}

func ClientDevicePoll(c *gin.Context) {
	var req dto.ClientDevicePollRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		ClientError(c, http.StatusOK, dto.ClientErrorInvalidRequest, err.Error())
		return
	}
	response, err := service.PollClientDeviceAuth(req.DeviceCode)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrClientDevicePending):
			common.ApiSuccess(c, dto.ClientDevicePollResponse{Status: "pending"})
		case errors.Is(err, service.ErrClientDeviceExpired):
			ClientError(c, http.StatusOK, dto.ClientErrorAuthExpired, err.Error())
		case errors.Is(err, service.ErrClientDeviceInvalid):
			ClientError(c, http.StatusOK, dto.ClientErrorInvalidRequest, err.Error())
		default:
			ClientError(c, http.StatusInternalServerError, dto.ClientErrorInternal, err.Error())
		}
		return
	}
	common.ApiSuccess(c, response)
}

func ClientDeviceAuthorize(c *gin.Context) {
	var req dto.ClientDeviceAuthorizeRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		ClientError(c, http.StatusOK, dto.ClientErrorInvalidRequest, err.Error())
		return
	}
	if err := service.AuthorizeClientDeviceAuth(req.UserCode, c.GetInt("id")); err != nil {
		switch {
		case errors.Is(err, service.ErrClientDeviceExpired):
			ClientError(c, http.StatusOK, dto.ClientErrorAuthExpired, err.Error())
		case errors.Is(err, service.ErrClientDeviceInvalid):
			ClientError(c, http.StatusOK, dto.ClientErrorInvalidRequest, err.Error())
		default:
			ClientError(c, http.StatusInternalServerError, dto.ClientErrorInternal, err.Error())
		}
		return
	}
	common.ApiSuccess(c, gin.H{"status": "authorized"})
}

func ClientHealth(c *gin.Context) {
	databaseStatus := "ok"
	message := "ok"
	if sqlDB, err := model.DB.DB(); err != nil {
		databaseStatus = "failed"
		message = err.Error()
	} else if err := sqlDB.Ping(); err != nil {
		databaseStatus = "failed"
		message = err.Error()
	}
	status := "ok"
	if databaseStatus != "ok" {
		status = "degraded"
	}
	common.ApiSuccess(c, dto.ClientHealthResponse{
		Status:         status,
		GatewayStatus:  "ok",
		DatabaseStatus: databaseStatus,
		Message:        message,
	})
}

func ClientGroups(c *gin.Context) {
	groups, err := service.GetClientGroups(c.GetInt("id"))
	if err != nil {
		ClientError(c, http.StatusInternalServerError, dto.ClientErrorInternal, err.Error())
		return
	}
	common.ApiSuccess(c, dto.ClientGroupsResponse{Groups: groups})
}

func ClientProfile(c *gin.Context) {
	profile, err := service.GetClientProfile(c.GetInt("id"))
	if err != nil {
		ClientError(c, http.StatusInternalServerError, dto.ClientErrorInternal, err.Error())
		return
	}
	common.ApiSuccess(c, profile)
}

func ClientModels(c *gin.Context) {
	models, err := service.GetClientModels(c.GetInt("id"), c.Query("app"), c.Query("group"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrClientAppNotSupported):
			ClientError(c, http.StatusOK, dto.ClientErrorAppNotSupported, err.Error())
		case errors.Is(err, service.ErrClientGroupNotAllowed):
			ClientError(c, http.StatusOK, dto.ClientErrorGroupNotAllowed, err.Error())
		default:
			ClientError(c, http.StatusInternalServerError, dto.ClientErrorInternal, err.Error())
		}
		return
	}
	common.ApiSuccess(c, dto.ClientModelsResponse{Models: models})
}

func ClientEnsureToken(c *gin.Context) {
	var req dto.ClientTokenEnsureRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		ClientError(c, http.StatusOK, dto.ClientErrorInvalidRequest, err.Error())
		return
	}
	response, err := service.EnsureClientToken(c.GetInt("id"), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrClientAppNotSupported):
			ClientError(c, http.StatusOK, dto.ClientErrorAppNotSupported, err.Error())
		case errors.Is(err, service.ErrClientGroupNotAllowed):
			ClientError(c, http.StatusOK, dto.ClientErrorGroupNotAllowed, err.Error())
		case errors.Is(err, service.ErrClientTokenLimitExceeded):
			ClientError(c, http.StatusOK, dto.ClientErrorTokenLimitExceeded, err.Error())
		default:
			ClientError(c, http.StatusOK, dto.ClientErrorTokenEnsureFailed, err.Error())
		}
		return
	}
	common.ApiSuccess(c, response)
}

func ClientTokenStatus(c *gin.Context) {
	response, err := service.GetClientTokenStatus(c.GetInt("id"), dto.ClientTokenStatusRequest{
		App:      c.Query("app"),
		Group:    c.Query("group"),
		DeviceID: c.Query("deviceId"),
		Name:     c.Query("name"),
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrClientAppNotSupported):
			ClientError(c, http.StatusOK, dto.ClientErrorAppNotSupported, err.Error())
		case errors.Is(err, service.ErrClientGroupNotAllowed):
			ClientError(c, http.StatusOK, dto.ClientErrorGroupNotAllowed, err.Error())
		case errors.Is(err, service.ErrClientTokenNotFound):
			ClientError(c, http.StatusOK, dto.ClientErrorTokenRevoked, err.Error())
		default:
			ClientError(c, http.StatusOK, dto.ClientErrorInvalidRequest, err.Error())
		}
		return
	}
	common.ApiSuccess(c, response)
}

func ClientUsageSummary(c *gin.Context) {
	response, err := service.GetClientUsageSummary(c.GetInt("id"), c.Query("app"), c.Query("group"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrClientAppNotSupported):
			ClientError(c, http.StatusOK, dto.ClientErrorAppNotSupported, err.Error())
		case errors.Is(err, service.ErrClientGroupNotAllowed):
			ClientError(c, http.StatusOK, dto.ClientErrorGroupNotAllowed, err.Error())
		default:
			ClientError(c, http.StatusInternalServerError, dto.ClientErrorInternal, err.Error())
		}
		return
	}
	common.ApiSuccess(c, response)
}

func getRequestBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host
}

func copyStringMap(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
