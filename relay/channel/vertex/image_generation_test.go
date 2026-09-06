package vertex

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenAIRequestPreservesImagenImageConfig(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	model := "imagen-4.0-generate-001"
	info := &relaycommon.RelayInfo{
		OriginModelName: model,
		RelayMode:       relayconstant.RelayModeChatCompletions,
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: model},
	}
	adaptor := &Adaptor{}
	adaptor.Init(info)

	extraBody, err := common.Marshal(map[string]any{
		"google": map[string]any{
			"image_config": map[string]string{
				"aspect_ratio": "16:9",
				"image_size":   "4K",
			},
		},
	})
	require.NoError(t, err)

	converted, err := adaptor.ConvertOpenAIRequest(c, info, &dto.GeneralOpenAIRequest{
		Model: model,
		Messages: []dto.Message{{
			Role:    "user",
			Content: "a cinematic landscape",
		}},
		ExtraBody: extraBody,
	})
	require.NoError(t, err)

	request, ok := converted.(dto.GeminiImageRequest)
	require.True(t, ok)
	assert.Equal(t, "16:9", request.Parameters.AspectRatio)
	assert.Equal(t, "4K", request.Parameters.ImageSize)
}
