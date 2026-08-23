package system_setting

import (
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

const (
	GlobalBannerOptionPrefix    = "global_banner."
	MarketingBannerOptionPrefix = "marketing_banner."
	maxMarketingBannerRunes     = 300
	maxMarketingBannerLinkRunes = 2048
	maxMarketingBannerTimestamp = 253402300799
)

var hexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var marketingBannerIcons = map[string]struct{}{
	"bell":      {},
	"coupon":    {},
	"crown":     {},
	"discount":  {},
	"fire":      {},
	"gift":      {},
	"heart":     {},
	"lightning": {},
	"megaphone": {},
	"party":     {},
	"rocket":    {},
	"sparkles":  {},
	"star":      {},
}

type BannerSettings struct {
	Enabled          bool   `json:"enabled"`
	Content          string `json:"content"`
	BackgroundColor  string `json:"background_color"`
	TextColor        string `json:"text_color"`
	Icon             string `json:"icon"`
	CountdownEnabled bool   `json:"countdown_enabled"`
	CountdownEndAt   int64  `json:"countdown_end_at"`
	LinkURL          string `json:"link_url"`
}

var defaultGlobalBannerSettings = BannerSettings{
	Enabled:          false,
	Content:          "",
	BackgroundColor:  "#0EA5E9",
	TextColor:        "#082F49",
	Icon:             "gift",
	CountdownEnabled: false,
	CountdownEndAt:   0,
	LinkURL:          "/pricing",
}

var defaultMarketingBannerSettings = BannerSettings{
	Enabled:          false,
	Content:          "",
	BackgroundColor:  "#A3E635",
	TextColor:        "#1A2E05",
	Icon:             "megaphone",
	CountdownEnabled: false,
	CountdownEndAt:   0,
	LinkURL:          "",
}

func init() {
	config.GlobalConfig.Register("global_banner", &defaultGlobalBannerSettings)
	config.GlobalConfig.Register("marketing_banner", &defaultMarketingBannerSettings)
}

func GetGlobalBannerSettings() BannerSettings {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	return defaultGlobalBannerSettings
}

func GetMarketingBannerSettings() BannerSettings {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	return defaultMarketingBannerSettings
}

func ValidateBannerOption(key string, value string) error {
	field, validPrefix := strings.CutPrefix(key, GlobalBannerOptionPrefix)
	if !validPrefix {
		field, validPrefix = strings.CutPrefix(key, MarketingBannerOptionPrefix)
	}
	if !validPrefix {
		return errors.New("unknown banner option")
	}

	switch field {
	case "enabled":
		if value != "true" && value != "false" {
			return errors.New("enabled must be true or false")
		}
	case "content":
		if utf8.RuneCountInString(strings.TrimSpace(value)) > maxMarketingBannerRunes {
			return errors.New("content must not exceed 300 characters")
		}
	case "background_color", "text_color":
		if !hexColorPattern.MatchString(value) {
			return errors.New("color must be a 6-digit hex value")
		}
	case "icon":
		if value == "" {
			return nil
		}
		if _, ok := marketingBannerIcons[value]; !ok {
			return errors.New("icon is not supported")
		}
	case "countdown_enabled":
		if value != "true" && value != "false" {
			return errors.New("countdown_enabled must be true or false")
		}
	case "countdown_end_at":
		timestamp, err := strconv.ParseInt(value, 10, 64)
		if err != nil || timestamp < 0 || timestamp > maxMarketingBannerTimestamp {
			return errors.New("countdown_end_at must be a valid Unix timestamp")
		}
	case "link_url":
		trimmed := strings.TrimSpace(value)
		if utf8.RuneCountInString(trimmed) > maxMarketingBannerLinkRunes {
			return errors.New("link_url must not exceed 2048 characters")
		}
		if trimmed == "" {
			return nil
		}
		if strings.HasPrefix(trimmed, "/") && !strings.HasPrefix(trimmed, "//") {
			return nil
		}
		parsed, err := url.ParseRequestURI(trimmed)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return errors.New("link_url must be an HTTP(S) URL or a site path starting with /")
		}
	}
	return nil
}

func ValidateMarketingBannerOption(key string, value string) error {
	return ValidateBannerOption(key, value)
}
