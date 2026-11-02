package model

import (
	"errors"
	"math"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type redemptionInviteReward struct {
	inviterID int
	quota     int
	ratio     float64
}

// The redemption's locked, unused -> used transition is the idempotency boundary.
// Credit and fulfillment share one transaction: a retry cannot lose or duplicate
// a reward. Quota cards use face quota; subscription cards use the granted snapshot.
func grantRedemptionInviteRewardTx(tx *gorm.DB, inviteeID int, grantedQuota int64) (*redemptionInviteReward, error) {
	common.OptionMapRWMutex.RLock()
	mode, ratio := common.NormalizeInviteRewardMode(common.InviteRewardMode), common.InviteRewardRatio
	common.OptionMapRWMutex.RUnlock()
	if mode != common.InviteRewardModeRatio || grantedQuota <= 0 {
		return nil, nil
	}
	var invitee User
	if err := tx.Select("id", "inviter_id").First(&invitee, inviteeID).Error; err != nil {
		return nil, err
	}
	if invitee.InviterId <= 0 || invitee.InviterId == inviteeID {
		return nil, nil
	}
	var inviter User
	err := tx.Select("id", "status", "invite_reward_ratio").First(&inviter, invitee.InviterId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if inviter.Status != common.UserStatusEnabled {
		return nil, nil
	}
	if inviter.InviteRewardRatio != nil {
		ratio = *inviter.InviteRewardRatio
	}
	if math.IsNaN(ratio) || math.IsInf(ratio, 0) || ratio < 0 || ratio > 1 {
		return nil, errors.New("invalid invitation reward ratio")
	}
	if ratio == 0 {
		return nil, nil
	}
	reward, err := common.TruncateWalletQuotaStrict(decimal.NewFromInt(grantedQuota).Mul(decimal.NewFromFloat(ratio)))
	if err != nil {
		return nil, err
	}
	if reward <= 0 {
		return nil, nil
	}
	max := int(^uint(0) >> 1)
	update := tx.Model(&User{}).Where("id = ? AND status = ? AND aff_quota <= ? AND aff_history <= ?", inviter.Id, common.UserStatusEnabled, max-reward, max-reward).Updates(map[string]interface{}{
		"aff_quota":   gorm.Expr("aff_quota + ?", reward),
		"aff_history": gorm.Expr("aff_history + ?", reward),
	})
	if update.Error != nil {
		return nil, update.Error
	}
	if update.RowsAffected != 1 {
		return nil, errors.New("invitation reward could not be credited safely")
	}
	return &redemptionInviteReward{inviterID: inviter.Id, quota: reward, ratio: ratio}, nil
}
