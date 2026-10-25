package model

import (
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGetPricingExportsVideoResolutionPricesOnlyForPerCallModels(t *testing.T) {
	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(k, v string) error { saved[k] = v; return nil }))
	oldPricing, oldTime := pricingMap, lastGetPricingTime
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
		pricingMap, lastGetPricingTime = oldPricing, oldTime
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.price_schedules":         `{}`,
		"billing_setting.video_resolution_prices": `{"video":{"720p":0,"768p":0.2},"token":{"720p":9}}`,
	}))
	pricingMap = []Pricing{{ModelName: "video", QuotaType: 1, ModelPrice: 0.1}, {ModelName: "token", QuotaType: 0, ModelRatio: 2}}
	lastGetPricingTime = time.Now()
	data := GetPricing()
	require.Len(t, data, 2)
	assert.Equal(t, map[string]float64{"720p": 0, "768p": 0.2}, data[0].VideoResolutionPrices)
	assert.Nil(t, data[1].VideoResolutionPrices)
	data[0].VideoResolutionPrices["720p"] = 100
	assert.Equal(t, 0.0, GetPricing()[0].VideoResolutionPrices["720p"])
}
