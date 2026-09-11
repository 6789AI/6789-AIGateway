package sora

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestVintedBuildRequestBodyUsesProviderProtocol(t *testing.T) {
	payload := `{
		"model":"seedance2.0fast",
		"prompt":"product showcase",
		"duration":10,
		"ratio":"16:9",
		"resolution":"720p",
		"image_urls":["https://example.com/first.png","https://example.com/second.png"],
		"seconds":"15",
		"metadata":{"unexpected":true}
	}`
	c, _ := newVintedTestContext(t, payload)
	adaptor, info := newVintedAdaptorInfo("seedance2.0fast")

	reqErr := adaptor.ValidateRequestAndSetAction(c, info)
	require.Nil(t, reqErr)
	require.Nil(t, adaptor.ValidateMappedTaskRequest(c, info))
	body, err := adaptor.BuildRequestBody(c, info)
	require.NoError(t, err)
	encoded, err := io.ReadAll(body)
	require.NoError(t, err)

	var request map[string]interface{}
	require.NoError(t, common.Unmarshal(encoded, &request))
	assert.Equal(t, "seedance2.0fast", request["model"])
	assert.Equal(t, "product showcase", request["prompt"])
	assert.Equal(t, float64(10), request["duration"])
	assert.Equal(t, "16:9", request["ratio"])
	assert.Equal(t, "720p", request["resolution"])
	assert.Equal(t, []interface{}{
		"https://example.com/first.png",
		"https://example.com/second.png",
	}, request["image_urls"])
	assert.Len(t, request, 6)
	assert.NotContains(t, request, "seconds")
	assert.NotContains(t, request, "metadata")
	assert.Equal(t, constant.TaskActionGenerate, info.Action)

	taskRequest, err := relaycommon.GetTaskRequest(c)
	require.NoError(t, err)
	assert.Equal(t, 10, taskRequest.Duration)
	assert.Equal(t, []string{
		"https://example.com/first.png",
		"https://example.com/second.png",
	}, taskRequest.Images)
	assert.Equal(t, map[string]float64{"seconds": 10, "size": 1}, adaptor.EstimateBilling(c, info))
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

func TestVintedBuildRequestBodyMergesReferenceAliases(t *testing.T) {
	payload := `{
		"model":"client-video-alias",
		"prompt":"product showcase",
		"duration":30,
		"camera_movement":"fixed",
		"image_ids":["img_first","img_first"],
		"image":"https://example.com/first.png",
		"images":["https://example.com/second.png"],
		"image_refs":["https://example.com/first.png"],
		"image_url":"https://example.com/third.png",
		"image_urls":["https://example.com/second.png"]
	}`
	c, _ := newVintedTestContext(t, payload)
	adaptor, info := newVintedAdaptorInfo("seedance2.5")

	require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info), "structural validation must run before model mapping")
	require.Nil(t, adaptor.ValidateMappedTaskRequest(c, info))
	body, err := adaptor.BuildRequestBody(c, info)
	require.NoError(t, err)
	encoded, err := io.ReadAll(body)
	require.NoError(t, err)

	var request vintedUpstreamRequest
	require.NoError(t, common.Unmarshal(encoded, &request))
	assert.Equal(t, "seedance2.5", request.Model)
	assert.Equal(t, "fixed", request.CameraMovement)
	assert.Equal(t, []string{"img_first"}, request.ImageIDs)
	assert.Equal(t, []string{
		"https://example.com/first.png",
		"https://example.com/third.png",
		"https://example.com/second.png",
	}, request.ImageURLs)
	assert.Equal(t, constant.TaskActionGenerate, info.Action)
}

func TestVintedReferenceLimitCountsIDsAndDeduplicatedURLsTogether(t *testing.T) {
	images := make([]string, 9)
	for i := range images {
		images[i] = `"https://example.com/` + strconv.Itoa(i) + `.png"`
	}
	payload := `{"model":"seedance2.0","prompt":"test","duration":5,"image_ids":["img_1"],"images":[` + strings.Join(images, ",") + `]}`
	c, _ := newVintedTestContext(t, payload)
	adaptor, info := newVintedAdaptorInfo("seedance2.0")

	require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))
	taskErr := adaptor.ValidateMappedTaskRequest(c, info)
	require.NotNil(t, taskErr)
	assert.Contains(t, taskErr.Message, "at most 9 reference images")
}

func TestVintedRequestValidationMatchesProviderLimits(t *testing.T) {
	tests := []struct {
		name          string
		payload       string
		upstreamModel string
		wantErr       string
	}{
		{
			name:          "unsupported legacy model",
			payload:       `{"model":"seedance-2.0","prompt":"test","duration":5,"resolution":"720p"}`,
			upstreamModel: "seedance-2.0",
			wantErr:       "unsupported Vinted video model",
		},
		{
			name:          "invalid model duration combination",
			payload:       `{"model":"seedance2.5","prompt":"test","duration":15,"resolution":"720p"}`,
			upstreamModel: "seedance2.5",
			wantErr:       "duration 15 is not supported",
		},
		{
			name:          "invalid ratio",
			payload:       `{"model":"seedance2.0","prompt":"test","duration":5,"ratio":"2:1","resolution":"720p"}`,
			upstreamModel: "seedance2.0",
			wantErr:       "unsupported Vinted video ratio",
		},
		{
			name:          "invalid resolution",
			payload:       `{"model":"seedance2.0","prompt":"test","duration":5,"resolution":"1080p"}`,
			upstreamModel: "seedance2.0",
			wantErr:       "unsupported Vinted video resolution",
		},
		{
			name:          "too many references",
			upstreamModel: "seedance2.0mini",
			payload: func() string {
				images := make([]string, 10)
				for i := range images {
					images[i] = `"https://example.com/reference-` + strconv.Itoa(i) + `.png"`
				}
				return `{"model":"seedance2.0mini","prompt":"test","duration":5,"resolution":"720p","image_urls":[` + strings.Join(images, ",") + `]}`
			}(),
			wantErr: "supports at most 9 reference images",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newVintedTestContext(t, tt.payload)
			adaptor, info := newVintedAdaptorInfo(tt.upstreamModel)
			taskErr := adaptor.ValidateRequestAndSetAction(c, info)
			require.Nil(t, taskErr, "structural validation must complete before mapped-model validation")
			taskErr = adaptor.ValidateMappedTaskRequest(c, info)
			require.NotNil(t, taskErr)
			assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
			assert.Contains(t, taskErr.Message, tt.wantErr)
		})
	}
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
			info.RequestId = tt.requestID
			info.PublicTaskID = tt.publicID
			info.UpstreamIdempotencyKey = tt.upstreamKey
			upstreamRequest := httptest.NewRequest(http.MethodPost, "https://vinted.example/v1/videos", nil)

			require.NoError(t, adaptor.BuildRequestHeader(c, upstreamRequest, info))
			assert.Equal(t, "Bearer upstream-key", upstreamRequest.Header.Get("Authorization"))
			assert.Equal(t, "application/json", upstreamRequest.Header.Get("Content-Type"))
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
