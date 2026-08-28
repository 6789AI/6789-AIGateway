package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type botProtectionRequestRecord struct {
	path  string
	query url.Values
	form  url.Values
	body  map[string]string
	err   error
}

func preserveBotProtectionGlobals(t *testing.T) {
	t.Helper()
	originalProvider := common.BotProtectionProvider
	originalEnabled := common.TurnstileCheckEnabled
	originalLoginEnabled := common.BotProtectionLoginEnabled
	originalTurnstileSecret := common.TurnstileSecretKey
	originalReCaptchaSecret := common.ReCaptchaSecretKey
	originalGeeTestCaptchaId := common.GeeTestCaptchaId
	originalGeeTestSecret := common.GeeTestSecretKey
	originalCapServerURL := common.CapServerURL
	originalCapSiteKey := common.CapSiteKey
	originalCapSecret := common.CapSecretKey
	originalTurnstileURL := turnstileVerifyURL
	originalReCaptchaURL := reCaptchaVerifyURL
	originalGeeTestURL := geeTestVerifyURL
	t.Cleanup(func() {
		common.BotProtectionProvider = originalProvider
		common.TurnstileCheckEnabled = originalEnabled
		common.BotProtectionLoginEnabled = originalLoginEnabled
		common.TurnstileSecretKey = originalTurnstileSecret
		common.ReCaptchaSecretKey = originalReCaptchaSecret
		common.GeeTestCaptchaId = originalGeeTestCaptchaId
		common.GeeTestSecretKey = originalGeeTestSecret
		common.CapServerURL = originalCapServerURL
		common.CapSiteKey = originalCapSiteKey
		common.CapSecretKey = originalCapSecret
		turnstileVerifyURL = originalTurnstileURL
		reCaptchaVerifyURL = originalReCaptchaURL
		geeTestVerifyURL = originalGeeTestURL
	})
}

func TestVerifyBotProtectionProviderProtocols(t *testing.T) {
	preserveBotProtectionGlobals(t)
	records := make(chan botProtectionRequestRecord, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record := botProtectionRequestRecord{path: r.URL.Path, query: r.URL.Query()}
		if r.Header.Get("Content-Type") == "application/json" {
			record.body = make(map[string]string)
			record.err = common.DecodeJson(r.Body, &record.body)
		} else {
			record.err = r.ParseForm()
			record.form = r.PostForm
		}
		records <- record
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/geetest" {
			_, _ = io.WriteString(w, `{"result":"success"}`)
			return
		}
		_, _ = io.WriteString(w, `{"success":true}`)
	}))
	t.Cleanup(server.Close)

	tests := []struct {
		name      string
		provider  string
		proof     string
		configure func()
		assert    func(*testing.T, botProtectionRequestRecord)
	}{
		{
			name:     "Turnstile form verification",
			provider: common.BotProtectionProviderTurnstile,
			proof:    "turnstile-token",
			configure: func() {
				common.TurnstileSecretKey = "turnstile-secret"
				turnstileVerifyURL = server.URL + "/turnstile"
			},
			assert: func(t *testing.T, record botProtectionRequestRecord) {
				require.NoError(t, record.err)
				assert.Equal(t, "/turnstile", record.path)
				assert.Equal(t, "turnstile-secret", record.form.Get("secret"))
				assert.Equal(t, "turnstile-token", record.form.Get("response"))
				assert.Equal(t, "198.51.100.8", record.form.Get("remoteip"))
			},
		},
		{
			name:     "reCAPTCHA form verification",
			provider: common.BotProtectionProviderReCaptcha,
			proof:    "recaptcha-token",
			configure: func() {
				common.ReCaptchaSecretKey = "recaptcha-secret"
				reCaptchaVerifyURL = server.URL + "/recaptcha"
			},
			assert: func(t *testing.T, record botProtectionRequestRecord) {
				require.NoError(t, record.err)
				assert.Equal(t, "/recaptcha", record.path)
				assert.Equal(t, "recaptcha-secret", record.form.Get("secret"))
				assert.Equal(t, "recaptcha-token", record.form.Get("response"))
				assert.Equal(t, "198.51.100.8", record.form.Get("remoteip"))
			},
		},
		{
			name:     "GeeTest V4 signed verification",
			provider: common.BotProtectionProviderGeeTestV4,
			proof:    `{"lot_number":"lot-1","captcha_output":"output","pass_token":"pass","gen_time":"123"}`,
			configure: func() {
				common.GeeTestCaptchaId = "captcha-id"
				common.GeeTestSecretKey = "geetest-secret"
				geeTestVerifyURL = server.URL + "/geetest"
			},
			assert: func(t *testing.T, record botProtectionRequestRecord) {
				require.NoError(t, record.err)
				assert.Equal(t, "/geetest", record.path)
				assert.Equal(t, "captcha-id", record.query.Get("captcha_id"))
				assert.Equal(t, "lot-1", record.form.Get("lot_number"))
				assert.Equal(t, "output", record.form.Get("captcha_output"))
				assert.Equal(t, "pass", record.form.Get("pass_token"))
				assert.Equal(t, "123", record.form.Get("gen_time"))
				assert.Equal(t, "7cb3d1d3b17635bff7cf4a4b9cd3a232e52a0fadae11a65ccf4840f914f5182b", record.form.Get("sign_token"))
			},
		},
		{
			name:     "Cap Standalone siteverify",
			provider: common.BotProtectionProviderCap,
			proof:    "cap-token",
			configure: func() {
				common.CapServerURL = server.URL
				common.CapSiteKey = "site-key"
				common.CapSecretKey = "cap-secret"
			},
			assert: func(t *testing.T, record botProtectionRequestRecord) {
				require.NoError(t, record.err)
				assert.Equal(t, "/site-key/siteverify", record.path)
				assert.Equal(t, "cap-secret", record.body["secret"])
				assert.Equal(t, "cap-token", record.body["response"])
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			common.BotProtectionProvider = test.provider
			test.configure()
			verified, err := verifyBotProtection(context.Background(), test.proof, "198.51.100.8")
			require.NoError(t, err)
			assert.True(t, verified)
			test.assert(t, <-records)
		})
	}
}

func TestBotProtectionCheckHonorsScopeAndLegacyParameter(t *testing.T) {
	preserveBotProtectionGlobals(t)
	gin.SetMode(gin.TestMode)
	common.TurnstileCheckEnabled = true
	common.BotProtectionProvider = common.BotProtectionProviderTurnstile
	common.TurnstileSecretKey = "secret"

	verificationServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true}`)
	}))
	t.Cleanup(verificationServer.Close)
	turnstileVerifyURL = verificationServer.URL

	common.BotProtectionLoginEnabled = false
	bypassRouter := gin.New()
	bypassRouter.GET("/", BotProtectionCheck(common.BotProtectionScopeLogin), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	bypassRecorder := httptest.NewRecorder()
	bypassRouter.ServeHTTP(bypassRecorder, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusNoContent, bypassRecorder.Code)

	common.BotProtectionLoginEnabled = true
	for _, parameter := range []string{"bot_protection", "turnstile"} {
		t.Run(parameter, func(t *testing.T) {
			router := gin.New()
			router.GET("/", BotProtectionCheck(common.BotProtectionScopeLogin), func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/?"+parameter+"=token", nil)
			router.ServeHTTP(recorder, request)
			assert.Equal(t, http.StatusNoContent, recorder.Code)
		})
	}
}
