package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveVintedVideoURLRequestsOriginalSignedURL(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/videos/job_upstream/signed_url", r.URL.Path)
		assert.Empty(t, r.URL.RawQuery)
		assert.Equal(t, "Bearer upstream-key", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, err := fmt.Fprintf(w, `{"url":"/v1/videos/job_upstream/file?exp=123&sig=secret","expires_at":123,"quality":"original"}`)
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	result, err := resolveVintedVideoURL(context.Background(), server.Client(), server.URL, "upstream-key", "job_upstream")

	require.NoError(t, err)
	assert.Equal(t, server.URL+"/v1/videos/job_upstream/file?exp=123&sig=secret", result)
}

func TestResolveVintedVideoURLAcceptsCurrentContractWithoutQuality(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer upstream-key", r.Header.Get("Authorization"))
		assert.Empty(t, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"url":"/v1/videos/job_upstream/file?sig=signed"}`))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	result, err := resolveVintedVideoURL(context.Background(), server.Client(), server.URL, "upstream-key", "job_upstream")

	require.NoError(t, err)
	assert.Equal(t, server.URL+"/v1/videos/job_upstream/file?sig=signed", result)
}

func TestResolveVintedVideoURLRejectsInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"expires_at":123,"quality":"original"}`))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	result, err := resolveVintedVideoURL(context.Background(), server.Client(), server.URL, "upstream-key", "job_upstream")

	require.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "missing url")
}

func TestResolveVintedVideoURLRejectsPreviewFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"url":"/preview.mp4","expires_at":123,"quality":"preview"}`))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	result, err := resolveVintedVideoURL(context.Background(), server.Client(), server.URL, "upstream-key", "job_upstream")

	require.Error(t, err)
	assert.Empty(t, result)
	assert.ErrorIs(t, err, errVintedOriginalNotReady)
}

func TestResolveVintedVideoURLTreatsConflictAsOriginalNotReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	t.Cleanup(server.Close)

	result, err := resolveVintedVideoURL(context.Background(), server.Client(), server.URL, "upstream-key", "job_upstream")

	require.Error(t, err)
	assert.Empty(t, result)
	assert.ErrorIs(t, err, errVintedOriginalNotReady)
}

func TestIsForwardableVideoResponseStatusIncludesRangeResponses(t *testing.T) {
	assert.True(t, isForwardableVideoResponseStatus(http.StatusOK))
	assert.True(t, isForwardableVideoResponseStatus(http.StatusPartialContent))
	assert.True(t, isForwardableVideoResponseStatus(http.StatusRequestedRangeNotSatisfiable))
	assert.False(t, isForwardableVideoResponseStatus(http.StatusBadGateway))
}

func TestCopyVideoProxyHeadersAllowsDirectVideoPlayback(t *testing.T) {
	upstream := http.Header{
		"Content-Type":            {"video/mp4"},
		"Content-Length":          {"1024"},
		"Content-Range":           {"bytes 0-1023/35551474"},
		"Content-Security-Policy": {"sandbox; default-src 'none'"},
	}
	forwarded := make(http.Header)

	copyVideoProxyHeaders(forwarded, upstream)

	assert.Equal(t, "video/mp4", forwarded.Get("Content-Type"))
	assert.Equal(t, "1024", forwarded.Get("Content-Length"))
	assert.Equal(t, "bytes 0-1023/35551474", forwarded.Get("Content-Range"))
	assert.Empty(t, forwarded.Get("Content-Security-Policy"))
	assert.Equal(t, "nosniff", forwarded.Get("X-Content-Type-Options"))
}

func TestCopyVideoProxyHeadersPreservesNonVideoSecurityPolicy(t *testing.T) {
	upstream := http.Header{
		"Content-Type":            {"text/html"},
		"Content-Security-Policy": {"sandbox; default-src 'none'"},
	}
	forwarded := make(http.Header)

	copyVideoProxyHeaders(forwarded, upstream)

	assert.Equal(t, "sandbox; default-src 'none'", forwarded.Get("Content-Security-Policy"))
}
