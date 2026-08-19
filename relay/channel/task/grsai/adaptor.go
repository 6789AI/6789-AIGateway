package grsai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	taskdto "github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	relayhelper "github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const ChannelName = "Grsai"

var ModelList = []string{
	"nano-banana",
	"nano-banana-fast",
	"nano-banana-2",
	"nano-banana-2-cl",
	"nano-banana-2-2k-cl",
	"nano-banana-2-4k-cl",
	"nano-banana-pro",
	"nano-banana-pro-vt",
	"nano-banana-pro-cl",
	"nano-banana-pro-vip",
	"nano-banana-pro-4k-vip",
	"gpt-image-2",
	"gpt-image-2-vip",
}

type generateRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	Images      []string `json:"images"`
	AspectRatio string   `json:"aspectRatio,omitempty"`
	ImageSize   string   `json:"imageSize,omitempty"`
	ReplyType   string   `json:"replyType"`
}

type result struct {
	URL string `json:"url"`
}

type response struct {
	ID       string          `json:"id"`
	Status   string          `json:"status"`
	Progress *int            `json:"progress,omitempty"`
	Results  []result        `json:"results,omitempty"`
	Error    json.RawMessage `json:"error,omitempty"`
}

type TaskAdaptor struct {
	taskcommon.BaseBilling
	apiKey          string
	baseURL         string
	imageRequest    *dto.ImageRequest
	upstreamRequest *generateRequest
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.apiKey = info.ApiKey
	a.baseURL = info.ChannelBaseUrl
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *taskdto.TaskError {
	if info.RelayMode != relayconstant.RelayModeImageSubmit {
		return service.TaskErrorWrapperLocal(fmt.Errorf("Grsai task adaptor only supports asynchronous image generation"), "invalid_relay_mode", http.StatusBadRequest)
	}

	imageRequest, err := relayhelper.GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesGenerations)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_image_request", http.StatusBadRequest)
	}
	if strings.TrimSpace(imageRequest.Prompt) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("prompt is required"), "invalid_image_request", http.StatusBadRequest)
	}
	if imageRequest.Stream != nil && *imageRequest.Stream {
		return service.TaskErrorWrapperLocal(fmt.Errorf("stream and asynchronous image generation cannot be used together"), "invalid_image_request", http.StatusBadRequest)
	}
	if imageRequest.ResponseFormat != "" && !strings.EqualFold(imageRequest.ResponseFormat, "url") {
		return service.TaskErrorWrapperLocal(fmt.Errorf("asynchronous image generation only supports response_format=url"), "invalid_image_request", http.StatusBadRequest)
	}

	a.imageRequest = imageRequest
	info.Request = imageRequest
	info.Action = constant.TaskActionImageGenerate
	return nil
}

func (a *TaskAdaptor) ValidateMappedTaskRequest(_ *gin.Context, info *relaycommon.RelayInfo) *taskdto.TaskError {
	if !constant.UpdateTask {
		return service.TaskErrorWrapperLocal(fmt.Errorf("asynchronous task polling is disabled"), "async_task_polling_disabled", http.StatusServiceUnavailable)
	}
	if a.imageRequest == nil {
		return service.TaskErrorWrapperLocal(fmt.Errorf("image request is not initialized"), "invalid_image_request", http.StatusBadRequest)
	}
	if !model.IsGrsaiImageModel(info.UpstreamModelName) {
		return service.TaskErrorWrapperLocal(fmt.Errorf("model %s is not supported by the Grsai asynchronous image API", info.OriginModelName), "async_image_not_supported", http.StatusBadRequest)
	}
	if a.imageRequest.N != nil && *a.imageRequest.N != 1 {
		return service.TaskErrorWrapperLocal(fmt.Errorf("Grsai asynchronous image generation only supports n=1"), "invalid_image_request", http.StatusBadRequest)
	}

	upstreamRequest, err := convertImageRequest(*a.imageRequest, info.UpstreamModelName)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_image_request", http.StatusBadRequest)
	}
	a.upstreamRequest = upstreamRequest
	return nil
}

func convertImageRequest(request dto.ImageRequest, upstreamModel string) (*generateRequest, error) {
	images, err := parseImages(request.Images)
	if err != nil {
		return nil, fmt.Errorf("images must be a string or an array of strings: %w", err)
	}
	if len(images) == 0 {
		images, err = parseImages(request.Image)
		if err != nil {
			return nil, fmt.Errorf("image must be a string or an array of strings: %w", err)
		}
	}

	aspectRatio, hasAspectRatio, err := extraString(request.Extra, "aspectRatio", "aspect_ratio")
	if err != nil {
		return nil, err
	}
	if !hasAspectRatio {
		aspectRatio = strings.TrimSpace(request.Size)
	}
	if strings.HasPrefix(strings.ToLower(upstreamModel), "nano-banana") {
		aspectRatio, err = normalizeNanoBananaAspectRatio(aspectRatio)
		if err != nil {
			return nil, err
		}
	}

	imageSize := ""
	if strings.HasPrefix(strings.ToLower(upstreamModel), "nano-banana") {
		var hasImageSize bool
		imageSize, hasImageSize, err = extraString(request.Extra, "imageSize", "image_size")
		if err != nil {
			return nil, err
		}
		if !hasImageSize {
			imageSize, err = normalizeImageSize(request.Quality)
			if err != nil {
				return nil, err
			}
		} else {
			imageSize, err = normalizeImageSize(imageSize)
			if err != nil {
				return nil, err
			}
		}
	}

	return &generateRequest{
		Model:       upstreamModel,
		Prompt:      request.Prompt,
		Images:      images,
		AspectRatio: aspectRatio,
		ImageSize:   imageSize,
		ReplyType:   "async",
	}, nil
}

func parseImages(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || common.GetJsonType(raw) == "null" {
		return []string{}, nil
	}
	switch common.GetJsonType(raw) {
	case "string":
		var image string
		if err := common.Unmarshal(raw, &image); err != nil {
			return nil, err
		}
		if strings.TrimSpace(image) == "" {
			return []string{}, nil
		}
		return []string{image}, nil
	case "array":
		var images []string
		if err := common.Unmarshal(raw, &images); err != nil {
			return nil, err
		}
		return images, nil
	default:
		return nil, fmt.Errorf("unexpected JSON type %s", common.GetJsonType(raw))
	}
}

func extraString(extra map[string]json.RawMessage, keys ...string) (string, bool, error) {
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

func normalizeNanoBananaAspectRatio(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "auto") || strings.Contains(value, ":") {
		return value, nil
	}
	mapped := map[string]string{
		"256x256":   "1:1",
		"512x512":   "1:1",
		"1024x1024": "1:1",
		"1280x720":  "16:9",
		"720x1280":  "9:16",
		"1536x1024": "3:2",
		"1024x1536": "2:3",
		"1792x1024": "16:9",
		"1024x1792": "9:16",
	}
	if aspectRatio, ok := mapped[strings.ToLower(value)]; ok {
		return aspectRatio, nil
	}
	return "", fmt.Errorf("unsupported nano-banana size %q; use a supported aspect ratio", value)
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
		return "", fmt.Errorf("imageSize must be one of 1K, 2K, or 4K")
	}
}

func (a *TaskAdaptor) EstimateBilling(_ *gin.Context, _ *relaycommon.RelayInfo) map[string]float64 {
	return map[string]float64{"n": 1}
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return strings.TrimRight(a.baseURL, "/") + "/v1/api/generate", nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(_ *gin.Context, _ *relaycommon.RelayInfo) (io.Reader, error) {
	if a.upstreamRequest == nil {
		return nil, fmt.Errorf("Grsai image request is not initialized")
	}
	body, err := common.Marshal(a.upstreamRequest)
	if err != nil {
		return nil, fmt.Errorf("marshal Grsai image request failed: %w", err)
	}
	return bytes.NewReader(body), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(_ *gin.Context, resp *http.Response, _ *relaycommon.RelayInfo) (string, []byte, *taskdto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	var grsaiResponse response
	if err := common.Unmarshal(responseBody, &grsaiResponse); err != nil {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("unmarshal Grsai response failed: %w", err), "invalid_response", http.StatusBadGateway)
	}
	status := strings.ToLower(strings.TrimSpace(grsaiResponse.Status))
	if status == "" {
		if message := responseErrorMessage(grsaiResponse); message != "" {
			return "", nil, service.TaskErrorWrapper(fmt.Errorf("%s", message), "grsai_api_error", http.StatusBadGateway)
		}
	}
	switch status {
	case "failed", "violation":
		message := responseErrorMessage(grsaiResponse)
		if message == "" {
			message = "Grsai image generation failed"
		}
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("%s", message), "grsai_api_error", http.StatusBadGateway)
	case "queued", "pending", "submitted", "running", "processing", "in_progress":
	case "succeeded", "success", "completed":
		if len(responseURLs(grsaiResponse.Results)) == 0 {
			return "", nil, service.TaskErrorWrapper(fmt.Errorf("Grsai task succeeded without image results"), "invalid_response", http.StatusBadGateway)
		}
	default:
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("unknown Grsai response status %q", grsaiResponse.Status), "invalid_response", http.StatusBadGateway)
	}
	if grsaiResponse.ID == "" {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("Grsai task id is empty"), "invalid_response", http.StatusBadGateway)
	}
	return grsaiResponse.ID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri, err := url.Parse(strings.TrimRight(baseURL, "/") + "/v1/api/result")
	if err != nil {
		return nil, fmt.Errorf("build Grsai result URL failed: %w", err)
	}
	query := uri.Query()
	query.Set("id", taskID)
	uri.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, uri.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(responseBody []byte) (*relaycommon.TaskInfo, error) {
	var grsaiResponse response
	if err := common.Unmarshal(responseBody, &grsaiResponse); err != nil {
		return nil, fmt.Errorf("unmarshal Grsai task result failed: %w", err)
	}

	taskResult := &relaycommon.TaskInfo{TaskID: grsaiResponse.ID}
	if grsaiResponse.Progress != nil {
		progress := min(max(*grsaiResponse.Progress, 0), 100)
		taskResult.Progress = strconv.Itoa(progress) + "%"
	}

	status := strings.ToLower(strings.TrimSpace(grsaiResponse.Status))
	if status == "" {
		if message := responseErrorMessage(grsaiResponse); message != "" {
			taskResult.Status = model.TaskStatusFailure
			taskResult.Reason = message
			return taskResult, nil
		}
	}
	switch status {
	case "queued", "pending", "submitted":
		taskResult.Status = model.TaskStatusQueued
	case "running", "processing", "in_progress":
		taskResult.Status = model.TaskStatusInProgress
	case "succeeded", "success", "completed":
		urls := responseURLs(grsaiResponse.Results)
		if len(urls) == 0 {
			taskResult.Status = model.TaskStatusFailure
			taskResult.Reason = "Grsai task succeeded without image results"
			return taskResult, nil
		}
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Url = urls[0]
		taskResult.BillingRatios = map[string]float64{"n": float64(len(urls))}
	case "failed", "violation":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Reason = responseErrorMessage(grsaiResponse)
		if taskResult.Reason == "" {
			taskResult.Reason = "Grsai image generation " + strings.ToLower(strings.TrimSpace(grsaiResponse.Status))
		}
	default:
		return nil, fmt.Errorf("unknown Grsai task status %q", grsaiResponse.Status)
	}
	return taskResult, nil
}

func responseURLs(results []result) []string {
	urls := make([]string, 0, min(len(results), dto.MaxImageN))
	for _, item := range results {
		if strings.TrimSpace(item.URL) == "" {
			continue
		}
		urls = append(urls, item.URL)
		if len(urls) == dto.MaxImageN {
			break
		}
	}
	return urls
}

func responseErrorMessage(grsaiResponse response) string {
	if len(grsaiResponse.Error) == 0 || common.GetJsonType(grsaiResponse.Error) == "null" {
		return ""
	}
	if common.GetJsonType(grsaiResponse.Error) == "string" {
		var message string
		if err := common.Unmarshal(grsaiResponse.Error, &message); err == nil {
			return message
		}
	}
	var payload struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := common.Unmarshal(grsaiResponse.Error, &payload); err == nil {
		message := payload.Message
		if message == "" {
			message = payload.Error
		}
		if payload.Code != "" && message != "" {
			return payload.Code + ": " + message
		}
		if message != "" {
			return message
		}
		return payload.Code
	}
	return ""
}

func (a *TaskAdaptor) ConvertToOpenAIImageTask(task *model.Task) ([]byte, error) {
	imageTask := dto.NewImageTaskResponse(task.TaskID)
	imageTask.Model = task.Properties.OriginModelName
	imageTask.CreatedAt = task.CreatedAt
	if imageTask.CreatedAt == 0 {
		imageTask.CreatedAt = task.SubmitTime
	}
	if progress, err := strconv.Atoi(strings.TrimSuffix(task.Progress, "%")); err == nil {
		imageTask.Progress = min(max(progress, 0), 100)
	}

	switch task.Status {
	case model.TaskStatusNotStart, model.TaskStatusSubmitted, model.TaskStatusQueued:
		imageTask.Status = dto.ImageTaskStatusQueued
	case model.TaskStatusInProgress:
		imageTask.Status = dto.ImageTaskStatusInProgress
	case model.TaskStatusSuccess:
		imageTask.Status = dto.ImageTaskStatusCompleted
		imageTask.Progress = 100
		imageTask.CompletedAt = task.FinishTime
		if imageTask.CompletedAt == 0 {
			imageTask.CompletedAt = task.UpdatedAt
		}
	case model.TaskStatusFailure:
		imageTask.Status = dto.ImageTaskStatusFailed
		imageTask.Progress = 100
		imageTask.CompletedAt = task.FinishTime
		if imageTask.CompletedAt == 0 {
			imageTask.CompletedAt = task.UpdatedAt
		}
	default:
		imageTask.Status = dto.ImageTaskStatusQueued
	}

	var grsaiResponse response
	if len(task.Data) > 0 {
		if err := common.Unmarshal(task.Data, &grsaiResponse); err != nil {
			return nil, fmt.Errorf("unmarshal Grsai image task response failed: %w", err)
		}
	}
	if task.Status == model.TaskStatusSuccess {
		urls := responseURLs(grsaiResponse.Results)
		imageTask.Data = make([]dto.ImageData, 0, len(urls))
		for _, imageURL := range urls {
			imageTask.Data = append(imageTask.Data, dto.ImageData{Url: imageURL})
		}
	}
	if task.Status == model.TaskStatusFailure {
		message := task.FailReason
		if message == "" {
			message = responseErrorMessage(grsaiResponse)
		}
		if message == "" {
			message = "Grsai image generation failed"
		}
		code := strings.ToLower(strings.TrimSpace(grsaiResponse.Status))
		imageTask.Error = &dto.ImageTaskError{Code: code, Message: message}
	}

	return common.Marshal(imageTask)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}
