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
	promotions := billing_setting.GetActiveModelPromotions(now)
	globalBanner := system_setting.GetGlobalBannerSettings()
	globalBanner.Enabled = globalBanner.Enabled &&
		strings.TrimSpace(globalBanner.Content) != "" &&
		(!globalBanner.CountdownEnabled || globalBanner.CountdownEndAt > now.Unix())
	freeModelBanner := system_setting.GetMarketingBannerSettings()
	nextChangeAt := int64(0)
	if globalBanner.Enabled && globalBanner.CountdownEnabled {
		nextChangeAt = globalBanner.CountdownEndAt
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
			"active":        len(promotions) > 0,
			"global_banner": globalBanner,
			"free_model_banner": gin.H{
				"background_color": freeModelBanner.BackgroundColor,
				"text_color":       freeModelBanner.TextColor,
				"icon":             freeModelBanner.Icon,
				"link_url":         freeModelBanner.LinkURL,
			},
			"server_time":    now.Unix(),
			"next_change_at": nextChangeAt,
			"models":         promotions,
		},
	})
}
