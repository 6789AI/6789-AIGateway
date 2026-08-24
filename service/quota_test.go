package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreWssConsumeQuotaKeepsFrozenFreeActivityFree(t *testing.T) {
	truncate(t)
	const (
		userID   = 810
		tokenID  = 811
		tokenKey = "realtime-free"
	)
	seedUser(t, userID, 10_000)
	seedToken(t, tokenID, userID, tokenKey, 10_000)

	relayInfo := &relaycommon.RelayInfo{
		UserId:          userID,
		TokenId:         tokenID,
		TokenKey:        tokenKey,
		OriginModelName: "gpt-4o-realtime-preview",
		PriceData: types.PriceData{
			FreeModel:  true,
			ModelRatio: 0,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
	}
	ctx, _ := gin.CreateTestContext(nil)

	require.NoError(t, PreWssConsumeQuota(ctx, relayInfo, &dto.RealtimeUsage{
		TotalTokens: 1_000,
		InputTokens: 1_000,
		InputTokenDetails: dto.InputTokenDetails{
			TextTokens: 1_000,
		},
	}))

	userQuota, err := model.GetUserQuota(userID, true)
	require.NoError(t, err)
	assert.Equal(t, 10_000, userQuota)
	var token model.Token
	err = model.DB.Where("id = ?", tokenID).First(&token).Error
	require.NoError(t, err)
	assert.Equal(t, 10_000, token.RemainQuota)
	assert.Nil(t, relayInfo.Billing)
}

func TestPreWssConsumeQuotaReservesCumulativeFrozenDiscountOnce(t *testing.T) {
	truncate(t)
	const userID = 812
	seedUser(t, userID, 10_000)

	relayInfo := &relaycommon.RelayInfo{
		UserId:          userID,
		IsPlayground:    true,
		ForcePreConsume: true,
		OriginModelName: "gpt-4o-realtime-preview",
		UserSetting: dto.UserSetting{
			BillingPreference: "wallet_only",
		},
		PriceData: types.PriceData{
			ModelRatio: 0.5,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
	}
	ctx, _ := gin.CreateTestContext(nil)
	require.Nil(t, PreConsumeBilling(ctx, 50, relayInfo))

	require.NoError(t, PreWssConsumeQuota(ctx, relayInfo, &dto.RealtimeUsage{
		TotalTokens: 200,
		InputTokens: 200,
		InputTokenDetails: dto.InputTokenDetails{
			TextTokens: 200,
		},
	}))
	assert.Equal(t, 100, relayInfo.Billing.GetPreConsumedQuota())
	assert.Equal(t, 100, relayInfo.FinalPreConsumedQuota)

	require.NoError(t, SettleBilling(ctx, relayInfo, 100))
	userQuota, err := model.GetUserQuota(userID, true)
	require.NoError(t, err)
	assert.Equal(t, 9_900, userQuota)
}

func TestPreWssConsumeQuotaUsesFrozenTieredDiscount(t *testing.T) {
	discountRate := 0.5
	billing := &recordingBillingSettler{}
	relayInfo := &relaycommon.RelayInfo{
		Billing: billing,
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:  "tiered_expr",
			ExprString:   `tier("realtime", p)`,
			ExprHash:     billingexpr.ExprHashString(`tier("realtime", p)`),
			GroupRatio:   1,
			QuotaPerUnit: 500_000,
			DiscountRate: &discountRate,
		},
	}
	ctx, _ := gin.CreateTestContext(nil)

	require.NoError(t, PreWssConsumeQuota(ctx, relayInfo, &dto.RealtimeUsage{
		TotalTokens: 1_000,
		InputTokens: 1_000,
		InputTokenDetails: dto.InputTokenDetails{
			TextTokens: 1_000,
		},
	}))

	assert.Equal(t, []int{250}, billing.reserveTargets)
	assert.Equal(t, 250, relayInfo.FinalPreConsumedQuota)
}
