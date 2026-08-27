package newapi

import (
	"bytes"
	"encoding/json"
	"errors"
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
	"github.com/QuantumNous/new-api/relay/channel/advancedcustom"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	relayhelper "github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const (
	ChannelName = "New API async image"
	imagePath   = "/v1/images/generations"
)

type TaskAdaptor struct {
	taskcommon.BaseBilling
	apiKey       string
	baseURL      string
	channelType  int
	imageRequest *dto.ImageRequest
	submitURL    string
	submitHeader http.Header
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.apiKey = info.ApiKey
	a.baseURL = info.ChannelBaseUrl
	a.channelType = info.ChannelType
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *taskdto.TaskError {
	if info.RelayMode != relayconstant.RelayModeImageSubmit {
		return service.TaskErrorWrapperLocal(errors.New("New API async image adaptor only supports asynchronous image generation"), "invalid_relay_mode", http.StatusBadRequest)
	}
	imageRequest, err := relayhelper.GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesGenerations)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_image_request", http.StatusBadRequest)
	}
	if imageRequest.Stream != nil && *imageRequest.Stream {
		return service.TaskErrorWrapperLocal(errors.New("stream and asynchronous image generation cannot be used together"), "invalid_image_request", http.StatusBadRequest)
	}
	if imageRequest.ResponseFormat != "" && !strings.EqualFold(imageRequest.ResponseFormat, "url") {
		return service.TaskErrorWrapperLocal(errors.New("asynchronous image generation only supports response_format=url"), "invalid_image_request", http.StatusBadRequest)
	}
	a.imageRequest = imageRequest
	info.Request = imageRequest
	info.Action = constant.TaskActionImageGenerate
	return nil
}

func (a *TaskAdaptor) ValidateMappedTaskRequest(_ *gin.Context, _ *relaycommon.RelayInfo) *taskdto.TaskError {
	if !constant.UpdateTask {
		return service.TaskErrorWrapperLocal(errors.New("asynchronous task polling is disabled"), "async_task_polling_disabled", http.StatusServiceUnavailable)
	}
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseURL := strings.TrimSpace(a.baseURL)
	if baseURL == "" {
		return "", errors.New("channel base URL is required for New API asynchronous image generation")
	}

	switch a.channelType {
	case constant.ChannelTypeCustom:
		a.submitURL = baseURL
		a.submitHeader = nil
	case constant.ChannelTypeAdvancedCustom:
		requestURL, header, err := advancedcustom.ResolveOpenAICompatibleRoute(info, imagePath)
		if err != nil {
			return "", err
		}
		a.submitURL = requestURL
		a.submitHeader = header
	default:
		a.submitURL = strings.TrimRight(baseURL, "/") + imagePath
		a.submitHeader = nil
	}
	parsed, err := url.Parse(a.submitURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid New API asynchronous image URL %q", a.submitURL)
	}
	return a.submitURL, nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, &req.Header)
	for name, values := range a.submitHeader {
		req.Header.Del(name)
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	if a.submitHeader == nil && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	a.submitHeader = req.Header.Clone()
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	if a.imageRequest == nil {
		return nil, errors.New("image request is not initialized")
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, err
	}
	body, err := storage.Bytes()
	if err != nil {
		return nil, err
	}
	payload := make(map[string]json.RawMessage)
	if err := common.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode asynchronous image request: %w", err)
	}
	modelName := info.UpstreamModelName
	if modelName == "" {
		modelName = info.OriginModelName
	}
	modelJSON, err := common.Marshal(modelName)
	if err != nil {
		return nil, err
	}
	asyncJSON, err := common.Marshal(true)
	if err != nil {
		return nil, err
	}
	payload["model"] = modelJSON
	payload["async"] = asyncJSON
	delete(payload, "stream")
	requestBody, err := common.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode asynchronous image request: %w", err)
	}
	return bytes.NewReader(requestBody), nil
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
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(string(responseBody))
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		return "", nil, service.TaskErrorWrapper(errors.New(message), "new_api_async_image_error", resp.StatusCode)
	}
	var upstream dto.ImageTaskResponse
	if err := common.Unmarshal(responseBody, &upstream); err != nil {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("decode New API image task response: %w", err), "invalid_response", http.StatusBadGateway)
	}
	taskID := strings.TrimSpace(upstream.TaskID)
	if taskID == "" {
		taskID = strings.TrimSpace(upstream.ID)
	}
	if taskID == "" {
		return "", nil, service.TaskErrorWrapper(errors.New("task_id is empty"), "invalid_response", http.StatusBadGateway)
	}
	return taskID, responseBody, nil
}

func (a *TaskAdaptor) TaskPollingConfig(upstreamTaskID string) (*model.TaskPollingConfig, error) {
	parsed, err := url.Parse(a.submitURL)
	if err != nil {
		return nil, err
	}
	basePath := strings.TrimRight(parsed.Path, "/")
	baseEscapedPath := strings.TrimRight(parsed.EscapedPath(), "/")
	parsed.Path = basePath + "/" + upstreamTaskID
	parsed.RawPath = baseEscapedPath + "/" + url.PathEscape(upstreamTaskID)
	header := a.submitHeader.Clone()
	header.Del("Content-Length")
	header.Del("Content-Type")
	header.Del("Prefer")
	return &model.TaskPollingConfig{URL: parsed.String(), Headers: header}, nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, errors.New("invalid task_id")
	}
	pollURL := strings.TrimRight(baseURL, "/") + imagePath + "/" + url.PathEscape(taskID)
	header := http.Header{}
	if config, ok := body["polling_config"].(*model.TaskPollingConfig); ok && config != nil && config.URL != "" {
		pollURL = config.URL
		for name, values := range config.Headers {
			for _, value := range values {
				header.Add(name, value)
			}
		}
	}
	if header.Get("Authorization") == "" {
		header.Set("Authorization", "Bearer "+key)
	}
	header.Set("Accept", "application/json")
	req, err := http.NewRequest(http.MethodGet, pollURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header = header
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(responseBody []byte) (*relaycommon.TaskInfo, error) {
	var response dto.ImageTaskResponse
	if err := common.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("decode New API image task result: %w", err)
	}
	taskID := strings.TrimSpace(response.TaskID)
	if taskID == "" {
		taskID = strings.TrimSpace(response.ID)
	}
	result := &relaycommon.TaskInfo{TaskID: taskID}
	if response.Progress > 0 {
		result.Progress = strconv.Itoa(min(response.Progress, 100)) + "%"
	}
	switch strings.ToLower(strings.TrimSpace(response.Status)) {
	case dto.ImageTaskStatusQueued:
		result.Status = model.TaskStatusQueued
	case dto.ImageTaskStatusInProgress:
		result.Status = model.TaskStatusInProgress
	case dto.ImageTaskStatusCompleted:
		result.Status = model.TaskStatusSuccess
		imageCount := 0
		for _, image := range response.Data {
			if strings.TrimSpace(image.Url) == "" {
				continue
			}
			if result.Url == "" {
				result.Url = image.Url
			}
			imageCount++
			if imageCount == dto.MaxImageN {
				break
			}
		}
		if imageCount == 0 {
			result.Status = model.TaskStatusFailure
			result.Reason = "New API image task completed without image results"
		} else {
			result.BillingRatios = map[string]float64{"n": float64(imageCount)}
		}
	case dto.ImageTaskStatusFailed:
		result.Status = model.TaskStatusFailure
		if response.Error != nil {
			result.Reason = response.Error.Message
		}
		if result.Reason == "" {
			result.Reason = "New API image generation failed"
		}
	default:
		return nil, fmt.Errorf("unknown New API image task status %q", response.Status)
	}
	return result, nil
}

func (a *TaskAdaptor) ConvertToOpenAIImageTask(task *model.Task) ([]byte, error) {
	response := dto.NewImageTaskResponse(task.TaskID)
	if len(task.Data) > 0 {
		if err := common.Unmarshal(task.Data, response); err != nil {
			return nil, fmt.Errorf("decode stored New API image task response: %w", err)
		}
	}
	response.ID = task.TaskID
	response.TaskID = task.TaskID
	response.Object = "image.generation.task"
	response.Model = task.Properties.OriginModelName
	response.CreatedAt = task.CreatedAt
	if response.CreatedAt == 0 {
		response.CreatedAt = task.SubmitTime
	}
	if progress, err := strconv.Atoi(strings.TrimSuffix(task.Progress, "%")); err == nil {
		response.Progress = min(max(progress, 0), 100)
	}
	switch task.Status {
	case model.TaskStatusInProgress:
		response.Status = dto.ImageTaskStatusInProgress
	case model.TaskStatusSuccess:
		response.Status = dto.ImageTaskStatusCompleted
		response.Progress = 100
		response.CompletedAt = task.FinishTime
		if len(response.Data) > dto.MaxImageN {
			response.Data = response.Data[:dto.MaxImageN]
		}
	case model.TaskStatusFailure:
		response.Status = dto.ImageTaskStatusFailed
		response.Progress = 100
		response.CompletedAt = task.FinishTime
		if response.Error == nil {
			response.Error = &dto.ImageTaskError{Message: task.FailReason}
		}
	default:
		response.Status = dto.ImageTaskStatusQueued
	}
	return common.Marshal(response)
}

func (a *TaskAdaptor) GetModelList() []string {
	return nil
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}
