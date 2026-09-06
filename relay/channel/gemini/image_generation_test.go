package gemini

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func geminiImageTestInfo(model string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName: model,
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: model},
	}
}

func TestConvertImageRequestKeepsImagenProtocol(t *testing.T) {
	adaptor := &Adaptor{}
	n := uint(2)

	converted, err := adaptor.ConvertImageRequest(nil, geminiImageTestInfo("imagen-4.0-generate-001"), dto.ImageRequest{
		Prompt:  "a city at sunset",
		N:       &n,
		Size:    "1536x1024",
		Quality: "high",
	})
	require.NoError(t, err)

	request, ok := converted.(dto.GeminiImageRequest)
	require.True(t, ok)
	require.Len(t, request.Instances, 1)
	assert.Equal(t, "a city at sunset", request.Instances[0].Prompt)
	assert.Equal(t, 2, request.Parameters.SampleCount)
	assert.Equal(t, "3:2", request.Parameters.AspectRatio)
	assert.Equal(t, "2K", request.Parameters.ImageSize)
}

func TestConvertImageRequestUsesGeminiGenerateContentForImageModel(t *testing.T) {
	adaptor := &Adaptor{}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	converted, err := adaptor.ConvertImageRequest(c, geminiImageTestInfo("gemini-3-pro-image-preview"), dto.ImageRequest{
		Prompt:  "a watercolor lighthouse",
		Size:    "1024x1536",
		Quality: "high",
	})
	require.NoError(t, err)

	request, ok := converted.(*dto.GeminiChatRequest)
	require.True(t, ok)
	require.Len(t, request.Contents, 1)
	require.Len(t, request.Contents[0].Parts, 1)
	assert.Equal(t, "user", request.Contents[0].Role)
	assert.Equal(t, "a watercolor lighthouse", request.Contents[0].Parts[0].Text)
	assert.Equal(t, []string{"TEXT", "IMAGE"}, request.GenerationConfig.ResponseModalities)

	var imageConfig map[string]string
	require.NoError(t, common.Unmarshal(request.GenerationConfig.ImageConfig, &imageConfig))
	assert.Equal(t, "2:3", imageConfig["aspectRatio"])
	assert.Equal(t, "2K", imageConfig["imageSize"])
}

func TestConvertImageRequestOmitsUnsupportedGeminiImageSize(t *testing.T) {
	adaptor := &Adaptor{}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	converted, err := adaptor.ConvertImageRequest(c, geminiImageTestInfo("gemini-2.5-flash-image"), dto.ImageRequest{
		Prompt:  "a watercolor lighthouse",
		Size:    "1024x1536",
		Quality: "high",
	})
	require.NoError(t, err)

	request, ok := converted.(*dto.GeminiChatRequest)
	require.True(t, ok)
	var imageConfig map[string]string
	require.NoError(t, common.Unmarshal(request.GenerationConfig.ImageConfig, &imageConfig))
	assert.Equal(t, "2:3", imageConfig["aspectRatio"])
	assert.NotContains(t, imageConfig, "imageSize")
}

func TestConvertImageRequestRejectsMultipleGeminiImages(t *testing.T) {
	adaptor := &Adaptor{}
	n := uint(2)

	_, err := adaptor.ConvertImageRequest(nil, geminiImageTestInfo("gemini-2.5-flash-image"), dto.ImageRequest{
		Prompt: "two images",
		N:      &n,
	})
	require.EqualError(t, err, "Gemini image models only support n=1")
}

func TestConvertImageRequestRejectsGeminiImageStreaming(t *testing.T) {
	adaptor := &Adaptor{}
	stream := true

	_, err := adaptor.ConvertImageRequest(nil, geminiImageTestInfo("gemini-2.5-flash-image"), dto.ImageRequest{
		Prompt: "streaming image",
		Stream: &stream,
	})
	require.EqualError(t, err, "Gemini image models do not support streaming through this endpoint")
}

func TestGeminiGetRequestURLKeepsSeparateImageProtocols(t *testing.T) {
	adaptor := &Adaptor{}

	imagenInfo := geminiImageTestInfo("imagen-4.0-generate-001")
	imagenInfo.ChannelBaseUrl = "https://generativelanguage.googleapis.com"
	imagenURL, err := adaptor.GetRequestURL(imagenInfo)
	require.NoError(t, err)
	assert.Equal(t, "https://generativelanguage.googleapis.com/v1beta/models/imagen-4.0-generate-001:predict", imagenURL)

	geminiInfo := geminiImageTestInfo("gemini-2.5-flash-image")
	geminiInfo.ChannelBaseUrl = "https://generativelanguage.googleapis.com"
	geminiURL, err := adaptor.GetRequestURL(geminiInfo)
	require.NoError(t, err)
	assert.Equal(t, "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-image:generateContent", geminiURL)
}

func TestGeminiGenerateContentImageHandlerReturnsOpenAIImageResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{
			"candidates": [{
				"content": {"parts": [
					{"text": "Here is your image."},
					{"inlineData": {"mimeType": "image/png", "data": "aGVsbG8="}}
				]},
				"finishReason": "STOP"
			}],
			"usageMetadata": {"promptTokenCount": 3, "candidatesTokenCount": 4, "totalTokenCount": 7}
		}`)),
	}

	usageValue, apiErr := (&Adaptor{}).DoResponse(c, resp, geminiImageTestInfo("gemini-2.5-flash-image"))
	require.Nil(t, apiErr)
	usage, ok := usageValue.(*dto.Usage)
	require.True(t, ok)
	require.NotNil(t, usage)
	assert.Equal(t, 3, usage.PromptTokens)
	assert.Equal(t, 4, usage.CompletionTokens)
	assert.Equal(t, 7, usage.TotalTokens)

	var imageResponse dto.ImageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &imageResponse))
	require.Len(t, imageResponse.Data, 1)
	assert.Equal(t, "aGVsbG8=", imageResponse.Data[0].B64Json)
}
