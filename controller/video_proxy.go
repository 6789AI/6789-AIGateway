package controller

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

type vintedSignedURLResponse struct {
	URL     string `json:"url"`
	Quality string `json:"quality"`
}

var errVintedOriginalNotReady = errors.New("Vinted original video is not ready")

// videoProxyError returns a standardized OpenAI-style error response.
func videoProxyError(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"message": message,
			"type":    errType,
		},
	})
}

func resolveVintedVideoURL(ctx context.Context, client *http.Client, baseURL, apiKey, upstreamTaskID string) (string, error) {
	requestURL := fmt.Sprintf(
		"%s/v1/videos/%s/signed_url?download=1",
		strings.TrimRight(baseURL, "/"),
		url.PathEscape(upstreamTaskID),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return "", fmt.Errorf("%w: signed URL endpoint returned status %d", errVintedOriginalNotReady, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("signed URL endpoint returned status %d", resp.StatusCode)
	}

	var payload vintedSignedURLResponse
	if err := common.DecodeJson(io.LimitReader(resp.Body, 1<<20), &payload); err != nil {
		return "", fmt.Errorf("decode signed URL response: %w", err)
	}
	payload.URL = strings.TrimSpace(payload.URL)
	if payload.URL == "" {
		return "", fmt.Errorf("signed URL response is missing url")
	}
	if strings.TrimSpace(payload.Quality) != "original" {
		return "", fmt.Errorf("%w: signed URL quality is %q", errVintedOriginalNotReady, payload.Quality)
	}

	base, err := url.Parse(strings.TrimRight(baseURL, "/") + "/")
	if err != nil {
		return "", fmt.Errorf("parse Vinted base URL: %w", err)
	}
	reference, err := url.Parse(payload.URL)
	if err != nil {
		return "", fmt.Errorf("parse signed video URL: %w", err)
	}
	return base.ResolveReference(reference).String(), nil
}

func isForwardableVideoResponseStatus(status int) bool {
	return status == http.StatusOK ||
		status == http.StatusPartialContent ||
		status == http.StatusRequestedRangeNotSatisfiable
}

func VideoProxy(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error", "task_id is required")
		return
	}

	task, exists, err := model.GetTaskByPublicID(taskID)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to query task %s: %s", taskID, err.Error()))
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to query task")
		return
	}
	if !exists || task == nil {
		videoProxyError(c, http.StatusNotFound, "invalid_request_error", "Task not found")
		return
	}

	if task.Status != model.TaskStatusSuccess {
		videoProxyError(c, http.StatusBadRequest, "invalid_request_error",
			fmt.Sprintf("Task is not completed yet, current status: %s", task.Status))
		return
	}

	channel, err := model.CacheGetChannel(task.ChannelId)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to get channel for task %s: %s", taskID, err.Error()))
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to retrieve channel information")
		return
	}
	baseURL := channel.GetBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}

	var videoURL string
	proxy := channel.GetSetting().Proxy
	client := service.GetSSRFProtectedHTTPClient()
	if proxy != "" {
		// 渠道代理路径的连接由代理侧建立，无法做拨号时逐 IP 校验，
		// 因此后面对 videoURL 保留请求前的一次性 SSRF 校验。
		client, err = service.GetHttpClientWithProxy(proxy)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to create proxy client for task %s: %s", taskID, err.Error()))
			videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to create proxy client")
			return
		}
	}

	isVinted := model.IsVintedVideoChannel(channel.Type, baseURL, channel.GetOtherSettings())
	requestHeaders := make(http.Header)
	if isVinted {
		apiKey := task.PrivateData.Key
		if apiKey == "" {
			apiKey = channel.Key
		}
		lookupCtx, cancelLookup := context.WithTimeout(c.Request.Context(), 30*time.Second)
		videoURL, err = resolveVintedVideoURL(lookupCtx, client, baseURL, apiKey, task.GetUpstreamTaskID())
		cancelLookup()
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to resolve Vinted video URL for task %s: %s", taskID, err.Error()))
			if errors.Is(err, errVintedOriginalNotReady) {
				c.Header("Retry-After", "5")
				videoProxyError(c, http.StatusConflict, "video_not_ready", "Original video is not ready")
				return
			}
			videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to resolve video URL")
			return
		}
	} else {
		switch channel.Type {
		case constant.ChannelTypeGemini:
			apiKey := task.PrivateData.Key
			if apiKey == "" {
				logger.LogError(c.Request.Context(), fmt.Sprintf("Missing stored API key for Gemini task %s", taskID))
				videoProxyError(c, http.StatusInternalServerError, "server_error", "API key not stored for task")
				return
			}
			videoURL, err = getGeminiVideoURL(channel, task, apiKey)
			if err != nil {
				logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to resolve Gemini video URL for task %s: %s", taskID, err.Error()))
				videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to resolve Gemini video URL")
				return
			}
			requestHeaders.Set("x-goog-api-key", apiKey)
		case constant.ChannelTypeVertexAi:
			videoURL, err = getVertexVideoURL(channel, task)
			if err != nil {
				logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to resolve Vertex video URL for task %s: %s", taskID, err.Error()))
				videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to resolve Vertex video URL")
				return
			}
		case constant.ChannelTypeOpenAI, constant.ChannelTypeSora:
			videoURL = fmt.Sprintf("%s/v1/videos/%s/content", baseURL, task.GetUpstreamTaskID())
			requestHeaders.Set("Authorization", "Bearer "+channel.Key)
		default:
			// Video URL is stored in PrivateData.ResultURL (fallback to FailReason for old data)
			videoURL = task.GetResultURL()
		}
	}

	videoURL = strings.TrimSpace(videoURL)
	if videoURL == "" {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Video URL is empty for task %s", taskID))
		videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to fetch video content")
		return
	}

	if strings.HasPrefix(videoURL, "data:") {
		if err := writeVideoDataURL(c, videoURL); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to decode video data URL for task %s: %s", taskID, err.Error()))
			videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to fetch video content")
		}
		return
	}

	var validateErr error
	if proxy == "" {
		validateErr = service.ValidateSSRFProtectedFetchURL(videoURL)
	} else {
		fetchSetting := system_setting.GetFetchSetting()
		validateErr = common.ValidateURLWithFetchSetting(videoURL, fetchSetting.EnableSSRFProtection, fetchSetting.AllowPrivateIp, fetchSetting.DomainFilterMode, fetchSetting.IpFilterMode, fetchSetting.DomainList, fetchSetting.IpList, fetchSetting.AllowedPorts, fetchSetting.ApplyIPFilterForDomain)
	}
	if validateErr != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Video URL blocked for task %s: %v", taskID, validateErr))
		videoProxyError(c, http.StatusForbidden, "server_error", fmt.Sprintf("request blocked: %v", validateErr))
		return
	}

	downloadTimeout := 60 * time.Second
	if isVinted {
		downloadTimeout = 10 * time.Minute
	}
	downloadCtx, cancelDownload := context.WithTimeout(c.Request.Context(), downloadTimeout)
	defer cancelDownload()
	req, err := http.NewRequestWithContext(downloadCtx, c.Request.Method, videoURL, nil)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to parse URL %s: %s", relaycommon.SanitizeURLForLog(videoURL), err.Error()))
		videoProxyError(c, http.StatusInternalServerError, "server_error", "Failed to create proxy request")
		return
	}
	for key, values := range requestHeaders {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if rangeHeader := c.GetHeader("Range"); rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	if ifRangeHeader := c.GetHeader("If-Range"); ifRangeHeader != "" {
		req.Header.Set("If-Range", ifRangeHeader)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to fetch video from %s: %s", relaycommon.SanitizeURLForLog(videoURL), err.Error()))
		videoProxyError(c, http.StatusBadGateway, "server_error", "Failed to fetch video content")
		return
	}
	defer resp.Body.Close()

	if !isForwardableVideoResponseStatus(resp.StatusCode) {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Upstream returned status %d for %s", resp.StatusCode, relaycommon.SanitizeURLForLog(videoURL)))
		videoProxyError(c, http.StatusBadGateway, "server_error",
			fmt.Sprintf("Upstream service returned status %d", resp.StatusCode))
		return
	}
	if isVinted && strings.EqualFold(strings.TrimSpace(resp.Header.Get("X-Video-Quality")), "preview") {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Vinted returned preview video content for task %s", taskID))
		c.Header("Retry-After", "5")
		videoProxyError(c, http.StatusConflict, "video_not_ready", "Original video is not ready")
		return
	}

	for key, values := range resp.Header {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	if resp.StatusCode != http.StatusRequestedRangeNotSatisfiable {
		c.Writer.Header().Set("Cache-Control", "public, max-age=86400")
	}
	c.Writer.WriteHeader(resp.StatusCode)
	if c.Request.Method == http.MethodHead {
		return
	}
	if _, err = io.Copy(c.Writer, resp.Body); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to stream video content: %s", err.Error()))
	}
}

func writeVideoDataURL(c *gin.Context, dataURL string) error {
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid data url")
	}

	header := parts[0]
	payload := parts[1]
	if !strings.HasPrefix(header, "data:") || !strings.Contains(header, ";base64") {
		return fmt.Errorf("unsupported data url")
	}

	mimeType := strings.TrimPrefix(header, "data:")
	mimeType = strings.TrimSuffix(mimeType, ";base64")
	if mimeType == "" {
		mimeType = "video/mp4"
	}

	videoBytes, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		videoBytes, err = base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			return err
		}
	}

	c.Writer.Header().Set("Content-Type", mimeType)
	c.Writer.Header().Set("Cache-Control", "public, max-age=86400")
	c.Writer.WriteHeader(http.StatusOK)
	if c.Request.Method == http.MethodHead {
		return nil
	}
	_, err = c.Writer.Write(videoBytes)
	return err
}
