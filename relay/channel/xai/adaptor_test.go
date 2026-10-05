package xai

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
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

func TestConvertImageRequestPreservesOfficialImageOptions(t *testing.T) {
	var request dto.ImageRequest
	require.NoError(t, common.Unmarshal([]byte(`{
		"model":"grok-imagine-image",
		"prompt":"a studio portrait",
		"aspect_ratio":"1:1",
		"resolution":"2k",
		"quality":"medium",
		"response_format":"b64_json"
	}`), &request))

	converted, err := (&Adaptor{}).ConvertImageRequest(nil, &relaycommon.RelayInfo{}, request)
	require.NoError(t, err)

	body, err := common.Marshal(converted)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(body, &payload))
	assert.Equal(t, "1:1", payload["aspect_ratio"])
	assert.Equal(t, "2k", payload["resolution"])
	assert.NotContains(t, payload, "quality")
	assert.Equal(t, "b64_json", payload["response_format"])
}

func TestConvertImageRequestMapsStandardSize(t *testing.T) {
	converted, err := (&Adaptor{}).ConvertImageRequest(nil, &relaycommon.RelayInfo{}, dto.ImageRequest{
		Model:  "grok-imagine-image",
		Prompt: "a studio portrait",
		Size:   "1536x1024",
	})
	require.NoError(t, err)

	body, err := common.Marshal(converted)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(body, &payload))
	assert.Equal(t, "3:2", payload["aspect_ratio"])
	assert.Equal(t, "2k", payload["resolution"])
}

func TestConvertImageRequestReadsExtraBodyOptions(t *testing.T) {
	var request dto.ImageRequest
	require.NoError(t, common.Unmarshal([]byte(`{
		"model":"grok-imagine-image",
		"prompt":"a studio portrait",
		"extra_body":{"aspect_ratio":"16:9","resolution":"2K"}
	}`), &request))

	converted, err := (&Adaptor{}).ConvertImageRequest(nil, &relaycommon.RelayInfo{}, request)
	require.NoError(t, err)

	body, err := common.Marshal(converted)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(body, &payload))
	assert.Equal(t, "16:9", payload["aspect_ratio"])
	assert.Equal(t, "2k", payload["resolution"])
}

func TestConvertImageRequestPreservesJSONEditImage(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	image := json.RawMessage(`{"url":"https://example.com/source.png","type":"image_url"}`)
	converted, err := (&Adaptor{}).ConvertImageRequest(c, &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesEdits,
	}, dto.ImageRequest{
		Model:  "grok-imagine-image",
		Prompt: "make it cinematic",
		Image:  image,
	})
	require.NoError(t, err)

	body, err := common.Marshal(converted)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(body, &payload))
	assert.Equal(t, map[string]any{
		"url":  "https://example.com/source.png",
		"type": "image_url",
	}, payload["image"])
}

func TestConvertImageRequestConvertsMultipartEditImage(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "grok-imagine-image"))
	require.NoError(t, writer.WriteField("prompt", "make it cinematic"))
	part, err := writer.CreateFormFile("image", "source.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("\x89PNG\r\n\x1a\nsource"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, c.Request.ParseMultipartForm(1<<20))

	converted, err := (&Adaptor{}).ConvertImageRequest(c, &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesEdits,
	}, dto.ImageRequest{
		Model:  "grok-imagine-image",
		Prompt: "make it cinematic",
	})
	require.NoError(t, err)

	encoded, err := common.Marshal(converted)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, common.Unmarshal(encoded, &payload))

	imagePayload, ok := payload["image"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "image_url", imagePayload["type"])
	assert.True(t, strings.HasPrefix(imagePayload["url"].(string), "data:image/png;base64,"))
}

func TestConvertImageRequestOmitsAbsentImageOptions(t *testing.T) {
	request := dto.ImageRequest{
		Model:  "grok-imagine-image",
		Prompt: "a studio portrait",
	}

	converted, err := (&Adaptor{}).ConvertImageRequest(nil, &relaycommon.RelayInfo{}, request)
	require.NoError(t, err)

	body, err := common.Marshal(converted)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(body, &payload))
	assert.NotContains(t, payload, "aspect_ratio")
	assert.NotContains(t, payload, "resolution")
	assert.NotContains(t, payload, "quality")
}
