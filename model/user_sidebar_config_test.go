package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateDefaultSidebarConfigIncludesRedemptionTopup(t *testing.T) {
	var config map[string]map[string]bool
	require.NoError(t, common.UnmarshalJsonStr(generateDefaultSidebarConfigForRole(common.RoleCommonUser), &config))

	assert.True(t, config["personal"]["redemptionTopup"])
}
