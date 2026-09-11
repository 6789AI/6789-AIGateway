package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelValidateSettingsRejectsInvalidHTTPTransport(t *testing.T) {
	tests := []struct {
		name    string
		setting dto.ChannelSettings
		wantErr string
	}{
		{
			name:    "auto with shards is valid",
			setting: dto.ChannelSettings{HTTPProtocol: "auto", HTTP2ConnectionShards: 4},
		},
		{
			name:    "http1 with shards greater than one rejected",
			setting: dto.ChannelSettings{HTTPProtocol: "http1", HTTP2ConnectionShards: 2},
			wantErr: "http2_connection_shards",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := &Channel{}
			channel.SetSetting(tt.setting)
			err := channel.ValidateSettings()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestChannelValidateSettingsValidatesAsyncImageProvider(t *testing.T) {
	for _, provider := range []string{
		dto.AsyncImageProviderAli,
		dto.AsyncImageProviderNewAPI,
		dto.AsyncImageProviderGrsai,
	} {
		channel := &Channel{Type: constant.ChannelTypeOpenAI}
		channel.SetOtherSettings(dto.ChannelOtherSettings{AsyncImageEnabled: true, AsyncImageProvider: provider})
		require.NoError(t, channel.ValidateSettings())
	}

	t.Run("legacy provider remains inert while disabled", func(t *testing.T) {
		channel := &Channel{Type: constant.ChannelTypeOpenAI}
		channel.SetOtherSettings(dto.ChannelOtherSettings{AsyncImageProvider: dto.AsyncImageProviderGrsai})
		require.NoError(t, channel.ValidateSettings())
	})

	t.Run("unknown provider is rejected", func(t *testing.T) {
		channel := &Channel{Type: constant.ChannelTypeOpenAI}
		channel.SetOtherSettings(dto.ChannelOtherSettings{AsyncImageProvider: "unknown"})
		err := channel.ValidateSettings()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "async_image_provider")
	})

	t.Run("enabled setting requires provider", func(t *testing.T) {
		channel := &Channel{Type: constant.ChannelTypeOpenAI}
		channel.SetOtherSettings(dto.ChannelOtherSettings{AsyncImageEnabled: true})
		err := channel.ValidateSettings()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "async_image_provider")
	})

	t.Run("unsupported channel type is rejected", func(t *testing.T) {
		channel := &Channel{Type: constant.ChannelTypeAnthropic}
		channel.SetOtherSettings(dto.ChannelOtherSettings{AsyncImageEnabled: true, AsyncImageProvider: dto.AsyncImageProviderNewAPI})
		err := channel.ValidateSettings()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "channel type")
	})
}

func TestChannelValidateSettingsValidatesVideoProtocol(t *testing.T) {
	channel := &Channel{Type: constant.ChannelTypeSora}
	channel.SetOtherSettings(dto.ChannelOtherSettings{VideoProtocol: dto.VideoProtocolVinted})
	require.NoError(t, channel.ValidateSettings())
	channel.SetOtherSettings(dto.ChannelOtherSettings{VideoProtocol: dto.VideoProtocolOpenAI})
	require.NoError(t, channel.ValidateSettings())

	channel.SetOtherSettings(dto.ChannelOtherSettings{VideoProtocol: "unknown"})
	err := channel.ValidateSettings()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "video_protocol")

	channel.Type = constant.ChannelTypeAnthropic
	channel.SetOtherSettings(dto.ChannelOtherSettings{VideoProtocol: dto.VideoProtocolVinted})
	err = channel.ValidateSettings()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "channel type")
}

func TestIsVintedVideoChannelUsesSettingAndLegacyHostnameFallback(t *testing.T) {
	assert.True(t, IsVintedVideoChannel(
		constant.ChannelTypeSora,
		"https://example.com",
		dto.ChannelOtherSettings{VideoProtocol: dto.VideoProtocolVinted},
	))
	assert.True(t, IsVintedVideoChannel(
		constant.ChannelTypeOpenAI,
		"https://api.vinted.cam/base",
		dto.ChannelOtherSettings{},
	))
	assert.False(t, IsVintedVideoChannel(
		constant.ChannelTypeSora,
		"https://vinted.cam",
		dto.ChannelOtherSettings{VideoProtocol: dto.VideoProtocolOpenAI},
	))
	assert.False(t, IsVintedVideoChannel(
		constant.ChannelTypeAnthropic,
		"https://vinted.cam",
		dto.ChannelOtherSettings{},
	))
	assert.False(t, IsVintedVideoChannel(
		constant.ChannelTypeSora,
		"https://not-vinted.example",
		dto.ChannelOtherSettings{},
	))
}

func TestGetKeyByIndexPreservesSingleKeyAndSelectsMultiKey(t *testing.T) {
	single := &Channel{Key: "single-key\nwith-newline"}
	key, err := single.GetKeyByIndex(0)
	require.NoError(t, err)
	assert.Equal(t, single.Key, key)
	_, err = single.GetKeyByIndex(1)
	require.Error(t, err)

	multi := &Channel{Key: "first\nsecond", ChannelInfo: ChannelInfo{IsMultiKey: true}}
	key, err = multi.GetKeyByIndex(1)
	require.NoError(t, err)
	assert.Equal(t, "second", key)
}

func TestAdvancedCustomChannelRequiresModelListRouteOnlyWhenUpdateChecksEnabled(t *testing.T) {
	inferenceRoute := dto.AdvancedCustomRoute{
		IncomingPath: "/v1/chat/completions",
		UpstreamPath: "/v1/chat/completions",
		Converter:    "none",
	}

	tests := []struct {
		name          string
		checksEnabled bool
		routes        []dto.AdvancedCustomRoute
		wantErr       string
	}{
		{
			name:   "legacy channel without discovery route remains valid",
			routes: []dto.AdvancedCustomRoute{inferenceRoute},
		},
		{
			name:          "enabled checks require discovery route",
			checksEnabled: true,
			routes:        []dto.AdvancedCustomRoute{inferenceRoute},
			wantErr:       dto.AdvancedCustomModelListPath,
		},
		{
			name:          "enabled checks accept discovery route",
			checksEnabled: true,
			routes: []dto.AdvancedCustomRoute{
				inferenceRoute,
				{
					IncomingPath: dto.AdvancedCustomModelListPath,
					UpstreamPath: dto.AdvancedCustomModelListPath,
					Converter:    "none",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := &Channel{Type: constant.ChannelTypeAdvancedCustom}
			channel.SetOtherSettings(dto.ChannelOtherSettings{
				UpstreamModelUpdateCheckEnabled: tt.checksEnabled,
				AdvancedCustom: &dto.AdvancedCustomConfig{
					Routes: tt.routes,
				},
			})

			err := channel.ValidateSettings()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
