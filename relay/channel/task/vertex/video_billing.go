package vertex

import (
	geminitask "github.com/QuantumNous/new-api/relay/channel/task/gemini"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func (a *TaskAdaptor) VideoBillingResolution(c *gin.Context, info *relaycommon.RelayInfo) (string, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return "", err
	}
	return geminitask.ResolveVeoResolution(req.Metadata, req.Size), nil
}
