package model

import (
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRedeemCreditsAffiliateReward(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.AutoMigrate(&Redemption{}))

	inviter := &User{Username: "affiliate-inviter", Password: "password", Status: common.UserStatusEnabled, AffCode: "affiliate-inviter-code"}
	require.NoError(t, DB.Create(inviter).Error)
	invitee := &User{Username: "affiliate-invitee", Password: "password", Status: common.UserStatusEnabled, InviterId: inviter.Id, AffCode: "affiliate-invitee-code"}
	require.NoError(t, DB.Create(invitee).Error)
	redemption := &Redemption{
		Key:         "affiliate-redemption-key",
		Status:      common.RedemptionCodeStatusEnabled,
		Quota:       1_000,
		CreatedTime: time.Now().Unix(),
	}
	require.NoError(t, DB.Create(redemption).Error)

	quota, err := Redeem(redemption.Key, invitee.Id)
	require.NoError(t, err)
	assert.Equal(t, 1_000, quota)

	var updatedInviter User
	require.NoError(t, DB.First(&updatedInviter, inviter.Id).Error)
	assert.Equal(t, 100, updatedInviter.AffQuota)
	assert.Equal(t, 100, updatedInviter.AffHistoryQuota)

	var rewards []AffiliateReward
	require.NoError(t, DB.Where("invitee_id = ?", invitee.Id).Find(&rewards).Error)
	require.Len(t, rewards, 1)
	assert.Equal(t, AffiliateRewardSourceRedemption, rewards[0].SourceType)
	assert.Equal(t, strconv.Itoa(redemption.Id), rewards[0].SourceId)
	assert.Equal(t, 100, rewards[0].Quota)
}

func TestCompleteEpayTopUpAffiliateRewardIsIdempotent(t *testing.T) {
	truncateTables(t)

	originalQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500_000
	t.Cleanup(func() { common.QuotaPerUnit = originalQuotaPerUnit })

	inviter := &User{Username: "epay-affiliate-inviter", Password: "password", Status: common.UserStatusEnabled, AffCode: "epay-inviter-code"}
	require.NoError(t, DB.Create(inviter).Error)
	invitee := &User{Username: "epay-affiliate-invitee", Password: "password", Status: common.UserStatusEnabled, InviterId: inviter.Id, AffCode: "epay-invitee-code"}
	require.NoError(t, DB.Create(invitee).Error)
	topUp := &TopUp{
		UserId:          invitee.Id,
		Amount:          1,
		Money:           1,
		TradeNo:         "epay-affiliate-trade",
		PaymentProvider: PaymentProviderEpay,
		PaymentMethod:   "alipay",
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, DB.Create(topUp).Error)

	completed, quota, err := CompleteEpayTopUp(topUp.TradeNo, "wechat")
	require.NoError(t, err)
	require.NotNil(t, completed)
	assert.Equal(t, 500_000, quota)

	_, quota, err = CompleteEpayTopUp(topUp.TradeNo, "wechat")
	require.NoError(t, err)
	assert.Zero(t, quota)

	var updatedInvitee, updatedInviter User
	require.NoError(t, DB.First(&updatedInvitee, invitee.Id).Error)
	require.NoError(t, DB.First(&updatedInviter, inviter.Id).Error)
	assert.Equal(t, 500_000, updatedInvitee.Quota)
	assert.Equal(t, 50_000, updatedInviter.AffQuota)
	assert.Equal(t, 50_000, updatedInviter.AffHistoryQuota)

	var rewardCount int64
	require.NoError(t, DB.Model(&AffiliateReward{}).
		Where("source_type = ? AND source_id = ?", AffiliateRewardSourcePayment, topUp.TradeNo).
		Count(&rewardCount).Error)
	assert.Equal(t, int64(1), rewardCount)
}

func TestManualCompleteTopUpRepeatedCallDoesNotWriteZeroUserLog(t *testing.T) {
	truncateTables(t)

	user := &User{Username: "manual-topup-user", Password: "password", Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(user).Error)
	topUp := &TopUp{
		UserId:          user.Id,
		Amount:          2,
		Money:           2,
		TradeNo:         "manual-idempotent-trade",
		PaymentProvider: PaymentProviderEpay,
		PaymentMethod:   "alipay",
		Status:          common.TopUpStatusPending,
		CreateTime:      time.Now().Unix(),
	}
	require.NoError(t, DB.Create(topUp).Error)

	require.NoError(t, ManualCompleteTopUp(topUp.TradeNo, "127.0.0.1"))
	var firstLogCount int64
	require.NoError(t, DB.Model(&Log{}).Where("type = ?", LogTypeTopup).Count(&firstLogCount).Error)
	assert.Equal(t, int64(1), firstLogCount)

	require.NoError(t, ManualCompleteTopUp(topUp.TradeNo, "127.0.0.1"))
	var secondLogCount int64
	require.NoError(t, DB.Model(&Log{}).Where("type = ?", LogTypeTopup).Count(&secondLogCount).Error)
	assert.Equal(t, firstLogCount, secondLogCount)

	var zeroUserLogs int64
	require.NoError(t, DB.Model(&Log{}).Where("type = ? AND user_id = 0", LogTypeTopup).Count(&zeroUserLogs).Error)
	assert.Zero(t, zeroUserLogs)
}

func TestAffiliateRewardSkipsMissingInviter(t *testing.T) {
	truncateTables(t)

	invitee := &User{Username: "small-affiliate-invitee", Password: "password", Status: common.UserStatusEnabled, InviterId: 99999, AffCode: "small-invitee-code"}
	require.NoError(t, DB.Create(invitee).Error)

	var rewarded int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		rewarded, err = grantAffiliateRewardTx(tx, invitee.Id, AffiliateRewardSourcePayment, "missing-inviter-trade", 500)
		return err
	})
	require.NoError(t, err)
	assert.Zero(t, rewarded)
}

func TestAffiliateRewardHasNoMinimumQuota(t *testing.T) {
	truncateTables(t)

	inviter := &User{Username: "minimum-free-inviter", Password: "password", Status: common.UserStatusEnabled, AffCode: "minimum-free-inviter-code"}
	require.NoError(t, DB.Create(inviter).Error)
	invitee := &User{Username: "minimum-free-invitee", Password: "password", Status: common.UserStatusEnabled, InviterId: inviter.Id, AffCode: "minimum-free-invitee-code"}
	require.NoError(t, DB.Create(invitee).Error)

	var rewarded int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		rewarded, err = grantAffiliateRewardTx(tx, invitee.Id, AffiliateRewardSourceRedemption, "minimum-free-redemption", 1)
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, 1, rewarded)

	var updatedInviter User
	require.NoError(t, DB.First(&updatedInviter, inviter.Id).Error)
	assert.Equal(t, 1, updatedInviter.AffQuota)
	assert.Equal(t, 1, updatedInviter.AffHistoryQuota)
}
