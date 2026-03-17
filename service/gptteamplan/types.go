package gptteamplan

type redeemRequestPayload struct {
	Email  string `json:"email"`
	Code   string `json:"code"`
	TeamID *int   `json:"team_id,omitempty"`
}

type RedeemResult struct {
	Message           string `json:"message,omitempty"`
	TeamName          string `json:"team_name,omitempty"`
	SubscriptionPlan  string `json:"subscription_plan,omitempty"`
	RemainingSeats    int    `json:"remaining_seats"`
	TotalSeats        int    `json:"total_seats"`
	UsedSeats         int    `json:"used_seats"`
	ExpiresAt         string `json:"expires_at,omitempty"`
	TeamExpiresAt     string `json:"team_expires_at,omitempty"`
	HasWarranty       bool   `json:"has_warranty"`
	WarrantyValid     bool   `json:"warranty_valid"`
	WarrantyExpiresAt string `json:"warranty_expires_at,omitempty"`
}

type WarrantyRecord struct {
	Code              string `json:"code,omitempty"`
	Status            string `json:"status,omitempty"`
	UsedAt            string `json:"used_at,omitempty"`
	Email             string `json:"email,omitempty"`
	TeamName          string `json:"team_name,omitempty"`
	TeamStatus        string `json:"team_status,omitempty"`
	HasWarranty       bool   `json:"has_warranty"`
	WarrantyValid     bool   `json:"warranty_valid"`
	WarrantyExpiresAt string `json:"warranty_expires_at,omitempty"`
	UserExpiresAt     string `json:"user_expires_at,omitempty"`
	TeamExpiresAt     string `json:"team_expires_at,omitempty"`
}

type WarrantyResult struct {
	Message           string           `json:"message,omitempty"`
	HasWarranty       bool             `json:"has_warranty"`
	WarrantyValid     bool             `json:"warranty_valid"`
	WarrantyExpiresAt string           `json:"warranty_expires_at,omitempty"`
	CanReuse          bool             `json:"can_reuse"`
	OriginalCode      string           `json:"original_code,omitempty"`
	Records           []WarrantyRecord `json:"records"`
}
