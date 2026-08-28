package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/console_setting"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

var completionRatioMetaOptionKeys = []string{
	"ModelPrice",
	"ModelRatio",
	"CompletionRatio",
	"CacheRatio",
	"CreateCacheRatio",
	"ImageRatio",
	"AudioRatio",
	"AudioCompletionRatio",
}

func isPaymentComplianceOptionKey(key string) bool {
	return strings.HasPrefix(key, "payment_setting.compliance_")
}

func isPositiveOptionValue(value string) bool {
	intValue, err := strconv.Atoi(strings.TrimSpace(value))
	if err == nil {
		return intValue > 0
	}
	floatValue, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return err == nil && floatValue > 0
}

func validateBotProtectionConfiguration(provider string) error {
	switch provider {
	case common.BotProtectionProviderTurnstile:
		if common.TurnstileSiteKey == "" || common.TurnstileSecretKey == "" {
			return fmt.Errorf("无法启用机器人保护，请先填写 Turnstile Site Key 和 Secret Key")
		}
	case common.BotProtectionProviderReCaptcha:
		if common.ReCaptchaSiteKey == "" || common.ReCaptchaSecretKey == "" {
			return fmt.Errorf("无法启用机器人保护，请先填写 reCAPTCHA Site Key 和 Secret Key")
		}
	case common.BotProtectionProviderGeeTestV4:
		if common.GeeTestCaptchaId == "" || common.GeeTestSecretKey == "" {
			return fmt.Errorf("无法启用机器人保护，请先填写极验 V4 Captcha ID 和 Captcha Key")
		}
	case common.BotProtectionProviderCap:
		if common.CapServerURL == "" || common.CapSiteKey == "" || common.CapSecretKey == "" {
			return fmt.Errorf("无法启用机器人保护，请先填写 Cap 服务地址、Site Key 和 Secret Key")
		}
	default:
		return fmt.Errorf("不支持的机器人保护提供商")
	}
	return nil
}

func validateCapServerURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil ||
		parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.User != nil ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" {
		return fmt.Errorf("Cap 服务地址必须是有效的 HTTP 或 HTTPS URL")
	}
	return nil
}

func collectModelNamesFromOptionValue(raw string, modelNames map[string]struct{}) {
	if strings.TrimSpace(raw) == "" {
		return
	}

	var parsed map[string]any
	if err := common.UnmarshalJsonStr(raw, &parsed); err != nil {
		return
	}

	for modelName := range parsed {
		modelNames[modelName] = struct{}{}
	}
}

func buildCompletionRatioMetaValue(optionValues map[string]string) string {
	modelNames := make(map[string]struct{})
	for _, key := range completionRatioMetaOptionKeys {
		collectModelNamesFromOptionValue(optionValues[key], modelNames)
	}

	meta := make(map[string]ratio_setting.CompletionRatioInfo, len(modelNames))
	for modelName := range modelNames {
		meta[modelName] = ratio_setting.GetCompletionRatioInfo(modelName)
	}

	jsonBytes, err := common.Marshal(meta)
	if err != nil {
		return "{}"
	}
	return string(jsonBytes)
}

func GetOptions(c *gin.Context) {
	var options []*model.Option
	optionValues := make(map[string]string)
	common.OptionMapRWMutex.Lock()
	for k, v := range common.OptionMap {
		if k == "theme.frontend" {
			continue
		}
		value := common.Interface2String(v)
		isPublicBotProtectionKey := k == "TurnstileSiteKey" ||
			k == "ReCaptchaSiteKey" ||
			k == "GeeTestCaptchaId" ||
			k == "CapSiteKey"
		isSensitiveKey := !isPublicBotProtectionKey && (strings.HasSuffix(k, "Token") ||
			strings.HasSuffix(k, "Secret") ||
			strings.HasSuffix(k, "Key") ||
			strings.HasSuffix(k, "Password") ||
			strings.HasSuffix(k, "secret") ||
			strings.HasSuffix(k, "api_key"))
		if isSensitiveKey {
			continue
		}
		options = append(options, &model.Option{
			Key:   k,
			Value: value,
		})
		for _, optionKey := range completionRatioMetaOptionKeys {
			if optionKey == k {
				optionValues[k] = value
				break
			}
		}
	}
	common.OptionMapRWMutex.Unlock()
	options = append(options, &model.Option{
		Key:   "CompletionRatioMeta",
		Value: buildCompletionRatioMetaValue(optionValues),
	})
	options = append(options,
		&model.Option{Key: "TurnstileSecretKeyConfigured", Value: strconv.FormatBool(common.TurnstileSecretKey != "")},
		&model.Option{Key: "ReCaptchaSecretKeyConfigured", Value: strconv.FormatBool(common.ReCaptchaSecretKey != "")},
		&model.Option{Key: "GeeTestSecretKeyConfigured", Value: strconv.FormatBool(common.GeeTestSecretKey != "")},
		&model.Option{Key: "CapSecretKeyConfigured", Value: strconv.FormatBool(common.CapSecretKey != "")},
		&model.Option{Key: "BTMailPasswordConfigured", Value: strconv.FormatBool(common.BTMailPassword != "")},
	)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    options,
	})
}

type OptionUpdateRequest struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

func UpdateOption(c *gin.Context) {
	var option OptionUpdateRequest
	err := common.DecodeJson(c.Request.Body, &option)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}
	switch option.Value.(type) {
	case bool:
		option.Value = common.Interface2String(option.Value.(bool))
	case float64:
		option.Value = common.Interface2String(option.Value.(float64))
	case int:
		option.Value = common.Interface2String(option.Value.(int))
	default:
		option.Value = fmt.Sprintf("%v", option.Value)
	}
	switch option.Key {
	case "QuotaForInviter", "QuotaForInvitee":
		if isPositiveOptionValue(option.Value.(string)) && !operation_setting.IsPaymentComplianceConfirmed() {
			common.ApiErrorI18n(c, i18n.MsgPaymentComplianceRequired)
			return
		}
	default:
		if isPaymentComplianceOptionKey(option.Key) {
			common.ApiErrorMsg(c, "合规确认字段不允许通过通用设置接口修改")
			return
		}
	}
	switch option.Key {
	case "EmailSenderProvider":
		provider := option.Value.(string)
		if !common.IsEmailSenderProvider(provider) {
			common.ApiErrorMsg(c, "不支持的发件方式")
			return
		}
		if provider == common.EmailSenderProviderBTMail {
			if err := common.ValidateBTMailConfiguration(); err != nil {
				common.ApiErrorMsg(c, err.Error())
				return
			}
		}
	case "BTMailAPIURL":
		if option.Value != "" {
			if err := common.ValidateBTMailAPIURL(option.Value.(string)); err != nil {
				common.ApiErrorMsg(c, err.Error())
				return
			}
		}
	case "BotProtectionProvider":
		provider := option.Value.(string)
		if !common.IsBotProtectionProvider(provider) {
			common.ApiErrorMsg(c, "不支持的机器人保护提供商")
			return
		}
		if common.TurnstileCheckEnabled {
			if err := validateBotProtectionConfiguration(provider); err != nil {
				common.ApiErrorMsg(c, err.Error())
				return
			}
		}
	case "CapServerURL":
		if option.Value != "" {
			if err := validateCapServerURL(option.Value.(string)); err != nil {
				common.ApiErrorMsg(c, err.Error())
				return
			}
		}
	case "GitHubOAuthEnabled":
		if option.Value == "true" && common.GitHubClientId == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 GitHub OAuth，请先填入 GitHub Client Id 以及 GitHub Client Secret！",
			})
			return
		}
	case "discord.enabled":
		if option.Value == "true" && system_setting.GetDiscordSettings().ClientId == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 Discord OAuth，请先填入 Discord Client Id 以及 Discord Client Secret！",
			})
			return
		}
	case "oidc.enabled":
		if option.Value == "true" && system_setting.GetOIDCSettings().ClientId == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 OIDC 登录，请先填入 OIDC Client Id 以及 OIDC Client Secret！",
			})
			return
		}
	case "LinuxDOOAuthEnabled":
		if option.Value == "true" && common.LinuxDOClientId == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 LinuxDO OAuth，请先填入 LinuxDO Client Id 以及 LinuxDO Client Secret！",
			})
			return
		}
	case "EmailDomainRestrictionEnabled":
		if option.Value == "true" && len(common.EmailDomainWhitelist) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用邮箱域名限制，请先填入限制的邮箱域名！",
			})
			return
		}
	case "WeChatAuthEnabled":
		if option.Value == "true" && common.WeChatServerAddress == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用微信登录，请先填入微信登录相关配置信息！",
			})
			return
		}
	case "TurnstileCheckEnabled":
		if option.Value == "true" {
			if err := validateBotProtectionConfiguration(common.BotProtectionProvider); err != nil {
				common.ApiErrorMsg(c, err.Error())
				return
			}
		}
		if option.Value == "true" && !common.BotProtectionLoginEnabled &&
			!common.BotProtectionRegisterEnabled &&
			!common.BotProtectionEmailVerificationEnabled &&
			!common.BotProtectionPasswordResetEnabled &&
			!common.BotProtectionCheckinEnabled {
			common.ApiErrorMsg(c, "请至少启用一个机器人保护功能")
			return
		}
	case "TelegramOAuthEnabled":
		if option.Value == "true" && common.TelegramBotToken == "" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无法启用 Telegram OAuth，请先填入 Telegram Bot Token！",
			})
			return
		}
	case "theme.frontend":
		if option.Value != "default" {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "Classic 前端已移除，主题只能设置为 default",
			})
			return
		}
	case "GroupRatio":
		err = ratio_setting.CheckGroupRatio(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "gemini.safety_settings":
		err = model_setting.ValidateGeminiSafetySettings(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "claude.default_max_tokens":
		err = model_setting.ValidateClaudeDefaultMaxTokens(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case operation_setting.ToolPriceOptionKey:
		err = operation_setting.ValidateToolPricesJSON(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "billing_setting." + billing_setting.PriceSchedulesField:
		err = billing_setting.ValidatePriceSchedulesJSON(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "模型时段价格设置失败: " + err.Error(),
			})
			return
		}
	case system_setting.GlobalBannerOptionPrefix + "enabled",
		system_setting.GlobalBannerOptionPrefix + "content",
		system_setting.GlobalBannerOptionPrefix + "background_color",
		system_setting.GlobalBannerOptionPrefix + "text_color",
		system_setting.GlobalBannerOptionPrefix + "icon",
		system_setting.GlobalBannerOptionPrefix + "countdown_enabled",
		system_setting.GlobalBannerOptionPrefix + "countdown_end_at",
		system_setting.GlobalBannerOptionPrefix + "link_url",
		system_setting.MarketingBannerOptionPrefix + "enabled",
		system_setting.MarketingBannerOptionPrefix + "content",
		system_setting.MarketingBannerOptionPrefix + "background_color",
		system_setting.MarketingBannerOptionPrefix + "text_color",
		system_setting.MarketingBannerOptionPrefix + "icon",
		system_setting.MarketingBannerOptionPrefix + "countdown_enabled",
		system_setting.MarketingBannerOptionPrefix + "countdown_end_at",
		system_setting.MarketingBannerOptionPrefix + "link_url":
		if option.Key == system_setting.GlobalBannerOptionPrefix+"icon" ||
			option.Key == system_setting.MarketingBannerOptionPrefix+"icon" {
			option.Value = strings.TrimSpace(option.Value.(string))
		}
		err = system_setting.ValidateBannerOption(option.Key, option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "横幅设置失败: " + err.Error(),
			})
			return
		}
	case "ImageRatio":
		err = ratio_setting.UpdateImageRatioByJSONString(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "图片倍率设置失败: " + err.Error(),
			})
			return
		}
	case "AudioRatio":
		err = ratio_setting.UpdateAudioRatioByJSONString(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "音频倍率设置失败: " + err.Error(),
			})
			return
		}
	case "AudioCompletionRatio":
		err = ratio_setting.UpdateAudioCompletionRatioByJSONString(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "音频补全倍率设置失败: " + err.Error(),
			})
			return
		}
	case "CreateCacheRatio":
		err = ratio_setting.UpdateCreateCacheRatioByJSONString(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "缓存创建倍率设置失败: " + err.Error(),
			})
			return
		}
	case "ModelRequestRateLimitGroup":
		err = setting.CheckModelRequestRateLimitGroup(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "AutomaticDisableStatusCodes":
		_, err = operation_setting.ParseHTTPStatusCodeRanges(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "AutomaticRetryStatusCodes":
		_, err = operation_setting.ParseHTTPStatusCodeRanges(option.Value.(string))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.api_info":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "ApiInfo")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.announcements":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "Announcements")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.faq":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "FAQ")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	case "console_setting.uptime_kuma_groups":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "UptimeKumaGroups")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}
	err = model.UpdateOption(option.Key, option.Value.(string))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// 出于安全考虑只记录被修改的配置项名称，不记录配置值（可能含密钥等敏感信息）。
	recordManageAudit(c, "option.update", map[string]interface{}{
		"key": option.Key,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}
