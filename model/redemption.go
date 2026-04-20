package model

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"gorm.io/gorm"
)

const (
	RedemptionTypeQuota        = common.RedemptionTypeQuota
	RedemptionTypeSubscription = common.RedemptionTypeSubscription
)

type Redemption struct {
	Id                     int            `json:"id"`
	UserId                 int            `json:"user_id"`
	Key                    string         `json:"key" gorm:"type:char(32);uniqueIndex"`
	Status                 int            `json:"status" gorm:"default:1"`
	Name                   string         `json:"name" gorm:"index"`
	Quota                  int            `json:"quota" gorm:"default:100"`
	RedeemType             string         `json:"redeem_type" gorm:"type:varchar(32);not null;default:quota"`
	SubscriptionPlanId     int            `json:"subscription_plan_id" gorm:"type:int;default:0;index"`
	SubscriptionPlanTitle  string         `json:"subscription_plan_title" gorm:"-:all"`
	RedeemedSubscriptionId int            `json:"redeemed_subscription_id" gorm:"type:int;default:0;index"`
	CreatedTime            int64          `json:"created_time" gorm:"bigint"`
	RedeemedTime           int64          `json:"redeemed_time" gorm:"bigint"`
	Count                  int            `json:"count" gorm:"-:all"` // only for api request
	UsedUserId             int            `json:"used_user_id"`
	DeletedAt              gorm.DeletedAt `gorm:"index"`
	ExpiredTime            int64          `json:"expired_time" gorm:"bigint"` // 过期时间，0 表示不过期
}

type RedemptionSubscriptionGrant struct {
	SubscriptionId int    `json:"subscription_id"`
	PlanId         int    `json:"plan_id"`
	PlanTitle      string `json:"plan_title"`
	StartTime      int64  `json:"start_time"`
	EndTime        int64  `json:"end_time"`
}

type RedemptionRedeemResult struct {
	RedeemType   string                       `json:"redeem_type"`
	Quota        int                          `json:"quota"`
	Subscription *RedemptionSubscriptionGrant `json:"subscription,omitempty"`
}

func NormalizeRedemptionType(redeemType string) string {
	switch redeemType {
	case "", common.RedemptionTypeQuota:
		return common.RedemptionTypeQuota
	case common.RedemptionTypeSubscription:
		return common.RedemptionTypeSubscription
	default:
		return redeemType
	}
}

func attachSubscriptionPlanTitles(query *gorm.DB, redemptions []*Redemption) error {
	if len(redemptions) == 0 {
		return nil
	}
	if query == nil {
		query = DB
	}
	planIDSet := make(map[int]struct{})
	for _, redemption := range redemptions {
		if redemption == nil {
			continue
		}
		if NormalizeRedemptionType(redemption.RedeemType) != common.RedemptionTypeSubscription {
			continue
		}
		if redemption.SubscriptionPlanId <= 0 {
			continue
		}
		planIDSet[redemption.SubscriptionPlanId] = struct{}{}
	}
	if len(planIDSet) == 0 {
		return nil
	}

	planIDs := make([]int, 0, len(planIDSet))
	for id := range planIDSet {
		planIDs = append(planIDs, id)
	}

	var plans []SubscriptionPlan
	if err := query.Select("id", "title").Where("id IN ?", planIDs).Find(&plans).Error; err != nil {
		return err
	}
	planTitles := make(map[int]string, len(plans))
	for _, plan := range plans {
		planTitles[plan.Id] = plan.Title
	}
	for _, redemption := range redemptions {
		if redemption == nil {
			continue
		}
		redemption.SubscriptionPlanTitle = planTitles[redemption.SubscriptionPlanId]
	}
	return nil
}

func GetAllRedemptions(startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	// 开始事务
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取总数
	err = tx.Model(&Redemption{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 获取分页数据
	err = tx.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}
	if err = attachSubscriptionPlanTitles(tx, redemptions); err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 提交事务
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return redemptions, total, nil
}

func SearchRedemptions(keyword string, startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Build query based on keyword type
	query := tx.Model(&Redemption{})

	// Only try to convert to ID if the string represents a valid integer
	if id, err := strconv.Atoi(keyword); err == nil {
		query = query.Where("id = ? OR name LIKE ?", id, keyword+"%")
	} else {
		query = query.Where("name LIKE ?", keyword+"%")
	}

	// Get total count
	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated data
	err = query.Order("id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}
	if err = attachSubscriptionPlanTitles(tx, redemptions); err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return redemptions, total, nil
}

func GetRedemptionById(id int) (*Redemption, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	var err error = nil
	err = DB.First(&redemption, "id = ?", id).Error
	if err == nil {
		err = attachSubscriptionPlanTitles(DB, []*Redemption{&redemption})
	}
	return &redemption, err
}

func Redeem(key string, userId int) (result RedemptionRedeemResult, err error) {
	if key == "" {
		return result, errors.New("未提供兑换码")
	}
	if userId == 0 {
		return result, errors.New("无效的 user id")
	}
	redemption := &Redemption{}
	result = RedemptionRedeemResult{
		RedeemType: common.RedemptionTypeQuota,
	}

	keyCol := "`key`"
	if common.UsingPostgreSQL {
		keyCol = `"key"`
	}
	common.RandomSleep()
	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(keyCol+" = ?", key).First(redemption).Error
		if err != nil {
			return errors.New("无效的兑换码")
		}
		if redemption.Status != common.RedemptionCodeStatusEnabled {
			return errors.New("该兑换码已被使用")
		}
		if redemption.ExpiredTime != 0 && redemption.ExpiredTime < common.GetTimestamp() {
			return errors.New("该兑换码已过期")
		}
		redemption.RedeemType = NormalizeRedemptionType(redemption.RedeemType)
		result.RedeemType = redemption.RedeemType
		switch redemption.RedeemType {
		case common.RedemptionTypeQuota:
			err = tx.Model(&User{}).Where("id = ?", userId).Update("quota", gorm.Expr("quota + ?", redemption.Quota)).Error
			if err != nil {
				return err
			}
			result.Quota = redemption.Quota
		case common.RedemptionTypeSubscription:
			plan, err := getSubscriptionPlanByIdTx(tx, redemption.SubscriptionPlanId)
			if err != nil {
				return err
			}
			sub, err := CreateUserSubscriptionFromPlanTx(tx, userId, plan, "redemption")
			if err != nil {
				return err
			}
			redemption.RedeemedSubscriptionId = sub.Id
			result.Subscription = &RedemptionSubscriptionGrant{
				SubscriptionId: sub.Id,
				PlanId:         plan.Id,
				PlanTitle:      plan.Title,
				StartTime:      sub.StartTime,
				EndTime:        sub.EndTime,
			}
		default:
			return errors.New("不支持的兑换码类型")
		}
		redemption.RedeemedTime = common.GetTimestamp()
		redemption.Status = common.RedemptionCodeStatusUsed
		redemption.UsedUserId = userId
		err = tx.Save(redemption).Error
		return err
	})
	if err != nil {
		common.SysError("redemption failed: " + err.Error())
		return RedemptionRedeemResult{}, ErrRedeemFailed
	}
	switch result.RedeemType {
	case common.RedemptionTypeSubscription:
		if result.Subscription != nil {
			RecordLog(userId, LogTypeTopup, fmt.Sprintf("通过兑换码兑换订阅套餐 %s，兑换码ID %d，订阅ID %d", result.Subscription.PlanTitle, redemption.Id, result.Subscription.SubscriptionId))
		}
	default:
		RecordLog(userId, LogTypeTopup, fmt.Sprintf("通过兑换码充值 %s，兑换码ID %d", logger.LogQuota(redemption.Quota), redemption.Id))
	}
	return result, nil
}

func (redemption *Redemption) Insert() error {
	var err error
	err = DB.Create(redemption).Error
	return err
}

func (redemption *Redemption) SelectUpdate() error {
	// This can update zero values
	return DB.Model(redemption).Select("redeemed_time", "status").Updates(redemption).Error
}

// Update Make sure your token's fields is completed, because this will update non-zero values
func (redemption *Redemption) Update() error {
	var err error
	err = DB.Model(redemption).Select("name", "status", "quota", "redeem_type", "subscription_plan_id", "redeemed_subscription_id", "redeemed_time", "expired_time").Updates(redemption).Error
	return err
}

func (redemption *Redemption) Delete() error {
	var err error
	err = DB.Delete(redemption).Error
	return err
}

func DeleteRedemptionById(id int) (err error) {
	if id == 0 {
		return errors.New("id 为空！")
	}
	redemption := Redemption{Id: id}
	err = DB.Where(redemption).First(&redemption).Error
	if err != nil {
		return err
	}
	return redemption.Delete()
}

func DeleteInvalidRedemptions() (int64, error) {
	now := common.GetTimestamp()
	result := DB.Where("status IN ? OR (status = ? AND expired_time != 0 AND expired_time < ?)", []int{common.RedemptionCodeStatusUsed, common.RedemptionCodeStatusDisabled}, common.RedemptionCodeStatusEnabled, now).Delete(&Redemption{})
	return result.RowsAffected, result.Error
}
