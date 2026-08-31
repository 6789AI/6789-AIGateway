package controller

import (
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

func TestGetUserIncludesPromotionUsageForAuthorizedAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB := model.DB
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.PromotionUsage{}))
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
	startAt := now.Add(-time.Hour).Unix()
	endAt := now.Add(time.Hour).Unix()
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{"promotion-detail-model":"scheduled_price"}`,
		"billing_setting.price_schedules": fmt.Sprintf(
			`{"promotion-detail-model":[{"type":"absolute","price":0,"start_at":%d,"end_at":%d}]}`,
			startAt, endAt,
		),
	}))

	user := model.User{
		Username:  "promotion-detail-user",
		Password:  "password",
		Role:      common.RoleCommonUser,
		Status:    common.UserStatusEnabled,
		Group:     "default",
		UsedQuota: int(common.QuotaPerUnit * 10),
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.PromotionUsage{
		UserId:        user.Id,
		ActivityKey:   fmt.Sprintf("v1:absolute:%d:%d", startAt, endAt),
		UsedCount:     2,
		ReservedCount: 1,
	}).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/user/"+fmt.Sprint(user.Id), nil)
	context.Params = gin.Params{{Key: "id", Value: fmt.Sprint(user.Id)}}
	context.Set("role", common.RoleAdminUser)

	GetUser(context)

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
	assert.Equal(t, 30, response.Data.FreeUsageLimit)
	assert.Equal(t, 3, response.Data.FreeUsageUsed)
	assert.Equal(t, 27, response.Data.FreeUsageRemaining)
	assert.True(t, response.Data.FreeUsageActive)
}
