package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestChannelAffinityUsesAsyncImageSelectionPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setting := operation_setting.GetChannelAffinitySetting()
	require.NotNil(t, setting)

	rule := operation_setting.ChannelAffinityRule{
		Name:       "async image affinity",
		ModelRegex: []string{"^wanx-v1$"},
		PathRegex:  []string{"^/v1/images/generations:async$"},
		KeySources: []operation_setting.ChannelAffinityKeySource{
			{Type: "request_header", Key: "X-Image-Affinity"},
		},
		IncludeRuleName: true,
	}
	originalEnabled := setting.Enabled
	originalRules := setting.Rules
	setting.Enabled = true
	setting.Rules = append([]operation_setting.ChannelAffinityRule{rule}, originalRules...)
	t.Cleanup(func() {
		setting.Enabled = originalEnabled
		setting.Rules = originalRules
	})

	affinityValue := fmt.Sprintf("async-image-%d", time.Now().UnixNano())
	cacheKey := buildChannelAffinityCacheKeySuffix(rule, "wanx-v1", "default", affinityValue)
	cache := getChannelAffinityCache()
	require.NoError(t, cache.SetWithTTL(cacheKey, 1717, time.Minute))
	t.Cleanup(func() { _, _ = cache.DeleteMany([]string{cacheKey}) })

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	ctx.Request.Header.Set("X-Image-Affinity", affinityValue)
	common.SetContextKey(ctx, constant.ContextKeyChannelSelectionPath, relayconstant.AsyncImageGenerationSelectionPath)

	channelID, found := GetPreferredChannelByAffinity(ctx, "wanx-v1", "default")
	require.True(t, found)
	require.Equal(t, 1717, channelID)
	meta, ok := getChannelAffinityMeta(ctx)
	require.True(t, ok)
	require.Equal(t, relayconstant.AsyncImageGenerationSelectionPath, meta.RequestPath)
}
