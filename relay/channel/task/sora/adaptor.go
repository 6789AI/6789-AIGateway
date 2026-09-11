package sora

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relaykitdto "github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/tidwall/sjson"
)

// ============================
// Request / Response structures
// ============================

type ContentItem struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // for text type
	ImageURL *ImageURL `json:"image_url,omitempty"` // for image_url type
}

type ImageURL struct {
	URL string `json:"url"`
}

type responseTask struct {
	ID                 string `json:"id"`
	TaskID             string `json:"task_id,omitempty"` //兼容旧接口
	Object             string `json:"object"`
	Model              string `json:"model"`
	Status             string `json:"status"`
	Progress           int    `json:"progress"`
	CreatedAt          int64  `json:"created_at"`
	CompletedAt        int64  `json:"completed_at,omitempty"`
	ExpiresAt          int64  `json:"expires_at,omitempty"`
	Seconds            string `json:"seconds,omitempty"`
	Size               string `json:"size,omitempty"`
	RemixedFromVideoID string `json:"remixed_from_video_id,omitempty"`
	Error              *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

var vintedVideoDurations = map[string]map[int]bool{
	"seedance2.0":     {5: true, 10: true, 15: true},
	"seedance2.0fast": {5: true, 10: true, 15: true},
	"seedance2.5":     {30: true},
	"seedance2.0mini": {5: true, 10: true},
}

var vintedVideoRatios = map[string]bool{
	"": true, "1:1": true, "3:4": true, "4:3": true,
	"9:16": true, "16:9": true, "21:9": true,
}

var vintedImageIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)

type vintedVideoRequest struct {
	Prompt         string   `json:"prompt"`
	Model          string   `json:"model,omitempty"`
	Duration       *int     `json:"duration,omitempty"`
	Ratio          *string  `json:"ratio,omitempty"`
	Resolution     *string  `json:"resolution,omitempty"`
	CameraMovement *string  `json:"camera_movement,omitempty"`
	ImageIDs       []string `json:"image_ids,omitempty"`
	Image          string   `json:"image,omitempty"`
	Images         []string `json:"images,omitempty"`
	ImageRefs      []string `json:"image_refs,omitempty"`
	ImageURL       string   `json:"image_url,omitempty"`
	ImageURLs      []string `json:"image_urls,omitempty"`
}

type vintedUpstreamRequest struct {
	Prompt         string   `json:"prompt"`
	Model          string   `json:"model"`
	Duration       int      `json:"duration"`
	Ratio          string   `json:"ratio"`
	Resolution     string   `json:"resolution"`
	CameraMovement string   `json:"camera_movement,omitempty"`
	ImageIDs       []string `json:"image_ids,omitempty"`
	ImageURLs      []string `json:"image_urls,omitempty"`
}

type vintedAcceptedResponse struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Idempotent string `json:"idempotent,omitempty"`
}

type vintedResponseTask struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	File        string `json:"file"`
	DownloadURL string `json:"download_url"`
	Error       string `json:"error"`
}

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
	isVinted    bool
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
	a.isVinted = model.IsVintedVideoChannel(info.ChannelType, info.ChannelBaseUrl, info.ChannelOtherSettings)
}

func collectVintedReferences(req vintedVideoRequest) ([]string, []string, error) {
	imageIDs := make([]string, 0, len(req.ImageIDs))
	seenIDs := make(map[string]struct{}, len(req.ImageIDs))
	for _, imageID := range req.ImageIDs {
		imageID = strings.TrimSpace(imageID)
		if !vintedImageIDPattern.MatchString(imageID) {
			return nil, nil, fmt.Errorf("image_ids contains an invalid value")
		}
		if _, exists := seenIDs[imageID]; exists {
			continue
		}
		seenIDs[imageID] = struct{}{}
		imageIDs = append(imageIDs, imageID)
	}

	for _, alias := range [][]string{req.Images, req.ImageRefs, req.ImageURLs} {
		for _, imageURL := range alias {
			if strings.TrimSpace(imageURL) == "" {
				return nil, nil, fmt.Errorf("image URL lists must not contain empty values")
			}
		}
	}
	urlAliases := make([]string, 0, 2+len(req.Images)+len(req.ImageRefs)+len(req.ImageURLs))
	if strings.TrimSpace(req.Image) != "" {
		urlAliases = append(urlAliases, req.Image)
	}
	if strings.TrimSpace(req.ImageURL) != "" {
		urlAliases = append(urlAliases, req.ImageURL)
	}
	urlAliases = append(urlAliases, req.Images...)
	urlAliases = append(urlAliases, req.ImageRefs...)
	urlAliases = append(urlAliases, req.ImageURLs...)
	imageURLs := make([]string, 0, len(urlAliases))
	seenURLs := make(map[string]struct{}, len(urlAliases))
	for _, imageURL := range urlAliases {
		imageURL = strings.TrimSpace(imageURL)
		if imageURL == "" {
			continue
		}
		if utf8.RuneCountInString(imageURL) > 2048 {
			return nil, nil, fmt.Errorf("image URL must not exceed 2048 characters")
		}
		if _, exists := seenURLs[imageURL]; exists {
			continue
		}
		seenURLs[imageURL] = struct{}{}
		imageURLs = append(imageURLs, imageURL)
	}
	return imageIDs, imageURLs, nil
}

func validateVintedRequestShape(req vintedVideoRequest) error {
	if strings.TrimSpace(req.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	if utf8.RuneCountInString(req.Prompt) > 6000 {
		return fmt.Errorf("prompt must not exceed 6000 characters")
	}
	cameraMovement := ""
	if req.CameraMovement != nil {
		cameraMovement = strings.TrimSpace(*req.CameraMovement)
	}
	if cameraMovement != "" && cameraMovement != "auto" && cameraMovement != "fixed" {
		return fmt.Errorf("unsupported Vinted camera movement: %s", cameraMovement)
	}
	_, _, err := collectVintedReferences(req)
	return err
}

func normalizeVintedRequest(req vintedVideoRequest, modelName string) (vintedUpstreamRequest, error) {
	if err := validateVintedRequestShape(req); err != nil {
		return vintedUpstreamRequest{}, err
	}

	modelName = strings.TrimSpace(modelName)
	allowedDurations, ok := vintedVideoDurations[modelName]
	if !ok {
		return vintedUpstreamRequest{}, fmt.Errorf("unsupported Vinted video model: %s", modelName)
	}

	duration := 5
	if req.Duration != nil {
		duration = *req.Duration
	}
	if !allowedDurations[duration] {
		return vintedUpstreamRequest{}, fmt.Errorf("duration %d is not supported for model %s", duration, modelName)
	}

	ratio := ""
	if req.Ratio != nil {
		ratio = strings.TrimSpace(*req.Ratio)
	}
	if !vintedVideoRatios[ratio] {
		return vintedUpstreamRequest{}, fmt.Errorf("unsupported Vinted video ratio: %s", ratio)
	}

	resolution := "720p"
	if req.Resolution != nil {
		resolution = strings.TrimSpace(*req.Resolution)
	}
	if resolution != "720p" {
		return vintedUpstreamRequest{}, fmt.Errorf("unsupported Vinted video resolution: %s", resolution)
	}
	cameraMovement := ""
	if req.CameraMovement != nil {
		cameraMovement = strings.TrimSpace(*req.CameraMovement)
	}
	imageIDs, imageURLs, err := collectVintedReferences(req)
	if err != nil {
		return vintedUpstreamRequest{}, err
	}

	maxImages := 9
	if modelName == "seedance2.5" {
		maxImages = 30
	}
	if len(imageIDs)+len(imageURLs) > maxImages {
		return vintedUpstreamRequest{}, fmt.Errorf("model %s supports at most %d reference images", modelName, maxImages)
	}

	return vintedUpstreamRequest{
		Prompt:         req.Prompt,
		Model:          modelName,
		Duration:       duration,
		Ratio:          ratio,
		Resolution:     resolution,
		CameraMovement: cameraMovement,
		ImageIDs:       imageIDs,
		ImageURLs:      imageURLs,
	}, nil
}

func validateRemixRequest(c *gin.Context) *dto.TaskError {
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("field prompt is required"), "invalid_request", http.StatusBadRequest)
	}
	// 存储原始请求到 context，与 ValidateMultipartDirect 路径保持一致
	c.Set("task_request", req)
	return nil
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	if a.isVinted {
		if info.Action == constant.TaskActionRemix {
			return service.TaskErrorWrapperLocal(errors.New("Vinted video protocol does not support remix"), "invalid_request", http.StatusBadRequest)
		}
		var req vintedVideoRequest
		if err := common.UnmarshalBodyReusable(c, &req); err != nil {
			return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
		}
		if err := validateVintedRequestShape(req); err != nil {
			return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
		}
		if utf8.RuneCountInString(c.GetHeader("Idempotency-Key")) > 200 {
			return service.TaskErrorWrapperLocal(fmt.Errorf("Idempotency-Key must not exceed 200 characters"), "invalid_request", http.StatusBadRequest)
		}

		imageIDs, imageURLs, err := collectVintedReferences(req)
		if err != nil {
			return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
		}
		duration := 5
		if req.Duration != nil {
			duration = *req.Duration
		}
		info.Action = constant.TaskActionTextGenerate
		if len(imageIDs)+len(imageURLs) > 0 {
			info.Action = constant.TaskActionGenerate
		}
		c.Set("task_request", relaycommon.TaskSubmitReq{
			Prompt:   req.Prompt,
			Model:    req.Model,
			Images:   imageURLs,
			Duration: duration,
		})
		return nil
	}
	if info.Action == constant.TaskActionRemix {
		return validateRemixRequest(c)
	}
	return relaycommon.ValidateMultipartDirect(c, info)
}

func (a *TaskAdaptor) ValidateMappedTaskRequest(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if !a.isVinted {
		return nil
	}
	var req vintedVideoRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if _, err := normalizeVintedRequest(req, info.UpstreamModelName); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	return nil
}

// EstimateBilling 根据用户请求的 seconds 和 size 计算 OtherRatios。
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	// remix 路径的 OtherRatios 已在 ResolveOriginTask 中设置
	if info.Action == constant.TaskActionRemix {
		return nil
	}

	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}

	seconds, _ := strconv.Atoi(req.Seconds)
	if seconds == 0 {
		seconds = req.Duration
	}
	if seconds <= 0 {
		if a.isVinted {
			seconds = 5
		} else {
			seconds = 4
		}
	}

	size := req.Size
	if size == "" {
		size = "720x1280"
	}

	ratios := map[string]float64{
		"seconds": float64(seconds),
		"size":    1,
	}
	if size == "1792x1024" || size == "1024x1792" {
		ratios["size"] = 1.666667
	}
	return ratios
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if a.isVinted {
		return fmt.Sprintf("%s/v1/videos", strings.TrimRight(a.baseURL, "/")), nil
	}
	if info.Action == constant.TaskActionRemix {
		return fmt.Sprintf("%s/v1/videos/%s/remix", a.baseURL, info.OriginTaskID), nil
	}
	return fmt.Sprintf("%s/v1/videos", a.baseURL), nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	if a.isVinted {
		req.Header.Set("Content-Type", "application/json")
		idempotencyKey := strings.TrimSpace(info.UpstreamIdempotencyKey)
		if idempotencyKey == "" {
			source := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
			if source == "" {
				source = strings.TrimSpace(info.RequestId)
			}
			if source == "" {
				source = strings.TrimSpace(info.PublicTaskID)
			}
			if source != "" {
				sum := sha256.Sum256([]byte(fmt.Sprintf("%d\x00%d\x00%s\x00%s", info.UserId, info.TokenId, c.Request.URL.Path, source)))
				idempotencyKey = fmt.Sprintf("newapi-%x", sum)
			}
		}
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}
		return nil
	}
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_request_body_failed")
	}
	cachedBody, err := storage.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "read_body_bytes_failed")
	}
	contentType := c.GetHeader("Content-Type")
	if a.isVinted {
		var request vintedVideoRequest
		if err := common.Unmarshal(cachedBody, &request); err != nil {
			return nil, errors.Wrap(err, "unmarshal_vinted_request_failed")
		}
		normalized, err := normalizeVintedRequest(request, info.UpstreamModelName)
		if err != nil {
			return nil, err
		}
		newBody, err := common.Marshal(normalized)
		if err != nil {
			return nil, errors.Wrap(err, "marshal_vinted_request_failed")
		}
		return bytes.NewReader(newBody), nil
	}

	if strings.HasPrefix(contentType, "application/json") {
		var bodyMap map[string]interface{}
		if err := common.Unmarshal(cachedBody, &bodyMap); err == nil {
			bodyMap["model"] = info.UpstreamModelName
			if newBody, err := common.Marshal(bodyMap); err == nil {
				return bytes.NewReader(newBody), nil
			}
		}
		return bytes.NewReader(cachedBody), nil
	}

	if strings.Contains(contentType, "multipart/form-data") {
		formData, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return bytes.NewReader(cachedBody), nil
		}
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		writer.WriteField("model", info.UpstreamModelName)
		for key, values := range formData.Value {
			if key == "model" {
				continue
			}
			for _, v := range values {
				writer.WriteField(key, v)
			}
		}
		for fieldName, fileHeaders := range formData.File {
			for _, fh := range fileHeaders {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				ct := fh.Header.Get("Content-Type")
				if ct == "" || ct == "application/octet-stream" {
					buf512 := make([]byte, 512)
					n, _ := io.ReadFull(f, buf512)
					ct = http.DetectContentType(buf512[:n])
					// Re-open after sniffing so the full content is copied below
					f.Close()
					f, err = fh.Open()
					if err != nil {
						continue
					}
				}
				h := make(textproto.MIMEHeader)
				h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fh.Filename))
				h.Set("Content-Type", ct)
				part, err := writer.CreatePart(h)
				if err != nil {
					f.Close()
					continue
				}
				io.Copy(part, f)
				f.Close()
			}
		}
		writer.Close()
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		return &buf, nil
	}

	return common.NewReplayableBodyReader(storage), nil
}

// DoRequest delegates to common helper.
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse handles upstream response, returns taskID etc.
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()
	if a.isVinted {
		var dResp vintedAcceptedResponse
		if err := common.Unmarshal(responseBody, &dResp); err != nil {
			taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
			return
		}
		if strings.TrimSpace(dResp.ID) == "" {
			taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
			return
		}
		upstreamID := dResp.ID
		dResp.ID = info.PublicTaskID
		clientBody, err := common.Marshal(dResp)
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError)
			return
		}
		info.TaskClientResponseStatus = resp.StatusCode
		info.TaskClientResponseBody = clientBody
		return upstreamID, responseBody, nil
	}

	// Parse Sora response
	var dResp responseTask
	if err := common.Unmarshal(responseBody, &dResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	upstreamID := dResp.ID
	if upstreamID == "" {
		upstreamID = dResp.TaskID
	}
	if upstreamID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	// 使用公开 task_xxxx ID 返回给客户端
	dResp.ID = info.PublicTaskID
	dResp.TaskID = info.PublicTaskID
	c.JSON(http.StatusOK, dResp)
	return upstreamID, responseBody, nil
}

// FetchTask fetch task status
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/v1/videos/%s", baseUrl, taskID)
	if a.isVinted {
		uri = fmt.Sprintf("%s/v1/videos/%s", strings.TrimRight(baseUrl, "/"), url.PathEscape(taskID))
	}
	header := http.Header{}
	if config, ok := body["polling_config"].(*model.TaskPollingConfig); ok && config != nil && config.URL != "" {
		uri = config.URL
		for name, values := range config.Headers {
			for _, value := range values {
				header.Add(name, value)
			}
		}
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header = header
	if req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) TaskPollingConfig(upstreamTaskID string) (*model.TaskPollingConfig, error) {
	if strings.TrimSpace(upstreamTaskID) == "" {
		return nil, errors.New("task_id is required")
	}
	protocol := relaykitdto.VideoProtocolOpenAI
	if a.isVinted {
		protocol = relaykitdto.VideoProtocolVinted
	}
	return &model.TaskPollingConfig{
		URL: fmt.Sprintf("%s/v1/videos/%s", strings.TrimRight(a.baseURL, "/"), url.PathEscape(upstreamTaskID)),
		Headers: map[string][]string{
			"Authorization": {"Bearer " + a.apiKey},
		},
		VideoProtocol: protocol,
	}, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	if a.isVinted {
		var resTask vintedResponseTask
		if err := common.Unmarshal(respBody, &resTask); err != nil {
			return nil, errors.Wrap(err, "unmarshal Vinted task result failed")
		}
		taskResult := relaycommon.TaskInfo{Code: 0}
		switch resTask.Status {
		case "queued":
			taskResult.Status = model.TaskStatusQueued
		case "running":
			taskResult.Status = model.TaskStatusInProgress
		case "succeeded":
			taskResult.Status = model.TaskStatusSuccess
			taskResult.RemoteUrl = strings.TrimSpace(resTask.File)
			if taskResult.RemoteUrl == "" {
				taskResult.RemoteUrl = strings.TrimSpace(resTask.DownloadURL)
			}
		case "failed":
			taskResult.Status = model.TaskStatusFailure
			taskResult.Reason = strings.TrimSpace(resTask.Error)
			if taskResult.Reason == "" {
				taskResult.Reason = "task failed"
			}
		}
		return &taskResult, nil
	}

	resTask := responseTask{}
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := relaycommon.TaskInfo{
		Code: 0,
	}

	switch resTask.Status {
	case "queued", "pending":
		taskResult.Status = model.TaskStatusQueued
	case "processing", "in_progress":
		taskResult.Status = model.TaskStatusInProgress
	case "unknown":
		// The minimax_h3 provider can temporarily return a valid task envelope with
		// status=unknown while generation is still running. Keep polling instead of
		// turning that transitional response into a failed task and refund.
		if resTask.Model == "minimax_h3" && (resTask.ID != "" || resTask.TaskID != "") && resTask.Error == nil {
			taskResult.Status = model.TaskStatusInProgress
		}
	case "completed":
		taskResult.Status = model.TaskStatusSuccess
		// Url intentionally left empty — the caller constructs the proxy URL using the public task ID
	case "failed", "cancelled":
		taskResult.Status = model.TaskStatusFailure
		if resTask.Error != nil {
			taskResult.Reason = resTask.Error.Message
		} else {
			taskResult.Reason = "task failed"
		}
	default:
	}
	if resTask.Progress > 0 && resTask.Progress < 100 {
		taskResult.Progress = fmt.Sprintf("%d%%", resTask.Progress)
	}

	return &taskResult, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	data := task.Data
	var err error
	if data, err = sjson.SetBytes(data, "id", task.TaskID); err != nil {
		return nil, errors.Wrap(err, "set id failed")
	}
	return data, nil
}
