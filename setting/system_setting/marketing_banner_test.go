package system_setting

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateMarketingBannerOption(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr string
	}{
		{
			name:  "accepts a six digit hex background",
			key:   MarketingBannerOptionPrefix + "background_color",
			value: "#A3E635",
		},
		{
			name:  "accepts a six digit hex button color",
			key:   MarketingBannerOptionPrefix + "button_color",
			value: "#123456",
		},
		{
			name:    "rejects arbitrary CSS colors",
			key:     MarketingBannerOptionPrefix + "text_color",
			value:   "var(--foreground)",
			wantErr: "6-digit hex",
		},
		{
			name:    "rejects invalid enabled values",
			key:     MarketingBannerOptionPrefix + "enabled",
			value:   "yes",
			wantErr: "true or false",
		},
		{
			name:    "rejects content beyond the display limit",
			key:     MarketingBannerOptionPrefix + "content",
			value:   strings.Repeat("界", 301),
			wantErr: "300 characters",
		},
		{
			name:  "accepts custom button text",
			key:   MarketingBannerOptionPrefix + "button_text",
			value: "点击查看",
		},
		{
			name:    "rejects button text beyond the display limit",
			key:     MarketingBannerOptionPrefix + "button_text",
			value:   strings.Repeat("界", 51),
			wantErr: "50 characters",
		},
		{
			name:  "accepts a supported icon",
			key:   MarketingBannerOptionPrefix + "icon",
			value: "coupon",
		},
		{
			name:  "accepts no icon",
			key:   MarketingBannerOptionPrefix + "icon",
			value: "",
		},
		{
			name:  "accepts custom icon characters",
			key:   MarketingBannerOptionPrefix + "icon",
			value: "限时福利 🎉",
		},
		{
			name:  "accepts one hundred Unicode icon characters",
			key:   MarketingBannerOptionPrefix + "icon",
			value: strings.Repeat("🎉", 100),
		},
		{
			name:    "rejects custom icon beyond the display limit",
			key:     MarketingBannerOptionPrefix + "icon",
			value:   strings.Repeat("🎉", 101),
			wantErr: "100 characters",
		},
		{
			name:  "accepts an absolute HTTPS link",
			key:   MarketingBannerOptionPrefix + "link_url",
			value: "https://example.com/promotion",
		},
		{
			name:  "accepts a site relative link",
			key:   MarketingBannerOptionPrefix + "link_url",
			value: "/pricing?campaign=summer",
		},
		{
			name:    "rejects a dangerous link scheme",
			key:     MarketingBannerOptionPrefix + "link_url",
			value:   "javascript:alert(1)",
			wantErr: "HTTP(S)",
		},
		{
			name:    "rejects a protocol relative link",
			key:     MarketingBannerOptionPrefix + "link_url",
			value:   "//example.com/promotion",
			wantErr: "HTTP(S)",
		},
		{
			name:  "accepts a valid countdown timestamp",
			key:   MarketingBannerOptionPrefix + "countdown_end_at",
			value: "1787500000",
		},
		{
			name:    "rejects a negative countdown timestamp",
			key:     MarketingBannerOptionPrefix + "countdown_end_at",
			value:   "-1",
			wantErr: "valid Unix timestamp",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateMarketingBannerOption(test.key, test.value)
			if test.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestValidateGlobalBannerOptionUsesSharedRules(t *testing.T) {
	assert.NoError(t, ValidateBannerOption(GlobalBannerOptionPrefix+"icon", "📣 公告"))
	assert.ErrorContains(
		t,
		ValidateBannerOption(GlobalBannerOptionPrefix+"link_url", "javascript:alert(1)"),
		"HTTP(S)",
	)
}
