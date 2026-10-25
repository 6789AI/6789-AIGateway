package doubao

import (
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func (a *TaskAdaptor) VideoBillingResolution(c *gin.Context, info *relaycommon.RelayInfo) (string, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return "", err
	}
	body, err := a.convertToRequestPayload(&req)
	if err != nil {
		return "", err
	}
	return body.Resolution, nil
}
