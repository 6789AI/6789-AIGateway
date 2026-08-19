package model

import (
	"net/url"
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

// ChannelMetadataSupportsAsyncImage reports whether the selected upstream
// model has a native asynchronous image API on this channel.
func ChannelMetadataSupportsAsyncImage(channelType int, baseURL string, settings dto.ChannelOtherSettings, upstreamModel string) bool {
	if IsGrsaiAsyncImageChannel(baseURL, settings) {
		return IsGrsaiImageModel(upstreamModel)
	}
	return channelType == constant.ChannelTypeAli && !model_setting.IsSyncImageModel(upstreamModel)
}

// IsGrsaiAsyncImageChannel recognizes the public Grsai endpoints and supports
// an explicit override for deployments that proxy Grsai through another host.
func IsGrsaiAsyncImageChannel(baseURL string, settings dto.ChannelOtherSettings) bool {
	provider := strings.ToLower(strings.TrimSpace(settings.AsyncImageProvider))
	if provider != "" {
		return provider == "grsai"
	}

	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return false
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	return host == "grsaiapi.com" || host == "grsai.dakka.com.cn"
}

func IsGrsaiImageModel(modelName string) bool {
	modelName = strings.ToLower(strings.TrimSpace(modelName))
	return strings.HasPrefix(modelName, "nano-banana") ||
		modelName == "gpt-image-2" ||
		strings.HasPrefix(modelName, "gpt-image-2-")
}
