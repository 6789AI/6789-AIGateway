package common

import (
	"net/url"
	"strings"
)

const (
	BotProtectionProviderTurnstile = "turnstile"
	BotProtectionProviderReCaptcha = "recaptcha"
	BotProtectionProviderGeeTestV4 = "geetest_v4"
	BotProtectionProviderCap       = "cap"

	BotProtectionScopeLogin             = "login"
	BotProtectionScopeRegister          = "register"
	BotProtectionScopeEmailVerification = "email_verification"
	BotProtectionScopePasswordReset     = "password_reset"
	BotProtectionScopeCheckin           = "checkin"
)

func IsBotProtectionProvider(provider string) bool {
	switch provider {
	case BotProtectionProviderTurnstile,
		BotProtectionProviderReCaptcha,
		BotProtectionProviderGeeTestV4,
		BotProtectionProviderCap:
		return true
	default:
		return false
	}
}

func IsBotProtectionEnabled(scope string) bool {
	if !TurnstileCheckEnabled {
		return false
	}

	switch scope {
	case BotProtectionScopeLogin:
		return BotProtectionLoginEnabled
	case BotProtectionScopeRegister:
		return BotProtectionRegisterEnabled
	case BotProtectionScopeEmailVerification:
		return BotProtectionEmailVerificationEnabled
	case BotProtectionScopePasswordReset:
		return BotProtectionPasswordResetEnabled
	case BotProtectionScopeCheckin:
		return BotProtectionCheckinEnabled
	default:
		return false
	}
}

func GetBotProtectionSiteKey() string {
	switch BotProtectionProvider {
	case BotProtectionProviderTurnstile:
		return TurnstileSiteKey
	case BotProtectionProviderReCaptcha:
		return ReCaptchaSiteKey
	case BotProtectionProviderGeeTestV4:
		return GeeTestCaptchaId
	default:
		return ""
	}
}

func GetCapAPIEndpoint() string {
	serverURL := strings.TrimRight(strings.TrimSpace(CapServerURL), "/")
	siteKey := strings.Trim(strings.TrimSpace(CapSiteKey), "/")
	if serverURL == "" || siteKey == "" {
		return ""
	}
	return serverURL + "/" + url.PathEscape(siteKey) + "/"
}
