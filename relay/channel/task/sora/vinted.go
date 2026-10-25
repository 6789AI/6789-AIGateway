package sora

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// The catalog and quote endpoint, rather than gateway model aliases, define
// upstream capabilities. Quotes also validate media content and combinations.
type vintedCatalog struct {
	Status     string               `json:"status"`
	Revision   string               `json:"revision"`
	ValidUntil string               `json:"valid_until"`
	Models     []vintedCatalogModel `json:"models"`
}

type vintedCatalogModel struct {
	ID                string            `json:"id"`
	Available         bool              `json:"available"`
	UnavailableReason string            `json:"unavailable_reason"`
	InputModes        []string          `json:"input_modes"`
	CanvasInputs      map[string]string `json:"x-canvas-inputs"`
	InputSchema       vintedSchema      `json:"input_schema"`
	ParameterSchema   vintedSchema      `json:"parameter_schema"`
	CombinationSchema vintedSchema      `json:"combination_schema"`
}

// These catalog constraints catch invalid parameters before billing or uploads.
// The upstream quote remains authoritative for the full JSON Schema contract.
type vintedSchema struct {
	Type                 string                  `json:"type"`
	Properties           map[string]vintedSchema `json:"properties"`
	Required             []string                `json:"required"`
	AdditionalProperties *bool                   `json:"additionalProperties"`
	Items                *vintedSchema           `json:"items"`
	Default              json.RawMessage         `json:"default,omitempty"`
	Const                json.RawMessage         `json:"const,omitempty"`
	Enum                 []json.RawMessage       `json:"enum"`
	Minimum              *float64                `json:"minimum"`
	Maximum              *float64                `json:"maximum"`
	MinLength            *int                    `json:"minLength"`
	MaxLength            *int                    `json:"maxLength"`
	MinItems             *int                    `json:"minItems"`
	MaxItems             *int                    `json:"maxItems"`
	AnyOf                []vintedSchema          `json:"anyOf"`
	AllOf                []vintedSchema          `json:"allOf"`
	If                   *vintedSchema           `json:"if"`
	Then                 *vintedSchema           `json:"then"`
	Else                 *vintedSchema           `json:"else"`
}

func (s vintedSchema) validate(value any, path string) error {
	if len(s.Const) > 0 {
		var expected any
		if common.Unmarshal(s.Const, &expected) != nil || !reflect.DeepEqual(expected, value) {
			return fmt.Errorf("%s does not match the catalog constant", path)
		}
	}
	if len(s.Enum) > 0 {
		matched := false
		for _, option := range s.Enum {
			var expected any
			if common.Unmarshal(option, &expected) == nil && reflect.DeepEqual(expected, value) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("%s is not in the catalog enum", path)
		}
	}
	validType := s.Type == ""
	switch v := value.(type) {
	case map[string]any:
		validType = validType || s.Type == "object"
		for _, name := range s.Required {
			if _, exists := v[name]; !exists {
				return fmt.Errorf("%s.%s is required by the catalog", path, name)
			}
		}
		for name, field := range v {
			property, declared := s.Properties[name]
			if !declared && s.AdditionalProperties != nil && !*s.AdditionalProperties {
				return fmt.Errorf("%s.%s is not declared in the catalog", path, name)
			}
			if declared {
				if err := property.validate(field, path+"."+name); err != nil {
					return err
				}
			}
		}
	case []any:
		validType = validType || s.Type == "array"
		if s.MinItems != nil && len(v) < *s.MinItems || s.MaxItems != nil && len(v) > *s.MaxItems {
			return fmt.Errorf("%s reference count is outside the catalog limits", path)
		}
		if s.Items != nil {
			for i, item := range v {
				if err := s.Items.validate(item, fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
			}
		}
	case string:
		validType = validType || s.Type == "string"
		length := utf8.RuneCountInString(v)
		if s.MinLength != nil && length < *s.MinLength || s.MaxLength != nil && length > *s.MaxLength {
			return fmt.Errorf("%s length is outside the catalog limits", path)
		}
	case float64:
		validType = validType || s.Type == "number" || s.Type == "integer" && math.Trunc(v) == v
		if s.Minimum != nil && v < *s.Minimum || s.Maximum != nil && v > *s.Maximum {
			return fmt.Errorf("%s is outside the catalog range", path)
		}
	case bool:
		validType = validType || s.Type == "boolean"
	case nil:
		validType = validType || s.Type == "null"
	}
	if !validType {
		return fmt.Errorf("%s must have catalog type %s", path, s.Type)
	}
	for _, branch := range s.AllOf {
		if err := branch.validate(value, path); err != nil {
			return err
		}
	}
	if len(s.AnyOf) > 0 {
		matched := false
		for _, branch := range s.AnyOf {
			if branch.validate(value, path) == nil {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("%s does not match a permitted catalog combination", path)
		}
	}
	if s.If != nil {
		branch := s.Else
		if s.If.validate(value, path) == nil {
			branch = s.Then
		}
		if branch != nil {
			return branch.validate(value, path)
		}
	}
	return nil
}

type vintedAPIError struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
}

func (req *vintedVideoRequest) normalizeAliases() error {
	if req.Duration == nil && req.Seconds != nil {
		seconds, err := req.Seconds.Int64()
		if err != nil || seconds < 1 || seconds > relaycommon.MaxTaskDurationSeconds {
			return fmt.Errorf("seconds must be between 1 and %d integer seconds", relaycommon.MaxTaskDurationSeconds)
		}
		duration := int(seconds)
		req.Duration = &duration
	}
	if req.Ratio == nil {
		req.Ratio = req.AspectRatio
	}
	if req.Metadata != nil {
		if req.Ratio == nil {
			req.Ratio = req.Metadata.Ratio
		}
		if req.Resolution == nil {
			req.Resolution = req.Metadata.Resolution
		}
	}
	return nil
}

func (e *vintedAPIError) Error() string {
	return fmt.Sprintf("Vinted HTTP %d [%s]: %s (request_id=%s)", e.StatusCode, e.Code, e.Message, e.RequestID)
}

func (a *TaskAdaptor) vintedJSON(c *gin.Context, info *relaycommon.RelayInfo, method, path string, body []byte, result any) error {
	client, err := service.GetHttpClientWithProxySettings(info.ChannelSetting.Proxy, info.ChannelSetting)
	if err != nil {
		return &vintedAPIError{StatusCode: http.StatusBadGateway, Code: "vinted_transport_error", Message: "cannot initialize channel transport"}
	}
	// Credential-bearing catalog/upload/quote calls must never follow redirects.
	apiClient := *client
	apiClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(a.baseURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return &vintedAPIError{StatusCode: http.StatusBadGateway, Code: "vinted_transport_error", Message: "invalid channel API URL"}
	}
	if path != "/v1/catalog" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := apiClient.Do(req)
	if err != nil {
		return &vintedAPIError{StatusCode: http.StatusBadGateway, Code: "vinted_transport_error", Message: "channel API request did not complete"}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiError := &vintedAPIError{StatusCode: resp.StatusCode, Code: "vinted_api_error", Message: "channel API rejected the request", RequestID: resp.Header.Get("X-Request-ID")}
		if resp.StatusCode < 400 {
			apiError.StatusCode = http.StatusBadGateway
		}
		var payload struct {
			Error struct {
				Code      string `json:"code"`
				Message   string `json:"message"`
				RequestID string `json:"request_id"`
			} `json:"error"`
		}
		if common.DecodeJson(io.LimitReader(resp.Body, 32768), &payload) == nil {
			if payload.Error.Code != "" {
				apiError.Code = payload.Error.Code
			}
			if payload.Error.Message != "" {
				apiError.Message = payload.Error.Message
			}
			if payload.Error.RequestID != "" {
				apiError.RequestID = payload.Error.RequestID
			}
		}
		return apiError
	}
	if err := common.DecodeJson(io.LimitReader(resp.Body, 2<<20), result); err != nil {
		return &vintedAPIError{StatusCode: http.StatusBadGateway, Code: "vinted_invalid_response", Message: "channel API returned invalid JSON"}
	}
	return nil
}

func (a *TaskAdaptor) prepareVintedRequest(c *gin.Context, info *relaycommon.RelayInfo) error {
	if a.vintedBody != nil {
		return nil
	}
	var req vintedVideoRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return err
	}
	if err := req.normalizeAliases(); err != nil {
		return err
	}
	if err := validateVintedRequestShape(req); err != nil {
		return err
	}
	var catalog vintedCatalog
	if err := a.vintedJSON(c, info, http.MethodGet, "/v1/catalog", nil, &catalog); err != nil {
		return err
	}
	if catalog.Status != "ready" || strings.TrimSpace(catalog.Revision) == "" {
		return &vintedAPIError{StatusCode: http.StatusServiceUnavailable, Code: "vinted_catalog_unavailable", Message: "model catalog is not ready"}
	}
	if catalog.ValidUntil != "" {
		deadline, err := time.Parse(time.RFC3339Nano, catalog.ValidUntil)
		if err != nil || !deadline.After(time.Now()) {
			return &vintedAPIError{StatusCode: http.StatusServiceUnavailable, Code: "vinted_catalog_expired", Message: "model catalog has expired"}
		}
	}
	var selected *vintedCatalogModel
	for i := range catalog.Models {
		if catalog.Models[i].ID == strings.TrimSpace(info.UpstreamModelName) {
			selected = &catalog.Models[i]
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("mapped Vinted model %q is not in the current catalog", info.UpstreamModelName)
	}
	if !selected.Available {
		return fmt.Errorf("mapped Vinted model %q is unavailable: %s", selected.ID, selected.UnavailableReason)
	}
	parameters := make(map[string]any)
	for name, schema := range selected.ParameterSchema.Properties {
		defaultValue := schema.Default
		if len(defaultValue) == 0 {
			defaultValue = schema.Const
		}
		if len(defaultValue) > 0 {
			var value any
			if err := common.Unmarshal(defaultValue, &value); err != nil {
				return fmt.Errorf("invalid catalog default for %s", name)
			}
			parameters[name] = value
		}
		switch name {
		case "duration":
			if req.Duration != nil {
				parameters[name] = float64(*req.Duration)
			}
		case "ratio":
			if req.Ratio != nil {
				parameters[name] = strings.TrimSpace(*req.Ratio)
			}
		case "resolution":
			if req.Resolution != nil {
				parameters[name] = strings.TrimSpace(*req.Resolution)
			}
		case "camera_movement":
			if req.CameraMovement != nil {
				parameters[name] = strings.TrimSpace(*req.CameraMovement)
			}
		}
	}
	if req.CameraMovement != nil && strings.TrimSpace(*req.CameraMovement) != "" {
		if _, declared := selected.ParameterSchema.Properties["camera_movement"]; !declared {
			return fmt.Errorf("camera_movement is not declared for mapped Vinted model %q", selected.ID)
		}
	}
	if err := selected.ParameterSchema.validate(parameters, "parameters"); err != nil {
		return err
	}
	// Catalog defaults become billing inputs too; never silently charge a
	// different duration from the request actually sent upstream.
	duration, ok := parameters["duration"].(float64)
	if !ok || math.Trunc(duration) != duration || duration < 1 || duration > relaycommon.MaxTaskDurationSeconds {
		return fmt.Errorf("catalog duration must be between 1 and %d integer seconds", relaycommon.MaxTaskDurationSeconds)
	}
	imageIDs, imageURLs, err := collectVintedReferences(req)
	if err != nil {
		return err
	}
	body := vintedUpstreamRequest{
		ModelID: selected.ID, CatalogRevision: catalog.Revision,
		InputMode: "text", Input: map[string]any{"prompt": req.Prompt}, Parameters: parameters,
	}
	if len(imageIDs)+len(imageURLs) > 0 || !common.StringsContains(selected.InputModes, "text") {
		body.InputMode = "references"
	}
	if req.InputMode != nil {
		body.InputMode = strings.TrimSpace(*req.InputMode)
	}
	if !common.StringsContains(selected.InputModes, body.InputMode) {
		return fmt.Errorf("input_mode %q is not supported by mapped Vinted model %q", body.InputMode, selected.ID)
	}
	imageField := selected.CanvasInputs["image"]
	references := make([]any, 0, len(imageIDs)+len(imageURLs))
	for i := 0; i < len(imageIDs)+len(imageURLs); i++ {
		references = append(references, map[string]any{"asset_id": fmt.Sprintf("pending_%d", i), "kind": "image"})
	}
	if len(references) > 0 {
		if imageField == "" {
			return fmt.Errorf("mapped Vinted model %q does not accept image references", selected.ID)
		}
		body.Input[imageField] = references
	}
	if err := selected.InputSchema.validate(body.Input, "input"); err != nil {
		return err
	}
	combination := map[string]any{"input_mode": body.InputMode, "input": body.Input, "parameters": body.Parameters}
	if err := selected.CombinationSchema.validate(combination, "request"); err != nil {
		return err
	}
	for i := range references {
		imageID, imageURL := "", ""
		if i < len(imageIDs) {
			imageID = imageIDs[i]
		} else {
			imageURL = imageURLs[i-len(imageIDs)]
		}
		assetID, err := a.uploadVintedImage(c, info, imageID, imageURL)
		if err != nil {
			return err
		}
		references[i] = map[string]any{"asset_id": assetID, "kind": "image"}
	}
	encoded, err := common.Marshal(body)
	if err != nil {
		return err
	}
	var quote struct {
		ID       string `json:"quote_id"`
		Revision string `json:"catalog_revision"`
	}
	if err := a.vintedJSON(c, info, http.MethodPost, "/v1/videos/quote", encoded, &quote); err != nil {
		return err
	}
	if strings.TrimSpace(quote.ID) == "" || quote.Revision != catalog.Revision {
		return &vintedAPIError{StatusCode: http.StatusBadGateway, Code: "vinted_invalid_quote", Message: "quote ID or catalog revision is invalid"}
	}
	taskRequest, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return err
	}
	taskRequest.Duration = int(duration)
	a.vintedResolution, _ = parameters["resolution"].(string)
	c.Set("task_request", taskRequest)
	a.vintedBody, a.vintedQuote = encoded, quote.ID
	return nil
}

func (a *TaskAdaptor) uploadVintedImage(c *gin.Context, info *relaycommon.RelayInfo, imageID, imageURL string) (string, error) {
	var resp *http.Response
	var err error
	if imageID != "" {
		client, clientErr := service.GetHttpClientWithProxySettings(info.ChannelSetting.Proxy, info.ChannelSetting)
		if clientErr != nil {
			return "", fmt.Errorf("cannot initialize Vinted image transport")
		}
		apiClient := *client
		apiClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
		req, requestErr := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, strings.TrimRight(a.baseURL, "/")+"/v1/reference-images/"+url.PathEscape(imageID), nil)
		if requestErr != nil {
			return "", fmt.Errorf("invalid Vinted reference image URL")
		}
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
		resp, err = apiClient.Do(req)
	} else {
		// Reuse the gateway's protected downloader for customer-supplied URLs.
		resp, err = service.DoDownloadRequest(imageURL, "vinted_reference_upload")
	}
	if err != nil {
		return "", fmt.Errorf("Vinted reference image download did not complete")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Vinted reference image download returned HTTP %d", resp.StatusCode)
	}
	const maxImageBytes = 10 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxImageBytes {
		return "", fmt.Errorf("Vinted reference image must contain 1 byte to 10 MiB")
	}
	extension := ""
	switch http.DetectContentType(data) {
	case "image/png":
		extension = ".png"
	case "image/jpeg":
		extension = ".jpg"
	case "image/webp":
		extension = ".webp"
	default:
		return "", fmt.Errorf("Vinted reference images must be PNG, JPEG or WebP")
	}
	digest := sha256.Sum256(data)
	encoded, err := common.Marshal(map[string]string{
		"name": fmt.Sprintf("reference-%x%s", digest, extension), "image_b64": base64.StdEncoding.EncodeToString(data),
	})
	if err != nil {
		return "", err
	}
	var uploaded struct {
		AssetID string `json:"asset_id"`
		Kind    string `json:"kind"`
	}
	if err := a.vintedJSON(c, info, http.MethodPost, "/v1/files", encoded, &uploaded); err != nil {
		return "", err
	}
	if strings.TrimSpace(uploaded.AssetID) == "" || len(uploaded.AssetID) > 128 || uploaded.Kind != "image" {
		return "", &vintedAPIError{StatusCode: http.StatusBadGateway, Code: "vinted_invalid_asset", Message: "image upload did not return a valid image asset_id"}
	}
	return uploaded.AssetID, nil
}
