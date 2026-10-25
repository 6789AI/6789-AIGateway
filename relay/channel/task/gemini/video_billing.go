package gemini

import (
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func (a *TaskAdaptor) VideoBillingResolution(c *gin.Context, info *relaycommon.RelayInfo) (string, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return "", err
	}
	return ResolveVeoResolution(req.Metadata, req.Size), nil
}
