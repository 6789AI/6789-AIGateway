package dto

import (
	"strconv"
	"testing"

	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestImageRequestAsyncIsGatewayOnly(t *testing.T) {
	for _, value := range []bool{false, true} {
		t.Run(strconv.FormatBool(value), func(t *testing.T) {
			raw := []byte(`{"model":"wanx-v1","prompt":"draw","async":` + strconv.FormatBool(value) + `}`)
			var request ImageRequest
			require.NoError(t, kitutil.Unmarshal(raw, &request))
			require.NotNil(t, request.Async)
			assert.Equal(t, value, *request.Async)
			assert.NotContains(t, request.Extra, "async")

			encoded, err := kitutil.Marshal(request)
			require.NoError(t, err)
			assert.False(t, gjson.GetBytes(encoded, "async").Exists())
			assert.Equal(t, "wanx-v1", gjson.GetBytes(encoded, "model").String())
		})
	}
}

func TestImageRequestRejectsNonBooleanAsync(t *testing.T) {
	for _, value := range []string{`null`, `"true"`, `1`, `{}`, `[]`} {
		t.Run(value, func(t *testing.T) {
			var request ImageRequest
			err := kitutil.Unmarshal([]byte(`{"prompt":"draw","async":`+value+`}`), &request)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "field async must be a boolean")
		})
	}
}

func TestNewImageTaskResponseUsesTaskContract(t *testing.T) {
	response := NewImageTaskResponse("task-123")
	require.Equal(t, "task-123", response.ID)
	require.Equal(t, response.ID, response.TaskID)
	require.Equal(t, "image.generation.task", response.Object)
	require.Equal(t, ImageTaskStatusQueued, response.Status)
}
