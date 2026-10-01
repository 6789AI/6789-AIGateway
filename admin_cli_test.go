package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfirmRootPromotionRequiresExactTargetID(t *testing.T) {
	user := &model.User{Id: 42, Username: "operator", Role: common.RoleAdminUser}
	var output bytes.Buffer
	require.NoError(t, confirmRootPromotion(strings.NewReader("PROMOTE 42\n"), &output, user))
	assert.Contains(t, output.String(), "operator (ID 42")
	assert.ErrorContains(t, confirmRootPromotion(strings.NewReader("PROMOTE 43\n"), &output, user), "cancelled")
	assert.ErrorContains(t, confirmRootPromotion(strings.NewReader("yes\n"), &output, user), "cancelled")
}

func TestAdminCLIRequiresInteractiveTerminalBeforeOpeningDatabase(t *testing.T) {
	input, err := os.CreateTemp(t.TempDir(), "promotion-input-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = input.Close() })
	var output bytes.Buffer
	assert.ErrorContains(t, runAdminCLI([]string{"promote-root", "--user-id", "42"}, input, &output),
		"interactive terminal")
	assert.ErrorContains(t, runAdminCLI([]string{"promote-root", "--user-id", "0"}, input, &output),
		"usage")
}
