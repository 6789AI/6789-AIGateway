package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestFilterChannelsForAsyncImageSelection(t *testing.T) {
	oldChannels := channelsIDM
	oldConfigs := channel2advancedCustomConfig
	toSyncModel := `{"image-alias":"qwen-image-plus"}`
	toAsyncModel := `{"image-alias":"wanx-v1"}`
	toGrsaiModel := `{"image-alias":"nano-banana-2"}`
	grsaiBaseURL := "https://grsaiapi.com"
	channelsIDM = map[int]*Channel{
		1: {Id: 1, Type: constant.ChannelTypeAli, ModelMapping: &toSyncModel},
		2: {Id: 2, Type: constant.ChannelTypeAli, ModelMapping: &toAsyncModel},
		3: {Id: 3, Type: constant.ChannelTypeOpenAI},
		4: {Id: 4, Type: constant.ChannelTypeAdvancedCustom},
		5: {Id: 5, Type: constant.ChannelTypeGemini, BaseURL: &grsaiBaseURL, ModelMapping: &toGrsaiModel},
	}
	channel2advancedCustomConfig = nil
	t.Cleanup(func() {
		channelsIDM = oldChannels
		channel2advancedCustomConfig = oldConfigs
	})

	got := filterChannelsByRequestPathAndModel(
		[]int{1, 2, 3, 4, 5, 404},
		relayconstant.AsyncImageGenerationSelectionPath,
		"image-alias",
	)
	assert.Equal(t, []int{2, 5}, got)
}

func TestFilterAbilitiesForAsyncImageSelection(t *testing.T) {
	oldDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Channel{}))
	DB = db
	t.Cleanup(func() { DB = oldDB })

	toSyncModel := `{"image-alias":"qwen-image-plus"}`
	toAsyncModel := `{"image-alias":"wanx-v1"}`
	toGrsaiModel := `{"image-alias":"nano-banana-2"}`
	grsaiBaseURL := "https://grsai.dakka.com.cn"
	require.NoError(t, DB.Create([]*Channel{
		{Id: 11, Type: constant.ChannelTypeAli, Key: "ali-sync", ModelMapping: &toSyncModel},
		{Id: 12, Type: constant.ChannelTypeAli, Key: "ali-async", ModelMapping: &toAsyncModel},
		{Id: 13, Type: constant.ChannelTypeOpenAI, Key: "openai"},
		{Id: 14, Type: constant.ChannelTypeGemini, Key: "grsai", BaseURL: &grsaiBaseURL, ModelMapping: &toGrsaiModel},
	}).Error)

	abilities := []Ability{{ChannelId: 11}, {ChannelId: 12}, {ChannelId: 13}, {ChannelId: 14}, {ChannelId: 404}}
	got := filterAbilitiesByRequestPathAndModel(
		abilities,
		relayconstant.AsyncImageGenerationSelectionPath,
		"image-alias",
	)
	require.Len(t, got, 2)
	assert.Equal(t, 12, got[0].ChannelId)
	assert.Equal(t, 14, got[1].ChannelId)
}

func TestChannelSupportsAsyncImageModelMapping(t *testing.T) {
	tests := []struct {
		name         string
		channelType  int
		model        string
		modelMapping string
		want         bool
	}{
		{
			name:        "ali async model without mapping",
			channelType: constant.ChannelTypeAli,
			model:       "wanx-v1",
			want:        true,
		},
		{
			name:         "alias mapped to synchronous model",
			channelType:  constant.ChannelTypeAli,
			model:        "image-alias",
			modelMapping: `{"image-alias":"qwen-image-plus"}`,
			want:         false,
		},
		{
			name:         "alias mapped to asynchronous model",
			channelType:  constant.ChannelTypeAli,
			model:        "image-alias",
			modelMapping: `{"image-alias":"wanx-v1"}`,
			want:         true,
		},
		{
			name:         "chain mapped to asynchronous model",
			channelType:  constant.ChannelTypeAli,
			model:        "image-alias",
			modelMapping: `{"image-alias":"image-v2","image-v2":"wanx-v1"}`,
			want:         true,
		},
		{
			name:         "mapping cycle",
			channelType:  constant.ChannelTypeAli,
			model:        "image-alias",
			modelMapping: `{"image-alias":"image-v2","image-v2":"image-alias"}`,
			want:         false,
		},
		{
			name:         "invalid mapping json",
			channelType:  constant.ChannelTypeAli,
			model:        "image-alias",
			modelMapping: `{`,
			want:         false,
		},
		{
			name:        "non ali channel",
			channelType: constant.ChannelTypeOpenAI,
			model:       "wanx-v1",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := &Channel{Type: tt.channelType}
			if tt.modelMapping != "" {
				channel.ModelMapping = &tt.modelMapping
			}
			assert.Equal(t, tt.want, ChannelSupportsAsyncImage(channel, tt.model))
		})
	}
}

func TestChannelSupportsGrsaiAsyncImage(t *testing.T) {
	publicBaseURL := "https://grsaiapi.com"
	mappedModel := `{"image-alias":"nano-banana-2"}`
	publicChannel := &Channel{
		Type:         constant.ChannelTypeGemini,
		BaseURL:      &publicBaseURL,
		ModelMapping: &mappedModel,
	}
	assert.True(t, ChannelSupportsAsyncImage(publicChannel, "image-alias"))
	assert.False(t, ChannelSupportsAsyncImage(publicChannel, "imagen-4.0-generate-001"))

	ordinaryGeminiURL := "https://generativelanguage.googleapis.com"
	ordinaryGemini := &Channel{Type: constant.ChannelTypeGemini, BaseURL: &ordinaryGeminiURL}
	assert.False(t, ChannelSupportsAsyncImage(ordinaryGemini, "nano-banana-2"))

	proxyURL := "https://images.example.com"
	proxiedGrsai := &Channel{Type: constant.ChannelTypeGemini, BaseURL: &proxyURL}
	proxiedGrsai.SetOtherSettings(dto.ChannelOtherSettings{AsyncImageProvider: "grsai"})
	assert.True(t, ChannelSupportsAsyncImage(proxiedGrsai, "gpt-image-2-vip"))
}
