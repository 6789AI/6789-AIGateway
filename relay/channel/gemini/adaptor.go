package gemini

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/reasoning"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type Adaptor struct {
}

func isImagenModel(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "imagen")
}

func isGeminiImageGenerationModel(model string) bool {
	model = strings.TrimSpace(model)
	return !isImagenModel(model) && model_setting.IsGeminiModelSupportImagine(model)
}

func normalizeImageAspectRatio(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "auto" {
		return "", nil
	}

	// OpenAI's landscape and portrait sizes are close to, but not exactly,
	// Gemini's documented ratios. Preserve their established meaning.
	switch value {
	case "1792x1024":
		return "16:9", nil
	case "1024x1792":
		return "9:16", nil
	}

	separator := "x"
	if strings.Contains(value, ":") {
		separator = ":"
	}
	parts := strings.Split(value, separator)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid image aspect ratio %q; use a ratio such as 16:9 or dimensions such as 1920x1080", value)
	}
	width, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32)
	if err != nil || width == 0 {
		return "", fmt.Errorf("invalid image aspect ratio %q; width must be a positive integer", value)
	}
	height, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 32)
	if err != nil || height == 0 {
		return "", fmt.Errorf("invalid image aspect ratio %q; height must be a positive integer", value)
	}

	a, b := width, height
	for b != 0 {
		a, b = b, a%b
	}
	return fmt.Sprintf("%d:%d", width/a, height/a), nil
}

func normalizeImageSize(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto":
		return "", nil
	case "1k", "standard", "medium", "low":
		return "1K", nil
	case "2k", "hd", "high":
		return "2K", nil
	case "4k":
		return "4K", nil
	default:
		return "", fmt.Errorf("invalid image size %q; use auto, 1K, 2K, or 4K", value)
	}
}

func supportsGeminiImageSize(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(model, "gemini-3-pro-image") ||
		strings.HasPrefix(model, "gemini-3.1-flash-image") ||
		strings.HasPrefix(model, "nano-banana-pro")
}

func imageRequestExtraString(extra map[string]json.RawMessage, keys ...string) (string, bool, error) {
	for _, key := range keys {
		raw, ok := extra[key]
		if !ok {
			continue
		}
		if common.GetJsonType(raw) != "string" {
			return "", true, fmt.Errorf("%s must be a string", key)
		}
		var value string
		if err := common.Unmarshal(raw, &value); err != nil {
			return "", true, fmt.Errorf("invalid %s: %w", key, err)
		}
		return strings.TrimSpace(value), true, nil
	}
	return "", false, nil
}

func imageConfigString(config map[string]interface{}, keys ...string) (string, bool, error) {
	for _, key := range keys {
		value, ok := config[key]
		if !ok {
			continue
		}
		text, ok := value.(string)
		if !ok {
			return "", true, fmt.Errorf("extra_body.google.image_config.%s must be a string", key)
		}
		return strings.TrimSpace(text), true, nil
	}
	return "", false, nil
}

func imageRequestOptions(request dto.ImageRequest) (string, string, error) {
	aspectRatioValue := request.Size
	imageSizeValue := request.Quality

	if value, exists, err := imageRequestExtraString(request.Extra, "aspect_ratio", "aspectRatio"); err != nil {
		return "", "", err
	} else if exists {
		aspectRatioValue = value
	}
	if value, exists, err := imageRequestExtraString(request.Extra, "image_size", "imageSize"); err != nil {
		return "", "", err
	} else if exists {
		imageSizeValue = value
	}

	if rawExtraBody, exists := request.Extra["extra_body"]; exists && common.GetJsonType(rawExtraBody) != "null" {
		var extraBody map[string]interface{}
		if err := common.Unmarshal(rawExtraBody, &extraBody); err != nil {
			return "", "", fmt.Errorf("invalid extra_body: %w", err)
		}
		if googleValue, exists := extraBody["google"]; exists {
			googleBody, ok := googleValue.(map[string]interface{})
			if !ok {
				return "", "", errors.New("extra_body.google must be an object")
			}
			configValue, exists := googleBody["image_config"]
			if !exists {
				configValue, exists = googleBody["imageConfig"]
			}
			if exists {
				imageConfig, ok := configValue.(map[string]interface{})
				if !ok {
					return "", "", errors.New("extra_body.google.image_config must be an object")
				}
				if value, exists, err := imageConfigString(imageConfig, "aspect_ratio", "aspectRatio"); err != nil {
					return "", "", err
				} else if exists {
					aspectRatioValue = value
				}
				if value, exists, err := imageConfigString(imageConfig, "image_size", "imageSize"); err != nil {
					return "", "", err
				} else if exists {
					imageSizeValue = value
				}
			}
		}
	}

	aspectRatio, err := normalizeImageAspectRatio(aspectRatioValue)
	if err != nil {
		return "", "", err
	}
	imageSize, err := normalizeImageSize(imageSizeValue)
	if err != nil {
		return "", "", err
	}
	return aspectRatio, imageSize, nil
}

func (a *Adaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	if len(request.Contents) > 0 {
		for i, content := range request.Contents {
			if i == 0 {
				if request.Contents[0].Role == "" {
					request.Contents[0].Role = "user"
				}
			}
			for _, part := range content.Parts {
				if part.FileData != nil {
					if part.FileData.MimeType == "" && strings.Contains(part.FileData.FileUri, "www.youtube.com") {
						part.FileData.MimeType = "video/webm"
					}
				}
			}
		}
	}
	return request, nil
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, req *dto.ClaudeRequest) (any, error) {
	result, err := relayconvert.ConvertRequest(c, info, types.RelayFormatGemini, req)
	if err != nil {
		return nil, err
	}
	geminiRequest, ok := result.Value.(*dto.GeminiChatRequest)
	if !ok {
		return nil, fmt.Errorf("expected Gemini generateContent request, got %T", result.Value)
	}
	return geminiRequest, nil
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	if info == nil {
		return nil, errors.New("relay info is required for image generation")
	}

	if isImagenModel(info.UpstreamModelName) {
		aspectRatio, imageSize, err := imageRequestOptions(request)
		if err != nil {
			return nil, err
		}
		geminiRequest := dto.GeminiImageRequest{
			Instances: []dto.GeminiImageInstance{{Prompt: request.Prompt}},
			Parameters: dto.GeminiImageParameters{
				SampleCount:      int(lo.FromPtrOr(request.N, uint(1))),
				AspectRatio:      aspectRatio,
				PersonGeneration: "allow_adult",
			},
		}
		if imageSize != "" {
			geminiRequest.Parameters.ImageSize = imageSize
		}
		return geminiRequest, nil
	}

	if !isGeminiImageGenerationModel(info.UpstreamModelName) {
		return nil, errors.New("not supported model for image generation, use an Imagen or configured Gemini image model")
	}
	if info.RelayMode == constant.RelayModeImagesEdits {
		return nil, errors.New("Gemini image models do not support image edits through this endpoint")
	}
	if request.Stream != nil && *request.Stream {
		return nil, errors.New("Gemini image models do not support streaming through this endpoint")
	}
	if request.N != nil && *request.N > 1 {
		return nil, errors.New("Gemini image models only support n=1")
	}

	aspectRatio, imageSize, err := imageRequestOptions(request)
	if err != nil {
		return nil, err
	}
	if imageSize != "" && !supportsGeminiImageSize(info.UpstreamModelName) {
		if imageSize == "1K" {
			imageSize = ""
		} else {
			return nil, fmt.Errorf("model %q does not support adjustable image size %s", info.UpstreamModelName, imageSize)
		}
	}

	imageConfig := make(map[string]interface{}, 2)
	if aspectRatio != "" {
		imageConfig["aspect_ratio"] = aspectRatio
	}
	if imageSize != "" {
		imageConfig["image_size"] = imageSize
	}
	var extraBody json.RawMessage
	if len(imageConfig) > 0 {
		extraBody, err = common.Marshal(map[string]interface{}{
			"google": map[string]interface{}{
				"image_config": imageConfig,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Gemini image config: %w", err)
		}
	}

	return relayconvert.OpenAIChatRequestToGeminiGenerateContent(c, dto.GeneralOpenAIRequest{
		Model: info.UpstreamModelName,
		Messages: []dto.Message{{
			Role:    "user",
			Content: request.Prompt,
		}},
		ExtraBody: extraBody,
	}, info)
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {

}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {

	if model_setting.GetGeminiSettings().ThinkingAdapterEnabled &&
		!model_setting.ShouldPreserveThinkingSuffix(info.OriginModelName) {
		// 新增逻辑：处理 -thinking-<budget> 格式
		if strings.Contains(info.UpstreamModelName, "-thinking-") {
			parts := strings.Split(info.UpstreamModelName, "-thinking-")
			info.UpstreamModelName = parts[0]
		} else if strings.HasSuffix(info.UpstreamModelName, "-thinking") { // 旧的适配
			info.UpstreamModelName = strings.TrimSuffix(info.UpstreamModelName, "-thinking")
		} else if strings.HasSuffix(info.UpstreamModelName, "-nothinking") {
			info.UpstreamModelName = strings.TrimSuffix(info.UpstreamModelName, "-nothinking")
		} else if baseModel, level, ok := reasoning.TrimEffortSuffix(info.UpstreamModelName); ok && level != "" {
			info.UpstreamModelName = baseModel
		}
	}

	version := model_setting.GetGeminiVersionSetting(info.UpstreamModelName)

	if isImagenModel(info.UpstreamModelName) {
		return fmt.Sprintf("%s/%s/models/%s:predict", info.ChannelBaseUrl, version, info.UpstreamModelName), nil
	}

	if strings.HasPrefix(info.UpstreamModelName, "text-embedding") ||
		strings.HasPrefix(info.UpstreamModelName, "embedding") ||
		strings.HasPrefix(info.UpstreamModelName, "gemini-embedding") {
		action := "embedContent"
		if info.IsGeminiBatchEmbedding {
			action = "batchEmbedContents"
		}
		return fmt.Sprintf("%s/%s/models/%s:%s", info.ChannelBaseUrl, version, info.UpstreamModelName, action), nil
	}

	action := "generateContent"
	if info.IsStream {
		action = "streamGenerateContent?alt=sse"
		if info.RelayMode == constant.RelayModeGemini {
			info.DisablePing = true
		}
	}
	return fmt.Sprintf("%s/%s/models/%s:%s", info.ChannelBaseUrl, version, info.UpstreamModelName, action), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("x-goog-api-key", info.ApiKey)
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	result, err := relayconvert.ConvertRequest(c, info, types.RelayFormatGemini, request)
	if err != nil {
		return nil, err
	}
	return result.Value, nil
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	if request.Input == nil {
		return nil, errors.New("input is required")
	}

	inputs := request.ParseInput()
	if len(inputs) == 0 {
		return nil, errors.New("input is empty")
	}
	// We always build a batch-style payload with `requests`, so ensure we call the
	// batch endpoint upstream to avoid payload/endpoint mismatches.
	info.IsGeminiBatchEmbedding = true
	// process all inputs
	geminiRequests := make([]map[string]interface{}, 0, len(inputs))
	for _, input := range inputs {
		geminiRequest := map[string]interface{}{
			"model": fmt.Sprintf("models/%s", info.UpstreamModelName),
			"content": dto.GeminiChatContent{
				Parts: []dto.GeminiPart{
					{
						Text: input,
					},
				},
			},
		}

		// set specific parameters for different models
		// https://ai.google.dev/api/embeddings?hl=zh-cn#method:-models.embedcontent
		switch info.UpstreamModelName {
		case "text-embedding-004", "gemini-embedding-exp-03-07", "gemini-embedding-001":
			// Only newer models introduced after 2024 support OutputDimensionality
			dimensions := lo.FromPtrOr(request.Dimensions, 0)
			if dimensions > 0 {
				geminiRequest["outputDimensionality"] = dimensions
			}
		}
		geminiRequests = append(geminiRequests, geminiRequest)
	}

	return map[string]interface{}{
		"requests": geminiRequests,
	}, nil
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	result, err := relayconvert.ConvertRequest(c, info, types.RelayFormatGemini, &request)
	if err != nil {
		return nil, err
	}
	geminiRequest, ok := result.Value.(*dto.GeminiChatRequest)
	if !ok {
		return nil, fmt.Errorf("expected Gemini generateContent request, got %T", result.Value)
	}
	return geminiRequest, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	if info.RelayMode == constant.RelayModeResponses {
		if info.IsStream {
			return GeminiResponsesStreamHandler(c, info, resp)
		}
		return GeminiResponsesHandler(c, info, resp)
	}

	if info.RelayMode == constant.RelayModeGemini {
		if strings.Contains(info.RequestURLPath, ":embedContent") ||
			strings.Contains(info.RequestURLPath, ":batchEmbedContents") {
			return NativeGeminiEmbeddingHandler(c, resp, info)
		}
		if info.IsStream {
			return GeminiTextGenerationStreamHandler(c, info, resp)
		} else {
			return GeminiTextGenerationHandler(c, info, resp)
		}
	}

	if isImagenModel(info.UpstreamModelName) {
		return GeminiImageHandler(c, info, resp)
	}
	if info.RelayMode == constant.RelayModeImagesGenerations && isGeminiImageGenerationModel(info.UpstreamModelName) {
		return GeminiGenerateContentImageHandler(c, info, resp)
	}

	// check if the model is an embedding model
	if strings.HasPrefix(info.UpstreamModelName, "text-embedding") ||
		strings.HasPrefix(info.UpstreamModelName, "embedding") ||
		strings.HasPrefix(info.UpstreamModelName, "gemini-embedding") {
		return GeminiEmbeddingHandler(c, info, resp)
	}

	if info.IsStream {
		return GeminiChatStreamHandler(c, info, resp)
	} else {
		return GeminiChatHandler(c, info, resp)
	}

}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
