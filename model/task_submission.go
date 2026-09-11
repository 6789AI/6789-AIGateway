package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	taskSubmissionPending   = "pending"
	taskSubmissionCompleted = "completed"
	taskSubmissionFailed    = "failed"
	taskSubmissionLease     = 10 * time.Minute
	taskSubmissionAttempts  = 3
)

// TaskSubmission serializes client-idempotent asynchronous task creation.
// ScopeHash and RequestHash contain digests only; raw client keys and payloads
// are never persisted.
type TaskSubmission struct {
	ID              int64  `json:"id" gorm:"primaryKey"`
	ScopeHash       string `json:"scope_hash" gorm:"type:varchar(64);not null;uniqueIndex"`
	RequestHash     string `json:"request_hash" gorm:"type:varchar(64);not null"`
	TaskID          string `json:"task_id" gorm:"type:varchar(191);not null;uniqueIndex"`
	UserID          int    `json:"user_id" gorm:"not null;index"`
	TokenID         int    `json:"token_id" gorm:"not null;index"`
	ChannelID       int    `json:"channel_id" gorm:"not null"`
	ChannelKeyIndex int    `json:"channel_key_index" gorm:"not null"`
	Status          string `json:"status" gorm:"type:varchar(20);not null;index"`
	LeaseToken      string `json:"lease_token" gorm:"type:varchar(64);not null"`
	LeaseExpiresAt  int64  `json:"lease_expires_at" gorm:"not null;index"`
	CreatedAt       int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

type TaskSubmissionState string

const (
	TaskSubmissionOwner    TaskSubmissionState = "owner"
	TaskSubmissionInFlight TaskSubmissionState = "in_flight"
	TaskSubmissionReplay   TaskSubmissionState = "replay"
	TaskSubmissionConflict TaskSubmissionState = "conflict"
)

type TaskSubmissionReservation struct {
	State           TaskSubmissionState
	TaskID          string
	ChannelID       int
	ChannelKeyIndex int
	LeaseToken      string
}

// ReserveTaskSubmission creates or claims one durable task submission. Failed
// and expired reservations retain the same public task ID and channel binding.
func ReserveTaskSubmission(scopeHash, requestHash, taskID string, userID, tokenID int) (TaskSubmissionReservation, error) {
	scopeHash = strings.TrimSpace(scopeHash)
	requestHash = strings.TrimSpace(requestHash)
	taskID = strings.TrimSpace(taskID)
	if len(scopeHash) != 64 || len(requestHash) != 64 || taskID == "" || len(taskID) > 191 || userID <= 0 || tokenID <= 0 {
		return TaskSubmissionReservation{}, errors.New("invalid task submission identity")
	}
	leaseToken, err := common.GenerateRandomCharsKey(32)
	if err != nil {
		return TaskSubmissionReservation{}, fmt.Errorf("generate task submission lease: %w", err)
	}

	for attempt := 0; attempt < taskSubmissionAttempts; attempt++ {
		var reservation TaskSubmissionReservation
		now := time.Now().Unix()
		err := DB.Transaction(func(tx *gorm.DB) error {
			candidate := TaskSubmission{
				ScopeHash:      scopeHash,
				RequestHash:    requestHash,
				TaskID:         taskID,
				UserID:         userID,
				TokenID:        tokenID,
				Status:         taskSubmissionPending,
				LeaseToken:     leaseToken,
				LeaseExpiresAt: now + int64(taskSubmissionLease/time.Second),
			}
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&candidate)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 1 {
				reservation = TaskSubmissionReservation{State: TaskSubmissionOwner, TaskID: taskID, LeaseToken: leaseToken}
				return nil
			}

			var existing TaskSubmission
			if err := lockForUpdate(tx).Where("scope_hash = ?", scopeHash).Take(&existing).Error; err != nil {
				return err
			}
			reservation = TaskSubmissionReservation{
				TaskID:          existing.TaskID,
				ChannelID:       existing.ChannelID,
				ChannelKeyIndex: existing.ChannelKeyIndex,
				LeaseToken:      existing.LeaseToken,
			}
			if existing.UserID != userID || existing.TokenID != tokenID || existing.RequestHash != requestHash {
				reservation.State = TaskSubmissionConflict
				return nil
			}
			switch existing.Status {
			case taskSubmissionCompleted:
				reservation.State = TaskSubmissionReplay
				return nil
			case taskSubmissionPending:
				if existing.LeaseExpiresAt > now {
					reservation.State = TaskSubmissionInFlight
					return nil
				}
			case taskSubmissionFailed:
			default:
				return fmt.Errorf("invalid task submission status: %s", existing.Status)
			}

			if existing.Status == taskSubmissionFailed {
				result = tx.Model(&TaskSubmission{}).
					Where("id = ? AND status = ?", existing.ID, taskSubmissionFailed).
					Updates(map[string]any{
						"status":           taskSubmissionPending,
						"lease_token":      leaseToken,
						"lease_expires_at": now + int64(taskSubmissionLease/time.Second),
					})
			} else {
				result = tx.Model(&TaskSubmission{}).
					Where("id = ? AND status = ? AND lease_expires_at <= ?", existing.ID, taskSubmissionPending, now).
					Updates(map[string]any{
						"lease_token":      leaseToken,
						"lease_expires_at": now + int64(taskSubmissionLease/time.Second),
					})
			}
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 1 {
				reservation.State = TaskSubmissionOwner
				reservation.LeaseToken = leaseToken
			} else {
				reservation.State = TaskSubmissionInFlight
			}
			return nil
		})
		if err == nil {
			return reservation, nil
		}
		if !isSQLiteBusyError(err) || attempt == taskSubmissionAttempts-1 {
			return TaskSubmissionReservation{}, err
		}
		time.Sleep(time.Duration(attempt+1) * 25 * time.Millisecond)
	}
	return TaskSubmissionReservation{}, errors.New("task submission reservation attempts exhausted")
}

// BindTaskSubmissionChannel pins retries to the provider credential scope that
// received the original request.
func BindTaskSubmissionChannel(scopeHash, taskID, leaseToken string, channelID, channelKeyIndex int) (TaskSubmissionReservation, error) {
	if channelID <= 0 || channelKeyIndex < 0 {
		return TaskSubmissionReservation{}, errors.New("invalid task submission channel")
	}
	var reservation TaskSubmissionReservation
	err := DB.Transaction(func(tx *gorm.DB) error {
		var existing TaskSubmission
		if err := lockForUpdate(tx).
			Where("scope_hash = ? AND task_id = ? AND lease_token = ?", scopeHash, taskID, leaseToken).
			Take(&existing).Error; err != nil {
			return err
		}
		if existing.Status != taskSubmissionPending {
			return errors.New("task submission is not pending")
		}
		if existing.ChannelID == 0 {
			result := tx.Model(&TaskSubmission{}).
				Where(
					"id = ? AND channel_id = 0 AND lease_token = ? AND status = ?",
					existing.ID,
					leaseToken,
					taskSubmissionPending,
				).
				Updates(map[string]any{"channel_id": channelID, "channel_key_index": channelKeyIndex})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("task submission lease was lost")
			}
			existing.ChannelID = channelID
			existing.ChannelKeyIndex = channelKeyIndex
		}
		reservation = TaskSubmissionReservation{
			State:           TaskSubmissionOwner,
			TaskID:          existing.TaskID,
			ChannelID:       existing.ChannelID,
			ChannelKeyIndex: existing.ChannelKeyIndex,
			LeaseToken:      existing.LeaseToken,
		}
		return nil
	})
	return reservation, err
}

func FailTaskSubmission(scopeHash, taskID, leaseToken string) error {
	if scopeHash == "" || taskID == "" || leaseToken == "" {
		return nil
	}
	return DB.Model(&TaskSubmission{}).
		Where("scope_hash = ? AND task_id = ? AND lease_token = ? AND status = ?", scopeHash, taskID, leaseToken, taskSubmissionPending).
		Updates(map[string]any{"status": taskSubmissionFailed, "lease_expires_at": time.Now().Unix()}).Error
}

// InsertTaskWithSubmission commits the public task and idempotency replay state
// atomically, so a retry can never observe "completed" without its task row.
func InsertTaskWithSubmission(task *Task, scopeHash, leaseToken string) error {
	if task == nil {
		return errors.New("task is required")
	}
	if scopeHash == "" {
		return task.Insert()
	}
	if leaseToken == "" {
		return errors.New("task submission lease is required")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(task).Error; err != nil {
			return err
		}
		result := tx.Model(&TaskSubmission{}).
			Where("scope_hash = ? AND task_id = ? AND lease_token = ? AND status = ?", scopeHash, task.TaskID, leaseToken, taskSubmissionPending).
			Updates(map[string]any{"status": taskSubmissionCompleted, "lease_expires_at": int64(0)})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("task submission reservation is missing")
		}
		return nil
	})
}
