package doubao

import (
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func TestVideoQualityAbsolutePriceKeepsOnlySameResolutionInputDiscount(t *testing.T) {
	for _, hasVideo := range []bool{false, true} {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		metadata := map[string]any{"resolution": "1080p"}
		if hasVideo {
			metadata["content"] = []any{map[string]any{"type": "video_url", "video_url": map[string]any{"url": "https://example.com/ref.mp4"}}}
		}
		ctx.Set("task_request", relaycommon.TaskSubmitReq{Metadata: metadata})
		info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "doubao-seedance-2-0-260128"}, OriginModelName: "canvas-alias", TaskResolutionPrice: true}
		ratios := (&TaskAdaptor{}).EstimateBilling(ctx, info)
		if hasVideo {
			require.Contains(t, ratios, "video_input")
			assert.InDelta(t, 31.0/51.0, ratios["video_input"], 1e-12)
		} else {
			assert.Empty(t, ratios)
		}
	}
}
