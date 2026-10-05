package xai

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
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
	var images json.RawMessage
	if info != nil && info.RelayMode == constant.RelayModeImagesEdits {
		image, images, err = xAIEditImages(c, request)
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
		Images:         images,
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

func xAIEditImages(c *gin.Context, request dto.ImageRequest) (json.RawMessage, json.RawMessage, error) {
	hasImage := len(request.Image) > 0 && common.GetJsonType(request.Image) != "null"
	hasImages := len(request.Images) > 0 && common.GetJsonType(request.Images) != "null"
	if hasImage && hasImages {
		return nil, nil, errors.New("provide either image or images, but not both")
	}
	if hasImage {
		image, err := normalizeXAIImage(request.Image)
		return image, nil, err
	}
	if hasImages {
		var imageList []json.RawMessage
		if err := common.Unmarshal(request.Images, &imageList); err != nil {
			return nil, nil, fmt.Errorf("invalid xAI images: %w", err)
		}
		return normalizeXAIImageList(imageList)
	}

	if c == nil || c.Request == nil || c.Request.MultipartForm == nil {
		return nil, nil, errors.New("image is required for xAI image edits")
	}
	multipartForm := c.Request.MultipartForm
	rawImages := make([]json.RawMessage, 0)
	fieldNames := make([]string, 0, len(multipartForm.Value)+len(multipartForm.File))
	seenFields := make(map[string]struct{}, len(multipartForm.Value)+len(multipartForm.File))
	for fieldName := range multipartForm.Value {
		if isXAIImageMultipartField(fieldName) {
			seenFields[fieldName] = struct{}{}
			fieldNames = append(fieldNames, fieldName)
		}
	}
	for fieldName := range multipartForm.File {
		if isXAIImageMultipartField(fieldName) {
			if _, ok := seenFields[fieldName]; !ok {
				fieldNames = append(fieldNames, fieldName)
			}
		}
	}
	sort.Slice(fieldNames, func(i, j int) bool {
		return xAIImageMultipartFieldOrder(fieldNames[i]) < xAIImageMultipartFieldOrder(fieldNames[j])
	})
	for _, fieldName := range fieldNames {
		for _, value := range multipartForm.Value[fieldName] {
			if strings.TrimSpace(value) == "" {
				continue
			}
			raw, err := common.Marshal(value)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to encode xAI image URL: %w", err)
			}
			rawImages = append(rawImages, raw)
		}
		for _, fileHeader := range multipartForm.File[fieldName] {
			file, err := fileHeader.Open()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to open xAI image: %w", err)
			}
			data, readErr := io.ReadAll(file)
			closeErr := file.Close()
			if readErr != nil {
				return nil, nil, fmt.Errorf("failed to read xAI image: %w", readErr)
			}
			if closeErr != nil {
				return nil, nil, fmt.Errorf("failed to close xAI image: %w", closeErr)
			}
			mimeType := http.DetectContentType(data)
			encoded := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
			raw, err := common.Marshal(encoded)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to encode xAI image: %w", err)
			}
			rawImages = append(rawImages, raw)
		}
	}
	return normalizeXAIImageList(rawImages)
}

func isXAIImageMultipartField(fieldName string) bool {
	if fieldName == "image" || fieldName == "image[]" {
		return true
	}
	if !strings.HasPrefix(fieldName, "image[") || !strings.HasSuffix(fieldName, "]") {
		return false
	}
	index := strings.TrimSuffix(strings.TrimPrefix(fieldName, "image["), "]")
	if index == "" {
		return false
	}
	_, err := strconv.ParseUint(index, 10, 32)
	return err == nil
}

func xAIImageMultipartFieldOrder(fieldName string) uint64 {
	switch fieldName {
	case "image":
		return 0
	case "image[]":
		return 1
	default:
		index := strings.TrimSuffix(strings.TrimPrefix(fieldName, "image["), "]")
		parsed, err := strconv.ParseUint(index, 10, 32)
		if err != nil {
			return ^uint64(0)
		}
		return parsed + 2
	}
}

func normalizeXAIImageList(rawImages []json.RawMessage) (json.RawMessage, json.RawMessage, error) {
	if len(rawImages) == 0 {
		return nil, nil, errors.New("image is required for xAI image edits")
	}
	normalized := make([]json.RawMessage, 0, len(rawImages))
	for index, raw := range rawImages {
		image, err := normalizeXAIImage(raw)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid xAI image %d: %w", index+1, err)
		}
		normalized = append(normalized, image)
	}
	if len(normalized) == 1 {
		return normalized[0], nil, nil
	}
	images, err := common.Marshal(normalized)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encode xAI images: %w", err)
	}
	return nil, images, nil
}

func normalizeXAIImage(raw json.RawMessage) (json.RawMessage, error) {
	switch common.GetJsonType(raw) {
	case "string":
		return normalizeXAIImageReference("url", raw)
	case "object":
		var image map[string]json.RawMessage
		if err := common.Unmarshal(raw, &image); err != nil {
			return nil, fmt.Errorf("invalid xAI image: %w", err)
		}
		url, hasURL := image["url"]
		fileID, hasFileID := image["file_id"]
		if hasURL && hasFileID {
			return nil, errors.New("xAI image must contain either url or file_id, not both")
		}
		if hasURL {
			return normalizeXAIImageReference("url", url)
		}
		if hasFileID {
			return normalizeXAIImageReference("file_id", fileID)
		}
		if imageURL, ok := image["image_url"]; ok {
			switch common.GetJsonType(imageURL) {
			case "string":
				return normalizeXAIImageReference("url", imageURL)
			case "object":
				var imageURLObject map[string]json.RawMessage
				if err := common.Unmarshal(imageURL, &imageURLObject); err != nil {
					return nil, fmt.Errorf("invalid xAI image_url: %w", err)
				}
				if imageURLValue, ok := imageURLObject["url"]; ok {
					return normalizeXAIImageReference("url", imageURLValue)
				}
			}
		}
		return nil, errors.New("xAI image must contain url or file_id")
	default:
		return nil, errors.New("xAI image must be an object or URL string")
	}
}

func normalizeXAIImageReference(key string, raw json.RawMessage) (json.RawMessage, error) {
	if common.GetJsonType(raw) != "string" {
		return nil, fmt.Errorf("xAI image %s must be a string", key)
	}
	var value string
	if err := common.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("invalid xAI image %s: %w", key, err)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("image is required for xAI image edits")
	}
	return common.Marshal(map[string]string{key: value})
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
