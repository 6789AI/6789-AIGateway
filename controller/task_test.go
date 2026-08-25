package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetUserTaskReturnsChannelIDWithoutProviderPlatform(t *testing.T) {
	oldDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Task{}))
	model.DB = db
	t.Cleanup(func() { model.DB = oldDB })

	require.NoError(t, db.Create([]*model.Task{
		{
			TaskID:    "task_grsai",
			Platform:  constant.TaskPlatformGrsai,
			UserId:    42,
			ChannelId: 90,
			Status:    model.TaskStatusSuccess,
		},
		{
			TaskID:    "task_numeric_platform",
			Platform:  constant.TaskPlatform("55"),
			UserId:    42,
			ChannelId: 55,
			Status:    model.TaskStatusSuccess,
		},
	}).Error)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/task/self?p=1&page_size=20&platform=grsai",
		nil,
	)
	context.Set("id", 42)

	GetUserTask(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Total int            `json:"total"`
			Items []*dto.TaskDto `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.Equal(t, 2, response.Data.Total)
	require.Len(t, response.Data.Items, 2)

	channelIDs := make([]int, 0, len(response.Data.Items))
	for _, task := range response.Data.Items {
		channelIDs = append(channelIDs, task.ChannelId)
		assert.Empty(t, task.Platform)
	}
	assert.ElementsMatch(t, []int{55, 90}, channelIDs)
}
