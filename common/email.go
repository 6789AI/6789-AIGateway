package common

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	EmailSenderProviderSMTP   = "smtp"
	EmailSenderProviderBTMail = "bt_mail"
	btMailResponseLimit       = 1 << 20
)

var btMailHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

type btMailResponse struct {
	Status bool   `json:"status"`
	Msg    string `json:"msg"`
}

func IsEmailSenderProvider(provider string) bool {
	return provider == EmailSenderProviderSMTP || provider == EmailSenderProviderBTMail
}

func ValidateBTMailAPIURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil ||
		parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.User != nil ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" {
		return fmt.Errorf("宝塔邮局 API 地址必须是有效的 HTTP 或 HTTPS URL")
	}
	return nil
}

func ValidateBTMailConfiguration() error {
	if BTMailAPIURL == "" || BTMailFrom == "" || BTMailPassword == "" {
		return fmt.Errorf("请先填写宝塔邮局 API 地址、发件邮箱和邮箱密码")
	}
	return ValidateBTMailAPIURL(BTMailAPIURL)
}

func generateMessageID() (string, error) {
	split := strings.Split(SMTPFrom, "@")
	if len(split) < 2 {
		return "", fmt.Errorf("invalid SMTP account")
	}
	domain := strings.Split(SMTPFrom, "@")[1]
	return fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), GetRandomString(12), domain), nil
}

func shouldUseSMTPLoginAuth() bool {
	if SMTPForceAuthLogin {
		return true
	}
	return isOutlookServer(SMTPAccount) || slices.Contains(EmailLoginAuthServerList, SMTPServer)
}

func getSMTPAuth() smtp.Auth {
	return AutoSMTPAuth(SMTPAccount, SMTPToken)
}

func shouldAuthenticateSMTP() bool {
	return SMTPAccount != "" && SMTPToken != ""
}

func smtpTLSConfig() *tls.Config {
	return &tls.Config{
		ServerName:         SMTPServer,
		InsecureSkipVerify: SMTPInsecureSkipVerify, // #nosec G402 -- admin-controlled SMTP compatibility option.
	}
}

func newSMTPClient(addr string) (*smtp.Client, error) {
	if SMTPSSLEnabled || (SMTPPort == 465 && !SMTPStartTLSEnabled) {
		conn, err := tls.Dial("tcp", addr, smtpTLSConfig())
		if err != nil {
			return nil, err
		}
		client, err := smtp.NewClient(conn, SMTPServer)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		return client, nil
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, err
	}

	if SMTPStartTLSEnabled {
		startTLSSupported, _ := client.Extension("STARTTLS")
		if !startTLSSupported {
			_ = client.Close()
			return nil, fmt.Errorf("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(smtpTLSConfig()); err != nil {
			_ = client.Close()
			return nil, err
		}
	}

	return client, nil
}

func SendEmail(subject string, receiver string, content string) error {
	switch EmailSenderProvider {
	case "", EmailSenderProviderSMTP:
		return sendSMTPEmail(subject, receiver, content)
	case EmailSenderProviderBTMail:
		return sendBTMailEmail(subject, receiver, content)
	default:
		return fmt.Errorf("不支持的发件方式: %s", EmailSenderProvider)
	}
}

func decorateBTMailContent(senderName string, content string) string {
	senderName = strings.TrimSpace(senderName)
	if senderName == "" {
		return content
	}

	var identity strings.Builder
	identity.WriteString(`<table role="presentation" cellpadding="0" cellspacing="0" style="margin:0 0 20px;border-collapse:collapse"><tr>`)
	identity.WriteString(`<td style="padding:0;vertical-align:middle;font-family:Arial,sans-serif;font-size:16px;font-weight:600;line-height:1.4">`)
	identity.WriteString(html.EscapeString(senderName))
	identity.WriteString(`</td>`)
	identity.WriteString(`</tr></table>`)
	identityHTML := identity.String()
	bodyStart := strings.Index(strings.ToLower(content), "<body")
	if bodyStart >= 0 {
		bodyTagEnd := strings.Index(content[bodyStart:], ">")
		if bodyTagEnd >= 0 {
			insertAt := bodyStart + bodyTagEnd + 1
			return content[:insertAt] + identityHTML + content[insertAt:]
		}
	}
	return identityHTML + content
}

func sendSMTPEmail(subject string, receiver string, content string) error {
	if SMTPFrom == "" { // for compatibility
		SMTPFrom = SMTPAccount
	}
	id, err2 := generateMessageID()
	if err2 != nil {
		return err2
	}
	if SMTPServer == "" && SMTPAccount == "" {
		return fmt.Errorf("SMTP 服务器未配置")
	}
	encodedSubject := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
	senderName := strings.TrimSpace(SMTPFromName)
	if senderName == "" {
		senderName = SystemName
	}
	fromHeader := (&mail.Address{Name: senderName, Address: SMTPFrom}).String()
	mail := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"Date: %s\r\n"+
		"Message-ID: %s\r\n"+ // 添加 Message-ID 头
		"Content-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n",
		receiver, fromHeader, encodedSubject, time.Now().Format(time.RFC1123Z), id, content))
	auth := getSMTPAuth()
	addr := fmt.Sprintf("%s:%d", SMTPServer, SMTPPort)
	to := strings.Split(receiver, ";")
	var err error
	client, err := newSMTPClient(addr)
	if err != nil {
		return err
	}
	defer client.Close()
	if shouldAuthenticateSMTP() {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}
	if err = client.Mail(SMTPFrom); err != nil {
		return err
	}
	for _, receiver := range to {
		if err = client.Rcpt(receiver); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(mail)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	err = client.Quit()
	if err != nil {
		SysError(fmt.Sprintf("failed to send email to %s: %v", receiver, err))
	}
	return err
}

func sendBTMailEmail(subject string, receiver string, content string) error {
	if err := ValidateBTMailConfiguration(); err != nil {
		return err
	}

	recipients := strings.FieldsFunc(receiver, func(r rune) bool {
		return r == ';' || r == ','
	})
	if len(recipients) == 0 {
		return fmt.Errorf("收件邮箱不能为空")
	}
	for i := range recipients {
		recipients[i] = strings.TrimSpace(recipients[i])
	}
	content = decorateBTMailContent(BTMailFromName, content)

	form := url.Values{
		"mail_from": {BTMailFrom},
		"password":  {BTMailPassword},
		"mail_to":   {strings.Join(recipients, ",")},
		"subject":   {subject},
		"content":   {content},
		"subtype":   {"html"},
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimSpace(BTMailAPIURL), strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := btMailHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("宝塔邮局 API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result btMailResponse
	decodeErr := DecodeJson(io.LimitReader(resp.Body, btMailResponseLimit), &result)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if result.Msg != "" {
			return fmt.Errorf("宝塔邮局发送失败: %s", result.Msg)
		}
		return fmt.Errorf("宝塔邮局 API 返回 HTTP %d", resp.StatusCode)
	}
	if decodeErr != nil {
		return fmt.Errorf("解析宝塔邮局 API 响应失败: %w", decodeErr)
	}
	if !result.Status {
		if result.Msg == "" {
			return fmt.Errorf("宝塔邮局发送失败")
		}
		return fmt.Errorf("宝塔邮局发送失败: %s", result.Msg)
	}
	return nil
}
