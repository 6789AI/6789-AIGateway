package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
)

func TestInitTaskPreservesSelectedMultiKeyForPolling(t *testing.T) {
	info := &relaycommon.RelayInfo{
		UserId:        7,
		TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeAli,
			ChannelId:         11,
			ChannelIsMultiKey: true,
			ApiKey:            "selected-key",
		},
	}

	task := InitTask(constant.TaskPlatform("17"), info)

	assert.Equal(t, "selected-key", task.PrivateData.Key)
	assert.Equal(t, "task_public", task.TaskID)
}

func TestInitTaskPreservesGrsaiKeyForPolling(t *testing.T) {
	info := &relaycommon.RelayInfo{
		UserId:        7,
		TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeOpenAI,
			ChannelId:   60,
			ApiKey:      "selected-key",
		},
	}

	task := InitTask(constant.TaskPlatformGrsai, info)

	assert.Equal(t, "selected-key", task.PrivateData.Key)
}
