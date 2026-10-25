package sora

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relaykitdto "github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type vintedFixture struct {
	server       *httptest.Server
	catalog      vintedCatalog
	image        []byte
	requests     []string
	uploadNames  []string
	quoteBody    []byte
	createdBody  []byte
	quoteError   string
	quoteID      string
	uploadKind   string
	createHeader http.Header
}

func newVintedFixture(t *testing.T) *vintedFixture {
	t.Helper()
	service.InitHttpClient()
	fetchSettings := system_setting.GetFetchSetting()
	previousProtection := fetchSettings.EnableSSRFProtection
	fetchSettings.EnableSSRFProtection = false
	t.Cleanup(func() { fetchSettings.EnableSSRFProtection = previousProtection })
	f := &vintedFixture{quoteID: "quote-accepted", uploadKind: "image"}
	require.NoError(t, common.Unmarshal([]byte(`{
		"status":"ready","revision":"revision-current","models":[{
			"id":"frameflow:seedance-2.0","available":true,"input_modes":["text","references"],
			"x-canvas-inputs":{"image":"images"},
			"input_schema":{"type":"object","required":["prompt"],"additionalProperties":false,"properties":{
				"prompt":{"type":"string","minLength":1,"maxLength":2000},
				"images":{"type":"array","maxItems":9,"items":{"type":"object","required":["asset_id","kind"],"additionalProperties":false,"properties":{"asset_id":{"type":"string","minLength":1,"maxLength":128},"kind":{"const":"image"}}}}
			}},
			"parameter_schema":{"type":"object","required":["duration","ratio","resolution"],"additionalProperties":false,"properties":{
				"duration":{"type":"integer","enum":[5,10,15],"default":10},
				"ratio":{"type":"string","enum":["16:9","9:16","1:1"],"default":"1:1"},
				"resolution":{"const":"720p","default":"720p"}
			}}
		}]
	}`), &f.catalog))
	f.catalog.ValidUntil = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	var imageData bytes.Buffer
	require.NoError(t, png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	f.image = imageData.Bytes()
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.requests = append(f.requests, r.Method+" "+r.URL.Path)
		if strings.HasPrefix(r.URL.Path, "/image/") {
			assert.Empty(t, r.Header.Get("Authorization"), "API keys must not be sent to image hosts")
			w.Header().Set("Content-Type", "image/png")
			_, err := w.Write(f.image)
			assert.NoError(t, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/catalog" {
			assert.Empty(t, r.Header.Get("Authorization"))
			encoded, err := common.Marshal(f.catalog)
			assert.NoError(t, err)
			_, err = w.Write(encoded)
			assert.NoError(t, err)
			return
		}
		assert.Equal(t, "Bearer upstream-key", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/v1/reference-images/img_first":
			w.Header().Set("Content-Type", "image/png")
			_, err := w.Write(f.image)
			assert.NoError(t, err)
		case "/v1/files":
			var upload map[string]string
			if !assert.NoError(t, common.DecodeJson(r.Body, &upload)) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			assert.Len(t, upload, 2)
			digest := sha256.Sum256(f.image)
			assert.Equal(t, fmt.Sprintf("reference-%x.png", digest), upload["name"])
			assert.Equal(t, base64.StdEncoding.EncodeToString(f.image), upload["image_b64"])
			f.uploadNames = append(f.uploadNames, upload["name"])
			w.WriteHeader(http.StatusCreated)
			_, err := fmt.Fprintf(w, `{"asset_id":"asset_%d","kind":%q}`, len(f.uploadNames), f.uploadKind)
			assert.NoError(t, err)
		case "/v1/videos/quote":
			var err error
			f.quoteBody, err = io.ReadAll(r.Body)
			assert.NoError(t, err)
			if f.quoteError != "" {
				w.WriteHeader(http.StatusConflict)
				_, err = w.Write([]byte(f.quoteError))
			} else {
				_, err = fmt.Fprintf(w, `{"quote_id":%q,"catalog_revision":"revision-current","credits":30}`, f.quoteID)
			}
			assert.NoError(t, err)
		case "/v1/videos":
			var err error
			f.createdBody, err = io.ReadAll(r.Body)
			assert.NoError(t, err)
			f.createHeader = r.Header.Clone()
			w.WriteHeader(http.StatusAccepted)
			_, err = w.Write([]byte(`{"id":"job_upstream","status":"queued","provider":"frameflow"}`))
			assert.NoError(t, err)
		default:
			t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(f.server.Close)
	return f
}

func (f *vintedFixture) adaptor(modelID string) (*TaskAdaptor, *relaycommon.RelayInfo) {
	a, info := newVintedAdaptorInfo(modelID)
	info.ChannelBaseUrl = f.server.URL
	info.ChannelOtherSettings.VideoProtocol = relaykitdto.VideoProtocolVinted
	info.UpstreamIdempotencyKey = "newapi-stable-operation"
	info.PublicTaskID = "task_public"
	a.Init(info)
	return a, info
}

func TestVintedMappedCatalogModelUploadsQuotesAndCreatesExactRequest(t *testing.T) {
	f := newVintedFixture(t)
	a, info := f.adaptor("frameflow:seedance-2.0")
	c, _ := newVintedTestContext(t, fmt.Sprintf(`{"model":"canvas-alias","prompt":"product showcase","duration":10,"ratio":"9:16","resolution":"720p","image_urls":[%q],"seconds":"15","metadata":{"unexpected":true}}`, f.server.URL+"/image/first.png"))
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	require.Nil(t, a.ValidateMappedTaskRequest(c, info))
	resolution, resolutionErr := a.VideoBillingResolution(c, info)
	require.NoError(t, resolutionErr)
	assert.Equal(t, "720p", resolution)
	assert.Equal(t, constant.TaskActionGenerate, info.Action)
	assert.Equal(t, []string{"GET /v1/catalog", "GET /image/first.png", "POST /v1/files", "POST /v1/videos/quote"}, f.requests)

	body, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	resp, err := a.DoRequest(c, info, body)
	require.NoError(t, err)
	upstreamID, _, taskErr := a.DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	assert.Equal(t, "job_upstream", upstreamID)
	assert.Equal(t, "Bearer upstream-key", f.createHeader.Get("Authorization"))
	assert.Equal(t, "newapi-stable-operation", f.createHeader.Get("Idempotency-Key"))
	assert.Equal(t, "quote-accepted", f.createHeader.Get("Expected-Quote-Id"))
	assert.Equal(t, f.quoteBody, f.createdBody, "quote and create must use identical bytes")
	assert.JSONEq(t, `{"model_id":"frameflow:seedance-2.0","catalog_revision":"revision-current","input_mode":"references","input":{"prompt":"product showcase","images":[{"asset_id":"asset_1","kind":"image"}]},"parameters":{"duration":10,"ratio":"9:16","resolution":"720p"}}`, string(f.createdBody))

	replay, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	replayBody, err := io.ReadAll(replay)
	require.NoError(t, err)
	assert.Equal(t, f.createdBody, replayBody)
	assert.Len(t, f.uploadNames, 1, "rebuilding a request must reuse its assets and quote")
	assert.Equal(t, map[string]float64{"seconds": 10, "size": 1}, a.EstimateBilling(c, info))
}

func TestVintedMappedModelUsesCatalogDefaultsAndAcceptsNewNames(t *testing.T) {
	f := newVintedFixture(t)
	f.catalog.Models[0].ID = "future-provider:any-new-model"
	f.catalog.Models[0].ParameterSchema.Properties["duration"] = vintedSchema{Type: "integer", Default: []byte("12"), Minimum: vintedFloat(4), Maximum: vintedFloat(15)}
	delete(f.catalog.Models[0].ParameterSchema.Properties, "resolution")
	f.catalog.Models[0].ParameterSchema.Required = []string{"duration", "ratio"}
	a, info := f.adaptor("future-provider:any-new-model")
	c, _ := newVintedTestContext(t, `{"model":"any-client-alias","prompt":"test","resolution":"720p"}`)
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	require.Nil(t, a.ValidateMappedTaskRequest(c, info))
	assert.JSONEq(t, `{"model_id":"future-provider:any-new-model","catalog_revision":"revision-current","input_mode":"text","input":{"prompt":"test"},"parameters":{"duration":12,"ratio":"1:1"}}`, string(f.quoteBody))
	taskRequest, err := relaycommon.GetTaskRequest(c)
	require.NoError(t, err)
	assert.Equal(t, 12, taskRequest.Duration)
	assert.Equal(t, map[string]float64{"seconds": 12, "size": 1}, a.EstimateBilling(c, info))
	resolution, resolutionErr := a.VideoBillingResolution(c, info)
	require.NoError(t, resolutionErr)
	assert.Empty(t, resolution, "a quality ignored by the catalog must not affect billing")
	assert.Empty(t, f.uploadNames)
}

func TestVintedCanvasFieldAliasesUseTheSameMappedProtocol(t *testing.T) {
	for _, tt := range []struct {
		name    string
		fields  string
		seconds float64
		ratio   string
	}{
		{name: "full model fields", fields: `"seconds":10,"aspect_ratio":"9:16","resolution":"720p"`, seconds: 10, ratio: "9:16"},
		{name: "per second metadata", fields: `"seconds":"15","metadata":{"ratio":"16:9","resolution":"720p"}`, seconds: 15, ratio: "16:9"},
		{name: "explicit fields take precedence", fields: `"duration":5,"seconds":15,"ratio":"1:1","aspect_ratio":"9:16","metadata":{"ratio":"16:9"}`, seconds: 5, ratio: "1:1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newVintedFixture(t)
			a, info := f.adaptor("frameflow:seedance-2.0")
			c, _ := newVintedTestContext(t, fmt.Sprintf(`{"model":"any-canvas-name","prompt":"test",%s,"reference_images":[%q]}`, tt.fields, f.server.URL+"/image/first.png"))
			require.Nil(t, a.ValidateRequestAndSetAction(c, info))
			require.Nil(t, a.ValidateMappedTaskRequest(c, info))
			var body vintedUpstreamRequest
			require.NoError(t, common.Unmarshal(f.quoteBody, &body))
			assert.Equal(t, "frameflow:seedance-2.0", body.ModelID)
			assert.Equal(t, tt.seconds, body.Parameters["duration"])
			assert.Equal(t, tt.ratio, body.Parameters["ratio"])
			assert.Equal(t, "720p", body.Parameters["resolution"])
			assert.Equal(t, []any{map[string]any{"asset_id": "asset_1", "kind": "image"}}, body.Input["images"])
			assert.Equal(t, map[string]float64{"seconds": tt.seconds, "size": 1}, a.EstimateBilling(c, info))
		})
	}
}

func TestVintedReferenceAliasesMigrateLegacyImageIDsAndKeepOrder(t *testing.T) {
	f := newVintedFixture(t)
	a, info := f.adaptor("frameflow:seedance-2.0")
	payload := fmt.Sprintf(`{"model":"client-alias","prompt":"test","duration":5,"image_ids":["img_first","img_first"],"image":%q,"images":[%q],"image_refs":[%q],"image_url":%q,"image_urls":[%q]}`, f.server.URL+"/image/first.png", f.server.URL+"/image/second.png", f.server.URL+"/image/first.png", f.server.URL+"/image/third.png", f.server.URL+"/image/second.png")
	c, _ := newVintedTestContext(t, payload)
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	require.Nil(t, a.ValidateMappedTaskRequest(c, info))
	assert.Equal(t, []string{"GET /v1/catalog", "GET /v1/reference-images/img_first", "POST /v1/files", "GET /image/first.png", "POST /v1/files", "GET /image/third.png", "POST /v1/files", "GET /image/second.png", "POST /v1/files", "POST /v1/videos/quote"}, f.requests)
	var body vintedUpstreamRequest
	require.NoError(t, common.Unmarshal(f.quoteBody, &body))
	assert.Equal(t, []any{map[string]any{"asset_id": "asset_1", "kind": "image"}, map[string]any{"asset_id": "asset_2", "kind": "image"}, map[string]any{"asset_id": "asset_3", "kind": "image"}, map[string]any{"asset_id": "asset_4", "kind": "image"}}, body.Input["images"])
}

func vintedFloat(value float64) *float64 { return &value }

func TestVintedCatalogValidationPreventsInvalidSubmissions(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		modelID string
		change  func(*vintedFixture)
		wantErr string
	}{
		{name: "unlisted mapped model", payload: `{"prompt":"test"}`, modelID: "not-in-catalog", wantErr: "not in the current catalog"},
		{name: "model paused", payload: `{"prompt":"test"}`, change: func(f *vintedFixture) { f.catalog.Models[0].Available = false }, wantErr: "unavailable"},
		{name: "stale catalog", payload: `{"prompt":"test"}`, change: func(f *vintedFixture) { f.catalog.Status = "stale" }, wantErr: "not ready"},
		{name: "expired catalog", payload: `{"prompt":"test"}`, change: func(f *vintedFixture) { f.catalog.ValidUntil = "2000-01-01T00:00:00Z" }, wantErr: "expired"},
		{name: "invalid duration enum", payload: `{"prompt":"test","duration":6}`, wantErr: "parameters.duration"},
		{name: "invalid ratio", payload: `{"prompt":"test","ratio":"4:3"}`, wantErr: "parameters.ratio"},
		{name: "invalid resolution", payload: `{"prompt":"test","resolution":"1080p"}`, wantErr: "parameters.resolution"},
		{name: "prompt exceeds mapped limits", payload: `{"prompt":"` + strings.Repeat("字", 2001) + `"}`, wantErr: "input.prompt"},
		{name: "undeclared camera", payload: `{"prompt":"test","camera_movement":"fixed"}`, wantErr: "camera_movement is not declared"},
		{name: "unsupported mode", payload: `{"prompt":"test","input_mode":"frames"}`, wantErr: "input_mode"},
		{name: "reference count includes legacy ids", payload: `{"prompt":"test","image_ids":["img_first"],"image_urls":["https://a/1","https://a/2","https://a/3","https://a/4","https://a/5","https://a/6","https://a/7","https://a/8","https://a/9"]}`, wantErr: "reference count"},
		{name: "billing rejects unsafe catalog default", payload: `{"prompt":"test"}`, change: func(f *vintedFixture) {
			f.catalog.Models[0].ParameterSchema.Properties["duration"] = vintedSchema{Type: "integer", Default: []byte("2147483647")}
		}, wantErr: "catalog duration"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newVintedFixture(t)
			if tt.change != nil {
				tt.change(f)
			}
			if tt.modelID == "" {
				tt.modelID = "frameflow:seedance-2.0"
			}
			a, info := f.adaptor(tt.modelID)
			c, _ := newVintedTestContext(t, tt.payload)
			require.Nil(t, a.ValidateRequestAndSetAction(c, info))
			taskErr := a.ValidateMappedTaskRequest(c, info)
			require.NotNil(t, taskErr)
			assert.Contains(t, taskErr.Message, tt.wantErr)
			assert.Equal(t, []string{"GET /v1/catalog"}, f.requests, "invalid parameters must fail before uploads, quotes, or generation")
		})
	}
}

func TestVintedRejectsUnboundedDurationsBeforeCatalogAndBilling(t *testing.T) {
	for _, duration := range []string{"0", "-1", "3601", "9223372036854775807"} {
		t.Run(duration, func(t *testing.T) {
			a, info := newVintedAdaptorInfo("frameflow:seedance-2.0")
			c, _ := newVintedTestContext(t, `{"prompt":"test","duration":`+duration+`}`)
			taskErr := a.ValidateRequestAndSetAction(c, info)
			require.NotNil(t, taskErr)
			assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
			assert.Contains(t, taskErr.Message, "duration")
		})
	}
}

func TestVintedRejectsUnsafeSecondsAliasesBeforeBilling(t *testing.T) {
	for _, seconds := range []string{"0", "-1", "3601", "1.5", `"10.5"`, `"9223372036854775807"`} {
		t.Run(seconds, func(t *testing.T) {
			a, info := newVintedAdaptorInfo("frameflow:seedance-2.0")
			c, _ := newVintedTestContext(t, `{"prompt":"test","seconds":`+seconds+`}`)
			taskErr := a.ValidateRequestAndSetAction(c, info)
			require.NotNil(t, taskErr)
			assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
			assert.Contains(t, taskErr.Message, "seconds")
		})
	}
}

func TestVintedQuoteErrorsPreserveUpstreamCodeAndDoNotCreate(t *testing.T) {
	f := newVintedFixture(t)
	f.quoteError = `{"error":{"code":"pricing_changed","message":"refresh the price","request_id":"req_provider"}}`
	a, info := f.adaptor("frameflow:seedance-2.0")
	c, _ := newVintedTestContext(t, `{"prompt":"test"}`)
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	taskErr := a.ValidateMappedTaskRequest(c, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, http.StatusConflict, taskErr.StatusCode)
	assert.Contains(t, taskErr.Message, "pricing_changed")
	assert.Contains(t, taskErr.Message, "refresh the price")
	assert.Contains(t, taskErr.Message, "req_provider")
	assert.Empty(t, f.createdBody)
	assert.Nil(t, a.vintedBody)
	assert.Empty(t, a.vintedQuote)
}

func TestVintedRejectsMalformedUploadAndQuoteContracts(t *testing.T) {
	for _, stage := range []string{"upload", "quote"} {
		t.Run(stage, func(t *testing.T) {
			f := newVintedFixture(t)
			if stage == "upload" {
				f.uploadKind = "video"
			} else {
				f.quoteID = ""
			}
			a, info := f.adaptor("frameflow:seedance-2.0")
			c, _ := newVintedTestContext(t, fmt.Sprintf(`{"prompt":"test","image_urls":[%q]}`, f.server.URL+"/image/first.png"))
			require.Nil(t, a.ValidateRequestAndSetAction(c, info))
			taskErr := a.ValidateMappedTaskRequest(c, info)
			require.NotNil(t, taskErr)
			assert.Equal(t, http.StatusBadGateway, taskErr.StatusCode)
			assert.Nil(t, a.vintedBody)
			assert.Empty(t, f.createdBody)
		})
	}
}

func TestVintedCombinationValidationUsesCatalogInsteadOfModelNames(t *testing.T) {
	f := newVintedFixture(t)
	f.catalog.Models[0].ID = "multiref:sd-2-c4"
	f.catalog.Models[0].InputModes = []string{"references", "frames"}
	require.NoError(t, common.Unmarshal([]byte(`{
		"if": {"properties": {"input_mode": {"const": "frames"}}},
		"then": {"properties": {"input": {
			"required": ["images"],
			"properties": {"images": {"minItems": 1, "maxItems": 2}}
		}}}
	}`), &f.catalog.Models[0].CombinationSchema))
	a, info := f.adaptor("multiref:sd-2-c4")
	c, _ := newVintedTestContext(t, `{"prompt":"test","input_mode":"frames","image_urls":["https://example.com/1.png","https://example.com/2.png","https://example.com/3.png"]}`)
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	taskErr := a.ValidateMappedTaskRequest(c, info)
	require.NotNil(t, taskErr)
	assert.Contains(t, taskErr.Message, "reference count")
	assert.Equal(t, []string{"GET /v1/catalog"}, f.requests)

	// Seedance's multi-image allowance is combination-specific, even though
	// its input_schema alone allows nine images.
	f.catalog.Models[0].InputModes = []string{"text", "references"}
	require.NoError(t, common.Unmarshal([]byte(`{"anyOf":[{"properties":{"input_mode":{"const":"text"}}},{"properties":{"input_mode":{"const":"references"},"parameters":{"properties":{"duration":{"const":5},"ratio":{"const":"16:9"}}},"input":{"properties":{"images":{"maxItems":9}}}}},{"properties":{"input_mode":{"const":"references"},"input":{"properties":{"images":{"maxItems":1}}}}}]}`), &f.catalog.Models[0].CombinationSchema))
	a, info = f.adaptor("multiref:sd-2-c4")
	c, _ = newVintedTestContext(t, `{"prompt":"test","duration":10,"ratio":"9:16","image_urls":["https://example.com/1.png","https://example.com/2.png"]}`)
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	taskErr = a.ValidateMappedTaskRequest(c, info)
	require.NotNil(t, taskErr)
	assert.Contains(t, taskErr.Message, "catalog combination")
}

func TestVintedQueryKeepsUnarchivedVideoRunningForCanvas(t *testing.T) {
	a, _ := newVintedAdaptorInfo("frameflow:seedance-2.0")
	data, err := a.ConvertToOpenAIVideo(&model.Task{
		TaskID: "task_public", Status: model.TaskStatusInProgress,
		Data: []byte(`{"id":"job_upstream","status":"succeeded","has_file":false}`),
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"task_public","status":"running","has_file":false}`, string(data))
}

func TestVintedVideoBillingResolutionUsesCatalogDefault(t *testing.T) {
	f := newVintedFixture(t)
	f.catalog.Models[0].ParameterSchema.Properties["resolution"] = vintedSchema{Type: "string", Default: []byte(`"768p"`), Enum: []json.RawMessage{[]byte(`"768p"`), []byte(`"1080p"`)}}
	a, info := f.adaptor("frameflow:seedance-2.0")
	c, _ := newVintedTestContext(t, `{"model":"canvas-alias","prompt":"test"}`)
	require.Nil(t, a.ValidateRequestAndSetAction(c, info))
	require.Nil(t, a.ValidateMappedTaskRequest(c, info))
	resolution, err := a.VideoBillingResolution(c, info)
	require.NoError(t, err)
	assert.Equal(t, "768p", resolution)
}
