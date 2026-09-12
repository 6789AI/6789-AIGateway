package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func configureRegistrationRewardsForTest(t *testing.T, inviterEnabled bool, inviteeEnabled bool, inviterQuota int, inviteeQuota int, complianceConfirmed bool) {
	t.Helper()
	previousInviterEnabled := common.QuotaForInviterEnabled
	previousInviteeEnabled := common.QuotaForInviteeEnabled
	previousInviterQuota := common.QuotaForInviter
	previousInviteeQuota := common.QuotaForInvitee
	previousNewUserQuota := common.QuotaForNewUser
	paymentSetting := operation_setting.GetPaymentSetting()
	previousConfirmed := paymentSetting.ComplianceConfirmed
	previousTermsVersion := paymentSetting.ComplianceTermsVersion

	common.QuotaForInviterEnabled = inviterEnabled
	common.QuotaForInviteeEnabled = inviteeEnabled
	common.QuotaForInviter = inviterQuota
	common.QuotaForInvitee = inviteeQuota
	common.QuotaForNewUser = 0
	paymentSetting.ComplianceConfirmed = complianceConfirmed
	if complianceConfirmed {
		paymentSetting.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	} else {
		paymentSetting.ComplianceTermsVersion = ""
	}

	t.Cleanup(func() {
		common.QuotaForInviterEnabled = previousInviterEnabled
		common.QuotaForInviteeEnabled = previousInviteeEnabled
		common.QuotaForInviter = previousInviterQuota
		common.QuotaForInvitee = previousInviteeQuota
		common.QuotaForNewUser = previousNewUserQuota
		paymentSetting.ComplianceConfirmed = previousConfirmed
		paymentSetting.ComplianceTermsVersion = previousTermsVersion
	})
}

func TestInsertCountsInvitationWhenRegistrationRewardsDoNotRun(t *testing.T) {
	tests := []struct {
		name                string
		inviterEnabled      bool
		inviteeEnabled      bool
		inviterQuota        int
		inviteeQuota        int
		complianceConfirmed bool
	}{
		{name: "zero reward amounts", inviterEnabled: true, inviteeEnabled: true, complianceConfirmed: true},
		{name: "disabled reward switches", inviterQuota: 50, inviteeQuota: 25, complianceConfirmed: true},
		{name: "compliance not confirmed", inviterEnabled: true, inviteeEnabled: true, inviterQuota: 50, inviteeQuota: 25},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			truncateTables(t)
			configureRegistrationRewardsForTest(t, test.inviterEnabled, test.inviteeEnabled, test.inviterQuota, test.inviteeQuota, test.complianceConfirmed)

			inviter := &User{Username: "count-inviter-" + test.name, Password: "password", Status: common.UserStatusEnabled, AffCode: "count-code-" + test.name}
			require.NoError(t, DB.Create(inviter).Error)
			invitee := &User{Username: "count-invitee-" + test.name, Password: "password", Status: common.UserStatusEnabled, Role: common.RoleCommonUser}
			require.NoError(t, invitee.Insert(inviter.Id))

			var updatedInviter, updatedInvitee User
			require.NoError(t, DB.First(&updatedInviter, inviter.Id).Error)
			require.NoError(t, DB.First(&updatedInvitee, invitee.Id).Error)
			assert.Equal(t, 1, updatedInviter.AffCount)
			assert.Zero(t, updatedInviter.AffQuota)
			assert.Zero(t, updatedInviter.AffHistoryQuota)
			assert.Zero(t, updatedInvitee.Quota)
		})
	}
}

func TestInsertWithTxCountsOAuthInvitationOnce(t *testing.T) {
	truncateTables(t)
	configureRegistrationRewardsForTest(t, false, false, 50, 25, true)

	inviter := &User{Username: "oauth-count-inviter", Password: "password", Status: common.UserStatusEnabled, AffCode: "oauth-count-inviter-code"}
	require.NoError(t, DB.Create(inviter).Error)
	invitee := &User{Username: "oauth-count-invitee", Password: "password", Status: common.UserStatusEnabled, Role: common.RoleCommonUser}
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return invitee.InsertWithTx(tx, inviter.Id)
	}))
	invitee.FinalizeOAuthUserCreation(inviter.Id)

	var updatedInviter User
	require.NoError(t, DB.First(&updatedInviter, inviter.Id).Error)
	assert.Equal(t, 1, updatedInviter.AffCount)
	assert.Zero(t, updatedInviter.AffQuota)
}

func TestRegistrationRewardSwitchesAreIndependent(t *testing.T) {
	truncateTables(t)
	configureRegistrationRewardsForTest(t, true, false, 40, 20, true)

	inviter := &User{Username: "switch-inviter", Password: "password", Status: common.UserStatusEnabled, AffCode: "switch-inviter-code"}
	require.NoError(t, DB.Create(inviter).Error)
	firstInvitee := &User{Username: "switch-first-invitee", Password: "password", Status: common.UserStatusEnabled, Role: common.RoleCommonUser}
	require.NoError(t, firstInvitee.Insert(inviter.Id))

	var firstStoredInvitee, updatedInviter User
	require.NoError(t, DB.First(&firstStoredInvitee, firstInvitee.Id).Error)
	require.NoError(t, DB.First(&updatedInviter, inviter.Id).Error)
	assert.Zero(t, firstStoredInvitee.Quota)
	assert.Equal(t, 1, updatedInviter.AffCount)
	assert.Equal(t, 40, updatedInviter.AffQuota)
	assert.Equal(t, 40, updatedInviter.AffHistoryQuota)

	common.QuotaForInviterEnabled = false
	common.QuotaForInviteeEnabled = true
	secondInvitee := &User{Username: "switch-second-invitee", Password: "password", Status: common.UserStatusEnabled, Role: common.RoleCommonUser}
	require.NoError(t, secondInvitee.Insert(inviter.Id))

	var secondStoredInvitee User
	require.NoError(t, DB.First(&secondStoredInvitee, secondInvitee.Id).Error)
	require.NoError(t, DB.First(&updatedInviter, inviter.Id).Error)
	assert.Equal(t, 20, secondStoredInvitee.Quota)
	assert.Equal(t, 2, updatedInviter.AffCount)
	assert.Equal(t, 40, updatedInviter.AffQuota)
	assert.Equal(t, 40, updatedInviter.AffHistoryQuota)
}

func TestRecalculateInviteCountsRebuildsOnlyRelationshipTotals(t *testing.T) {
	truncateTables(t)

	owner := &User{Username: "recount-owner", Password: "password", Status: common.UserStatusEnabled, AffCode: "recount-owner-code", AffCount: 99, AffQuota: 70, AffHistoryQuota: 90, Quota: 110}
	otherOwner := &User{Username: "recount-other", Password: "password", Status: common.UserStatusEnabled, AffCode: "recount-other-code", AffCount: 5}
	require.NoError(t, DB.Create(owner).Error)
	require.NoError(t, DB.Create(otherOwner).Error)
	activeInvitee := &User{Username: "recount-active", Password: "password", Status: common.UserStatusEnabled, AffCode: "recount-active-code", InviterId: owner.Id}
	softDeletedInvitee := &User{Username: "recount-deleted", Password: "password", Status: common.UserStatusEnabled, AffCode: "recount-deleted-code", InviterId: owner.Id}
	selfInvited := &User{Username: "recount-self", Password: "password", Status: common.UserStatusEnabled, AffCode: "recount-self-code"}
	orphaned := &User{Username: "recount-orphan", Password: "password", Status: common.UserStatusEnabled, AffCode: "recount-orphan-code", InviterId: 999999}
	hardDeletedInvitee := &User{Username: "recount-hard-deleted", Password: "password", Status: common.UserStatusEnabled, AffCode: "recount-hard-deleted-code", InviterId: owner.Id}
	require.NoError(t, DB.Create(activeInvitee).Error)
	require.NoError(t, DB.Create(softDeletedInvitee).Error)
	require.NoError(t, DB.Create(selfInvited).Error)
	require.NoError(t, DB.Model(selfInvited).UpdateColumn("inviter_id", selfInvited.Id).Error)
	require.NoError(t, DB.Create(orphaned).Error)
	require.NoError(t, DB.Create(hardDeletedInvitee).Error)
	require.NoError(t, DB.Delete(softDeletedInvitee).Error)
	require.NoError(t, DB.Unscoped().Delete(hardDeletedInvitee).Error)

	result, err := RecalculateInviteCounts()
	require.NoError(t, err)
	assert.Equal(t, 6, result.UsersScanned)
	assert.Equal(t, 2, result.InvitationRelations)
	assert.Equal(t, 2, result.UsersUpdated)

	var updatedOwner, updatedOtherOwner User
	require.NoError(t, DB.First(&updatedOwner, owner.Id).Error)
	require.NoError(t, DB.First(&updatedOtherOwner, otherOwner.Id).Error)
	assert.Equal(t, 2, updatedOwner.AffCount)
	assert.Equal(t, 70, updatedOwner.AffQuota)
	assert.Equal(t, 90, updatedOwner.AffHistoryQuota)
	assert.Equal(t, 110, updatedOwner.Quota)
	assert.Zero(t, updatedOtherOwner.AffCount)

	secondResult, err := RecalculateInviteCounts()
	require.NoError(t, err)
	assert.Zero(t, secondResult.UsersUpdated)
}

func TestValidateAffiliateOptions(t *testing.T) {
	validValues := map[string][]string{
		"AffiliateRebatePercentage": {"0", "2.5", "10.00", "100"},
		"AffiliateRebateEnabled":    {"true", "false"},
		"QuotaForInviterEnabled":    {"true", "false"},
		"QuotaForInviteeEnabled":    {"true", "false"},
	}
	for key, values := range validValues {
		for _, value := range values {
			assert.NoError(t, validateOptionValue(key, value), "%s=%s", key, value)
		}
	}

	invalidValues := map[string][]string{
		"AffiliateRebatePercentage": {"-0.01", "1.234", "100.01", "not-a-number"},
		"AffiliateRebateEnabled":    {"1", "yes"},
		"QuotaForInviterEnabled":    {"TRUE", ""},
		"QuotaForInviteeEnabled":    {"0", "on"},
	}
	for key, values := range invalidValues {
		for _, value := range values {
			assert.Error(t, validateOptionValue(key, value), "%s=%s", key, value)
		}
	}
}
