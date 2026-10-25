package helper

import (
	"fmt"
	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPerCallVideoResolutionPricesUseClientAliasAndGroup(t *testing.T) {
	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(k, v string) error { saved[k] = v; return nil }))
	oldPrices := ratio_setting.ModelPrice2JSONString()
	oldRatios := ratio_setting.ModelRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(oldPrices))
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(oldRatios))
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.price_schedules":         `{}`,
		"billing_setting.video_resolution_prices": `{"canvas-alias":{"480p":0,"720p":0.1,"768p":0.2,"1080p":0.3,"2k":0.4},"token-model":{"720p":9}}`,
		"group_ratio_setting.group_ratio":         `{"default":1.5}`,
	}))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"canvas-alias":0.5,"upstream-name":9}`))
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"token-model":2}`))
	for _, tc := range []struct {
		name, resolution string
		price            float64
		matched          bool
	}{
		{"480p free", "480p", 0, true}, {"720p", "720p", 0.1, true}, {"768p", "768p", 0.2, true}, {"1080p", "1080P", 0.3, true}, {"2k alias", "1440p", 0.4, true},
		{"missing resolution falls back", "", 0.5, false}, {"unpriced resolution falls back", "4k", 0.5, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Set("group", "default")
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}, OriginModelName: "canvas-alias", UserGroup: "default", UsingGroup: "default", TaskVideoResolution: tc.resolution}
			info.UpstreamModelName = "upstream-name"
			price, err := ModelPriceHelperPerCall(ctx, info)
			require.NoError(t, err)
			assert.Equal(t, tc.price, price.ModelPrice)
			assert.Equal(t, tc.matched, info.TaskResolutionPrice)
			assert.Equal(t, common.QuotaFromFloat(tc.price*common.QuotaPerUnit*1.5), price.Quota)
		})
	}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("group", "default")
	tokenInfo := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}, OriginModelName: "token-model", UserGroup: "default", UsingGroup: "default", TaskVideoResolution: "720p"}
	tokenPrice, err := ModelPriceHelperPerCall(ctx, tokenInfo)
	require.NoError(t, err)
	assert.False(t, tokenPrice.UsePrice)
	assert.False(t, tokenInfo.TaskResolutionPrice)
	assert.Equal(t, 2.0, tokenPrice.ModelRatio)
	now := time.Now().Unix()
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{"billing_setting.price_schedules": fmt.Sprintf(`{"canvas-alias":[{"type":"absolute","adjustment_type":"discount","discount_rate":0.5,"start_at":%d,"end_at":%d}]}`, now-3600, now+3600)}))
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}, OriginModelName: "canvas-alias", UserGroup: "default", UsingGroup: "default", TaskVideoResolution: "720p"}
	price, err := ModelPriceHelperPerCall(ctx, info)
	require.NoError(t, err)
	assert.Equal(t, 0.05, price.ModelPrice)
	assert.Equal(t, 37500, price.Quota)
}
