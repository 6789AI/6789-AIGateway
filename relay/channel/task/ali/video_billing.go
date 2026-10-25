package ali

import (
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func (a *TaskAdaptor) VideoBillingResolution(c *gin.Context, info *relaycommon.RelayInfo) (string, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return "", err
	}
	body, err := a.convertToAliRequest(info, req)
	if err != nil {
		return "", err
	}
	if body.Parameters.Size != "" {
		return sizeToResolution(body.Parameters.Size)
	}
	return body.Parameters.Resolution, nil
}
