package relay

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTaskQuotaFromFixedPriceAppliesImageCountBeforeConversion(t *testing.T) {
	priceData := hosttypes.PriceData{
		UsePrice:   true,
		ModelPrice: 0.0000012,
		GroupRatioInfo: hosttypes.GroupRatioInfo{
			GroupRatio: 1,
		},
	}
	priceData.AddOtherRatio("n", 3)

	quota, clamp := taskQuotaFromPriceData(priceData, 0)

	require.Nil(t, clamp)
	assert.Equal(t, 1, quota)
}

func TestGetTaskAdaptorSupportsGrsaiPlatform(t *testing.T) {
	adaptor := GetTaskAdaptor(constant.TaskPlatformGrsai)
	require.NotNil(t, adaptor)
	assert.Equal(t, "Grsai", adaptor.GetChannelName())
}

func TestImageFetchByIDIsScopedToOwnerAndImageTasks(t *testing.T) {
	oldDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Task{}))
	model.DB = db
	t.Cleanup(func() { model.DB = oldDB })

	platform := constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeAli))
	require.NoError(t, db.Create([]*model.Task{
		{
			TaskID:     "task_image",
			UserId:     7,
			Platform:   platform,
			Action:     constant.TaskActionImageGenerate,
			Status:     model.TaskStatusQueued,
			Progress:   "0%",
			SubmitTime: 123,
			Properties: model.Properties{OriginModelName: "wanx-v1"},
		},
		{
			TaskID:   "task_video",
			UserId:   7,
			Platform: platform,
			Action:   constant.TaskActionGenerate,
			Status:   model.TaskStatusQueued,
			Progress: "0%",
		},
	}).Error)

	newContext := func(userID int, taskID string) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/v1/images/generations/"+taskID, nil)
		c.Params = gin.Params{{Key: "task_id", Value: taskID}}
		c.Set("id", userID)
		return c
	}

	t.Run("owner can fetch image task", func(t *testing.T) {
		body, taskErr := imageFetchByIDRespBodyBuilder(newContext(7, "task_image"))
		require.Nil(t, taskErr)
		var response dto.ImageTaskResponse
		require.NoError(t, common.Unmarshal(body, &response))
		assert.Equal(t, "task_image", response.ID)
		assert.Equal(t, dto.ImageTaskStatusQueued, response.Status)
	})

	t.Run("other user receives not found", func(t *testing.T) {
		_, taskErr := imageFetchByIDRespBodyBuilder(newContext(8, "task_image"))
		require.NotNil(t, taskErr)
		assert.Equal(t, http.StatusNotFound, taskErr.StatusCode)
		assert.Equal(t, "task_not_exist", taskErr.Code)
	})

	t.Run("non image task receives not found", func(t *testing.T) {
		_, taskErr := imageFetchByIDRespBodyBuilder(newContext(7, "task_video"))
		require.NotNil(t, taskErr)
		assert.Equal(t, http.StatusNotFound, taskErr.StatusCode)
		assert.Equal(t, "task_not_exist", taskErr.Code)
	})
}
