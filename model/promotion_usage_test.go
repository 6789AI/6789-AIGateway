package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromotionAllowanceForUsedQuotaRoundsDownAfterMultiplication(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500_000
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})

	tests := []struct {
		name      string
		usedQuota int
		want      int
	}{
		{name: "negative consumption uses minimum", usedQuota: -1, want: 20},
		{name: "below ten units uses minimum", usedQuota: 4_999_999, want: 20},
		{name: "exactly ten units gets thirty", usedQuota: 5_000_000, want: 30},
		{name: "fraction is multiplied before floor", usedQuota: 5_495_000, want: 32},
		{name: "large consumption is capped", usedQuota: 500_000_000, want: 2000},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, PromotionAllowanceForUsedQuota(test.usedQuota))
		})
	}
}

func TestPromotionReservationSharesAllowanceAndRestoresRefundedUse(t *testing.T) {
	truncateTables(t)
	originalQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500_000
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})

	user := User{Username: "promotion-user", UsedQuota: 0}
	require.NoError(t, DB.Create(&user).Error)
	const activityKey = "v1:absolute:100:200"

	for index := 0; index < 20; index++ {
		granted, err := ReservePromotionUse(user.Id, fmt.Sprintf("request-%d", index), activityKey)
		require.NoError(t, err)
		assert.True(t, granted)
	}
	granted, err := ReservePromotionUse(user.Id, "request-over-limit", activityKey)
	require.NoError(t, err)
	assert.False(t, granted)

	granted, err = ReservePromotionUse(user.Id, "request-0", activityKey)
	require.NoError(t, err)
	assert.True(t, granted, "repeated reservation must be idempotent")

	require.NoError(t, CommitPromotionUse("request-0"))
	require.NoError(t, CommitPromotionUse("request-0"))
	require.NoError(t, RefundPromotionUse("request-1"))
	require.NoError(t, RefundPromotionUse("request-1"))

	granted, err = ReservePromotionUse(user.Id, "request-after-refund", activityKey)
	require.NoError(t, err)
	assert.True(t, granted)

	var usage PromotionUsage
	require.NoError(t, DB.Where("user_id = ? AND activity_key = ?", user.Id, activityKey).Take(&usage).Error)
	assert.Equal(t, 1, usage.UsedCount)
	assert.Equal(t, 19, usage.ReservedCount)
	var terminalReservations int64
	require.NoError(t, DB.Model(&PromotionReservation{}).
		Where("request_id IN ?", []string{"request-0", "request-1"}).
		Count(&terminalReservations).Error)
	assert.Zero(t, terminalReservations)

	granted, err = ReservePromotionUse(user.Id, "request-other-activity", "v1:absolute:300:400")
	require.NoError(t, err)
	assert.True(t, granted, "a different activity occurrence has an independent allowance")
}

func TestPromotionAllowanceExpandsWithTotalConsumption(t *testing.T) {
	truncateTables(t)
	originalQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500_000
	t.Cleanup(func() {
		common.QuotaPerUnit = originalQuotaPerUnit
	})

	user := User{Username: "promotion-expanded-user", UsedQuota: 5_495_000}
	require.NoError(t, DB.Create(&user).Error)
	const activityKey = "v1:absolute:500:600"

	for index := 0; index < 32; index++ {
		granted, err := ReservePromotionUse(user.Id, fmt.Sprintf("expanded-request-%d", index), activityKey)
		require.NoError(t, err)
		assert.True(t, granted)
	}
	granted, err := ReservePromotionUse(user.Id, "expanded-over-limit", activityKey)
	require.NoError(t, err)
	assert.False(t, granted)
}
