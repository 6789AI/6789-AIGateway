package billing_setting

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// NormalizeVideoResolution names billing tiers, independently of provider spelling.
func NormalizeVideoResolution(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "480", "480p":
		return "480p"
	case "720", "720p":
		return "720p"
	case "768", "768p":
		return "768p"
	case "1080", "1080p":
		return "1080p"
	case "2k", "1440", "1440p":
		return "2k"
	}
	return ""
}

// VideoResolutionFromSize also accepts portrait and landscape pixel dimensions.
func VideoResolutionFromSize(value string) string {
	if resolution := NormalizeVideoResolution(value); resolution != "" {
		return resolution
	}
	parts := strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return r == 'x' || r == '*' || r == '×' })
	if len(parts) != 2 {
		return ""
	}
	width, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || width <= 0 {
		return ""
	}
	height, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || height <= 0 {
		return ""
	}
	return NormalizeVideoResolution(strconv.Itoa(min(width, height)))
}

func GetVideoResolutionPrice(model, resolution string) (float64, bool) {
	resolution = NormalizeVideoResolution(resolution)
	if resolution == "" {
		return 0, false
	}
	price, ok := billingSetting.VideoResolutionPrices[model][resolution]
	return price, ok && price >= 0 && !math.IsNaN(price) && !math.IsInf(price, 0)
}

func GetVideoResolutionPricesCopy() map[string]map[string]float64 {
	result := make(map[string]map[string]float64, len(billingSetting.VideoResolutionPrices))
	for model, prices := range billingSetting.VideoResolutionPrices {
		result[model] = make(map[string]float64, len(prices))
		for resolution, price := range prices {
			result[model][resolution] = price
		}
	}
	return result
}

func ValidateVideoResolutionPricesJSON(value string) error {
	var settings map[string]map[string]*float64
	if err := common.UnmarshalJsonStr(value, &settings); err != nil {
		return fmt.Errorf("invalid video resolution prices: %w", err)
	}
	if settings == nil {
		return fmt.Errorf("video resolution prices must be a JSON object")
	}
	for model, prices := range settings {
		if strings.TrimSpace(model) == "" || prices == nil {
			return fmt.Errorf("model name and resolution price object are required")
		}
		for resolution, price := range prices {
			if NormalizeVideoResolution(resolution) != resolution || resolution == "" {
				return fmt.Errorf("unsupported video resolution %q; use 480p, 720p, 768p, 1080p or 2k", resolution)
			}
			if price == nil || *price < 0 || math.IsNaN(*price) || math.IsInf(*price, 0) {
				return fmt.Errorf("video resolution price must be a finite number greater than or equal to zero")
			}
		}
	}
	return nil
}

// FilterTaskBillingRatios applies the submission policy again during settlement.
func FilterTaskBillingRatios(ratios map[string]float64, multiplyDuration, resolutionPrice bool) map[string]float64 {
	if multiplyDuration && !resolutionPrice {
		return ratios
	}
	filtered := make(map[string]float64, len(ratios))
	for name, ratio := range ratios {
		if !multiplyDuration && name == "seconds" {
			continue
		}
		if resolutionPrice && (name == "size" || name == "resolution" || strings.HasPrefix(name, "resolution-")) {
			continue
		}
		filtered[name] = ratio
	}
	return filtered
}
