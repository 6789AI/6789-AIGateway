package relay

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/ali"
	"github.com/QuantumNous/new-api/relay/channel/task/doubao"
	"github.com/QuantumNous/new-api/relay/channel/task/gemini"
	"github.com/QuantumNous/new-api/relay/channel/task/hailuo"
	"github.com/QuantumNous/new-api/relay/channel/task/vertex"
	"github.com/QuantumNous/new-api/relay/channel/task/vidu"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func TestVideoResolutionBillingReadsConvertedProviderParameters(t *testing.T) {
	for _, tc := range []struct {
		name     string
		provider channel.TaskVideoResolutionProvider
		req      relaycommon.TaskSubmitReq
		want     string
	}{
		{"Veo metadata beats size", &gemini.TaskAdaptor{}, relaycommon.TaskSubmitReq{Size: "1280x720", Metadata: map[string]any{"resolution": "1080p"}}, "1080p"},
		{"Vertex default", &vertex.TaskAdaptor{}, relaycommon.TaskSubmitReq{}, "720p"},
		{"Vidu metadata beats size", &vidu.TaskAdaptor{}, relaycommon.TaskSubmitReq{Size: "720p", Metadata: map[string]any{"resolution": "2K"}}, "2k"},
		{"Vidu default", &vidu.TaskAdaptor{}, relaycommon.TaskSubmitReq{}, "1080p"},
		{"Hailuo 768p", &hailuo.TaskAdaptor{}, relaycommon.TaskSubmitReq{Size: "720p", Metadata: map[string]any{"resolution": "768P"}}, "768p"},
		{"Doubao metadata", &doubao.TaskAdaptor{}, relaycommon.TaskSubmitReq{Metadata: map[string]any{"resolution": "1080p"}}, "1080p"},
		{"Ali nested parameters beat size", &ali.TaskAdaptor{}, relaycommon.TaskSubmitReq{Model: "wan2.5-i2v-preview", Size: "720p", Metadata: map[string]any{"parameters": map[string]any{"resolution": "1080P"}}}, "1080p"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Set("task_request", tc.req)
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}
			info.UpstreamModelName = "wan2.5-i2v-preview"
			resolution, err := tc.provider.VideoBillingResolution(ctx, info)
			require.NoError(t, err)
			assert.Equal(t, tc.want, billing_setting.NormalizeVideoResolution(resolution))
		})
	}
}

func TestVideoQualityPriceAppliesSecondsOnceAndSaturatesSafely(t *testing.T) {
	for _, tc := range []struct {
		name     string
		multiply bool
		want     int
	}{{"per request", false, 100000}, {"per second", true, 1000000}} {
		t.Run(tc.name, func(t *testing.T) {
			prices := hosttypes.PriceData{UsePrice: true, ModelPrice: 0.2, GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1}}
			ratios := billing_setting.FilterTaskBillingRatios(map[string]float64{"seconds": 10, "resolution": 2, "size": 1.6, "resolution-1080P": 3}, tc.multiply, true)
			prices.ReplaceOtherRatios(ratios)
			quota, clamp := taskQuotaFromPriceData(prices, 0)
			assert.Nil(t, clamp)
			assert.Equal(t, tc.want, quota)
			prices.ModelPrice = 1e30
			quota, clamp = taskQuotaFromPriceData(prices, 0)
			require.NotNil(t, clamp)
			assert.Positive(t, quota)
			assert.Equal(t, common.QuotaFromFloat(1e30), quota)
		})
	}
}
