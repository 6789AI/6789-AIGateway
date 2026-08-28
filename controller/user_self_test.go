package controller

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetSelfFallsBackWhenPromotionUsageQueryFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB := model.DB
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}))
	t.Cleanup(func() {
		model.DB = previousDB
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	savedConfig := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		savedConfig[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(savedConfig))
	})

	now := time.Now()
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{"fallback-model":"scheduled_price"}`,
		"billing_setting.price_schedules": fmt.Sprintf(
			`{"fallback-model":[{"type":"absolute","price":0,"start_at":%d,"end_at":%d}]}`,
			now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix(),
		),
	}))

	user := model.User{
		Username:  "promotion-fallback-user",
		Password:  "password",
		Role:      common.RoleCommonUser,
		Status:    common.UserStatusEnabled,
		Group:     "default",
		UsedQuota: int(common.QuotaPerUnit * 20),
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register("test:fail_promotion_usage_query", func(tx *gorm.DB) {
		if tx.Statement.Table == "promotion_usages" {
			tx.AddError(errors.New("promotion usage query failed"))
		}
	}))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/user/self", nil)
	context.Set("id", user.Id)
	context.Set("role", user.Role)

	GetSelf(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			FreeUsageLimit     int  `json:"free_usage_limit"`
			FreeUsageUsed      int  `json:"free_usage_used"`
			FreeUsageRemaining int  `json:"free_usage_remaining"`
			FreeUsageActive    bool `json:"free_usage_active"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, 60, response.Data.FreeUsageLimit)
	assert.Zero(t, response.Data.FreeUsageUsed)
	assert.Equal(t, 60, response.Data.FreeUsageRemaining)
	assert.True(t, response.Data.FreeUsageActive)
}
