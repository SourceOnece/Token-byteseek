package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/creative"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// oauthCreativeFixture 禁止 OAuth 流程调用普通 API Key 的地址构造函数。
func oauthCreativeFixture(t *testing.T, call func(*http.Request) (*http.Response, error)) *Target {
	t.Helper()
	return &Target{OpenAI: &OpenAIOptions{
		OAuth: true,
		Token: func(context.Context) (string, error) { return "fixture-oauth-token", nil },
		URL:   func(string) (string, error) { t.Fatal("OAuth reached API Key URL"); return "", nil },
		BuildOAuth: func(ctx context.Context, _ creative.CreativeRun, body []byte, token, url string) (*http.Request, error) {
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)
			return req, nil
		},
		Do: call,
	}}
}

func creativeFixtureResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestCreativeOAuthImageModelsAndOperations(t *testing.T) {
	for _, model := range []string{"gpt-image-1.5", "gpt-image-2", "gpt-image-2.5-flare", "gpt-image-2.5-sunburst", "gpt-image-2.5-flare-2026-09-08", "gpt-image-1"} {
		for _, operation := range []string{"generate", "edit", "inpaint"} {
			t.Run(model+"/"+operation, func(t *testing.T) {
				calls := 0
				image := []byte("fixture image bytes")
				encoded := base64.StdEncoding.EncodeToString(image)
				target := oauthCreativeFixture(t, func(req *http.Request) (*http.Response, error) {
					calls++
					require.Equal(t, "chatgpt.com", req.URL.Host)
					require.Equal(t, "Bearer fixture-oauth-token", req.Header.Get("Authorization"))
					require.Equal(t, "application/json", req.Header.Get("Content-Type"))
					body, err := io.ReadAll(req.Body)
					require.NoError(t, err)
					require.True(t, gjson.ValidBytes(body))
					if model == "gpt-image-1" {
						require.Equal(t, "/backend-api/codex/responses", req.URL.Path)
						require.Equal(t, "text/event-stream", req.Header.Get("Accept"))
						require.Contains(t, string(body), "image_generation")
						return creativeFixtureResponse(200, `data: {"type":"response.completed","response":{"status":"completed","output":[{"type":"image_generation_call","result":"`+encoded+`"}]}}`+"\n\n"), nil
					}
					require.Equal(t, model, gjson.GetBytes(body, "model").String())
					require.Equal(t, "cat", gjson.GetBytes(body, "prompt").String())
					require.Equal(t, "application/json", req.Header.Get("Accept"))
					if operation == "generate" {
						require.Equal(t, "/backend-api/codex/images/generations", req.URL.Path)
					} else {
						require.Equal(t, "/backend-api/codex/images/edits", req.URL.Path)
						require.Len(t, gjson.GetBytes(body, "images").Array(), 2)
						require.Contains(t, gjson.GetBytes(body, "images.0.image_url").String(), "data:image/png;base64,")
					}
					if operation == "inpaint" {
						require.Contains(t, gjson.GetBytes(body, "mask.image_url").String(), "data:image/png;base64,")
					}
					return creativeFixtureResponse(200, `{"data":[{"b64_json":"`+encoded+`"}]}`), nil
				})
				payload := creative.CreativeRunPayload{Prompt: "cat", Quality: "medium", Background: "auto"}
				if operation != "generate" {
					payload.Sources = []creative.CreativeInputImage{{Bytes: image, Mime: "image/png"}, {Bytes: image, Mime: "image/png"}}
				}
				if operation == "inpaint" {
					payload.Mask = &creative.CreativeInputImage{Bytes: image, Mime: "image/png"}
				}
				outputs, err := target.ExecuteOpenAI(context.Background(), creative.CreativeRun{Operation: operation, ImageSize: "1K"}, payload, model)
				require.NoError(t, err)
				require.Len(t, outputs, 1)
				require.Equal(t, image, outputs[0].Bytes)
				require.Equal(t, 1, calls)
			})
		}
	}
}

func TestCreativeOAuthFallbackAndFailures(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 405, 429, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			target := oauthCreativeFixture(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 2 {
					require.Equal(t, "/backend-api/codex/responses", req.URL.Path)
				}
				return creativeFixtureResponse(status, `{"error":{"message":"private prompt and credentials"}}`), nil
			})
			_, err := target.ExecuteOpenAI(context.Background(), creative.CreativeRun{Operation: "generate", ImageSize: "1K"}, creative.CreativeRunPayload{Prompt: "cat"}, "gpt-image-2")
			var upstreamErr *creative.CreativeUpstreamError
			require.ErrorAs(t, err, &upstreamErr)
			require.Equal(t, status, upstreamErr.StatusCode)
			require.NotContains(t, err.Error(), "private")
			require.Equal(t, status == 429 || status >= 500, creative.IsRetryableCreativeError(err))
			want := 1
			if status == 404 || status == 405 {
				want = 2
			}
			require.Equal(t, want, calls)
		})
	}
}

func TestCreativeOAuthFallbackSuccessClosesFirstResponse(t *testing.T) {
	first := &creativeImageResponseBody{Reader: strings.NewReader(`{}`)}
	calls := 0
	encoded := base64.StdEncoding.EncodeToString([]byte("image"))
	target := oauthCreativeFixture(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return &http.Response{StatusCode: 404, Body: first}, nil
		}
		require.True(t, first.closed)
		require.Equal(t, "/backend-api/codex/responses", req.URL.Path)
		return creativeFixtureResponse(200, `data: {"type":"response.completed","response":{"output":[{"type":"image_generation_call","result":"`+encoded+`"}]}}`+"\n\n"), nil
	})
	outputs, err := target.ExecuteOpenAI(context.Background(), creative.CreativeRun{Operation: "generate", ImageSize: "1K"}, creative.CreativeRunPayload{Prompt: "cat"}, "gpt-image-2")
	require.NoError(t, err)
	require.Len(t, outputs, 1)
	require.Equal(t, 2, calls)
}

func TestCreativeOAuthMalformedSuccessDoesNotRegenerate(t *testing.T) {
	for _, body := range []string{`{"data":[]}`, `{"data":[{"b64_json":"%%%"}]}`, `not json`} {
		target := oauthCreativeFixture(t, func(*http.Request) (*http.Response, error) { return creativeFixtureResponse(200, body), nil })
		_, err := target.ExecuteOpenAI(context.Background(), creative.CreativeRun{Operation: "generate", ImageSize: "1K"}, creative.CreativeRunPayload{Prompt: "cat"}, "gpt-image-2")
		require.Error(t, err)
		require.False(t, creative.IsRetryableCreativeError(err))
	}
}

func TestCreativeOAuthSSEErrorIsSanitized(t *testing.T) {
	target := oauthCreativeFixture(t, func(*http.Request) (*http.Response, error) {
		return creativeFixtureResponse(200, `data: {"type":"response.failed","response":{"error":{"code":"content_policy_violation","message":"private prompt"}}}`+"\n\n"), nil
	})
	_, err := target.ExecuteOpenAI(context.Background(), creative.CreativeRun{Operation: "generate", ImageSize: "1K"}, creative.CreativeRunPayload{Prompt: "cat"}, "gpt-image-1")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "private")
	require.False(t, creative.IsRetryableCreativeError(err))
}

type creativeBrokenImageBody struct{ closed bool }

func (b *creativeBrokenImageBody) Read([]byte) (int, error) {
	return 0, errors.New("private transport information")
}
func (b *creativeBrokenImageBody) Close() error { b.closed = true; return nil }

// 已接受的请求在结果读取阶段失败时结束任务，响应体也要释放。
func TestCreativeOAuthResultReadFailure(t *testing.T) {
	body := &creativeBrokenImageBody{}
	calls := 0
	target := oauthCreativeFixture(t, func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: body}, nil
	})
	_, err := target.ExecuteOpenAI(context.Background(), creative.CreativeRun{Operation: "generate", ImageSize: "1K"}, creative.CreativeRunPayload{Prompt: "cat"}, "gpt-image-2")
	var resultErr *creative.CreativeUpstreamError
	require.ErrorAs(t, err, &resultErr)
	require.Equal(t, "IMAGE_RESPONSE_READ_FAILED", resultErr.Code)
	require.False(t, resultErr.Retryable)
	require.True(t, body.closed)
	require.Equal(t, 1, calls)
	require.NotContains(t, err.Error(), "private")
}

func TestCreativeOAuthCancellationReachesTransport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	target := oauthCreativeFixture(t, func(req *http.Request) (*http.Response, error) {
		calls++
		require.ErrorIs(t, req.Context().Err(), context.Canceled)
		return nil, req.Context().Err()
	})
	_, err := target.ExecuteOpenAI(ctx, creative.CreativeRun{Operation: "generate", ImageSize: "1K"}, creative.CreativeRunPayload{Prompt: "cat"}, "gpt-image-2")
	require.Error(t, err)
	require.Equal(t, 1, calls)
}

// API Key 仍使用管理员的 Base URL 和原 JSON/multipart 图片参数。
func TestCreativeAPIKeyPathDoesNotUseCodex(t *testing.T) {
	for _, operation := range []string{"generate", "edit", "inpaint"} {
		t.Run(operation, func(t *testing.T) {
			target := &Target{OpenAI: &OpenAIOptions{
				BuildOAuth: func(context.Context, creative.CreativeRun, []byte, string, string) (*http.Request, error) {
					t.Fatal("unexpected Codex request")
					return nil, nil
				},
				Token:   func(context.Context) (string, error) { return "api-key", nil },
				URL:     func(endpoint string) (string, error) { return "https://relay.example.com" + endpoint, nil },
				Prepare: func(req *http.Request) *http.Request { return req },
				AuthHeaders: func(context.Context, string) (http.Header, error) {
					return http.Header{"Authorization": []string{"Bearer api-key"}}, nil
				},
				ApplyHeaders: func(http.Header) {},
				Do: func(req *http.Request) (*http.Response, error) {
					require.Equal(t, "relay.example.com", req.URL.Host)
					require.Equal(t, "Bearer api-key", req.Header.Get("Authorization"))
					if operation == "generate" {
						var body map[string]any
						require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
						require.Equal(t, "gpt-image-2", body["model"])
					} else {
						require.Contains(t, req.Header.Get("Content-Type"), "multipart/form-data")
					}
					return creativeFixtureResponse(200, `{"data":[{"b64_json":"aW1hZ2U="}]}`), nil
				},
			}}
			outputs, err := target.ExecuteOpenAI(context.Background(), creative.CreativeRun{Operation: operation, ImageSize: "1K"}, creative.CreativeRunPayload{Prompt: "cat", Sources: []creative.CreativeInputImage{{Bytes: []byte("image"), Mime: "image/png"}}}, "gpt-image-2")
			require.NoError(t, err)
			require.Len(t, outputs, 1)
		})
	}
}
