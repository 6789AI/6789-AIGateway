package billing_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldMultiplyTaskDurationDefaultsToEnabled(t *testing.T) {
	original := billingSetting.TaskDurationMultiplier
	t.Cleanup(func() { billingSetting.TaskDurationMultiplier = original })
	billingSetting.TaskDurationMultiplier = map[string]bool{"disabled-model": false}

	assert.True(t, ShouldMultiplyTaskDuration("unconfigured-model"))
	assert.False(t, ShouldMultiplyTaskDuration("disabled-model"))
}

func TestValidateTaskDurationMultiplierJSON(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "boolean map", value: `{"video-model":false}`},
		{name: "empty map", value: `{}`},
		{name: "non boolean value", value: `{"video-model":"false"}`, wantErr: true},
		{name: "blank model name", value: `{" ":false}`, wantErr: true},
		{name: "null", value: `null`, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateTaskDurationMultiplierJSON(test.value)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGetPricingSyncDataIncludesTaskDurationMultiplier(t *testing.T) {
	original := billingSetting.TaskDurationMultiplier
	t.Cleanup(func() { billingSetting.TaskDurationMultiplier = original })
	billingSetting.TaskDurationMultiplier = map[string]bool{"video-model": false}

	data := GetPricingSyncData(map[string]any{})
	durationMultipliers, ok := data[TaskDurationMultiplierField].(map[string]bool)

	require.True(t, ok)
	assert.Equal(t, map[string]bool{"video-model": false}, durationMultipliers)
}
