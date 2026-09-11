package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTaskSubmissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Task{}, &TaskSubmission{}))
	previousDB := DB
	previousType := common.MainDatabaseType()
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB = previousDB
		common.SetMainDatabaseType(previousType)
	})
	return db
}

func TestTaskSubmissionReservationPreventsDuplicateCreationAndReusesTask(t *testing.T) {
	setupTaskSubmissionTestDB(t)
	scopeHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	requestHash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	first, err := ReserveTaskSubmission(scopeHash, requestHash, "task_public", 10, 20)
	require.NoError(t, err)
	assert.Equal(t, TaskSubmissionOwner, first.State)

	bound, err := BindTaskSubmissionChannel(scopeHash, first.TaskID, first.LeaseToken, 89, 2)
	require.NoError(t, err)
	assert.Equal(t, 89, bound.ChannelID)
	assert.Equal(t, 2, bound.ChannelKeyIndex)

	duplicate, err := ReserveTaskSubmission(scopeHash, requestHash, "task_other", 10, 20)
	require.NoError(t, err)
	assert.Equal(t, TaskSubmissionInFlight, duplicate.State)
	assert.Equal(t, "task_public", duplicate.TaskID)

	require.NoError(t, FailTaskSubmission(scopeHash, first.TaskID, first.LeaseToken))
	retry, err := ReserveTaskSubmission(scopeHash, requestHash, "task_other", 10, 20)
	require.NoError(t, err)
	assert.Equal(t, TaskSubmissionOwner, retry.State)
	assert.Equal(t, "task_public", retry.TaskID)
	assert.Equal(t, 89, retry.ChannelID)
	assert.Equal(t, 2, retry.ChannelKeyIndex)

	task := &Task{TaskID: retry.TaskID, UserId: 10, Status: TaskStatusSubmitted}
	require.NoError(t, InsertTaskWithSubmission(task, scopeHash, retry.LeaseToken))
	replay, err := ReserveTaskSubmission(scopeHash, requestHash, "task_third", 10, 20)
	require.NoError(t, err)
	assert.Equal(t, TaskSubmissionReplay, replay.State)
	assert.Equal(t, "task_public", replay.TaskID)
	_, exists, err := GetByTaskId(10, replay.TaskID)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestTaskSubmissionRejectsBodyConflict(t *testing.T) {
	setupTaskSubmissionTestDB(t)
	scopeHash := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	requestHash := "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"

	_, err := ReserveTaskSubmission(scopeHash, requestHash, "task_public", 10, 20)
	require.NoError(t, err)
	conflict, err := ReserveTaskSubmission(
		scopeHash,
		"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		"task_other",
		10,
		20,
	)
	require.NoError(t, err)
	assert.Equal(t, TaskSubmissionConflict, conflict.State)
}

func TestTaskSubmissionExpiredLeaseRejectsStaleOwner(t *testing.T) {
	db := setupTaskSubmissionTestDB(t)
	scopeHash := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	requestHash := "1111111111111111111111111111111111111111111111111111111111111111"

	first, err := ReserveTaskSubmission(scopeHash, requestHash, "task_public", 10, 20)
	require.NoError(t, err)
	require.NoError(t, db.Model(&TaskSubmission{}).
		Where("scope_hash = ?", scopeHash).
		Update("lease_expires_at", time.Now().Add(-time.Minute).Unix()).Error)

	reclaimed, err := ReserveTaskSubmission(scopeHash, requestHash, "task_other", 10, 20)
	require.NoError(t, err)
	assert.Equal(t, TaskSubmissionOwner, reclaimed.State)
	assert.Equal(t, first.TaskID, reclaimed.TaskID)
	assert.NotEqual(t, first.LeaseToken, reclaimed.LeaseToken)
	_, err = BindTaskSubmissionChannel(scopeHash, first.TaskID, first.LeaseToken, 89, 0)
	require.Error(t, err)

	err = InsertTaskWithSubmission(
		&Task{TaskID: first.TaskID, UserId: 10, Status: TaskStatusSubmitted},
		scopeHash,
		first.LeaseToken,
	)
	require.Error(t, err)
	var taskCount int64
	require.NoError(t, db.Model(&Task{}).Where("task_id = ?", first.TaskID).Count(&taskCount).Error)
	assert.Zero(t, taskCount, "a stale owner must not leave a task row behind")

	require.NoError(t, InsertTaskWithSubmission(
		&Task{TaskID: reclaimed.TaskID, UserId: 10, Status: TaskStatusSubmitted},
		scopeHash,
		reclaimed.LeaseToken,
	))
}
