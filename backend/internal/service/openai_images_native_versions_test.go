package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 本次扩展的 1.5/2 与 2.5 共用原生编辑流，并保持 OAuth transport 配置。
func TestCodexDirectImagesAllVersionsStreamEdits(t *testing.T) {
	for _, model := range []string{"gpt-image-1.5", "gpt-image-2", "gpt-image-2.5-flare"} {
		t.Run(model, func(t *testing.T) {
			body := []byte(fmt.Sprintf(`{"model":%q,"prompt":"edit","images":[{"image_url":"data:image/png;base64,AA=="}],"stream":true}`, model))
			c, rec := newOpenAIImagesTestContext(t, body)
			c.Request.URL.Path = "/v1/images/edits"
			upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("data: {\"type\":\"image_generation.completed\",\"b64_json\":\"AA==\",\"usage\":{\"input_tokens\":5,\"output_tokens\":9}}\n\n"))}}
			svc := newOpenAIImagesTestService(upstream)
			parsed, err := svc.ParseOpenAIImagesRequest(c, body)
			require.NoError(t, err)
			result, err := svc.ForwardImages(context.Background(), c, directImagesTestAccount(), body, parsed, "")
			require.NoError(t, err)
			require.Equal(t, 1, result.ImageCount)
			require.Equal(t, 9, result.Usage.ImageOutputTokens)
			require.Equal(t, "/backend-api/codex/images/edits", upstream.lastReq.URL.Path)
			require.Equal(t, HTTPUpstreamProfileOpenAI, HTTPUpstreamProfileFromContext(upstream.lastReq.Context()))
			require.Contains(t, rec.Body.String(), "image_edit.completed")
		})
	}
}
