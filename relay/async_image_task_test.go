package relay

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
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

func TestTaskBillingRatiosRespectsFixedPriceDurationSetting(t *testing.T) {
	original := billing_setting.GetTaskDurationMultiplierCopy()
	restoreJSON, err := common.Marshal(original)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
			"billing_setting." + billing_setting.TaskDurationMultiplierField: string(restoreJSON),
		}))
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting." + billing_setting.TaskDurationMultiplierField: `{"fixed-video":false}`,
	}))
	ratioInput := map[string]float64{"seconds": 10, "resolution": 2}

	t.Run("missing setting keeps duration multiplier", func(t *testing.T) {
		assert.Equal(t, ratioInput, taskBillingRatios("default-video", true, ratioInput))
	})

	t.Run("fixed price can disable duration without removing other ratios", func(t *testing.T) {
		assert.Equal(t, map[string]float64{"resolution": 2}, taskBillingRatios("fixed-video", true, ratioInput))
	})

	t.Run("ratio pricing always keeps duration", func(t *testing.T) {
		assert.Equal(t, ratioInput, taskBillingRatios("fixed-video", false, ratioInput))
	})
}

func TestTaskDurationMultiplierFiltersSubmitAdjustment(t *testing.T) {
	original := billing_setting.GetTaskDurationMultiplierCopy()
	restoreJSON, err := common.Marshal(original)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
			"billing_setting." + billing_setting.TaskDurationMultiplierField: string(restoreJSON),
		}))
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting." + billing_setting.TaskDurationMultiplierField: `{"fixed-video":false}`,
	}))

	priceData := hosttypes.PriceData{UsePrice: true, ModelPrice: 0.001, Quota: 1000}
	priceData.GroupRatioInfo.GroupRatio = 1
	priceData.AddOtherRatio("resolution", 2)
	priceData.Quota = 1000
	info := &relaycommon.RelayInfo{PriceData: priceData}
	adjusted := taskBillingRatios(
		"fixed-video",
		true,
		map[string]float64{"seconds": 10, "resolution": 3},
	)

	quota, ok := recalcQuotaFromRatios(info, adjusted)

	require.True(t, ok)
	assert.Equal(t, 1500, quota)
	assert.Equal(t, map[string]float64{"resolution": 3}, adjusted)
}

func TestGetTaskAdaptorSupportsGrsaiPlatform(t *testing.T) {
	adaptor := GetTaskAdaptor(constant.TaskPlatformGrsai)
	require.NotNil(t, adaptor)
	assert.Equal(t, "Grsai", adaptor.GetChannelName())
}

func TestGetTaskAdaptorSupportsConfiguredAsyncImagePlatforms(t *testing.T) {
	aliAdaptor := GetTaskAdaptor(constant.TaskPlatformAsyncImageAli)
	require.NotNil(t, aliAdaptor)
	assert.Equal(t, "ali", aliAdaptor.GetChannelName())

	newAPIAdaptor := GetTaskAdaptor(constant.TaskPlatformAsyncImageNewAPI)
	require.NotNil(t, newAPIAdaptor)
	assert.Equal(t, "New API async image", newAPIAdaptor.GetChannelName())
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
