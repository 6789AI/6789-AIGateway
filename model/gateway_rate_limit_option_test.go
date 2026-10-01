package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGatewayRateLimitOptionPersistsAndReloads(t *testing.T) {
	previousDB := DB
	previousOptions := common.OptionMap
	previousConfig := common.GetGatewayRateLimits()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Option{}))
	DB = db
	common.OptionMap = map[string]string{}
	t.Cleanup(func() {
		DB = previousDB
		common.OptionMap = previousOptions
		common.SetGatewayRateLimits(previousConfig)
	})

	config := common.GatewayRateLimitConfig{
		API:      common.GatewayRateLimitRule{Enabled: true, Limit: 900, WindowSeconds: 180},
		Web:      common.GatewayRateLimitRule{Enabled: true, Limit: 300, WindowSeconds: 180},
		Critical: common.GatewayRateLimitRule{Enabled: true, Limit: 40, WindowSeconds: 1200},
		Login:    common.GatewayRateLimitRule{Enabled: true, Limit: 100, WindowSeconds: 1200},
		Session:  common.GatewayRateLimitRule{Enabled: true, Limit: 200, WindowSeconds: 1200},
	}
	common.SetGatewayRateLimits(config)
	value := common.GatewayRateLimitsJSON()
	common.SetGatewayRateLimits(previousConfig)
	require.NoError(t, UpdateOption(common.GatewayRateLimitOptionKey, value))
	assert.Equal(t, config, common.GetGatewayRateLimits())

	common.SetGatewayRateLimits(previousConfig)
	loadOptionsFromDatabase()
	assert.Equal(t, config, common.GetGatewayRateLimits())

	config.Login.Limit = 0
	common.SetGatewayRateLimits(config)
	assert.Error(t, UpdateOption(common.GatewayRateLimitOptionKey, common.GatewayRateLimitsJSON()))
	var persisted Option
	require.NoError(t, db.First(&persisted, "key = ?", common.GatewayRateLimitOptionKey).Error)
	assert.Equal(t, value, persisted.Value)
}
