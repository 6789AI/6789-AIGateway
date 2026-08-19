package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	projecti18n "github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newImageGenerationContext(t *testing.T, body string) *gin.Context {
	t.Helper()
	require.NoError(t, projecti18n.Init())
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	t.Cleanup(func() { common.CleanupBodyStorage(ctx) })
	return ctx
}

func TestGetModelRequestAsyncImageOptIn(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		prefer      string
		wantAsync   bool
		wantMode    int
		wantSelPath string
	}{
		{
			name:     "synchronous by default",
			body:     `{"model":"wanx-v1","prompt":"draw"}`,
			wantMode: relayconstant.RelayModeImagesGenerations,
		},
		{
			name:        "json opt in",
			body:        `{"model":"wanx-v1","prompt":"draw","async":true}`,
			wantAsync:   true,
			wantMode:    relayconstant.RelayModeImageSubmit,
			wantSelPath: relayconstant.AsyncImageGenerationSelectionPath,
		},
		{
			name:        "prefer opt in overrides explicit false",
			body:        `{"model":"wanx-v1","prompt":"draw","async":false}`,
			prefer:      "return=minimal, RESPOND-ASYNC; handling=strict",
			wantAsync:   true,
			wantMode:    relayconstant.RelayModeImageSubmit,
			wantSelPath: relayconstant.AsyncImageGenerationSelectionPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newImageGenerationContext(t, tt.body)
			if tt.prefer != "" {
				ctx.Request.Header.Set("Prefer", tt.prefer)
			}

			request, shouldSelect, err := getModelRequest(ctx)
			require.NoError(t, err)
			require.True(t, shouldSelect)
			assert.Equal(t, "wanx-v1", request.Model)
			assert.Equal(t, tt.wantAsync, common.GetContextKeyBool(ctx, constant.ContextKeyAsyncImageRequest))
			assert.Equal(t, tt.wantSelPath, common.GetContextKeyString(ctx, constant.ContextKeyChannelSelectionPath))
			assert.Equal(t, tt.wantMode, ctx.GetInt("relay_mode"))
		})
	}
}

func TestGetModelRequestRejectsNonBooleanImageAsync(t *testing.T) {
	ctx := newImageGenerationContext(t, `{"model":"wanx-v1","prompt":"draw","async":"true"}`)
	_, _, err := getModelRequest(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field async must be a boolean")
}

func TestGetModelRequestImageFetchSkipsChannelSelection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/images/generations/task-123", nil)
	ctx.Params = gin.Params{{Key: "task_id", Value: "task-123"}}

	_, shouldSelect, err := getModelRequest(ctx)
	require.NoError(t, err)
	assert.False(t, shouldSelect)
	assert.Equal(t, relayconstant.RelayModeImageFetch, ctx.GetInt("relay_mode"))
}

func TestChannelSupportsAsyncImageSelectionForNativeProviders(t *testing.T) {
	assert.True(t, channelSupportsRequestPath(
		&model.Channel{Type: constant.ChannelTypeAli},
		relayconstant.AsyncImageGenerationSelectionPath,
		"wanx-v1",
	))
	grsaiBaseURL := "https://grsaiapi.com"
	assert.True(t, channelSupportsRequestPath(
		&model.Channel{Type: constant.ChannelTypeGemini, BaseURL: &grsaiBaseURL},
		relayconstant.AsyncImageGenerationSelectionPath,
		"nano-banana-2",
	))
	assert.False(t, channelSupportsRequestPath(
		&model.Channel{Type: constant.ChannelTypeOpenAI},
		relayconstant.AsyncImageGenerationSelectionPath,
		"wanx-v1",
	))
}
