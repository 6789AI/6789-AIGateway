package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewayRateLimitConfigRejectsInvalidWindowAndLimit(t *testing.T) {
	config := GatewayRateLimitConfig{
		API:      GatewayRateLimitRule{true, 360, 180},
		Web:      GatewayRateLimitRule{true, 120, 180},
		Critical: GatewayRateLimitRule{true, 20, 1200},
		Login:    GatewayRateLimitRule{false, 20, 1200},
		Session:  GatewayRateLimitRule{false, 120, 1200},
	}
	require.NoError(t, ValidateGatewayRateLimits(config))

	config.Login.Limit = 0
	assert.ErrorContains(t, ValidateGatewayRateLimits(config), "login limit")
	config.Login.Limit = 20
	config.Session.WindowSeconds = 1201
	assert.ErrorContains(t, ValidateGatewayRateLimits(config), "session window_seconds")

	_, err := ParseGatewayRateLimits(`{"api":{"enabled":true,"limit":360,"window_seconds":180}}`)
	assert.Error(t, err, "incomplete configurations must not silently disable the remaining buckets")
}

func TestGatewayRateLimitSnapshotPreservesIndependentRules(t *testing.T) {
	previous := GetGatewayRateLimits()
	t.Cleanup(func() { SetGatewayRateLimits(previous) })
	config := GatewayRateLimitConfig{
		API:      GatewayRateLimitRule{true, 360, 180},
		Web:      GatewayRateLimitRule{true, 120, 180},
		Critical: GatewayRateLimitRule{true, 20, 1200},
		Login:    GatewayRateLimitRule{true, 30, 1200},
		Session:  GatewayRateLimitRule{true, 120, 1200},
	}
	SetGatewayRateLimits(config)
	config.Login.Limit = 999
	assert.Equal(t, 30, GetGatewayRateLimits().Login.Limit)
	parsed, err := ParseGatewayRateLimits(GatewayRateLimitsJSON())
	require.NoError(t, err)
	assert.Equal(t, GetGatewayRateLimits(), parsed)
}
