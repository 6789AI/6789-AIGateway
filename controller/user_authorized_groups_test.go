package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserGroupsWithoutLoginReturnsOnlyPublicGroups(t *testing.T) {
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = originalRedisEnabled })
	originalRatios := ratio_setting.GroupRatio2JSONString()
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"hidden":1}`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default"}`))
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/groups", nil)
	GetUserGroups(ctx)

	var response struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Contains(t, response.Data, "default")
	assert.NotContains(t, response.Data, "hidden")
}

func TestGetUserGroupsWithLoginIncludesAuthorizedHiddenGroup(t *testing.T) {
	db := setupManageUserTestDB(t)
	originalRatios := ratio_setting.GroupRatio2JSONString()
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"hidden":1}`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default"}`))
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
	})
	user := &model.User{
		Username: "hidden-group-user", Password: "password", Group: "default",
		AuthorizedGroups: []string{"hidden"}, Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(user).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/self/groups", nil)
	ctx.Set("id", user.Id)
	GetUserGroups(ctx)

	var response struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Contains(t, response.Data, "default")
	assert.Contains(t, response.Data, "hidden")
	assert.NotContains(t, response.Data, "auto")
}

func TestUpdateUserAuthorizedGroupsGrantAndRevoke(t *testing.T) {
	db := setupManageUserTestDB(t)
	originalRatios := ratio_setting.GroupRatio2JSONString()
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"hidden":1}`))
	t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios)) })
	user := &model.User{Username: "authorized-user", Password: "password", Group: "default", Role: common.RoleCommonUser, Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(user).Error)
	initialVersion := user.AuthVersion

	request := func(body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPut, "/api/user/", strings.NewReader(body))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Set("id", 9999)
		ctx.Set("role", common.RoleAdminUser)
		ctx.Set("username", "root-operator")
		UpdateUser(ctx)
		return recorder
	}

	grant := request(`{"id":` + stringInt(user.Id) + `,"username":"authorized-user","group":"default","authorized_groups":["hidden"]}`)
	assert.Contains(t, grant.Body.String(), `"success":true`)
	require.NoError(t, db.First(user, user.Id).Error)
	assert.Equal(t, []string{"hidden"}, user.AuthorizedGroups)
	assert.Greater(t, user.AuthVersion, initialVersion)

	reject := request(`{"id":` + stringInt(user.Id) + `,"username":"authorized-user","group":"default","authorized_groups":["unknown"]}`)
	assert.Contains(t, reject.Body.String(), `"success":false`)
	require.NoError(t, db.First(user, user.Id).Error)
	assert.Equal(t, []string{"hidden"}, user.AuthorizedGroups)

	revoke := request(`{"id":` + stringInt(user.Id) + `,"username":"authorized-user","group":"default","authorized_groups":[]}`)
	assert.Contains(t, revoke.Body.String(), `"success":true`)
	require.NoError(t, db.First(user, user.Id).Error)
	assert.Empty(t, user.AuthorizedGroups)
}

func TestUpdateUserRemovesPrimaryGroupFromExistingAdditionalGrants(t *testing.T) {
	db := setupManageUserTestDB(t)
	originalRatios := ratio_setting.GroupRatio2JSONString()
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"hidden":1}`))
	t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios)) })
	user := &model.User{
		Username: "primary-switch-user", Password: "password", Group: "default",
		AuthorizedGroups: []string{"hidden"}, Role: common.RoleCommonUser, Status: common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(user).Error)

	request := func(group, authorizedGroups string) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPut, "/api/user/", strings.NewReader(`{"id":`+stringInt(user.Id)+`,"username":"primary-switch-user","group":"`+group+`"`+authorizedGroups+`}`))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Set("id", 9999)
		ctx.Set("role", common.RoleAdminUser)
		ctx.Set("username", "root-operator")
		UpdateUser(ctx)
		require.Contains(t, recorder.Body.String(), `"success":true`)
	}

	request("hidden", "")
	require.NoError(t, db.First(user, user.Id).Error)
	assert.Empty(t, user.AuthorizedGroups)

	request("default", `,"authorized_groups":["hidden"]`)
	require.NoError(t, db.First(user, user.Id).Error)
	assert.Equal(t, []string{"hidden"}, user.AuthorizedGroups)

	request("hidden", `,"authorized_groups":["hidden"]`)
	require.NoError(t, db.First(user, user.Id).Error)
	assert.Empty(t, user.AuthorizedGroups)

	request("default", "")
	require.NoError(t, db.First(user, user.Id).Error)
	assert.Empty(t, user.AuthorizedGroups)
}
