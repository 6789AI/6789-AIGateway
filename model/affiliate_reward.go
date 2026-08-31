package model

import (
	"errors"

	"gorm.io/gorm"
)

const (
	AffiliateRewardSourcePayment    = "payment"
	AffiliateRewardSourceRedemption = "redemption"
	AffiliateRewardDivisor          = 10 // 10% reward, stored as whole quota units
)

// AffiliateReward records a single referral reward. The source key is unique
// so a successful payment callback or redemption retry cannot issue the same
// reward twice.
type AffiliateReward struct {
	Id         int    `json:"id"`
	InviterId  int    `json:"inviter_id" gorm:"not null;index"`
	InviteeId  int    `json:"invitee_id" gorm:"not null;index"`
	SourceType string `json:"source_type" gorm:"type:varchar(32);not null;uniqueIndex:idx_affiliate_reward_source,priority:1"`
	SourceId   string `json:"source_id" gorm:"type:varchar(255);not null;uniqueIndex:idx_affiliate_reward_source,priority:2"`
	Quota      int    `json:"quota" gorm:"not null"`
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime"`
}

// grantAffiliateRewardTx credits 10% of the quota just added to an invitee.
// Rewards are kept in AffQuota until the inviter transfers them to normal
// quota, matching the existing referral-reward workflow.
func grantAffiliateRewardTx(tx *gorm.DB, inviteeId int, sourceType string, sourceId string, creditedQuota int) (int, error) {
	if tx == nil {
		return 0, errors.New("nil transaction")
	}
	if inviteeId <= 0 || sourceType == "" || sourceId == "" || creditedQuota <= 0 {
		return 0, nil
	}

	var invitee User
	if err := tx.Select("id", "inviter_id").Where("id = ?", inviteeId).First(&invitee).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	if invitee.InviterId <= 0 || invitee.InviterId == inviteeId {
		return 0, nil
	}

	// The payment/redemption row is locked by its caller before this helper is
	// invoked. Checking the unique source inside the same transaction keeps the
	// operation idempotent while allowing retries of already completed orders.
	var existing AffiliateReward
	err := tx.Where("source_type = ? AND source_id = ?", sourceType, sourceId).First(&existing).Error
	if err == nil {
		return 0, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	rewardQuota := creditedQuota / AffiliateRewardDivisor
	// A positive recharge always earns a reward. The stored quota is an
	// integer, so retain the smallest unit when 10% truncates below one.
	if rewardQuota == 0 {
		rewardQuota = 1
	}

	result := tx.Model(&User{}).Where("id = ?", invitee.InviterId).Updates(map[string]interface{}{
		"aff_quota":   gorm.Expr("aff_quota + ?", rewardQuota),
		"aff_history": gorm.Expr("aff_history + ?", rewardQuota),
	})
	if result.Error != nil {
		return 0, result.Error
	}
	if result.RowsAffected == 0 {
		// A deleted inviter should not make a valid recharge fail.
		return 0, nil
	}

	reward := &AffiliateReward{
		InviterId:  invitee.InviterId,
		InviteeId:  inviteeId,
		SourceType: sourceType,
		SourceId:   sourceId,
		Quota:      rewardQuota,
	}
	if err := tx.Create(reward).Error; err != nil {
		return 0, err
	}
	return rewardQuota, nil
}
