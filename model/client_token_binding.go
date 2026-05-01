package model

type ClientTokenBinding struct {
	Id        int    `json:"id"`
	UserID    int    `json:"user_id" gorm:"uniqueIndex:idx_client_token_binding_user_device_app_group"`
	DeviceID  string `json:"device_id" gorm:"type:varchar(191);uniqueIndex:idx_client_token_binding_user_device_app_group"`
	App       string `json:"app" gorm:"type:varchar(64);uniqueIndex:idx_client_token_binding_user_device_app_group"`
	Group     string `json:"group" gorm:"column:group;type:varchar(191);uniqueIndex:idx_client_token_binding_user_device_app_group"`
	TokenID   int    `json:"token_id" gorm:"index"`
	TokenName string `json:"token_name" gorm:"type:varchar(191)"`
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt int64  `json:"updated_at" gorm:"bigint"`
}

func (ClientTokenBinding) TableName() string {
	return "client_token_bindings"
}
