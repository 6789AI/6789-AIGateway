package newapi

import (
	"bytes"
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

func newImageContext(method, target string, body []byte) *gin.Context {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, target, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c
}

func TestBuildRequestBodyPreservesFieldsAndForcesAsync(t *testing.T) {
	body := []byte(`{"model":"image-alias","prompt":"draw","custom_field":{"enabled":true},"stream":false}`)
	c := newImageContext(http.MethodPost, imagePath, body)
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeImageSubmit,
		OriginModelName: "image-alias",
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "upstream-image",
		},
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))

	requestBody, err := adaptor.BuildRequestBody(c, info)
	require.NoError(t, err)
	encoded, err := io.ReadAll(requestBody)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, common.Unmarshal(encoded, &payload))
	assert.Equal(t, "upstream-image", payload["model"])
	assert.Equal(t, true, payload["async"])
	assert.NotContains(t, payload, "stream")
	assert.Equal(t, map[string]any{"enabled": true}, payload["custom_field"])
}

func TestAdvancedCustomRouteAndPollingConfigAreSnapshotted(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeAdvancedCustom,
			ChannelBaseUrl:    "https://images.example.com/root",
			ApiKey:            "secret",
			UpstreamModelName: "image-model",
			ChannelOtherSettings: dto.ChannelOtherSettings{
				AdvancedCustom: &dto.AdvancedCustomConfig{Routes: []dto.AdvancedCustomRoute{
					{
						IncomingPath: imagePath,
						UpstreamPath: "/gateway/images",
						Converter:    "none",
						Auth: &dto.AdvancedCustomRouteAuth{
							Type:  dto.AdvancedCustomAuthTypeHeader,
							Name:  "X-API-Key",
							Value: "{api_key}",
						},
					},
				}},
			},
		},
	}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	requestURL, err := adaptor.BuildRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://images.example.com/root/gateway/images", requestURL)

	c := newImageContext(http.MethodPost, imagePath, nil)
	req := httptest.NewRequest(http.MethodPost, requestURL, nil)
	require.NoError(t, adaptor.BuildRequestHeader(c, req, info))
	assert.Equal(t, "secret", req.Header.Get("X-API-Key"))
	assert.Empty(t, req.Header.Get("Authorization"))

	config, err := adaptor.TaskPollingConfig("upstream-task")
	require.NoError(t, err)
	assert.Equal(t, "https://images.example.com/root/gateway/images/upstream-task", config.URL)
	assert.Equal(t, []string{"secret"}, config.Headers["X-Api-Key"])
}

func TestDoResponseAcceptsTaskIDAndPollingUsesSnapshot(t *testing.T) {
	var requestedPath string
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.EscapedPath()
		authorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"upstream-1","task_id":"upstream-1","status":"queued","progress":0,"created_at":1,"model":"image-model"}`))
	}))
	t.Cleanup(server.Close)

	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelType:    constant.ChannelTypeNewAPI,
		ChannelBaseUrl: server.URL,
		ApiKey:         "submit-key",
	}}
	adaptor := &TaskAdaptor{}
	adaptor.Init(info)
	require.Equal(t, server.URL+imagePath, func() string {
		value, err := adaptor.BuildRequestURL(info)
		require.NoError(t, err)
		return value
	}())
	c := newImageContext(http.MethodPost, imagePath, nil)
	req := httptest.NewRequest(http.MethodPost, server.URL+imagePath, nil)
	require.NoError(t, adaptor.BuildRequestHeader(c, req, info))

	upstream := httptest.NewRecorder()
	upstream.Code = http.StatusAccepted
	_, _ = upstream.WriteString(`{"id":"fallback-id","task_id":"upstream-1","status":"queued"}`)
	taskID, _, taskErr := adaptor.DoResponse(c, upstream.Result(), info)
	require.Nil(t, taskErr)
	assert.Equal(t, "upstream-1", taskID)

	config, err := adaptor.TaskPollingConfig(taskID)
	require.NoError(t, err)
	resp, err := adaptor.FetchTask("https://changed.example.com", "changed-key", map[string]any{
		"task_id":        taskID,
		"polling_config": config,
	}, "")
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, imagePath+"/upstream-1", requestedPath)
	assert.Equal(t, "Bearer submit-key", authorization)
}

func TestParseTaskResultMapsStatusAndCapsImageCount(t *testing.T) {
	data := make([]dto.ImageData, dto.MaxImageN+2)
	for index := range data {
		data[index].Url = "https://images.example.com/result.png"
	}
	body, err := common.Marshal(dto.ImageTaskResponse{
		ID:       "upstream-1",
		TaskID:   "upstream-1",
		Status:   dto.ImageTaskStatusCompleted,
		Progress: 100,
		Data:     data,
	})
	require.NoError(t, err)

	result, err := (&TaskAdaptor{}).ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, result.Status)
	assert.Equal(t, float64(dto.MaxImageN), result.BillingRatios["n"])
	assert.Equal(t, "https://images.example.com/result.png", result.Url)
}
