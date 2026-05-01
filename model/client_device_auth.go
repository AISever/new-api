package model

type ClientDeviceAuth struct {
	Id             int    `json:"id"`
	DeviceCode     string `json:"device_code" gorm:"type:char(64);uniqueIndex"`
	UserCode       string `json:"user_code" gorm:"type:varchar(16);uniqueIndex"`
	UserId         int    `json:"user_id" gorm:"index"`
	Status         string `json:"status" gorm:"type:varchar(16);index"`
	CreatedTime    int64  `json:"created_time" gorm:"bigint"`
	ExpiresTime    int64  `json:"expires_time" gorm:"bigint;index"`
	AuthorizedTime int64  `json:"authorized_time" gorm:"bigint;default:0"`
	PolledTime     int64  `json:"polled_time" gorm:"bigint;default:0"`
}
