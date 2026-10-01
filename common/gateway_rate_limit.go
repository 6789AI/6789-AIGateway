package common

import (
	"fmt"
	"sync/atomic"
)

const GatewayRateLimitOptionKey = "GatewayRateLimits"

type GatewayRateLimitRule struct {
	Enabled       bool  `json:"enabled"`
	Limit         int   `json:"limit"`
	WindowSeconds int64 `json:"window_seconds"`
}

type GatewayRateLimitConfig struct {
	API      GatewayRateLimitRule `json:"api"`
	Web      GatewayRateLimitRule `json:"web"`
	Critical GatewayRateLimitRule `json:"critical"`
	Login    GatewayRateLimitRule `json:"login"`
	Session  GatewayRateLimitRule `json:"session"`
}

var gatewayRateLimitConfig atomic.Pointer[GatewayRateLimitConfig]

func InitGatewayRateLimitsFromEnv() {
	config := GatewayRateLimitConfig{
		API:      GatewayRateLimitRule{GlobalApiRateLimitEnable, GlobalApiRateLimitNum, GlobalApiRateLimitDuration},
		Web:      GatewayRateLimitRule{GlobalWebRateLimitEnable, GlobalWebRateLimitNum, GlobalWebRateLimitDuration},
		Critical: GatewayRateLimitRule{CriticalRateLimitEnable, CriticalRateLimitNum, CriticalRateLimitDuration},
		Login:    GatewayRateLimitRule{false, CriticalRateLimitNum, CriticalRateLimitDuration},
		Session:  GatewayRateLimitRule{false, 120, CriticalRateLimitDuration},
	}
	gatewayRateLimitConfig.Store(&config)
}

func GetGatewayRateLimits() GatewayRateLimitConfig {
	if config := gatewayRateLimitConfig.Load(); config != nil {
		return *config
	}
	// Tests and callers that initialize only the legacy globals still receive
	// the same behavior until InitEnv publishes the first snapshot.
	return GatewayRateLimitConfig{
		API:      GatewayRateLimitRule{GlobalApiRateLimitEnable, GlobalApiRateLimitNum, GlobalApiRateLimitDuration},
		Web:      GatewayRateLimitRule{GlobalWebRateLimitEnable, GlobalWebRateLimitNum, GlobalWebRateLimitDuration},
		Critical: GatewayRateLimitRule{CriticalRateLimitEnable, CriticalRateLimitNum, CriticalRateLimitDuration},
		Login:    GatewayRateLimitRule{false, CriticalRateLimitNum, CriticalRateLimitDuration},
		Session:  GatewayRateLimitRule{false, 120, CriticalRateLimitDuration},
	}
}

func ValidateGatewayRateLimits(config GatewayRateLimitConfig) error {
	for name, rule := range map[string]GatewayRateLimitRule{
		"api": config.API, "web": config.Web, "critical": config.Critical,
		"login": config.Login, "session": config.Session,
	} {
		if rule.Limit < 1 || rule.Limit > 1_000_000 {
			return fmt.Errorf("%s limit must be between 1 and 1000000", name)
		}
		if rule.WindowSeconds < 1 || rule.WindowSeconds > int64(RateLimitKeyExpirationDuration.Seconds()) {
			return fmt.Errorf("%s window_seconds must be between 1 and %d", name, int64(RateLimitKeyExpirationDuration.Seconds()))
		}
	}
	return nil
}

func ParseGatewayRateLimits(value string) (GatewayRateLimitConfig, error) {
	var config GatewayRateLimitConfig
	if err := UnmarshalJsonStr(value, &config); err != nil {
		return config, err
	}
	return config, ValidateGatewayRateLimits(config)
}

func SetGatewayRateLimits(config GatewayRateLimitConfig) {
	gatewayRateLimitConfig.Store(&config)
}

func GatewayRateLimitsJSON() string {
	data, err := Marshal(GetGatewayRateLimits())
	if err != nil {
		SysError("failed to marshal gateway rate limits: " + err.Error())
		return "{}"
	}
	return string(data)
}
