package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func ChannelSupportsAsyncImage(channel *Channel, modelName string) bool {
	if channel == nil {
		return false
	}

	mappingJSON := channel.GetModelMapping()
	if mappingJSON == "" || mappingJSON == "{}" {
		return ChannelMetadataSupportsAsyncImage(
			channel.Type,
			channel.GetBaseURL(),
			channel.GetOtherSettings(),
			modelName,
		)
	}
	modelMap := make(map[string]string)
	if err := common.UnmarshalJsonStr(mappingJSON, &modelMap); err != nil {
		return false
	}

	currentModel := modelName
	visited := map[string]bool{currentModel: true}
	for {
		mappedModel, ok := modelMap[currentModel]
		if !ok || mappedModel == "" || mappedModel == currentModel {
			break
		}
		if visited[mappedModel] {
			return false
		}
		visited[mappedModel] = true
		currentModel = mappedModel
	}
	if currentModel == "" {
		return false
	}
	return ChannelMetadataSupportsAsyncImage(
		channel.Type,
		channel.GetBaseURL(),
		channel.GetOtherSettings(),
		currentModel,
	)
}

// ChannelMetadataSupportsAsyncImage reports whether this channel is explicitly
// enabled for the selected asynchronous image protocol and upstream model.
func ChannelMetadataSupportsAsyncImage(channelType int, _ string, settings dto.ChannelOtherSettings, upstreamModel string) bool {
	if !ChannelTypeSupportsImageGeneration(channelType) || !settings.AsyncImageEnabled {
		return false
	}
	switch AsyncImageProvider(settings) {
	case dto.AsyncImageProviderAli:
		return !model_setting.IsSyncImageModel(upstreamModel)
	case dto.AsyncImageProviderGrsai:
		return IsGrsaiImageModel(upstreamModel)
	case dto.AsyncImageProviderNewAPI:
		return true
	default:
		return false
	}
}

func AsyncImageProvider(settings dto.ChannelOtherSettings) string {
	return strings.ToLower(strings.TrimSpace(settings.AsyncImageProvider))
}

func IsGrsaiAsyncImageChannel(_ string, settings dto.ChannelOtherSettings) bool {
	return settings.AsyncImageEnabled && AsyncImageProvider(settings) == dto.AsyncImageProviderGrsai
}

func IsGrsaiImageModel(modelName string) bool {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	return strings.HasPrefix(modelName, "nano-banana") ||
		modelName == "gpt-image-2" ||
		strings.HasPrefix(modelName, "gpt-image-2-")
}

func ChannelTypeSupportsImageGeneration(channelType int) bool {
	switch channelType {
	case constant.ChannelTypeOpenAI,
		constant.ChannelTypeAzure,
		constant.ChannelTypeOhMyGPT,
		constant.ChannelTypeCustom,
		constant.ChannelTypeAli,
		constant.ChannelType360,
		constant.ChannelTypeOpenRouter,
		constant.ChannelTypeFastGPT,
		constant.ChannelTypeGemini,
		constant.ChannelTypeZhipu_v4,
		constant.ChannelTypeLingYiWanWu,
		constant.ChannelTypeMiniMax,
		constant.ChannelTypeSiliconFlow,
		constant.ChannelTypeVertexAi,
		constant.ChannelTypeVolcEngine,
		constant.ChannelTypeXinference,
		constant.ChannelTypeXai,
		constant.ChannelTypeJimeng,
		constant.ChannelTypeReplicate,
		constant.ChannelTypeAdvancedCustom,
		constant.ChannelTypeSub2API,
		constant.ChannelTypeNewAPI:
		return true
	default:
		return false
	}
}
