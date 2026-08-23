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
			name:  "accepts a supported icon",
			key:   MarketingBannerOptionPrefix + "icon",
			value: "coupon",
		},
		{
			name:    "rejects an unsupported icon",
			key:     MarketingBannerOptionPrefix + "icon",
			value:   "custom-svg",
			wantErr: "not supported",
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
