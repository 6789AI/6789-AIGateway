package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relay/channel/task/sora"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func main() {
	payload := []byte(`{"model":"minimax-h3","prompt":"protocol validation probe","seconds":16,"size":"1376x768","reference_images":["data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="]}`)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(payload))
	context.Request.Header.Set("Content-Type", "application/json")
	defer common.CleanupBodyStorage(context)

	body, err := (&sora.TaskAdaptor{}).BuildRequestBody(context, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "minimax_h3"},
	})
	if err != nil {
		panic(err)
	}
	request, err := http.NewRequest(http.MethodPost, "http://103.36.63.156:3000/v1/videos", body)
	if err != nil {
		panic(err)
	}
	request.Header.Set("Authorization", "Bearer "+os.Getenv("UPSTREAM_KEY"))
	request.Header.Set("Content-Type", context.Request.Header.Get("Content-Type"))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	fmt.Printf("HTTP %d\n%s\n", response.StatusCode, responseBody)
}
