package billing_setting

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
)

func TestVideoResolutionPricesValidateStoredContract(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		valid       bool
	}{
		{"all tiers including free", `{"mapped-alias":{"480p":0,"720p":0.1,"768p":0.2,"1080p":0.3,"2k":0.4}}`, true},
		{"clear", `{}`, true}, {"null object", `null`, false}, {"null model", `{"video":null}`, false},
		{"null price", `{"video":{"720p":null}}`, false}, {"negative", `{"video":{"720p":-1}}`, false},
		{"string price", `{"video":{"720p":"1"}}`, false}, {"overflow", `{"video":{"720p":1e400}}`, false},
		{"unsupported tier", `{"video":{"4k":1}}`, false}, {"empty model", `{" ":{}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateVideoResolutionPricesJSON(tc.value)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestVideoResolutionPricesNormalizeAndKeepZero(t *testing.T) {
	original := billingSetting.VideoResolutionPrices
	t.Cleanup(func() { billingSetting.VideoResolutionPrices = original })
	billingSetting.VideoResolutionPrices = map[string]map[string]float64{"canvas-alias": {"480p": 0, "720p": 0.1, "768p": 0.2, "1080p": 0.3, "2k": 0.4, "bad": math.Inf(1)}}
	for _, tc := range []struct {
		input string
		price float64
	}{{"480p", 0}, {"720P", 0.1}, {"768p", 0.2}, {"1080p", 0.3}, {"2K", 0.4}, {"1440p", 0.4}} {
		price, ok := GetVideoResolutionPrice("canvas-alias", tc.input)
		require.True(t, ok)
		assert.Equal(t, tc.price, price)
	}
	_, ok := GetVideoResolutionPrice("upstream-mapped-target", "720p")
	assert.False(t, ok)
	_, ok = GetVideoResolutionPrice("canvas-alias", "")
	assert.False(t, ok)
	_, ok = GetVideoResolutionPrice("canvas-alias", "4k")
	assert.False(t, ok)
	exported := GetPricingSyncData(nil)[VideoResolutionPricesField].(map[string]map[string]float64)
	exported["canvas-alias"]["720p"] = 100
	price, ok := GetVideoResolutionPrice("canvas-alias", "720p")
	require.True(t, ok)
	assert.Equal(t, 0.1, price)
}

func TestVideoResolutionSizesDoNotConfusePortraitWithQuality(t *testing.T) {
	for _, tc := range []struct{ size, resolution string }{{"1920x1080", "1080p"}, {"1080x1920", "1080p"}, {"1280*720", "720p"}, {"1366x768", "768p"}, {"2560x1440", "2k"}, {"2K", "2k"}, {"16:9", ""}, {"0x720", ""}} {
		assert.Equal(t, tc.resolution, VideoResolutionFromSize(tc.size), tc.size)
	}
}
