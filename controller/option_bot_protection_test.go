package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateBotProtectionConfiguration(t *testing.T) {
	originalTurnstileSiteKey := common.TurnstileSiteKey
	originalTurnstileSecretKey := common.TurnstileSecretKey
	originalReCaptchaSiteKey := common.ReCaptchaSiteKey
	originalReCaptchaSecretKey := common.ReCaptchaSecretKey
	originalGeeTestCaptchaId := common.GeeTestCaptchaId
	originalGeeTestSecretKey := common.GeeTestSecretKey
	originalCapServerURL := common.CapServerURL
	originalCapSiteKey := common.CapSiteKey
	originalCapSecretKey := common.CapSecretKey
	t.Cleanup(func() {
		common.TurnstileSiteKey = originalTurnstileSiteKey
		common.TurnstileSecretKey = originalTurnstileSecretKey
		common.ReCaptchaSiteKey = originalReCaptchaSiteKey
		common.ReCaptchaSecretKey = originalReCaptchaSecretKey
		common.GeeTestCaptchaId = originalGeeTestCaptchaId
		common.GeeTestSecretKey = originalGeeTestSecretKey
		common.CapServerURL = originalCapServerURL
		common.CapSiteKey = originalCapSiteKey
		common.CapSecretKey = originalCapSecretKey
	})

	common.TurnstileSiteKey = "site"
	common.TurnstileSecretKey = "secret"
	common.ReCaptchaSiteKey = "site"
	common.ReCaptchaSecretKey = "secret"
	common.GeeTestCaptchaId = "captcha-id"
	common.GeeTestSecretKey = "captcha-key"
	common.CapServerURL = "https://cap.example.com"
	common.CapSiteKey = "site"
	common.CapSecretKey = "secret"

	for _, provider := range []string{
		common.BotProtectionProviderTurnstile,
		common.BotProtectionProviderReCaptcha,
		common.BotProtectionProviderGeeTestV4,
		common.BotProtectionProviderCap,
	} {
		t.Run(provider, func(t *testing.T) {
			require.NoError(t, validateBotProtectionConfiguration(provider))
		})
	}
	assert.Error(t, validateBotProtectionConfiguration("unknown"))

	common.CapSecretKey = ""
	assert.ErrorContains(
		t,
		validateBotProtectionConfiguration(common.BotProtectionProviderCap),
		"Cap",
	)
}

func TestValidateCapServerURL(t *testing.T) {
	for _, validURL := range []string{
		"https://cap.example.com",
		"http://127.0.0.1:3000/cap",
	} {
		require.NoError(t, validateCapServerURL(validURL))
	}

	for _, invalidURL := range []string{
		"cap.example.com",
		"ftp://cap.example.com",
		"https://user:password@cap.example.com",
		"https://cap.example.com?token=secret",
	} {
		assert.Error(t, validateCapServerURL(invalidURL))
	}
}
