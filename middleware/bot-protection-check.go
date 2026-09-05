package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const maxBotProtectionProofLength = 16 * 1024

var botProtectionHTTPClient = &http.Client{Timeout: 10 * time.Second}
var turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
var reCaptchaVerifyURL = "https://www.recaptcha.net/recaptcha/api/siteverify"
var geeTestVerifyURL = "https://gcaptcha4.geetest.com/validate"

type botProtectionResponse struct {
	Success bool   `json:"success"`
	Result  string `json:"result"`
}

type geeTestProof struct {
	LotNumber     string `json:"lot_number"`
	CaptchaOutput string `json:"captcha_output"`
	PassToken     string `json:"pass_token"`
	GenTime       string `json:"gen_time"`
}

func BotProtectionCheck(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.IsBotProtectionEnabled(scope) {
			c.Next()
			return
		}
		if scope == common.BotProtectionScopeRegister &&
			common.EmailVerificationEnabled &&
			common.BotProtectionEmailVerificationEnabled {
			c.Next()
			return
		}

		proof := c.Query("bot_protection")
		if proof == "" {
			proof = c.Query("turnstile")
		}
		if proof == "" {
			abortBotProtection(c, "请先完成人机验证")
			return
		}
		if len(proof) > maxBotProtectionProofLength {
			abortBotProtection(c, "人机验证数据无效")
			return
		}

		verified, err := verifyBotProtection(c.Request.Context(), proof, c.ClientIP())
		if err != nil {
			common.SysLog(err.Error())
			abortBotProtection(c, "人机验证服务暂时不可用，请稍后重试")
			return
		}
		if !verified {
			abortBotProtection(c, "人机验证失败，请刷新后重试")
			return
		}

		c.Next()
	}
}

func abortBotProtection(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": message,
	})
	c.Abort()
}

func verifyBotProtection(ctx context.Context, proof string, remoteIP string) (bool, error) {
	switch common.BotProtectionProvider {
	case common.BotProtectionProviderTurnstile:
		return verifyFormToken(ctx, turnstileVerifyURL, common.TurnstileSecretKey, proof, remoteIP)
	case common.BotProtectionProviderReCaptcha:
		return verifyFormToken(ctx, reCaptchaVerifyURL, common.ReCaptchaSecretKey, proof, remoteIP)
	case common.BotProtectionProviderGeeTestV4:
		return verifyGeeTestV4(ctx, proof)
	case common.BotProtectionProviderCap:
		return verifyCap(ctx, proof)
	default:
		return false, fmt.Errorf("unsupported bot protection provider: %s", common.BotProtectionProvider)
	}
}

func verifyFormToken(ctx context.Context, endpoint string, secret string, proof string, remoteIP string) (bool, error) {
	form := url.Values{
		"secret":   {secret},
		"response": {proof},
	}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return sendBotProtectionRequest(req, false)
}

func verifyGeeTestV4(ctx context.Context, proofJSON string) (bool, error) {
	var proof geeTestProof
	if err := common.UnmarshalJsonStr(proofJSON, &proof); err != nil {
		return false, nil
	}
	if proof.LotNumber == "" || proof.CaptchaOutput == "" || proof.PassToken == "" || proof.GenTime == "" {
		return false, nil
	}

	signature := hmac.New(sha256.New, []byte(common.GeeTestSecretKey))
	_, _ = signature.Write([]byte(proof.LotNumber))
	form := url.Values{
		"lot_number":     {proof.LotNumber},
		"captcha_output": {proof.CaptchaOutput},
		"pass_token":     {proof.PassToken},
		"gen_time":       {proof.GenTime},
		"sign_token":     {hex.EncodeToString(signature.Sum(nil))},
	}
	endpoint := geeTestVerifyURL + "?captcha_id=" + url.QueryEscape(common.GeeTestCaptchaId)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return sendBotProtectionRequest(req, true)
}

func verifyCap(ctx context.Context, proof string) (bool, error) {
	endpoint := common.GetCapAPIEndpoint()
	if endpoint == "" {
		return false, fmt.Errorf("cap api endpoint is not configured")
	}
	body, err := common.Marshal(map[string]string{
		"secret":   common.CapSecretKey,
		"response": proof,
	})
	if err != nil {
		return false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"siteverify", bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	return sendBotProtectionRequest(req, false)
}

func sendBotProtectionRequest(req *http.Request, useResult bool) (bool, error) {
	response, err := botProtectionHTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return false, fmt.Errorf("bot protection provider returned HTTP %d", response.StatusCode)
	}

	var result botProtectionResponse
	if err := common.DecodeJson(response.Body, &result); err != nil {
		return false, err
	}
	if useResult {
		return result.Result == "success", nil
	}
	return result.Success, nil
}
