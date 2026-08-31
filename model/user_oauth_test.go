package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestInsertWithTxPersistsInviterID(t *testing.T) {
	setupUserUpdateTestState(t)

	inviter := &User{
		Username: "oauth-inviter",
		Password: "password",
		Status:   common.UserStatusEnabled,
		AffCode:  "oauth-inviter-code",
	}
	require.NoError(t, DB.Create(inviter).Error)

	invitee := &User{
		Username: "oauth-invitee",
		Password: "password",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return invitee.InsertWithTx(tx, inviter.Id)
	}))

	var stored User
	require.NoError(t, DB.First(&stored, invitee.Id).Error)
	assert.Equal(t, inviter.Id, stored.InviterId)
}
