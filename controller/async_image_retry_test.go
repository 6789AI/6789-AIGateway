package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestShouldRetryAsyncImageCapabilityMismatch(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	taskErr := &taskdto.TaskError{
		Code:       "async_image_not_supported",
		StatusCode: http.StatusBadRequest,
		LocalError: true,
	}

	assert.True(t, shouldRetryTaskRelay(c, 1, taskErr, 1))
	assert.False(t, shouldRetryTaskRelay(c, 1, taskErr, 0))

	c.Set("specific_channel_id", 1)
	assert.False(t, shouldRetryTaskRelay(c, 1, taskErr, 1))
}
