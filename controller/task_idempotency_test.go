package controller

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relaykitdto "github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newVintedIdempotencyContext(t *testing.T, userID, tokenID int, body string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Idempotency-Key", "client-operation")
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeSora)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, "https://vinted.cam")
	common.SetContextKey(c, constant.ContextKeyChannelOtherSetting, relaykitdto.ChannelOtherSettings{VideoProtocol: relaykitdto.VideoProtocolVinted})
	common.SetContextKey(c, constant.ContextKeyUserId, userID)
	common.SetContextKey(c, constant.ContextKeyTokenId, tokenID)
	t.Cleanup(func() { common.CleanupBodyStorage(c) })
	return c, recorder
}

func setupTaskIdempotencyControllerDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Task{}, &model.TaskSubmission{}))
	previousDB := model.DB
	previousType := common.MainDatabaseType()
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		model.DB = previousDB
		common.SetMainDatabaseType(previousType)
	})
}

func TestPrepareVintedTaskSubmissionIsolatesUsersAndBlocksConcurrentDuplicates(t *testing.T) {
	setupTaskIdempotencyControllerDB(t)
	body := `{"model":"seedance2.0","prompt":"test","duration":5}`
	c1, _ := newVintedIdempotencyContext(t, 10, 20, body)
	info1 := &relaycommon.RelayInfo{UserId: 10, TokenId: 20, RequestId: "request-1", TaskRelayInfo: &relaycommon.TaskRelayInfo{}}

	handled, taskErr := prepareVintedTaskSubmission(c1, info1)
	assert.False(t, handled)
	require.Nil(t, taskErr)
	assert.NotEmpty(t, info1.TaskSubmissionScope)

	cDuplicate, _ := newVintedIdempotencyContext(t, 10, 20, body)
	duplicateInfo := &relaycommon.RelayInfo{UserId: 10, TokenId: 20, RequestId: "request-2", TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	handled, taskErr = prepareVintedTaskSubmission(cDuplicate, duplicateInfo)
	assert.False(t, handled)
	require.NotNil(t, taskErr)
	assert.Equal(t, http.StatusConflict, taskErr.StatusCode)
	assert.Equal(t, "idempotency_in_progress", taskErr.Code)

	cOtherUser, _ := newVintedIdempotencyContext(t, 11, 21, body)
	otherInfo := &relaycommon.RelayInfo{UserId: 11, TokenId: 21, RequestId: "request-3", TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	handled, taskErr = prepareVintedTaskSubmission(cOtherUser, otherInfo)
	assert.False(t, handled)
	require.Nil(t, taskErr)
	assert.NotEqual(t, info1.TaskSubmissionScope, otherInfo.TaskSubmissionScope)
	assert.NotEqual(t, info1.UpstreamIdempotencyKey, otherInfo.UpstreamIdempotencyKey)
}

func TestPrepareVintedTaskSubmissionReplaysCompletedPublicTask(t *testing.T) {
	setupTaskIdempotencyControllerDB(t)
	body := `{"model":"seedance2.0","prompt":"test","duration":5}`
	c, _ := newVintedIdempotencyContext(t, 10, 20, body)
	info := &relaycommon.RelayInfo{UserId: 10, TokenId: 20, RequestId: "request-1", TaskRelayInfo: &relaycommon.TaskRelayInfo{}}

	handled, taskErr := prepareVintedTaskSubmission(c, info)
	assert.False(t, handled)
	require.Nil(t, taskErr)
	task := &model.Task{TaskID: info.PublicTaskID, UserId: 10, Status: model.TaskStatusQueued}
	require.NoError(t, model.InsertTaskWithSubmission(task, info.TaskSubmissionScope, info.TaskSubmissionLeaseToken))

	replayContext, recorder := newVintedIdempotencyContext(t, 10, 20, body)
	replayInfo := &relaycommon.RelayInfo{UserId: 10, TokenId: 20, RequestId: "request-2", TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	handled, taskErr = prepareVintedTaskSubmission(replayContext, replayInfo)

	assert.True(t, handled)
	require.Nil(t, taskErr)
	assert.Equal(t, http.StatusAccepted, recorder.Code)
	assert.JSONEq(t, `{"id":"`+info.PublicTaskID+`","status":"queued","idempotent":"1"}`, recorder.Body.String())
}

func TestPrepareVintedTaskSubmissionRestoresBoundChannelMetadataAndKey(t *testing.T) {
	setupTaskIdempotencyControllerDB(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Channel{}))
	baseURL := "https://vinted.cam"
	channel := &model.Channel{
		Id:      321,
		Type:    constant.ChannelTypeSora,
		Key:     "first-key\nsecond-key",
		BaseURL: &baseURL,
		ChannelInfo: model.ChannelInfo{
			IsMultiKey: true,
		},
	}
	channel.SetOtherSettings(relaykitdto.ChannelOtherSettings{VideoProtocol: relaykitdto.VideoProtocolVinted})
	require.NoError(t, model.DB.Create(channel).Error)

	body := `{"model":"seedance2.0","prompt":"test","duration":5}`
	firstContext, _ := newVintedIdempotencyContext(t, 10, 20, body)
	firstInfo := &relaycommon.RelayInfo{UserId: 10, TokenId: 20, RequestId: "request-1", TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	handled, taskErr := prepareVintedTaskSubmission(firstContext, firstInfo)
	assert.False(t, handled)
	require.Nil(t, taskErr)
	_, err := model.BindTaskSubmissionChannel(
		firstInfo.TaskSubmissionScope,
		firstInfo.PublicTaskID,
		firstInfo.TaskSubmissionLeaseToken,
		channel.Id,
		1,
	)
	require.NoError(t, err)
	require.NoError(t, model.FailTaskSubmission(
		firstInfo.TaskSubmissionScope,
		firstInfo.PublicTaskID,
		firstInfo.TaskSubmissionLeaseToken,
	))

	retryContext, _ := newVintedIdempotencyContext(t, 10, 20, body)
	retryInfo := &relaycommon.RelayInfo{UserId: 10, TokenId: 20, RequestId: "request-2", OriginModelName: "seedance2.0", TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	handled, taskErr = prepareVintedTaskSubmission(retryContext, retryInfo)

	assert.False(t, handled)
	require.Nil(t, taskErr)
	assert.True(t, retryInfo.TaskSubmissionChannelBound)
	assert.Equal(t, channel.Id, retryInfo.ChannelId)
	assert.Equal(t, constant.ChannelTypeSora, retryInfo.ChannelType)
	assert.Equal(t, baseURL, retryInfo.ChannelBaseUrl)
	assert.Equal(t, "second-key", retryInfo.ApiKey)
	assert.Equal(t, 1, retryInfo.ChannelMultiKeyIndex)
	assert.Equal(t, relaykitdto.VideoProtocolVinted, retryInfo.ChannelOtherSettings.VideoProtocol)
}

func TestPersistAcceptedTaskPropagatesInsertFailure(t *testing.T) {
	setupTaskIdempotencyControllerDB(t)
	rejectErr := errors.New("task insert rejected")
	require.NoError(t, model.DB.Callback().Create().Before("gorm:create").Register("test:reject_task_insert", func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*model.Task); ok {
			tx.AddError(rejectErr)
		}
	}))

	info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	err := persistAcceptedTask(&model.Task{TaskID: "task_unpersisted"}, info)

	require.ErrorIs(t, err, rejectErr)
	assert.False(t, info.TaskSubmissionCompleted)
}
