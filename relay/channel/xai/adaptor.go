package xai

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/QuantumNous/new-api/relay/constant"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type Adaptor struct {
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(*gin.Context, *relaycommon.RelayInfo, *dto.ClaudeRequest) (any, error) {
	//TODO implement me
	//panic("implement me")
	return nil, errors.New("not available")
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	//not available
	return nil, errors.New("not available")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	aspectRatio, resolution, err := xAIImageOptions(request)
	if err != nil {
		return nil, err
	}

	var image json.RawMessage
	if info != nil && info.RelayMode == constant.RelayModeImagesEdits {
		image, err = xAIEditImage(c, request)
		if err != nil {
			return nil, err
		}
	}

	xaiRequest := ImageRequest{
		Model:          request.Model,
		Prompt:         request.Prompt,
		N:              int(lo.FromPtrOr(request.N, uint(1))),
		AspectRatio:    aspectRatio,
		Resolution:     resolution,
		ResponseFormat: request.ResponseFormat,
		Image:          image,
	}
	return xaiRequest, nil
}

func xAIImageOptions(request dto.ImageRequest) (*string, *string, error) {
	aspectRatio, err := xAIImageOption(request.Extra, "aspect_ratio", "aspectRatio")
	if err != nil {
		return nil, nil, err
	}
	resolution, err := xAIImageOption(request.Extra, "resolution")
	if err != nil {
		return nil, nil, err
	}

	if aspectRatio == nil || resolution == nil {
		extraBody, err := xAIExtraBody(request.Extra)
		if err != nil {
			return nil, nil, err
		}
		if aspectRatio == nil {
			aspectRatio, err = xAIImageOption(extraBody, "aspect_ratio", "aspectRatio")
			if err != nil {
				return nil, nil, err
			}
		}
		if resolution == nil {
			resolution, err = xAIImageOption(extraBody, "resolution")
			if err != nil {
				return nil, nil, err
			}
		}
	}

	if request.Size != "" && aspectRatio == nil {
		derivedAspectRatio, derivedResolution, err := xAIImageOptionsFromSize(request.Size)
		if err != nil {
			return nil, nil, err
		}
		if derivedAspectRatio != "" {
			aspectRatio = &derivedAspectRatio
		}
		if resolution == nil && derivedResolution != "" {
			resolution = &derivedResolution
		}
	}

	if resolution != nil {
		normalizedResolution, err := normalizeXAIResolution(*resolution)
		if err != nil {
			return nil, nil, err
		}
		if normalizedResolution == "" {
			resolution = nil
		} else {
			resolution = &normalizedResolution
		}
	}
	return aspectRatio, resolution, nil
}

func xAIExtraBody(extra map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	raw, ok := extra["extra_body"]
	if !ok || common.GetJsonType(raw) == "null" {
		return nil, nil
	}
	var body map[string]json.RawMessage
	if err := common.Unmarshal(raw, &body); err != nil {
		return nil, fmt.Errorf("invalid xAI extra_body: %w", err)
	}
	return body, nil
}

func xAIImageOption(extra map[string]json.RawMessage, keys ...string) (*string, error) {
	for _, key := range keys {
		raw, ok := extra[key]
		if !ok {
			continue
		}
		if common.GetJsonType(raw) == "null" {
			continue
		}

		var value string
		if err := common.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("invalid xAI image option %q: %w", key, err)
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		return &value, nil
	}
	return nil, nil
}

func normalizeXAIResolution(value string) (string, error) {
	resolution := strings.ToLower(strings.TrimSpace(value))
	switch resolution {
	case "":
		return "", nil
	case "1k", "2k":
		return resolution, nil
	default:
		return "", fmt.Errorf("invalid xAI image resolution %q; use 1k or 2k", value)
	}
}

func xAIImageOptionsFromSize(size string) (string, string, error) {
	size = strings.TrimSpace(strings.ToLower(size))
	if size == "" || size == "auto" {
		return "", "", nil
	}
	if strings.Contains(size, ":") {
		parts := strings.Split(size, ":")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid xAI image size %q", size)
		}
		width, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32)
		if err != nil || width == 0 {
			return "", "", fmt.Errorf("invalid xAI image size %q", size)
		}
		height, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 32)
		if err != nil || height == 0 {
			return "", "", fmt.Errorf("invalid xAI image size %q", size)
		}
		return fmt.Sprintf("%d:%d", width/gcdUint32(width, height), height/gcdUint32(width, height)), "", nil
	}

	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid xAI image size %q", size)
	}
	width, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32)
	if err != nil || width == 0 {
		return "", "", fmt.Errorf("invalid xAI image size %q", size)
	}
	height, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 32)
	if err != nil || height == 0 {
		return "", "", fmt.Errorf("invalid xAI image size %q", size)
	}
	resolution := "1k"
	if width > 1024 || height > 1024 {
		resolution = "2k"
	}
	return fmt.Sprintf("%d:%d", width/gcdUint32(width, height), height/gcdUint32(width, height)), resolution, nil
}

func gcdUint32(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func xAIEditImage(c *gin.Context, request dto.ImageRequest) (json.RawMessage, error) {
	if len(request.Image) > 0 && common.GetJsonType(request.Image) != "null" {
		return normalizeXAIImage(request.Image)
	}
	if len(request.Images) > 0 && common.GetJsonType(request.Images) != "null" {
		var images []json.RawMessage
		if err := common.Unmarshal(request.Images, &images); err != nil {
			return nil, fmt.Errorf("invalid xAI images: %w", err)
		}
		if len(images) > 0 {
			return normalizeXAIImage(images[0])
		}
	}

	if c == nil || c.Request == nil || c.Request.MultipartForm == nil {
		return nil, errors.New("image is required for xAI image edits")
	}
	multipartForm := c.Request.MultipartForm
	if values := multipartForm.Value["image"]; len(values) > 0 && strings.TrimSpace(values[0]) != "" {
		raw, err := common.Marshal(values[0])
		if err != nil {
			return nil, fmt.Errorf("failed to encode xAI image URL: %w", err)
		}
		return normalizeXAIImage(raw)
	}
	files := multipartForm.File["image"]
	if len(files) == 0 {
		files = multipartForm.File["image[]"]
	}
	if len(files) == 0 {
		return nil, errors.New("image is required for xAI image edits")
	}
	file, err := files[0].Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open xAI image: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read xAI image: %w", err)
	}
	mimeType := http.DetectContentType(data)
	encoded := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
	raw, err := common.Marshal(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to encode xAI image: %w", err)
	}
	return normalizeXAIImage(raw)
}

func normalizeXAIImage(raw json.RawMessage) (json.RawMessage, error) {
	if common.GetJsonType(raw) == "string" {
		var value string
		if err := common.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("invalid xAI image: %w", err)
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, errors.New("image is required for xAI image edits")
		}
		return common.Marshal(map[string]string{"url": value, "type": "image_url"})
	}
	if common.GetJsonType(raw) != "object" {
		return nil, errors.New("xAI image must be an object or URL string")
	}
	return raw, nil
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return relaycommon.GetFullRequestURL(info.ChannelBaseUrl, info.RequestURLPath, info.ChannelType), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("Authorization", "Bearer "+info.ApiKey)
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	if strings.HasSuffix(info.UpstreamModelName, "-search") {
		info.UpstreamModelName = strings.TrimSuffix(info.UpstreamModelName, "-search")
		request.Model = info.UpstreamModelName
		toMap := request.ToMap()
		toMap["search_parameters"] = map[string]any{
			"mode": "on",
		}
		return toMap, nil
	}
	if strings.HasPrefix(request.Model, "grok-3-mini") {
		if lo.FromPtrOr(request.MaxCompletionTokens, uint(0)) == 0 && lo.FromPtrOr(request.MaxTokens, uint(0)) != 0 {
			request.MaxCompletionTokens = request.MaxTokens
			request.MaxTokens = nil
		}
		if strings.HasSuffix(request.Model, "-high") {
			request.ReasoningEffort = "high"
			request.Model = strings.TrimSuffix(request.Model, "-high")
		} else if strings.HasSuffix(request.Model, "-low") {
			request.ReasoningEffort = "low"
			request.Model = strings.TrimSuffix(request.Model, "-low")
		}
		info.ReasoningEffort = request.ReasoningEffort
		info.UpstreamModelName = request.Model
	}
	return request, nil
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	//not available
	return nil, errors.New("not available")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	if request.Model == "" && info != nil {
		request.Model = info.UpstreamModelName
	}
	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	switch info.RelayMode {
	case constant.RelayModeImagesGenerations, constant.RelayModeImagesEdits:
		usage, err = openai.OpenaiImageHandler(c, info, resp)
	case constant.RelayModeResponses:
		if info.IsStream {
			usage, err = openai.OaiResponsesStreamHandler(c, info, resp)
		} else {
			usage, err = openai.OaiResponsesHandler(c, info, resp)
		}
	default:
		if info.IsStream {
			usage, err = xAIStreamHandler(c, info, resp)
		} else {
			usage, err = xAIHandler(c, info, resp)
		}
	}
	return
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
