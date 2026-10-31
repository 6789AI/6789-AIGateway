package sora

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
	relaykitdto "github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newVintedTestContext(t *testing.T, payload string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	t.Cleanup(func() { common.CleanupBodyStorage(c) })
	return c, recorder
}

func newVintedAdaptorInfo(modelName string) (*TaskAdaptor, *relaycommon.RelayInfo) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeSora,
			ChannelId:         987,
			ChannelBaseUrl:    "https://vinted.cam",
			ApiKey:            "upstream-key",
			UpstreamModelName: modelName,
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	return adaptor, info
}

func TestSoraBuildRequestBodyReturnsReplayablePassThroughBody(t *testing.T) {
	payload := []byte("opaque-sora-request-body")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/octet-stream")
	defer common.CleanupBodyStorage(c)

	info := &relaycommon.RelayInfo{}
	body, err := (&TaskAdaptor{}).BuildRequestBody(c, info)
	require.NoError(t, err)
	replayable, ok := body.(common.ReplayableBody)
	require.True(t, ok)

	sent, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, payload, sent)
	assert.EqualValues(t, len(payload), replayable.Size())

	replayBody, err := replayable.NewReader()
	require.NoError(t, err)
	replay, err := io.ReadAll(replayBody)
	require.NoError(t, err)
	require.NoError(t, replayBody.Close())
	assert.Equal(t, payload, replay)
}

func TestSoraBuildRequestBodyKeepsMinimaxH3ImagesArrayUnchanged(t *testing.T) {
	payload := `{"model":"minimax-h3","prompt":"test","seconds":5,"size":"1376x768","images":["https://example.com/first.png","https://example.com/second.png"]}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	defer common.CleanupBodyStorage(c)

	body, err := (&TaskAdaptor{}).BuildRequestBody(c, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "minimax_h3"},
	})
	require.NoError(t, err)
	responseBody, err := io.ReadAll(body)
	require.NoError(t, err)

	assert.Equal(t, "application/json", c.Request.Header.Get("Content-Type"))
	var request map[string]interface{}
	require.NoError(t, common.Unmarshal(responseBody, &request))
	assert.Equal(t, "minimax_h3", request["model"])
	assert.Equal(t, "test", request["prompt"])
	assert.Equal(t, float64(5), request["seconds"])
	assert.Equal(t, "1376x768", request["size"])
	assert.Equal(t, []interface{}{
		"https://example.com/first.png",
		"https://example.com/second.png",
	}, request["images"])
	assert.NotContains(t, request, "input_reference")
	assert.NotContains(t, request, "reference_images")
}

func TestSoraDoResponseAcceptsNumericAndStringSeconds(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantSeconds any
	}{
		{
			name:        "numeric seconds",
			body:        `{"id":"vidtask_upstream","status":"queued","seconds":10,"video_available":false}`,
			wantSeconds: float64(10),
		},
		{
			name:        "string seconds",
			body:        `{"id":"vidtask_upstream","status":"queued","seconds":"10"}`,
			wantSeconds: "10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			info := &relaycommon.RelayInfo{
				TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
			}

			upstreamID, taskData, taskErr := (&TaskAdaptor{}).DoResponse(c, &http.Response{
				StatusCode: http.StatusAccepted,
				Body:       io.NopCloser(bytes.NewBufferString(tt.body)),
			}, info)

			require.Nil(t, taskErr)
			assert.Equal(t, "vidtask_upstream", upstreamID)
			assert.JSONEq(t, tt.body, string(taskData))
			assert.Equal(t, http.StatusOK, recorder.Code)
			var clientResponse map[string]any
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &clientResponse))
			assert.Equal(t, "task_public", clientResponse["id"])
			assert.Equal(t, "task_public", clientResponse["task_id"])
			assert.Equal(t, tt.wantSeconds, clientResponse["seconds"])
		})
	}
}

func TestSoraParseTaskResultWaitsForVideoAvailability(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus string
	}{
		{
			name:       "numeric seconds while queued",
			body:       `{"id":"vidtask_upstream","status":"queued","seconds":10,"video_available":false}`,
			wantStatus: model.TaskStatusQueued,
		},
		{
			name:       "string seconds while processing",
			body:       `{"id":"vidtask_upstream","status":"processing","seconds":"10"}`,
			wantStatus: model.TaskStatusInProgress,
		},
		{
			name:       "completed before content is ready",
			body:       `{"id":"vidtask_upstream","status":"completed","seconds":10,"asset_state":"generating","video_available":false}`,
			wantStatus: model.TaskStatusInProgress,
		},
		{
			name:       "completed with content ready",
			body:       `{"id":"vidtask_upstream","status":"completed","seconds":10,"video_available":true}`,
			wantStatus: model.TaskStatusSuccess,
		},
		{
			name:       "completed without availability field",
			body:       `{"id":"task_other_provider","status":"completed","seconds":"10"}`,
			wantStatus: model.TaskStatusSuccess,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := (&TaskAdaptor{}).ParseTaskResult([]byte(tt.body))
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantStatus, result.Status)
		})
	}
}

func TestSoraConvertToOpenAIVideoDoesNotExposePrematureCompletion(t *testing.T) {
	tests := []struct {
		name       string
		status     model.TaskStatus
		body       string
		wantStatus string
	}{
		{
			name:       "content still generating",
			status:     model.TaskStatusInProgress,
			body:       `{"id":"vidtask_upstream","status":"completed","video_available":false}`,
			wantStatus: "processing",
		},
		{
			name:       "content ready",
			status:     model.TaskStatusSuccess,
			body:       `{"id":"vidtask_upstream","status":"completed","video_available":true}`,
			wantStatus: "completed",
		},
		{
			name:       "other provider without availability field",
			status:     model.TaskStatusSuccess,
			body:       `{"id":"video_upstream","status":"completed"}`,
			wantStatus: "completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &model.Task{TaskID: "task_public", Status: tt.status, Data: []byte(tt.body)}
			responseBody, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
			require.NoError(t, err)
			var response map[string]any
			require.NoError(t, common.Unmarshal(responseBody, &response))
			assert.Equal(t, "task_public", response["id"])
			assert.Equal(t, tt.wantStatus, response["status"])
		})
	}
}

func TestSoraParseTaskResultKeepsMinimaxH3UnknownTaskInProgress(t *testing.T) {
	taskResult, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"task_minimax_h3",
		"object":"video",
		"model":"minimax_h3",
		"status":"unknown",
		"progress":0,
		"metadata":{"url":""}
	}`))

	require.NoError(t, err)
	require.NotNil(t, taskResult)
	assert.Equal(t, model.TaskStatusInProgress, taskResult.Status)
}

func TestSoraParseTaskResultDoesNotGeneralizeUnknownStatus(t *testing.T) {
	taskResult, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"task_other_model",
		"object":"video",
		"model":"other-model",
		"status":"unknown"
	}`))

	require.NoError(t, err)
	require.NotNil(t, taskResult)
	assert.Empty(t, taskResult.Status)
}

func TestSoraParseTaskResultKeepsMinimaxH3FailureTerminal(t *testing.T) {
	taskResult, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{
		"id":"task_minimax_h3_failed",
		"object":"video",
		"model":"minimax_h3",
		"status":"failed",
		"error":{"message":"provider rejected request"}
	}`))

	require.NoError(t, err)
	require.NotNil(t, taskResult)
	assert.Equal(t, model.TaskStatusFailure, taskResult.Status)
	assert.Equal(t, "provider rejected request", taskResult.Reason)
}

func TestSoraParseTaskResultPreservesTerminalFailureReason(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus string
		wantReason string
	}{
		{
			name:       "string error from upstream",
			body:       `{"id":"job_1","status":"failed","progress":30,"error":"生成服务报告视频生成异常"}`,
			wantStatus: model.TaskStatusFailure,
			wantReason: "生成服务报告视频生成异常",
		},
		{
			name:       "object error from upstream",
			body:       `{"id":"job_1","status":"failed","error":{"message":"provider rejected request","code":"rejected"}}`,
			wantStatus: model.TaskStatusFailure,
			wantReason: "provider rejected request",
		},
		{
			name:       "cancelled with string error",
			body:       `{"id":"job_1","status":"cancelled","error":"user cancelled"}`,
			wantStatus: model.TaskStatusFailure,
			wantReason: "user cancelled",
		},
		{
			name:       "failed without error",
			body:       `{"id":"job_1","status":"failed","error":null}`,
			wantStatus: model.TaskStatusFailure,
			wantReason: "task failed",
		},
		{
			name:       "processing with null error",
			body:       `{"id":"job_1","status":"processing","error":null}`,
			wantStatus: model.TaskStatusInProgress,
		},
		{
			name:       "minimax unknown with null error",
			body:       `{"id":"job_1","model":"minimax_h3","status":"unknown","error":null}`,
			wantStatus: model.TaskStatusInProgress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := (&TaskAdaptor{}).ParseTaskResult([]byte(tt.body))
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Equal(t, tt.wantReason, result.Reason)
		})
	}
}

func TestSoraDoResponseAcceptsStringError(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"}}
	upstreamBody := `{"id":"job_1","status":"failed","error":"provider rejected request"}`

	upstreamID, taskData, taskErr := (&TaskAdaptor{}).DoResponse(c, &http.Response{
		StatusCode: http.StatusAccepted,
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}, info)

	require.Nil(t, taskErr)
	assert.Equal(t, "job_1", upstreamID)
	assert.JSONEq(t, upstreamBody, string(taskData))
	var response map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "task_public", response["id"])
	assert.Equal(t, "provider rejected request", response["error"])
}

func TestVintedRejectsUnsupportedRemix(t *testing.T) {
	c, _ := newVintedTestContext(t, `{"prompt":"change the scene"}`)
	adaptor, info := newVintedAdaptorInfo("seedance2.0")
	info.Action = constant.TaskActionRemix

	taskErr := adaptor.ValidateRequestAndSetAction(c, info)

	require.NotNil(t, taskErr)
	assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
	assert.Contains(t, taskErr.Message, "does not support remix")
}
func TestVintedBuildRequestHeaderUsesGatewayScopedIdempotencyKey(t *testing.T) {
	tests := []struct {
		name        string
		clientKey   string
		requestID   string
		publicID    string
		upstreamKey string
		wantHeader  string
	}{
		{name: "reserved gateway key", clientKey: "client-operation", upstreamKey: "newapi-reserved", wantHeader: "newapi-reserved"},
		{name: "request id fallback", requestID: "request-operation"},
		{name: "public task id fallback", publicID: "task_public"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newVintedTestContext(t, `{"model":"seedance2.0","prompt":"test","duration":5}`)
			c.Request.Header.Set("Idempotency-Key", tt.clientKey)
			adaptor, info := newVintedAdaptorInfo("seedance2.0")
			adaptor.vintedQuote = "quote-test"
			info.RequestId = tt.requestID
			info.PublicTaskID = tt.publicID
			info.UpstreamIdempotencyKey = tt.upstreamKey
			upstreamRequest := httptest.NewRequest(http.MethodPost, "https://vinted.example/v1/videos", nil)

			require.NoError(t, adaptor.BuildRequestHeader(c, upstreamRequest, info))
			assert.Equal(t, "Bearer upstream-key", upstreamRequest.Header.Get("Authorization"))
			assert.Equal(t, "application/json", upstreamRequest.Header.Get("Content-Type"))
			assert.Equal(t, "quote-test", upstreamRequest.Header.Get("Expected-Quote-Id"))
			actual := upstreamRequest.Header.Get("Idempotency-Key")
			if tt.wantHeader != "" {
				assert.Equal(t, tt.wantHeader, actual)
			} else {
				assert.True(t, strings.HasPrefix(actual, "newapi-"))
				assert.NotEqual(t, tt.clientKey, actual)
			}
		})
	}
}

func TestVintedDoResponsePreservesAcceptedStatusAndHidesUpstreamID(t *testing.T) {
	c, recorder := newVintedTestContext(t, `{"model":"seedance2.0","prompt":"test","duration":5}`)
	adaptor, info := newVintedAdaptorInfo("seedance2.0")
	info.PublicTaskID = "task_public"
	responseBody := `{"id":"job_upstream","status":"queued","idempotent":"1"}`

	upstreamID, taskData, taskErr := adaptor.DoResponse(c, &http.Response{
		StatusCode: http.StatusAccepted,
		Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
	}, info)

	require.Nil(t, taskErr)
	assert.Equal(t, "job_upstream", upstreamID)
	assert.JSONEq(t, responseBody, string(taskData))
	assert.Empty(t, recorder.Body.String(), "the adaptor must not acknowledge a task before it is persisted")
	assert.Equal(t, http.StatusAccepted, info.TaskClientResponseStatus)
	assert.JSONEq(t, `{"id":"task_public","status":"queued","idempotent":"1"}`, string(info.TaskClientResponseBody))
}

func TestVintedParseTaskResultUsesProviderStatusesAndStableProxyFlow(t *testing.T) {
	adaptor, _ := newVintedAdaptorInfo("seedance2.0")
	tests := []struct {
		name       string
		body       string
		wantStatus string
		wantReason string
		wantRemote string
	}{
		{name: "queued", body: `{"id":"job_1","status":"queued","file":"","download_url":"","error":""}`, wantStatus: model.TaskStatusQueued},
		{name: "running", body: `{"id":"job_1","status":"running","file":"","download_url":"","error":""}`, wantStatus: model.TaskStatusInProgress},
		{name: "succeeded awaiting archive", body: `{"id":"job_1","status":"succeeded","has_file":false}`, wantStatus: model.TaskStatusInProgress},
		{name: "succeeded file", body: `{"id":"job_1","status":"succeeded","file":"/v1/videos/job_1/file?exp=1&sig=x","download_url":"","error":""}`, wantStatus: model.TaskStatusSuccess, wantRemote: "/v1/videos/job_1/file?exp=1&sig=x"},
		{name: "succeeded download URL", body: `{"id":"job_1","status":"succeeded","file":"","download_url":"https://vinted.example/video.mp4","error":""}`, wantStatus: model.TaskStatusSuccess, wantRemote: "https://vinted.example/video.mp4"},
		{name: "failed", body: `{"id":"job_1","status":"failed","file":"","download_url":"","error":"provider rejected prompt"}`, wantStatus: model.TaskStatusFailure, wantReason: "provider rejected prompt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := adaptor.ParseTaskResult([]byte(tt.body))
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Equal(t, tt.wantReason, result.Reason)
			assert.Equal(t, tt.wantRemote, result.RemoteUrl)
			assert.Empty(t, result.Url, "expiring Vinted URLs must not be persisted as the public result URL")
		})
	}

	nonVintedResult, err := (&TaskAdaptor{}).ParseTaskResult([]byte(`{"id":"job_1","status":"succeeded"}`))
	require.NoError(t, err)
	assert.Empty(t, nonVintedResult.Status, "Vinted status aliases must not affect other Sora channels")
}

func TestVintedPollingUsesSubmissionEndpointAndCredentialSnapshot(t *testing.T) {
	var requestedPath string
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.EscapedPath()
		authorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"job/upstream","status":"queued"}`))
	}))
	t.Cleanup(server.Close)

	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelType:          constant.ChannelTypeSora,
		ChannelBaseUrl:       server.URL + "/gateway",
		ApiKey:               "submit-key",
		ChannelOtherSettings: relaykitdto.ChannelOtherSettings{VideoProtocol: relaykitdto.VideoProtocolVinted},
	}}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	config, err := adaptor.TaskPollingConfig("job/upstream")
	require.NoError(t, err)
	assert.Equal(t, relaykitdto.VideoProtocolVinted, config.VideoProtocol)

	changedAdaptor := &TaskAdaptor{}
	changedAdaptor.Init(&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelType:          constant.ChannelTypeSora,
		ChannelBaseUrl:       "https://changed.example.com",
		ApiKey:               "changed-key",
		ChannelOtherSettings: relaykitdto.ChannelOtherSettings{VideoProtocol: relaykitdto.VideoProtocolOpenAI},
	}})
	resp, err := changedAdaptor.FetchTask("https://changed.example.com", "changed-key", map[string]any{
		"task_id":        "job/upstream",
		"polling_config": config,
	}, "")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	assert.Equal(t, "/gateway/v1/videos/job%2Fupstream", requestedPath)
	assert.Equal(t, "Bearer submit-key", authorization)
}

func TestSoraVideoQualityResolutionUsesRequestFieldsAndPortraitSizes(t *testing.T) {
	for _, tc := range []struct{ name, fields, want string }{
		{"flat resolution", `"resolution":"480p"`, "480p"},
		{"metadata resolution", `"metadata":{"resolution":"768p"}`, "768p"},
		{"JSON string metadata", `"metadata":"{\"resolution\":\"720p\"}"`, "720p"},
		{"portrait size", `"size":"1080x1920"`, "1080p"},
		{"2K resolution", `"resolution":"2K"`, "2K"},
		{"default", `"duration":5`, "720p"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newVintedTestContext(t, `{"model":"canvas-alias","prompt":"test",`+tc.fields+`}`)
			info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
			a := &TaskAdaptor{}
			require.Nil(t, a.ValidateRequestAndSetAction(ctx, info))
			resolution, err := a.VideoBillingResolution(ctx, info)
			require.NoError(t, err)
			assert.Equal(t, tc.want, resolution)
		})
	}
}

func TestSoraRemixBillingUsesOriginalQualityInsteadOfNewTaskDefault(t *testing.T) {
	for _, originalResolution := range []string{"1080p", ""} {
		ctx, _ := newVintedTestContext(t, `{"prompt":"change the lighting"}`)
		info := &relaycommon.RelayInfo{
			TaskRelayInfo:       &relaycommon.TaskRelayInfo{Action: constant.TaskActionRemix},
			TaskVideoResolution: originalResolution,
		}
		a := &TaskAdaptor{}
		require.Nil(t, a.ValidateRequestAndSetAction(ctx, info))
		resolution, err := a.VideoBillingResolution(ctx, info)
		require.NoError(t, err)
		assert.Equal(t, originalResolution, resolution)
	}
}
