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
		"global_banner.enabled":                     "false",
		"global_banner.content":                     "Service update",
		"global_banner.background_color":            "#0EA5E9",
		"global_banner.text_color":                  "#082F49",
		"global_banner.icon":                        "gift",
		"global_banner.countdown_enabled":           "false",
		"global_banner.countdown_end_at":            "0",
		"global_banner.link_url":                    "/pricing",
		"marketing_banner.background_color":         "#123456",
		"marketing_banner.text_color":               "#FEDCBA",
		"marketing_banner.icon":                     "rocket",
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
			Active       bool  `json:"active"`
			NextChangeAt int64 `json:"next_change_at"`
			GlobalBanner struct {
				Enabled          bool   `json:"enabled"`
				Content          string `json:"content"`
				BackgroundColor  string `json:"background_color"`
				TextColor        string `json:"text_color"`
				Icon             string `json:"icon"`
				CountdownEnabled bool   `json:"countdown_enabled"`
				CountdownEndAt   int64  `json:"countdown_end_at"`
				LinkURL          string `json:"link_url"`
			} `json:"global_banner"`
			FreeModelBanner struct {
				BackgroundColor string `json:"background_color"`
				TextColor       string `json:"text_color"`
				Icon            string `json:"icon"`
				LinkURL         string `json:"link_url"`
			} `json:"free_model_banner"`
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
	assert.False(t, response.Data.GlobalBanner.Enabled)
	assert.Equal(t, "Service update", response.Data.GlobalBanner.Content)
	assert.Equal(t, "#0EA5E9", response.Data.GlobalBanner.BackgroundColor)
	assert.Equal(t, "#082F49", response.Data.GlobalBanner.TextColor)
	assert.Equal(t, "gift", response.Data.GlobalBanner.Icon)
	assert.False(t, response.Data.GlobalBanner.CountdownEnabled)
	assert.Zero(t, response.Data.GlobalBanner.CountdownEndAt)
	assert.Equal(t, "/pricing", response.Data.GlobalBanner.LinkURL)
	assert.Equal(t, "#123456", response.Data.FreeModelBanner.BackgroundColor)
	assert.Equal(t, "#FEDCBA", response.Data.FreeModelBanner.TextColor)
	assert.Equal(t, "rocket", response.Data.FreeModelBanner.Icon)
	assert.Equal(t, "https://example.com/promotion", response.Data.FreeModelBanner.LinkURL)
}

func TestGetFreeModelPromotionsIgnoresLegacyMarketingCountdown(t *testing.T) {
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
	modelEndAt := now.Add(time.Hour).Unix()
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode":              `{"banner-model":"scheduled_price"}`,
		"billing_setting.free_model_banner_enabled": "true",
		"marketing_banner.enabled":                  "true",
		"marketing_banner.content":                  "Legacy campaign",
		"marketing_banner.countdown_enabled":        "true",
		"marketing_banner.countdown_end_at":         fmt.Sprintf("%d", now.Add(time.Minute).Unix()),
		"billing_setting.price_schedules": fmt.Sprintf(
			`{"banner-model":[{"type":"absolute","price":0,"start_at":%d,"end_at":%d,"show_banner":true}]}`,
			now.Add(-time.Hour).Unix(), modelEndAt,
		),
	}))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	GetFreeModelPromotions(context)

	var response struct {
		Data struct {
			NextChangeAt int64 `json:"next_change_at"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, modelEndAt, response.Data.NextChangeAt)
	assert.NotContains(t, recorder.Body.String(), `"marketing_banner"`)
	assert.NotContains(t, recorder.Body.String(), "Legacy campaign")
}
