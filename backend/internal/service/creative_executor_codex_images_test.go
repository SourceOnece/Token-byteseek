package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// 创作台独立调用供应商，不经过网关计费；覆盖原生生成、编辑与历史模型。
func TestCreativeCodexImagesProtocolAndIsolation(t *testing.T) {
	image := makeTestPNG(t, 2, 2)
	b64 := base64.StdEncoding.EncodeToString(image)
	for _, model := range []string{"gpt-image-1", "gpt-image-2", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst"} {
		for _, operation := range []string{CreativeOperationGenerate, CreativeOperationEdit, CreativeOperationInpaint} {
			t.Run(model+"/"+operation, func(t *testing.T) {
				response := `{"data":[{"b64_json":"` + b64 + `"}]}`
				if !usesCodexDirectImages(model) {
					response = "data: {\"type\":\"response.completed\",\"response\":{\"output\":[{\"type\":\"image_generation_call\",\"result\":\"" + b64 + "\"}]}}\n\n"
				}
				upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}}
				gateway := newOpenAIImagesTestService(upstream)
				gateway.tlsFPProfileService = &TLSFingerprintProfileService{}
				executor := &CreativeExecutor{gateway: gateway}
				account := directImagesTestAccount()
				account.Extra = map[string]any{"enable_tls_fingerprint": true}
				run := CreativeRun{RunID: "synthetic-run", APIKeyID: 12, UserID: 34, Operation: operation, ImageSize: "1K"}
				payload := CreativeRunPayload{Prompt: "  中文绘图  ", Quality: "high"}
				if operation != CreativeOperationGenerate {
					payload.Sources = []CreativeInputImage{{Bytes: image, Mime: "image/png"}}
				}
				if operation == CreativeOperationInpaint {
					payload.Mask = &CreativeInputImage{Bytes: image, Mime: "image/png"}
				}
				outputs, err := executor.executeOpenAI(context.Background(), run, payload, account, model)
				require.NoError(t, err)
				require.Len(t, outputs, 1)
				require.Equal(t, image, outputs[0].Bytes)
				require.NotNil(t, upstream.lastTLSProfile)
				require.Equal(t, "Bearer test-token", upstream.lastReq.Header.Get("Authorization"))
				require.Equal(t, "test-account", upstream.lastReq.Header.Get("Chatgpt-Account-Id"))
				if usesCodexDirectImages(model) {
					endpoint := "/backend-api/codex/images/generations"
					if operation != CreativeOperationGenerate {
						endpoint = "/backend-api/codex/images/edits"
					}
					require.Equal(t, endpoint, upstream.lastReq.URL.Path)
					require.Equal(t, payload.Prompt, gjson.GetBytes(upstream.lastBody, "prompt").String())
					require.False(t, gjson.GetBytes(upstream.lastBody, "tools").Exists())
					if operation == CreativeOperationInpaint {
						require.Contains(t, gjson.GetBytes(upstream.lastBody, "mask.image_url").String(), b64)
					}
				} else {
					require.Equal(t, "/backend-api/codex/responses", upstream.lastReq.URL.Path)
				}
				require.Len(t, upstream.requests, 1)
			})
		}
	}
}

// 只有 404/405 会切协议，429/5xx 保留原 worker 重试边界，取消不脱离执行上下文。
func TestCreativeCodexImagesFallbackAndCancellation(t *testing.T) {
	image := base64.StdEncoding.EncodeToString(makeTestPNG(t, 2, 2))
	for _, status := range []int{404, 405, 401, 429, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			upstream := &httpUpstreamRecorder{responses: []*http.Response{
				{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":{"message":"synthetic"}}`))},
				{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"output\":[{\"type\":\"image_generation_call\",\"result\":\"" + image + "\"}]}}\n\n"))},
			}}
			executor := &CreativeExecutor{gateway: newOpenAIImagesTestService(upstream)}
			_, err := executor.executeOpenAI(context.Background(), CreativeRun{RunID: "test", Operation: CreativeOperationGenerate}, CreativeRunPayload{Prompt: "test"}, directImagesTestAccount(), "gpt-image-2.5-flare")
			if status == 404 || status == 405 {
				require.NoError(t, err)
				require.Len(t, upstream.requests, 2)
				require.Equal(t, "/backend-api/codex/responses", upstream.requests[1].URL.Path)
			} else {
				require.Error(t, err)
				require.Len(t, upstream.requests, 1)
				require.Equal(t, status == 429 || status == 503, IsRetryableCreativeError(err))
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	upstream := &codexModelsHTTPUpstreamStub{do: func(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
		require.ErrorIs(t, req.Context().Err(), context.Canceled)
		return nil, req.Context().Err()
	}}
	executor := &CreativeExecutor{gateway: newOpenAIImagesTestService(upstream)}
	_, err := executor.executeOpenAI(ctx, CreativeRun{Operation: CreativeOperationGenerate}, CreativeRunPayload{Prompt: "test"}, directImagesTestAccount(), "gpt-image-2")
	require.Error(t, err)
}
