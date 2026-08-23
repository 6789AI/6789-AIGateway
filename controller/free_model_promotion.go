package controller

import (
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

func GetFreeModelPromotions(c *gin.Context) {
	now := time.Now()
	promotions := billing_setting.GetActiveFreeModelPromotions(now)
	marketingBanner := system_setting.GetMarketingBannerSettings()
	marketingBanner.Enabled = marketingBanner.Enabled &&
		strings.TrimSpace(marketingBanner.Content) != "" &&
		(!marketingBanner.CountdownEnabled || marketingBanner.CountdownEndAt > now.Unix())
	if !marketingBanner.Enabled {
		marketingBanner.Content = ""
		marketingBanner.LinkURL = ""
		marketingBanner.CountdownEnabled = false
		marketingBanner.CountdownEndAt = 0
	}
	nextChangeAt := int64(0)
	if marketingBanner.Enabled && marketingBanner.CountdownEnabled {
		nextChangeAt = marketingBanner.CountdownEndAt
	}
	for _, promotion := range promotions {
		if promotion.EndsAt > 0 && (nextChangeAt == 0 || promotion.EndsAt < nextChangeAt) {
			nextChangeAt = promotion.EndsAt
		}
	}

	c.Header("Cache-Control", "public, max-age=15")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"active":           len(promotions) > 0,
			"marketing_banner": marketingBanner,
			"server_time":      now.Unix(),
			"next_change_at":   nextChangeAt,
			"models":           promotions,
		},
	})
}
