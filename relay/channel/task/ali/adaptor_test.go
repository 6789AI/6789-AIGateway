package ali

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRelayInfo() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
	}
}

func TestConvertToAliRequestWan27I2VBuildsMediaFromImage(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:    "wan2.7-i2v",
		Prompt:   "animate the first frame",
		Image:    "https://example.com/first.png",
		Size:     "720p",
		Duration: 10,
	}

	aliReq, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.NoError(t, err)
	require.Equal(t, "wan2.7-i2v", aliReq.Model)
	require.Equal(t, "720P", aliReq.Parameters.Resolution)
	require.Equal(t, 10, aliReq.Parameters.Duration)
	require.Equal(t, []AliVideoMedia{
		{Type: "first_frame", URL: "https://example.com/first.png"},
	}, aliReq.Input.Media)
	require.Empty(t, aliReq.Input.ImgURL)

	body, err := common.Marshal(aliReq)
	require.NoError(t, err)
	require.Contains(t, string(body), `"media"`)
	require.NotContains(t, string(body), `"img_url"`)
}

func TestConvertToAliRequestWan27I2VBuildsFirstAndLastFrameFromImages(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  "wan2.7-i2v",
		Prompt: "interpolate between frames",
		Images: []string{
			"https://example.com/first.png",
			"https://example.com/last.png",
		},
	}

	aliReq, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.NoError(t, err)
	require.Equal(t, []AliVideoMedia{
		{Type: "first_frame", URL: "https://example.com/first.png"},
		{Type: "last_frame", URL: "https://example.com/last.png"},
	}, aliReq.Input.Media)
}

func TestConvertToAliRequestWan27I2VPrefersImageBeforeImagesAndInputReference(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:          "wan2.7-i2v",
		Prompt:         "use the direct image",
		Image:          " https://example.com/direct.png ",
		Images:         []string{"https://example.com/images-first.png", " https://example.com/images-last.png "},
		InputReference: "https://example.com/input-reference.png",
	}

	aliReq, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.NoError(t, err)
	require.Equal(t, []AliVideoMedia{
		{Type: "first_frame", URL: "https://example.com/direct.png"},
		{Type: "last_frame", URL: "https://example.com/images-last.png"},
	}, aliReq.Input.Media)
}

func TestConvertToAliRequestWan27I2VFallsBackToFirstNonEmptyImage(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  "wan2.7-i2v",
		Prompt: "skip blank images",
		Image:  " ",
		Images: []string{
			" ",
			" https://example.com/first.png ",
			" https://example.com/last.png ",
		},
		InputReference: "https://example.com/input-reference.png",
	}

	aliReq, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.NoError(t, err)
	require.Equal(t, []AliVideoMedia{
		{Type: "first_frame", URL: "https://example.com/first.png"},
		{Type: "last_frame", URL: "https://example.com/last.png"},
	}, aliReq.Input.Media)
}

func TestConvertToAliRequestWan27I2VKeepsExplicitMetadataMedia(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:          "wan2.7-i2v",
		Prompt:         "continue the clip",
		Image:          "https://example.com/direct.png",
		Images:         []string{"https://example.com/images-first.png", "https://example.com/images-last.png"},
		InputReference: "https://example.com/input-reference.png",
		Metadata: map[string]interface{}{
			"input": map[string]interface{}{
				"media": []interface{}{
					map[string]interface{}{
						"type": "first_clip",
						"url":  "https://example.com/input.mp4",
					},
				},
			},
		},
	}

	aliReq, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.NoError(t, err)
	require.Equal(t, []AliVideoMedia{
		{Type: "first_clip", URL: "https://example.com/input.mp4"},
	}, aliReq.Input.Media)
	require.Empty(t, aliReq.Input.ImgURL)

	body, err := common.Marshal(aliReq)
	require.NoError(t, err)
	require.Contains(t, string(body), `"media"`)
	require.NotContains(t, string(body), `"img_url"`)
}

func TestConvertToAliRequestWan27I2VRequiresMedia(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  "wan2.7-i2v",
		Prompt: "animate without a frame",
	}

	_, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "requires image"))
}

func TestConvertToAliRequestWan25I2VKeepsLegacyImgURL(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  "wan2.5-i2v-preview",
		Prompt: "animate the first frame",
		Image:  "https://example.com/first.png",
	}

	aliReq, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.NoError(t, err)
	require.Equal(t, "https://example.com/first.png", aliReq.Input.ImgURL)
	require.Empty(t, aliReq.Input.Media)

	body, err := common.Marshal(aliReq)
	require.NoError(t, err)
	require.Contains(t, string(body), `"img_url"`)
	require.NotContains(t, string(body), `"media"`)
}

func TestAsyncImageRequestValidationAndBillingRatios(t *testing.T) {
	oldUpdateTask := constant.UpdateTask
	constant.UpdateTask = true
	t.Cleanup(func() { constant.UpdateTask = oldUpdateTask })

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewBufferString(`{
		"model":"wanx-v1","prompt":"draw a lighthouse","n":3,"async":true
	}`))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeImageSubmit,
		OriginModelName: "wanx-v1",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeAli,
			UpstreamModelName: "wanx-v1",
		},
	}
	adaptor := &TaskAdaptor{}

	taskErr := adaptor.ValidateRequestAndSetAction(c, info)
	require.Nil(t, taskErr)
	require.Equal(t, constant.TaskActionImageGenerate, info.Action)
	require.IsType(t, &dto.ImageRequest{}, info.Request)
	require.Nil(t, adaptor.ValidateMappedTaskRequest(c, info))
	assert.Equal(t, map[string]float64{"n": 3}, adaptor.EstimateBilling(c, info))
}

func TestAsyncImageMappedValidationPreservesProviderBillingRatios(t *testing.T) {
	oldUpdateTask := constant.UpdateTask
	constant.UpdateTask = true
	t.Cleanup(func() { constant.UpdateTask = oldUpdateTask })
	qwenSettings := model_setting.GetQwenSettings()
	oldSyncImageModels := append([]string(nil), qwenSettings.SyncImageModels...)
	qwenSettings.SyncImageModels = nil
	t.Cleanup(func() { qwenSettings.SyncImageModels = oldSyncImageModels })

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewBufferString(`{
		"model":"customer-image-model",
		"prompt":"draw a lighthouse",
		"async":true,
		"parameters":{"n":2,"prompt_extend":true}
	}`))
	c.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeImageSubmit,
		OriginModelName: "customer-image-model",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeAli,
			UpstreamModelName: "z-image-turbo",
		},
	}
	adaptor := &TaskAdaptor{}

	require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))
	require.Nil(t, adaptor.ValidateMappedTaskRequest(c, info))
	assert.Equal(t, map[string]float64{"n": 2, "prompt_extend": 2}, adaptor.EstimateBilling(c, info))

	info.UpstreamModelName = "wanx-v1"
	require.Nil(t, adaptor.ValidateMappedTaskRequest(c, info))
	assert.Equal(t, map[string]float64{"n": 2}, adaptor.EstimateBilling(c, info))
}

func TestAsyncImageMappedSyncModelIsRejected(t *testing.T) {
	oldUpdateTask := constant.UpdateTask
	constant.UpdateTask = true
	t.Cleanup(func() { constant.UpdateTask = oldUpdateTask })

	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeImageSubmit,
		OriginModelName: "customer-image-model",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "qwen-image-plus",
		},
	}

	taskErr := (&TaskAdaptor{}).ValidateMappedTaskRequest(nil, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
	assert.Equal(t, "async_image_not_supported", taskErr.Code)
}

func TestAsyncImageDoResponseDefersClientResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	responseBody := []byte(`{"output":{"task_id":"upstream-task","task_status":"PENDING"}}`)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(responseBody)),
	}
	info := &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeImageSubmit}

	taskID, taskData, taskErr := (&TaskAdaptor{}).DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	assert.Equal(t, "upstream-task", taskID)
	assert.Equal(t, responseBody, taskData)
	assert.Empty(t, recorder.Body.String())
}

func TestAsyncImageDoResponseMapsSuccessfulAliErrorToBadGateway(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(bytes.NewReader([]byte(
			`{"code":"InvalidParameter","message":"prompt is invalid"}`,
		))),
	}
	info := &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeImageSubmit}

	_, _, taskErr := (&TaskAdaptor{}).DoResponse(c, resp, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "ali_api_error", taskErr.Code)
	assert.Equal(t, http.StatusBadGateway, taskErr.StatusCode)
}

func TestParseAsyncImageTaskResultPreservesMultipleImages(t *testing.T) {
	taskResult, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"output":{"task_status":"SUCCEEDED","results":[
			{"url":"https://example.com/one.png"},
			{"url":"https://example.com/two.png"}
		]},
		"usage":{"image_count":2}
	}`))

	require.NoError(t, err)
	assert.Equal(t, string(model.TaskStatusSuccess), taskResult.Status)
	assert.Equal(t, "https://example.com/one.png", taskResult.Url)
	assert.Equal(t, 1, taskResult.TotalTokens)
	assert.Equal(t, map[string]float64{"n": 2}, taskResult.BillingRatios)
}

func TestParseAsyncImageTaskResultCapsUpstreamImageCount(t *testing.T) {
	taskResult, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"output":{"task_status":"SUCCEEDED","results":[{"url":"https://example.com/one.png"}]},
		"usage":{"image_count":999999999}
	}`))

	require.NoError(t, err)
	assert.Equal(t, map[string]float64{"n": dto.MaxImageN}, taskResult.BillingRatios)
}

func TestParseAsyncImageTaskResultTreatsTopLevelAliErrorAsFailure(t *testing.T) {
	taskResult, err := (&TaskAdaptor{}).ParseTaskResult([]byte(
		`{"code":"InvalidApiKey","message":"invalid API key"}`,
	))

	require.NoError(t, err)
	assert.Equal(t, string(model.TaskStatusFailure), taskResult.Status)
	assert.Equal(t, "InvalidApiKey: invalid API key", taskResult.Reason)
}

func TestConvertToOpenAIImageTaskCompletedAndFailed(t *testing.T) {
	adaptor := &TaskAdaptor{}

	t.Run("completed", func(t *testing.T) {
		task := &model.Task{
			TaskID:     "task_public",
			Status:     model.TaskStatusSuccess,
			Progress:   "100%",
			CreatedAt:  10,
			FinishTime: 20,
			Properties: model.Properties{
				OriginModelName: "wanx-v1",
			},
			Data: []byte(`{"output":{"task_status":"SUCCEEDED","results":[
				{"url":"https://example.com/one.png"},
				{"url":"https://example.com/two.png"}
			]}}`),
		}

		body, err := adaptor.ConvertToOpenAIImageTask(task)
		require.NoError(t, err)
		var response dto.ImageTaskResponse
		require.NoError(t, common.Unmarshal(body, &response))
		assert.Equal(t, dto.ImageTaskStatusCompleted, response.Status)
		assert.Equal(t, 100, response.Progress)
		assert.Equal(t, int64(20), response.CompletedAt)
		require.Len(t, response.Data, 2)
		assert.Equal(t, "https://example.com/two.png", response.Data[1].Url)
	})

	t.Run("failed", func(t *testing.T) {
		task := &model.Task{
			TaskID:     "task_failed",
			Status:     model.TaskStatusFailure,
			Progress:   "100%",
			FailReason: "content policy rejected the prompt",
			Data:       []byte(`{"output":{"task_status":"FAILED","code":"DataInspectionFailed"}}`),
		}

		body, err := adaptor.ConvertToOpenAIImageTask(task)
		require.NoError(t, err)
		var response dto.ImageTaskResponse
		require.NoError(t, common.Unmarshal(body, &response))
		assert.Equal(t, dto.ImageTaskStatusFailed, response.Status)
		require.NotNil(t, response.Error)
		assert.Equal(t, "DataInspectionFailed", response.Error.Code)
		assert.Equal(t, task.FailReason, response.Error.Message)
	})
}
