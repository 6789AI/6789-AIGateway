package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminCompleteTopUpDefaultsToRebateAndAllowsOptOut(t *testing.T) {
	db := setupManageUserTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.TopUp{}, &model.AffiliateReward{}))

	previousQuotaPerUnit := common.QuotaPerUnit
	previousRebateEnabled := common.AffiliateRebateEnabled
	previousBasisPoints := common.AffiliateRebateBasisPoints
	paymentSetting := operation_setting.GetPaymentSetting()
	previousCompliance := paymentSetting.ComplianceConfirmed
	previousTermsVersion := paymentSetting.ComplianceTermsVersion
	common.QuotaPerUnit = 1_000
	common.AffiliateRebateEnabled = true
	common.AffiliateRebateBasisPoints = 1_000
	paymentSetting.ComplianceConfirmed = true
	paymentSetting.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	t.Cleanup(func() {
		common.QuotaPerUnit = previousQuotaPerUnit
		common.AffiliateRebateEnabled = previousRebateEnabled
		common.AffiliateRebateBasisPoints = previousBasisPoints
		paymentSetting.ComplianceConfirmed = previousCompliance
		paymentSetting.ComplianceTermsVersion = previousTermsVersion
	})

	inviter := model.User{Username: "complete-inviter", Password: "password", Status: common.UserStatusEnabled, AffCode: "complete-inviter-code"}
	require.NoError(t, db.Create(&inviter).Error)
	invitee := model.User{Username: "complete-invitee", Password: "password", Status: common.UserStatusEnabled, AffCode: "complete-invitee-code", InviterId: inviter.Id}
	require.NoError(t, db.Create(&invitee).Error)
	orders := []model.TopUp{
		{UserId: invitee.Id, Amount: 1, Money: 1, TradeNo: "complete-default-rebate", PaymentProvider: model.PaymentProviderEpay, Status: common.TopUpStatusPending},
		{UserId: invitee.Id, Amount: 1, Money: 1, TradeNo: "complete-no-rebate", PaymentProvider: model.PaymentProviderEpay, Status: common.TopUpStatusPending},
	}
	require.NoError(t, db.Create(&orders).Error)

	requests := []struct {
		body          string
		expectedGrant bool
	}{
		{body: `{"trade_no":"complete-default-rebate"}`, expectedGrant: true},
		{body: `{"trade_no":"complete-no-rebate","grant_affiliate_rebate":false}`},
	}
	for _, request := range requests {
		response := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(response)
		context.Request = httptest.NewRequest(http.MethodPost, "/api/user/topup/complete", strings.NewReader(request.body))
		context.Request.Header.Set("Content-Type", "application/json")
		AdminCompleteTopUp(context)

		assert.Equal(t, http.StatusOK, response.Code)
		var payload struct {
			Success bool                              `json:"success"`
			Data    model.ManualTopUpCompletionResult `json:"data"`
		}
		require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
		assert.True(t, payload.Success)
		assert.True(t, payload.Data.Completed)
		assert.Equal(t, request.expectedGrant, payload.Data.AffiliateRebateGranted)
	}

	var updatedInviter model.User
	require.NoError(t, db.First(&updatedInviter, inviter.Id).Error)
	assert.Equal(t, 100, updatedInviter.AffQuota)
	assert.Equal(t, 100, updatedInviter.AffHistoryQuota)
}
