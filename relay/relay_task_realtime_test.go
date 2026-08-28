package relay

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTryRealtimeFetchFinalizesPromotionAfterTerminalCAS(t *testing.T) {
	tests := []struct {
		name           string
		upstreamBody   string
		wantStatus     model.TaskStatus
		wantUsedCount  int
		wantFailReason string
	}{
		{
			name:          "success commits promotion use",
			upstreamBody:  `{"name":"operations/test","done":true,"response":{}}`,
			wantStatus:    model.TaskStatusSuccess,
			wantUsedCount: 1,
		},
		{
			name:           "failure refunds promotion use",
			upstreamBody:   `{"name":"operations/test","done":true,"error":{"message":"generation failed"}}`,
			wantStatus:     model.TaskStatusFailure,
			wantFailReason: "generation failed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(test.upstreamBody))
			}))
			t.Cleanup(server.Close)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			require.NoError(t, err)
			require.NoError(t, db.AutoMigrate(
				&model.Channel{},
				&model.User{},
				&model.Task{},
				&model.PromotionUsage{},
				&model.PromotionReservation{},
			))
			previousDB := model.DB
			model.DB = db
			t.Cleanup(func() {
				model.DB = previousDB
			})

			user := model.User{Username: "realtime-promotion-user"}
			require.NoError(t, model.DB.Create(&user).Error)
			baseURL := server.URL
			channel := model.Channel{
				Type:    constant.ChannelTypeGemini,
				Name:    "realtime-gemini",
				Key:     "test-key",
				BaseURL: &baseURL,
			}
			require.NoError(t, model.DB.Create(&channel).Error)

			requestID := "realtime-" + test.name
			granted, err := model.ReservePromotionUse(user.Id, requestID, "v1:absolute:100:200")
			require.NoError(t, err)
			require.True(t, granted)

			task := model.Task{
				TaskID:    "public-" + test.name,
				UserId:    user.Id,
				ChannelId: channel.Id,
				Status:    model.TaskStatusInProgress,
				Progress:  "50%",
				PrivateData: model.TaskPrivateData{
					UpstreamTaskID:     taskcommon.EncodeLocalTaskID("operations/test"),
					PromotionRequestId: requestID,
				},
			}
			require.NoError(t, task.Insert())

			responseBody := tryRealtimeFetch(context.Background(), &task, false)
			require.NotEmpty(t, responseBody)

			var storedTask model.Task
			require.NoError(t, model.DB.First(&storedTask, task.ID).Error)
			assert.Equal(t, test.wantStatus, storedTask.Status)
			assert.Equal(t, "100%", storedTask.Progress)
			assert.Equal(t, test.wantFailReason, storedTask.FailReason)

			var usage model.PromotionUsage
			require.NoError(t, model.DB.Where("user_id = ?", user.Id).Take(&usage).Error)
			assert.Equal(t, test.wantUsedCount, usage.UsedCount)
			assert.Zero(t, usage.ReservedCount)
			var reservationCount int64
			require.NoError(t, model.DB.Model(&model.PromotionReservation{}).Count(&reservationCount).Error)
			assert.Zero(t, reservationCount)
		})
	}
}
