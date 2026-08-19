package sora

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
