package grsai

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertImageRequestMapsGrsaiAsyncContract(t *testing.T) {
	request := dto.ImageRequest{
		Prompt:  "draw a lighthouse",
		Size:    "1792x1024",
		Quality: "high",
		Images:  []byte(`["https://example.com/reference.png","data:image/png;base64,AAAA"]`),
	}

	converted, err := convertImageRequest(request, "nano-banana-2")

	require.NoError(t, err)
	assert.Equal(t, "nano-banana-2", converted.Model)
	assert.Equal(t, "draw a lighthouse", converted.Prompt)
	assert.Equal(t, []string{"https://example.com/reference.png", "data:image/png;base64,AAAA"}, converted.Images)
	assert.Equal(t, "16:9", converted.AspectRatio)
	assert.Equal(t, "2K", converted.ImageSize)
	assert.Equal(t, "async", converted.ReplyType)
	body, err := common.Marshal(converted)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"replyType":"async"`)
	assert.Contains(t, string(body), `"aspectRatio":"16:9"`)
}

func TestConvertImageRequestAcceptsProviderOverrides(t *testing.T) {
	request := dto.ImageRequest{
		Prompt: "draw a lighthouse",
		Extra: map[string]json.RawMessage{
			"aspectRatio": []byte(`"1:4"`),
			"imageSize":   []byte(`"4K"`),
		},
	}

	converted, err := convertImageRequest(request, "nano-banana-pro-vip")

	require.NoError(t, err)
	assert.Equal(t, "1:4", converted.AspectRatio)
	assert.Equal(t, "4K", converted.ImageSize)
}

func TestValidateMappedTaskRequestRejectsUnsupportedCountAndModel(t *testing.T) {
	oldUpdateTask := constant.UpdateTask
	constant.UpdateTask = true
	t.Cleanup(func() { constant.UpdateTask = oldUpdateTask })

	t.Run("multiple outputs", func(t *testing.T) {
		adaptor := &TaskAdaptor{imageRequest: &dto.ImageRequest{Prompt: "draw", N: common.GetPointer(uint(2))}}
		info := &relaycommon.RelayInfo{
			OriginModelName: "image-alias",
			ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "nano-banana-2"},
		}

		taskErr := adaptor.ValidateMappedTaskRequest(nil, info)

		require.NotNil(t, taskErr)
		assert.Equal(t, "invalid_image_request", taskErr.Code)
	})

	t.Run("unsupported mapped model", func(t *testing.T) {
		adaptor := &TaskAdaptor{imageRequest: &dto.ImageRequest{Prompt: "draw", N: common.GetPointer(uint(1))}}
		info := &relaycommon.RelayInfo{
			OriginModelName: "image-alias",
			ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "imagen-4.0-generate-001"},
		}

		taskErr := adaptor.ValidateMappedTaskRequest(nil, info)

		require.NotNil(t, taskErr)
		assert.Equal(t, "async_image_not_supported", taskErr.Code)
	})
}

func TestDoResponseDefersClientResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	responseBody := []byte(`{"id":"upstream-task","status":"running"}`)
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(responseBody))}

	taskID, taskData, taskErr := (&TaskAdaptor{}).DoResponse(c, resp, &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeImageSubmit})

	require.Nil(t, taskErr)
	assert.Equal(t, "upstream-task", taskID)
	assert.Equal(t, responseBody, taskData)
	assert.Empty(t, recorder.Body.String())
}

func TestFetchTaskUsesGrsaiResultContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/api/result", r.URL.Path)
		assert.Equal(t, "task/with spaces", r.URL.Query().Get("id"))
		assert.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"id":"task/with spaces","status":"running"}`))
	}))
	t.Cleanup(server.Close)

	resp, err := (&TaskAdaptor{}).FetchTask(server.URL, "secret", map[string]any{"task_id": "task/with spaces"}, "")

	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
}

func TestParseTaskResultMapsStatusesAndActualImageCount(t *testing.T) {
	adaptor := &TaskAdaptor{}

	t.Run("running", func(t *testing.T) {
		result, err := adaptor.ParseTaskResult([]byte(`{"id":"upstream-task","status":"running","progress":42}`))

		require.NoError(t, err)
		assert.Equal(t, string(model.TaskStatusInProgress), result.Status)
		assert.Equal(t, "42%", result.Progress)
	})

	t.Run("succeeded", func(t *testing.T) {
		result, err := adaptor.ParseTaskResult([]byte(`{
			"id":"upstream-task","status":"succeeded","progress":100,
			"results":[{"url":"https://example.com/one.png"},{"url":"https://example.com/two.png"}]
		}`))

		require.NoError(t, err)
		assert.Equal(t, string(model.TaskStatusSuccess), result.Status)
		assert.Equal(t, "https://example.com/one.png", result.Url)
		assert.Zero(t, result.TotalTokens)
		assert.Equal(t, map[string]float64{"n": 2}, result.BillingRatios)
	})

	t.Run("violation", func(t *testing.T) {
		result, err := adaptor.ParseTaskResult([]byte(`{
			"id":"upstream-task","status":"violation",
			"error":{"code":"content_policy","message":"prompt rejected"}
		}`))

		require.NoError(t, err)
		assert.Equal(t, string(model.TaskStatusFailure), result.Status)
		assert.Equal(t, "content_policy: prompt rejected", result.Reason)
	})

	t.Run("top-level error", func(t *testing.T) {
		result, err := adaptor.ParseTaskResult([]byte(`{"error":"invalid API key"}`))

		require.NoError(t, err)
		assert.Equal(t, string(model.TaskStatusFailure), result.Status)
		assert.Equal(t, "invalid API key", result.Reason)
	})
}

func TestConvertToOpenAIImageTaskPreservesOrderedResults(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_public",
		Status:     model.TaskStatusSuccess,
		Progress:   "100%",
		CreatedAt:  10,
		FinishTime: 20,
		Properties: model.Properties{OriginModelName: "nano-banana-2"},
		Data: []byte(`{
			"id":"upstream-task","status":"succeeded",
			"results":[{"url":"https://example.com/one.png"},{"url":"https://example.com/two.png"}]
		}`),
	}

	body, err := (&TaskAdaptor{}).ConvertToOpenAIImageTask(task)

	require.NoError(t, err)
	var response dto.ImageTaskResponse
	require.NoError(t, common.Unmarshal(body, &response))
	assert.Equal(t, dto.ImageTaskStatusCompleted, response.Status)
	assert.Equal(t, int64(20), response.CompletedAt)
	require.Len(t, response.Data, 2)
	assert.Equal(t, "https://example.com/one.png", response.Data[0].Url)
	assert.Equal(t, "https://example.com/two.png", response.Data[1].Url)
}
