package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDifferencesSupportsTaskDurationMultiplier(t *testing.T) {
	local := map[string]any{
		billing_setting.TaskDurationMultiplierField: map[string]bool{"video-model": false},
	}
	upstreams := []struct {
		name string
		data map[string]any
	}{
		{
			name: "upstream",
			data: map[string]any{
				billing_setting.TaskDurationMultiplierField: map[string]bool{"video-model": true},
			},
		},
	}

	differences := buildDifferences(local, upstreams)
	item, ok := differences["video-model"][billing_setting.TaskDurationMultiplierField]

	require.True(t, ok)
	assert.Equal(t, false, item.Current)
	assert.Equal(t, true, item.Upstreams["upstream"])
}

func TestBuildDifferencesSupportsVideoQualityPriceMaps(t *testing.T) {
	local := map[string]any{billing_setting.VideoResolutionPricesField: map[string]map[string]float64{"video": {"720p": 0.1}}}
	upstream := []struct {
		name string
		data map[string]any
	}{{name: "upstream", data: map[string]any{billing_setting.VideoResolutionPricesField: map[string]map[string]float64{"video": {"720p": 0, "768p": 0.2}}}}}
	differences := buildDifferences(local, upstream)
	item, ok := differences["video"][billing_setting.VideoResolutionPricesField]
	require.True(t, ok)
	assert.Equal(t, map[string]float64{"720p": 0.1}, item.Current)
	assert.Equal(t, map[string]float64{"720p": 0, "768p": 0.2}, item.Upstreams["upstream"])
}
