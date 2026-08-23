package controller

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFreeModelPromotionsReturnsPublicCountdownData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	savedConfig := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		savedConfig[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(savedConfig))
	})

	now := time.Now()
	endAt := now.Add(time.Hour).Unix()
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode":              `{"banner-model":"scheduled_price"}`,
		"billing_setting.free_model_banner_enabled": "true",
		"marketing_banner.enabled":                  "true",
		"marketing_banner.content":                  "Weekend bonus",
		"marketing_banner.background_color":         "#123456",
		"marketing_banner.text_color":               "#FEDCBA",
		"marketing_banner.icon":                     "rocket",
		"marketing_banner.countdown_enabled":        "true",
		"marketing_banner.countdown_end_at":         fmt.Sprintf("%d", endAt),
		"marketing_banner.link_url":                 "https://example.com/promotion",
		"billing_setting.price_schedules": fmt.Sprintf(
			`{"banner-model":[{"type":"absolute","price":0,"start_at":%d,"end_at":%d,"show_banner":true}]}`,
			now.Add(-time.Hour).Unix(), endAt,
		),
	}))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	GetFreeModelPromotions(context)

	require.Equal(t, 200, recorder.Code)
	assert.Equal(t, "public, max-age=15", recorder.Header().Get("Cache-Control"))
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Active          bool  `json:"active"`
			NextChangeAt    int64 `json:"next_change_at"`
			MarketingBanner struct {
				Enabled          bool   `json:"enabled"`
				Content          string `json:"content"`
				BackgroundColor  string `json:"background_color"`
				TextColor        string `json:"text_color"`
				Icon             string `json:"icon"`
				CountdownEnabled bool   `json:"countdown_enabled"`
				CountdownEndAt   int64  `json:"countdown_end_at"`
				LinkURL          string `json:"link_url"`
			} `json:"marketing_banner"`
			Models []struct {
				ModelName string `json:"model_name"`
				EndsAt    int64  `json:"ends_at"`
			} `json:"models"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.True(t, response.Data.Active)
	require.Len(t, response.Data.Models, 1)
	assert.Equal(t, "banner-model", response.Data.Models[0].ModelName)
	assert.Equal(t, endAt, response.Data.Models[0].EndsAt)
	assert.Equal(t, endAt, response.Data.NextChangeAt)
	assert.True(t, response.Data.MarketingBanner.Enabled)
	assert.Equal(t, "Weekend bonus", response.Data.MarketingBanner.Content)
	assert.Equal(t, "#123456", response.Data.MarketingBanner.BackgroundColor)
	assert.Equal(t, "#FEDCBA", response.Data.MarketingBanner.TextColor)
	assert.Equal(t, "rocket", response.Data.MarketingBanner.Icon)
	assert.True(t, response.Data.MarketingBanner.CountdownEnabled)
	assert.Equal(t, endAt, response.Data.MarketingBanner.CountdownEndAt)
	assert.Equal(t, "https://example.com/promotion", response.Data.MarketingBanner.LinkURL)
}

func TestGetFreeModelPromotionsOmitsDisabledMarketingContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	savedConfig := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		savedConfig[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(savedConfig))
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"marketing_banner.enabled":           "false",
		"marketing_banner.content":           "Unpublished campaign",
		"marketing_banner.countdown_enabled": "true",
		"marketing_banner.countdown_end_at":  "1787500000",
		"marketing_banner.link_url":          "https://example.com/draft",
	}))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	GetFreeModelPromotions(context)

	var response struct {
		Data struct {
			MarketingBanner struct {
				Enabled          bool   `json:"enabled"`
				Content          string `json:"content"`
				CountdownEnabled bool   `json:"countdown_enabled"`
				CountdownEndAt   int64  `json:"countdown_end_at"`
				LinkURL          string `json:"link_url"`
			} `json:"marketing_banner"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Data.MarketingBanner.Enabled)
	assert.Empty(t, response.Data.MarketingBanner.Content)
	assert.False(t, response.Data.MarketingBanner.CountdownEnabled)
	assert.Zero(t, response.Data.MarketingBanner.CountdownEndAt)
	assert.Empty(t, response.Data.MarketingBanner.LinkURL)
}

func TestGetFreeModelPromotionsHidesExpiredMarketingCountdown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	savedConfig := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		savedConfig[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(savedConfig))
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"marketing_banner.enabled":           "true",
		"marketing_banner.content":           "Expired campaign",
		"marketing_banner.countdown_enabled": "true",
		"marketing_banner.countdown_end_at":  fmt.Sprintf("%d", time.Now().Add(-time.Minute).Unix()),
		"marketing_banner.link_url":          "https://example.com/expired",
	}))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	GetFreeModelPromotions(context)

	var response struct {
		Data struct {
			MarketingBanner struct {
				Enabled          bool   `json:"enabled"`
				Content          string `json:"content"`
				CountdownEnabled bool   `json:"countdown_enabled"`
				CountdownEndAt   int64  `json:"countdown_end_at"`
				LinkURL          string `json:"link_url"`
			} `json:"marketing_banner"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Data.MarketingBanner.Enabled)
	assert.Empty(t, response.Data.MarketingBanner.Content)
	assert.False(t, response.Data.MarketingBanner.CountdownEnabled)
	assert.Zero(t, response.Data.MarketingBanner.CountdownEndAt)
	assert.Empty(t, response.Data.MarketingBanner.LinkURL)
}
