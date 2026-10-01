package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var (
	ErrRootPromotionIneligible = errors.New("only an enabled, existing user or administrator can be promoted")
	ErrRootAlreadyPromoted     = errors.New("user is already a super administrator")
)

// PromoteUserToRoot is intentionally not exposed by any HTTP route. It is
// called only by the local maintenance command after operator confirmation.
func PromoteUserToRoot(userID int) error {
	if userID <= 0 {
		return ErrRootPromotionIneligible
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).Where("id = ?", userID).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRootPromotionIneligible
			}
			return err
		}
		if user.Role == common.RoleRootUser {
			return ErrRootAlreadyPromoted
		}
		if user.Status != common.UserStatusEnabled ||
			(user.Role != common.RoleCommonUser && user.Role != common.RoleAdminUser) {
			return ErrRootPromotionIneligible
		}
		if _, err := IncrementUserAuthVersionWithTx(tx, userID); err != nil {
			return err
		}
		result := tx.Model(&User{}).
			Where("id = ? AND role = ? AND status = ?", userID, user.Role, common.UserStatusEnabled).
			Update("role", common.RoleRootUser)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("root promotion changed no user rows")
		}
		return nil
	})
	if err != nil {
		return err
	}
	cacheErr := PublishUserAuthCache(userID)
	_, sessionErr := RevokeAllUserSessions(userID, "cli_root_promotion")
	var followUpErr error
	if cacheErr != nil {
		followUpErr = errors.Join(followUpErr, fmt.Errorf("role committed but auth cache publication failed: %w", cacheErr))
	}
	if sessionErr != nil {
		followUpErr = errors.Join(followUpErr, fmt.Errorf("role committed but session revocation failed: %w", sessionErr))
	}
	return followUpErr
}
