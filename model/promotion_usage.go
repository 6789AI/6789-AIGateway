package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	sqlitedriver "github.com/glebarez/go-sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	promotionReservationReserved = "reserved"
	promotionReservationConsumed = "consumed"
	promotionReservationRefunded = "refunded"
	promotionMinimumAllowance    = 20
	promotionMaximumAllowance    = 2000
	promotionReservationAttempts = 3
)

// PromotionUsage stores the aggregate activity usage for one user and one
// activity occurrence. Models with the same activity key share this counter.
type PromotionUsage struct {
	ID            int64  `json:"id" gorm:"primaryKey"`
	UserId        int    `json:"user_id" gorm:"not null;uniqueIndex:idx_promotion_usage_user_activity,priority:1"`
	ActivityKey   string `json:"activity_key" gorm:"type:varchar(191);not null;uniqueIndex:idx_promotion_usage_user_activity,priority:2"`
	UsedCount     int    `json:"used_count" gorm:"not null"`
	ReservedCount int    `json:"reserved_count" gorm:"not null"`
	CreatedAt     int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

// PromotionReservation keeps an in-flight request idempotent until it is
// consumed or refunded. Terminal rows are deleted in the same transaction that
// updates the aggregate counter, so failed requests cannot grow this table.
type PromotionReservation struct {
	ID          int64  `json:"id" gorm:"primaryKey"`
	RequestId   string `json:"request_id" gorm:"type:varchar(191);not null;uniqueIndex"`
	UserId      int    `json:"user_id" gorm:"not null;index"`
	ActivityKey string `json:"activity_key" gorm:"type:varchar(191);not null;index"`
	Status      string `json:"status" gorm:"type:varchar(20);not null;index"`
	CreatedAt   int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

type PromotionUsageSummary struct {
	Limit     int
	Used      int
	Remaining int
	Active    bool
}

func normalizePromotionActivityKeys(activityKeys []string) []string {
	uniqueKeys := make([]string, 0, len(activityKeys))
	seen := make(map[string]struct{}, len(activityKeys))
	for _, activityKey := range activityKeys {
		activityKey = strings.TrimSpace(activityKey)
		if activityKey == "" {
			continue
		}
		if _, exists := seen[activityKey]; exists {
			continue
		}
		seen[activityKey] = struct{}{}
		uniqueKeys = append(uniqueKeys, activityKey)
	}
	return uniqueKeys
}

// PromotionAllowanceForUsedQuota converts cumulative platform consumption to
// activity uses: below 10 units gets 20; otherwise floor(amount*3), capped at 2000.
func PromotionAllowanceForUsedQuota(usedQuota int) int {
	if usedQuota < 0 {
		usedQuota = 0
	}
	quotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
	if !quotaPerUnit.IsPositive() {
		return promotionMinimumAllowance
	}

	amount := decimal.NewFromInt(int64(usedQuota)).Div(quotaPerUnit)
	if amount.LessThan(decimal.NewFromInt(10)) {
		return promotionMinimumAllowance
	}
	allowance := amount.Mul(decimal.NewFromInt(3)).Floor()
	if allowance.GreaterThanOrEqual(decimal.NewFromInt(promotionMaximumAllowance)) {
		return promotionMaximumAllowance
	}
	return int(allowance.IntPart())
}

// GetPromotionUsageSummary reports one shared allowance across all currently
// active promotions. Between activities it reports the user's full allowance
// so the wallet can show the next entitlement.
func GetPromotionUsageSummary(userId int, usedQuota int, activityKeys []string) (PromotionUsageSummary, error) {
	uniqueKeys := normalizePromotionActivityKeys(activityKeys)
	allowance := PromotionAllowanceForUsedQuota(usedQuota)
	if len(uniqueKeys) == 0 {
		return PromotionUsageSummary{
			Limit:     allowance,
			Remaining: allowance,
		}, nil
	}

	summary := PromotionUsageSummary{
		Limit:     allowance,
		Remaining: allowance,
		Active:    true,
	}
	var usages []PromotionUsage
	if err := DB.Where("user_id = ? AND activity_key IN ?", userId, uniqueKeys).Find(&usages).Error; err != nil {
		return PromotionUsageSummary{}, err
	}
	for _, usage := range usages {
		used := usage.UsedCount + usage.ReservedCount
		if used < 0 {
			used = 0
		}
		summary.Used += used
	}
	summary.Remaining = allowance - min(summary.Used, allowance)
	return summary, nil
}

// ReservePromotionUse atomically reserves one use from the allowance shared by
// all currently active promotions. Exhaustion is a normal result: callers
// should restore the model's regular price. Database errors are returned so
// callers do not silently charge a request at the wrong price.
func ReservePromotionUse(userId int, requestId string, activityKey string, activeActivityKeys ...string) (bool, error) {
	requestId = strings.TrimSpace(requestId)
	activityKey = strings.TrimSpace(activityKey)
	if userId <= 0 || requestId == "" || activityKey == "" {
		return false, errors.New("invalid promotion reservation identity")
	}
	if len(requestId) > 191 || len(activityKey) > 191 {
		return false, errors.New("promotion reservation identity is too long")
	}
	poolKeys := make([]string, 0, len(activeActivityKeys)+1)
	poolKeys = append(poolKeys, activityKey)
	poolKeys = append(poolKeys, activeActivityKeys...)
	poolKeys = normalizePromotionActivityKeys(poolKeys)

	for attempt := 0; attempt < promotionReservationAttempts; attempt++ {
		granted := false
		err := DB.Transaction(func(tx *gorm.DB) error {
			var existing PromotionReservation
			err := tx.Where("request_id = ?", requestId).Take(&existing).Error
			if err == nil {
				if existing.UserId != userId || existing.ActivityKey != activityKey {
					return errors.New("promotion request id is already bound to another activity")
				}
				granted = existing.Status == promotionReservationReserved || existing.Status == promotionReservationConsumed
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			var user User
			result := lockForUpdate(tx).Select("used_quota").Take(&user, "id = ?", userId)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("user %d not found", userId)
			}
			allowance := PromotionAllowanceForUsedQuota(user.UsedQuota)
			var activeUsages []PromotionUsage
			if err := tx.Where("user_id = ? AND activity_key IN ?", userId, poolKeys).Find(&activeUsages).Error; err != nil {
				return err
			}
			poolUsed := 0
			for _, activeUsage := range activeUsages {
				used := activeUsage.UsedCount + activeUsage.ReservedCount
				if used > 0 {
					poolUsed += used
				}
			}
			if poolUsed >= allowance {
				return nil
			}

			usage := PromotionUsage{UserId: userId, ActivityKey: activityKey}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&usage).Error; err != nil {
				return err
			}
			result = tx.Model(&PromotionUsage{}).
				Where("user_id = ? AND activity_key = ?", userId, activityKey).
				Update("reserved_count", gorm.Expr("reserved_count + ?", 1))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return errors.New("promotion usage counter is missing")
			}

			reservation := PromotionReservation{
				RequestId:   requestId,
				UserId:      userId,
				ActivityKey: activityKey,
				Status:      promotionReservationReserved,
			}
			if err := tx.Create(&reservation).Error; err != nil {
				return err
			}
			granted = true
			return nil
		})
		if err == nil {
			return granted, nil
		}
		if !isSQLiteBusyError(err) || attempt == promotionReservationAttempts-1 {
			return false, err
		}
		// A short bounded retry handles transient SQLite writer contention while
		// keeping the request responsive when the database is genuinely stuck.
		time.Sleep(time.Duration(attempt+1) * 25 * time.Millisecond)
	}
	return false, errors.New("promotion reservation attempts exhausted")
}

func isSQLiteBusyError(err error) bool {
	var sqliteErr *sqlitedriver.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code()
	baseCode := code & 0xff
	return baseCode == 5 || baseCode == 6 // SQLITE_BUSY / SQLITE_LOCKED
}

func CommitPromotionUse(requestId string) error {
	return finishPromotionReservation(requestId, promotionReservationConsumed)
}

func RefundPromotionUse(requestId string) error {
	return finishPromotionReservation(requestId, promotionReservationRefunded)
}

func finishPromotionReservation(requestId string, targetStatus string) error {
	requestId = strings.TrimSpace(requestId)
	if requestId == "" {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var reservation PromotionReservation
		err := tx.Where("request_id = ?", requestId).Take(&reservation).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if reservation.Status != promotionReservationReserved {
			return nil
		}

		result := tx.Model(&PromotionReservation{}).
			Where("id = ? AND status = ?", reservation.ID, promotionReservationReserved).
			Update("status", targetStatus)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		updates := map[string]any{
			"reserved_count": gorm.Expr("reserved_count - ?", 1),
		}
		if targetStatus == promotionReservationConsumed {
			updates["used_count"] = gorm.Expr("used_count + ?", 1)
		}
		result = tx.Model(&PromotionUsage{}).
			Where("user_id = ? AND activity_key = ? AND reserved_count > 0", reservation.UserId, reservation.ActivityKey).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("promotion usage reservation counter is inconsistent")
		}
		result = tx.Delete(&PromotionReservation{}, reservation.ID)
		return result.Error
	})
}
