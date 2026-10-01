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

func setupRootPromotionDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := DB
	previousRedis := common.RedisEnabled
	previousType := common.MainDatabaseType()
	common.RedisEnabled = false
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&User{}, &UserSession{}))
	DB = db
	t.Cleanup(func() {
		DB = previousDB
		common.RedisEnabled = previousRedis
		common.SetMainDatabaseType(previousType)
		_ = sqlDB.Close()
	})
	return db
}

func TestPromoteUserToRootRevokesExistingSessions(t *testing.T) {
	db := setupRootPromotionDB(t)
	user := User{Username: "second-root", Password: "hash", Role: common.RoleAdminUser,
		Status: common.UserStatusEnabled, AuthVersion: 1}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&UserSession{
		SID: "old-root-session", UserID: user.Id, Version: 1, UserAuthVersion: 1,
		Status: UserSessionStatusActive, RefreshHash: "hash", LoginMethod: "password",
		LastActiveAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}).Error)

	require.NoError(t, PromoteUserToRoot(user.Id))
	var stored User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Equal(t, common.RoleRootUser, stored.Role)
	assert.EqualValues(t, 2, stored.AuthVersion)
	var session UserSession
	require.NoError(t, db.First(&session, "sid = ?", "old-root-session").Error)
	assert.Equal(t, UserSessionStatusRevoked, session.Status)
	assert.Equal(t, "cli_root_promotion", session.RevokedReason)
	assert.ErrorIs(t, PromoteUserToRoot(user.Id), ErrRootAlreadyPromoted)
}

func TestPromoteUserToRootRejectsMissingOrDisabledAccount(t *testing.T) {
	db := setupRootPromotionDB(t)
	assert.ErrorIs(t, PromoteUserToRoot(999), ErrRootPromotionIneligible)
	user := User{Username: "disabled-root-candidate", Password: "hash", Role: common.RoleCommonUser,
		Status: common.UserStatusDisabled, AuthVersion: 1}
	require.NoError(t, db.Create(&user).Error)
	assert.ErrorIs(t, PromoteUserToRoot(user.Id), ErrRootPromotionIneligible)
	var stored User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Equal(t, common.RoleCommonUser, stored.Role)
	assert.EqualValues(t, 1, stored.AuthVersion)
}

func TestEnabledRootUsersIncludesEveryEligibleRecipient(t *testing.T) {
	db := setupRootPromotionDB(t)
	users := []User{
		{Username: "root-one", AffCode: "root-one-code", Role: common.RoleRootUser, Status: common.UserStatusEnabled},
		{Username: "root-two", AffCode: "root-two-code", Role: common.RoleRootUser, Status: common.UserStatusEnabled},
		{Username: "disabled-root", AffCode: "disabled-root-code", Role: common.RoleRootUser, Status: common.UserStatusDisabled},
		{Username: "admin", AffCode: "admin-code", Role: common.RoleAdminUser, Status: common.UserStatusEnabled},
	}
	require.NoError(t, db.Create(&users).Error)
	recipients, err := GetEnabledRootUsers()
	require.NoError(t, err)
	require.Len(t, recipients, 2)
	assert.Equal(t, users[0].Id, recipients[0].Id)
	assert.Equal(t, users[1].Id, recipients[1].Id)
}
