package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOptionsHidesBTMailPassword(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	common.OptionMap = map[string]string{
		"BTMailAPIURL":   "https://panel.example.com/mail_sys/send_mail_http.json",
		"BTMailFrom":     "sender@example.com",
		"BTMailPassword": "mail-password",
	}
	common.OptionMapRWMutex.Unlock()
	originalPassword := common.BTMailPassword
	common.BTMailPassword = "mail-password"
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
		common.BTMailPassword = originalPassword
	})

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	GetOptions(context)

	require.Equal(t, 200, recorder.Code)
	var response struct {
		Data []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	values := make(map[string]string, len(response.Data))
	for _, option := range response.Data {
		values[option.Key] = option.Value
	}

	assert.NotContains(t, values, "BTMailPassword")
	assert.Equal(t, "true", values["BTMailPasswordConfigured"])
	assert.Equal(
		t,
		"https://panel.example.com/mail_sys/send_mail_http.json",
		values["BTMailAPIURL"],
	)
}
