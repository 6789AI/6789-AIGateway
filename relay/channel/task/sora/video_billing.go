package sora

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/gin-gonic/gin"
)

func (a *TaskAdaptor) VideoBillingResolution(c *gin.Context, info *relaycommon.RelayInfo) (string, error) {
	if a.isVinted {
		return a.vintedResolution, nil
	}
	var fields struct {
		Resolution string `json:"resolution"`
		Size       string `json:"size"`
	}
	if strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
		if err := common.UnmarshalBodyReusable(c, &fields); err != nil {
			return "", err
		}
	} else {
		form, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return "", err
		}
		if values := form.Value["resolution"]; len(values) > 0 {
			fields.Resolution = values[0]
		}
		if values := form.Value["size"]; len(values) > 0 {
			fields.Size = values[0]
		}
	}
	if fields.Resolution != "" {
		return fields.Resolution, nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return "", err
	}
	if resolution, ok := req.Metadata["resolution"].(string); ok && resolution != "" {
		return resolution, nil
	}
	if fields.Size != "" {
		return billing_setting.VideoResolutionFromSize(fields.Size), nil
	}
	if info.Action == constant.TaskActionRemix {
		return info.TaskVideoResolution, nil
	}
	return "720p", nil
}
