package dto

const (
	ClientProtocolVersion         = 1
	ClientMinVersion              = "3.13.0"
	ClientRecommendedVersion      = "3.13.0"
	ClientErrorAuthRequired       = "AUTH_REQUIRED"
	ClientErrorAuthExpired        = "AUTH_EXPIRED"
	ClientErrorAuthRevoked        = "AUTH_REVOKED"
	ClientErrorGroupNotFound      = "GROUP_NOT_FOUND"
	ClientErrorGroupNotAllowed    = "GROUP_NOT_ALLOWED"
	ClientErrorTokenRevoked       = "TOKEN_REVOKED"
	ClientErrorTokenLimitExceeded = "TOKEN_LIMIT_EXCEEDED"
	ClientErrorTokenEnsureFailed  = "TOKEN_ENSURE_FAILED"
	ClientErrorQuotaExceeded      = "QUOTA_EXCEEDED"
	ClientErrorModelNotAllowed    = "MODEL_NOT_ALLOWED"
	ClientErrorAppNotSupported    = "APP_NOT_SUPPORTED"
	ClientErrorVersionUnsupported = "CLIENT_VERSION_UNSUPPORTED"
	ClientErrorSiteMaintenance    = "SITE_MAINTENANCE"
	ClientErrorRateLimited        = "RATE_LIMITED"
	ClientErrorInternal           = "INTERNAL_ERROR"
	ClientErrorInvalidRequest     = "INVALID_REQUEST"
)

var ClientSupportedApps = []string{"claude", "codex", "gemini", "opencode", "openclaw"}

var ClientDefaultModelsByApp = map[string]string{
	"claude":   "claude-sonnet-4-6",
	"codex":    "gpt-5.4",
	"gemini":   "gemini-3-flash",
	"opencode": "gpt-5.4",
	"openclaw": "gpt-5.4",
}

var ClientPreferredGroupByApp = map[string]string{
	"claude":   "claudecode-03",
	"codex":    "codex-01",
	"gemini":   "gemini-01",
	"opencode": "codex-01",
	"openclaw": "codex-01",
}

type ClientErrorResponse struct {
	Success   bool   `json:"success"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}

type ClientBootstrapResponse struct {
	ProtocolVersion          int                `json:"protocolVersion"`
	MinClientVersion         string             `json:"minClientVersion"`
	RecommendedClientVersion string             `json:"recommendedClientVersion"`
	BrandName                string             `json:"brandName"`
	HomepageURL              string             `json:"homepageUrl"`
	ConsoleURL               string             `json:"consoleUrl"`
	GatewayBaseURL           string             `json:"gatewayBaseUrl"`
	TokenURL                 string             `json:"tokenUrl"`
	SupportedApps            []string           `json:"supportedApps"`
	DefaultModelsByApp       map[string]string  `json:"defaultModelsByApp"`
	PreferredGroupByApp      map[string]string  `json:"preferredGroupByApp"`
	Capabilities             ClientCapabilities `json:"capabilities"`
}

type ClientCapabilities struct {
	DeviceLogin       bool `json:"deviceLogin"`
	TokenEnsure       bool `json:"tokenEnsure"`
	Usage             bool `json:"usage"`
	ModelCatalog      bool `json:"modelCatalog"`
	GroupCapabilities bool `json:"groupCapabilities"`
}

type ClientProfileResponse struct {
	ID           int     `json:"id"`
	Username     string  `json:"username"`
	DisplayName  string  `json:"displayName"`
	Group        string  `json:"group"`
	Quota        float64 `json:"quota"`
	RequestCount int     `json:"requestCount"`
	Role         int     `json:"role"`
	Status       int     `json:"status"`
}

type ClientGroupsResponse struct {
	Groups []ClientGroup `json:"groups"`
}

type ClientGroup struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	Ratio              float64           `json:"ratio"`
	SupportedApps      []string          `json:"supportedApps"`
	DefaultModelsByApp map[string]string `json:"defaultModelsByApp"`
	QuotaStatus        string            `json:"quotaStatus"`
}

type ClientModelsResponse struct {
	Models []ClientModel `json:"models"`
}

type ClientModel struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Protocol      string  `json:"protocol"`
	Recommended   bool    `json:"recommended"`
	Enabled       bool    `json:"enabled"`
	ContextWindow int     `json:"contextWindow"`
	InputPrice    float64 `json:"inputPrice"`
	OutputPrice   float64 `json:"outputPrice"`
}

type ClientTokenEnsureRequest struct {
	App      string `json:"app"`
	Group    string `json:"group"`
	DeviceID string `json:"deviceId"`
	Name     string `json:"name"`
}

type ClientTokenEnsureResponse struct {
	TokenID   int    `json:"tokenId"`
	TokenName string `json:"tokenName"`
	Group     string `json:"group"`
	Key       string `json:"key"`
	Created   bool   `json:"created"`
	ExpiresAt *int64 `json:"expiresAt"`
	Status    string `json:"status"`
}

type ClientTokenStatusRequest struct {
	App      string `json:"app"`
	Group    string `json:"group"`
	DeviceID string `json:"deviceId"`
	Name     string `json:"name"`
}

type ClientTokenStatusResponse struct {
	TokenID     int    `json:"tokenId"`
	Status      string `json:"status"`
	Group       string `json:"group"`
	QuotaStatus string `json:"quotaStatus"`
}

type ClientUsageSummaryResponse struct {
	TodayRequests  int     `json:"todayRequests"`
	TodayTokens    int     `json:"todayTokens"`
	MonthRequests  int     `json:"monthRequests"`
	MonthTokens    int     `json:"monthTokens"`
	QuotaStatus    string  `json:"quotaStatus"`
	RemainingQuota float64 `json:"remainingQuota"`
	Currency       string  `json:"currency"`
}

type ClientHealthResponse struct {
	Status         string `json:"status"`
	GatewayStatus  string `json:"gatewayStatus"`
	DatabaseStatus string `json:"databaseStatus"`
	Message        string `json:"message"`
}

type ClientDeviceStartResponse struct {
	DeviceCode      string `json:"deviceCode"`
	UserCode        string `json:"userCode"`
	VerificationURL string `json:"verificationUrl"`
	ExpiresIn       int    `json:"expiresIn"`
	Interval        int    `json:"interval"`
}

type ClientDevicePollRequest struct {
	DeviceCode string `json:"deviceCode"`
}

type ClientDevicePollResponse struct {
	Status       string `json:"status,omitempty"`
	UserID       int    `json:"userId,omitempty"`
	Username     string `json:"username,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresAt    string `json:"expiresAt,omitempty"`
}

type ClientDeviceAuthorizeRequest struct {
	UserCode string `json:"userCode"`
}
