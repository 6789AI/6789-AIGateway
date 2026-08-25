package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/setting/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPricingIncludesPromotionFromSamePriceSnapshot(t *testing.T) {
	savedConfig := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		savedConfig[key] = value
		return nil
	}))
	originalPricingMap := pricingMap
	originalLastGetPricingTime := lastGetPricingTime
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(savedConfig))
		pricingMap = originalPricingMap
		lastGetPricingTime = originalLastGetPricingTime
	})

	now := time.Now()
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.free_model_banner_enabled": "false",
		"billing_setting.price_schedules": fmt.Sprintf(
			`{"discount-model":[{"type":"absolute","adjustment_type":"discount","discount_rate":0.8,"show_banner":false,"start_at":%d,"end_at":%d}]}`,
			now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix(),
		),
	}))
	pricingMap = []Pricing{
		{ModelName: "discount-model", QuotaType: 0, ModelRatio: 0.5},
		{ModelName: "regular-model", QuotaType: 0, ModelRatio: 0.5},
	}
	lastGetPricingTime = now

	pricing := GetPricing()

	require.Len(t, pricing, 2)
	assert.InDelta(t, 0.4, pricing[0].ModelRatio, 1e-12)
	require.NotNil(t, pricing[0].ActivePromotion)
	assert.Equal(t, "discount-model", pricing[0].ActivePromotion.ModelName)
	assert.Equal(t, "discount", pricing[0].ActivePromotion.PromotionType)
	require.NotNil(t, pricing[0].ActivePromotion.DiscountRate)
	assert.InDelta(t, 0.8, *pricing[0].ActivePromotion.DiscountRate, 1e-12)
	assert.Nil(t, pricing[1].ActivePromotion)
}
