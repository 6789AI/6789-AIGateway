package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func performAffiliateSearchRequest(t *testing.T, requestURL string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, requestURL, nil)
	SearchAffiliateUsers(c)
	return recorder
}

func decodeAffiliateSearchResponse(t *testing.T, recorder *httptest.ResponseRecorder) struct {
	Success bool                    `json:"success"`
	Code    string                  `json:"code"`
	Message string                  `json:"message"`
	Data    affiliateLookupResponse `json:"data"`
} {
	t.Helper()
	var response struct {
		Success bool                    `json:"success"`
		Code    string                  `json:"code"`
		Message string                  `json:"message"`
		Data    affiliateLookupResponse `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestParseAffiliateLookupQuery(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantType   model.AffiliateLookupType
		wantValue  string
		wantUserId int
		wantErr    bool
	}{
		{name: "automatic value", query: " 46fD ", wantType: model.AffiliateLookupTypeAuto, wantValue: "46fD"},
		{name: "automatic numeric id", query: "42", wantType: model.AffiliateLookupTypeAuto, wantValue: "42", wantUserId: 42},
		{name: "absolute signup link", query: "https://www.6789api.top/sign-up?aff=46fD", wantType: model.AffiliateLookupTypeAffCode, wantValue: "46fD"},
		{name: "relative signup link", query: "/sign-up?next=%2Fconsole&aff=46fD", wantType: model.AffiliateLookupTypeAffCode, wantValue: "46fD"},
		{name: "forced id", query: "ID: 42", wantType: model.AffiliateLookupTypeUserId, wantValue: "42", wantUserId: 42},
		{name: "forced email", query: "email: OWNER@EXAMPLE.COM ", wantType: model.AffiliateLookupTypeEmail, wantValue: "owner@example.com"},
		{name: "forced affiliate code", query: "aff: 46fD", wantType: model.AffiliateLookupTypeAffCode, wantValue: "46fD"},
		{name: "empty", query: " ", wantErr: true},
		{name: "link without code", query: "https://www.6789api.top/sign-up", wantErr: true},
		{name: "link with empty code", query: "https://www.6789api.top/sign-up?aff=", wantErr: true},
		{name: "forced code exceeds database limit", query: "aff:" + strings.Repeat("a", 33), wantErr: true},
		{name: "automatic value exceeds field limits", query: strings.Repeat("a", 51), wantErr: true},
		{name: "invalid forced id", query: "id:not-a-number", wantErr: true},
		{name: "unsupported username prefix", query: "username:owner", wantErr: true},
		{name: "malformed link", query: "https://%", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAffiliateLookupQuery(tt.query)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, got.Type)
			assert.Equal(t, tt.wantValue, got.Value)
			assert.Equal(t, tt.wantUserId, got.UserId)
		})
	}
}

func TestSearchAffiliateUsersReturnsOwnerAndDirectInviteesFromDatabase(t *testing.T) {
	db := setupManageUserTestDB(t)
	inviter := model.User{
		Username: "upstream-inviter", Password: "inviter-secret", DisplayName: "Upstream Inviter",
		Email: "inviter@example.com", Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
		Group: "default", AffCode: "UP01",
	}
	require.NoError(t, db.Create(&inviter).Error)
	owner := model.User{
		Username: "affiliate-owner", Password: "owner-secret", DisplayName: "Affiliate Owner",
		Email: "owner@example.com", Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
		Group: "default", AffCode: "46fD", InviterId: inviter.Id,
	}
	require.NoError(t, db.Create(&owner).Error)
	directActive := model.User{
		Username: "direct-active", Password: "active-secret", DisplayName: "Direct Active",
		Email: "active@example.com", Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
		Group: "default", AffCode: "act1", InviterId: owner.Id,
	}
	require.NoError(t, db.Create(&directActive).Error)
	directDeleted := model.User{
		Username: "direct-deleted", Password: "deleted-secret", DisplayName: "Direct Deleted",
		Email: "deleted@example.com", Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
		Group: "default", AffCode: "del1", InviterId: owner.Id,
	}
	require.NoError(t, db.Create(&directDeleted).Error)
	require.NoError(t, db.Delete(&directDeleted).Error)
	grandchild := model.User{
		Username: "grandchild", Password: "grandchild-secret", DisplayName: "Grandchild",
		Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default",
		AffCode: "gran", InviterId: directActive.Id,
	}
	require.NoError(t, db.Create(&grandchild).Error)
	unrelated := model.User{
		Username: "unrelated", Password: "unrelated-secret", DisplayName: "Unrelated",
		Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default",
		AffCode: "none",
	}
	require.NoError(t, db.Create(&unrelated).Error)
	require.NoError(t, db.Delete(&inviter).Error)

	queryCount := 0
	callbackName := "test:count_affiliate_lookup_queries"
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(callbackName, func(_ *gorm.DB) {
		queryCount++
	}))
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(callbackName)
	})
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = true
	t.Cleanup(func() {
		common.RedisEnabled = originalRedisEnabled
	})

	link := "https://www.6789api.top/sign-up?aff=46fD"
	recorder := performAffiliateSearchRequest(t, fmt.Sprintf(
		"/api/user/aff/search?q=%s&p=1&page_size=1",
		url.QueryEscape(link),
	))
	response := decodeAffiliateSearchResponse(t, recorder)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, response.Success)
	assert.Equal(t, "46fD", response.Data.AffCode)
	require.NotNil(t, response.Data.Owner)
	assert.Equal(t, owner.Id, response.Data.Owner.Id)
	assert.Equal(t, "affiliate-owner", response.Data.Owner.Username)
	assert.Equal(t, inviter.Id, response.Data.Owner.InviterId)
	require.NotNil(t, response.Data.Inviter)
	assert.Equal(t, inviter.Id, response.Data.Inviter.Id)
	assert.Equal(t, "UP01", response.Data.Inviter.AffCode)
	assert.NotNil(t, response.Data.Inviter.DeletedAt)
	assert.Equal(t, int64(2), response.Data.Invitees.Total)
	assert.Equal(t, 1, response.Data.Invitees.Page)
	assert.Equal(t, 1, response.Data.Invitees.PageSize)
	require.Len(t, response.Data.Invitees.Items, 1)
	assert.Equal(t, directDeleted.Id, response.Data.Invitees.Items[0].Id)
	assert.NotNil(t, response.Data.Invitees.Items[0].DeletedAt)
	assert.GreaterOrEqual(t, queryCount, 3)
	assert.NotContains(t, recorder.Body.String(), `"password"`)
	assert.NotContains(t, recorder.Body.String(), `"access_token"`)
	assert.NotContains(t, recorder.Body.String(), `"github_id"`)
	assert.NotContains(t, recorder.Body.String(), `"quota"`)

	linkResponse := response.Data
	queries := []string{
		"46fD",
		"aff:46fD",
		strconv.Itoa(owner.Id),
		"id:" + strconv.Itoa(owner.Id),
		"OWNER@EXAMPLE.COM",
		"email:OWNER@EXAMPLE.COM",
	}
	for _, query := range queries {
		recorder = performAffiliateSearchRequest(t, fmt.Sprintf(
			"/api/user/aff/search?q=%s&p=1&page_size=1",
			url.QueryEscape(query),
		))
		response = decodeAffiliateSearchResponse(t, recorder)
		assert.True(t, response.Success, query)
		assert.Equal(t, linkResponse, response.Data, query)
	}

	recorder = performAffiliateSearchRequest(t, "/api/user/aff/search?q=affiliate-owner")
	response = decodeAffiliateSearchResponse(t, recorder)
	assert.True(t, response.Success)
	assert.Nil(t, response.Data.Owner)

	recorder = performAffiliateSearchRequest(t, "/api/user/aff/search?q=46fD&p=2&page_size=1")
	response = decodeAffiliateSearchResponse(t, recorder)
	require.Len(t, response.Data.Invitees.Items, 1)
	assert.Equal(t, directActive.Id, response.Data.Invitees.Items[0].Id)
	assert.Nil(t, response.Data.Invitees.Items[0].DeletedAt)
	assert.NotEqual(t, grandchild.Id, response.Data.Invitees.Items[0].Id)
	assert.NotEqual(t, unrelated.Id, response.Data.Invitees.Items[0].Id)
}

func TestSearchAffiliateUsersRejectsAmbiguousAutomaticLookupAndAcceptsPrefixes(t *testing.T) {
	db := setupManageUserTestDB(t)
	emailMatch := model.User{
		Username: "email-match", Password: "password", Email: "collision@example.com", Status: common.UserStatusEnabled,
		Group: "default", AffCode: "user-code",
	}
	affCodeMatch := model.User{
		Username: "other-user", Password: "password", Status: common.UserStatusEnabled,
		Group: "default", AffCode: "collision@example.com",
	}
	require.NoError(t, db.Create(&emailMatch).Error)
	require.NoError(t, db.Create(&affCodeMatch).Error)

	recorder := performAffiliateSearchRequest(t, "/api/user/aff/search?q=collision%40example.com")
	response := decodeAffiliateSearchResponse(t, recorder)
	assert.False(t, response.Success)
	assert.Equal(t, affiliateLookupAmbiguousErrorCode, response.Code)
	assert.NotEmpty(t, response.Message)

	recorder = performAffiliateSearchRequest(t, "/api/user/aff/search?q=email%3Acollision%40example.com")
	response = decodeAffiliateSearchResponse(t, recorder)
	require.NotNil(t, response.Data.Owner)
	assert.Equal(t, emailMatch.Id, response.Data.Owner.Id)

	recorder = performAffiliateSearchRequest(t, "/api/user/aff/search?q=aff%3Acollision%40example.com")
	response = decodeAffiliateSearchResponse(t, recorder)
	require.NotNil(t, response.Data.Owner)
	assert.Equal(t, affCodeMatch.Id, response.Data.Owner.Id)
}

func TestSearchAffiliateUsersReturnsEmptyResultAndRejectsInvalidInput(t *testing.T) {
	setupManageUserTestDB(t)

	recorder := performAffiliateSearchRequest(t, "/api/user/aff/search?q=missing")
	response := decodeAffiliateSearchResponse(t, recorder)
	assert.True(t, response.Success)
	assert.Nil(t, response.Data.Owner)
	assert.Empty(t, response.Data.Invitees.Items)
	assert.Zero(t, response.Data.Invitees.Total)
	assert.Equal(t, 20, response.Data.Invitees.PageSize)

	invalidRequests := []string{
		"/api/user/aff/search?q=",
		"/api/user/aff/search?q=46fD&p=0",
		"/api/user/aff/search?q=46fD&p=invalid",
		"/api/user/aff/search?q=46fD&page_size=0",
		"/api/user/aff/search?q=46fD&page_size=invalid",
	}
	for _, requestURL := range invalidRequests {
		recorder = performAffiliateSearchRequest(t, requestURL)
		response = decodeAffiliateSearchResponse(t, recorder)
		assert.False(t, response.Success, requestURL)
		assert.NotEmpty(t, response.Message, requestURL)
	}
}
