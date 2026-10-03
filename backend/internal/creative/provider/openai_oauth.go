package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/creative"
	"github.com/TokenFlux/TokenRouter/internal/upstream"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/tidwall/gjson"
)

// executeOpenAIOAuth 使用图片网关的 Codex 请求和响应转换器。
// 原生端点返回 404/405 时，同一任务可尝试一次 Responses 图片工具。
func (e *Target) executeOpenAIOAuth(ctx context.Context, run creative.CreativeRun, payload creative.CreativeRunPayload, model string) ([]creative.CreativeOutput, error) {
	if e.OpenAI.BuildOAuth == nil {
		return nil, creative.CreativeNonRetryableError("creative Codex image transport is not configured")
	}
	parsed := creativeOAuthImageRequest(run, payload, model)
	body, targetURL, err := openai.BuildImagesOAuthPayload(parsed, model)
	if err != nil {
		return nil, creative.CreativeNonRetryableError("creative image parameters are invalid")
	}
	token, err := e.OpenAI.Token(ctx)
	if err != nil {
		return nil, creative.CreativeHTTPStatusError(0, "")
	}
	direct := openai.UsesCodexDirectImages(model)
	for attempt := 0; attempt < 2; attempt++ {
		req, err := e.OpenAI.BuildOAuth(ctx, run, body, token, targetURL)
		if err != nil {
			return nil, creative.CreativeHTTPStatusError(0, "")
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("OpenAI-Beta", "responses=experimental")
		req.Header.Set("Accept", "text/event-stream")
		if direct {
			req.Header.Set("Accept", "application/json")
		}
		resp, err := e.OpenAI.Do(req)
		if err != nil {
			return nil, creative.CreativeHTTPStatusError(0, "")
		}
		limit := int64(64 << 20)
		if resp.StatusCode >= 400 {
			limit = 2 << 20
		}
		responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, limit+1))
		_ = resp.Body.Close()
		if direct && attempt == 0 && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed) && ctx.Err() == nil {
			body, err = openai.BuildOpenAIImagesResponsesRequest(parsed, model)
			if err != nil {
				return nil, creative.CreativeNonRetryableError("creative image parameters are invalid")
			}
			targetURL = "https://chatgpt.com/backend-api/codex/responses"
			direct = false
			continue
		}
		if resp.StatusCode >= 400 {
			return nil, creative.CreativeHTTPStatusError(resp.StatusCode, "")
		}
		if readErr != nil || int64(len(responseBody)) > limit {
			// 请求已被接受，读取结果失败结束任务，防止再次生成并消耗额度。
			return nil, creative.CreativeImageResultError("IMAGE_RESPONSE_READ_FAILED", "creative image response could not be read")
		}
		return e.parseOAuthImageOutputs(ctx, responseBody, direct)
	}
	return nil, creative.CreativeHTTPStatusError(http.StatusBadGateway, "")
}

// creativeOAuthImageRequest 将创作台参数转换为网关图片输入。
func creativeOAuthImageRequest(run creative.CreativeRun, payload creative.CreativeRunPayload, model string) *upstream.ImageRequest {
	parsed := &upstream.ImageRequest{
		Endpoint: upstream.OpenAIImagesGenerationsEndpoint, Model: model, Prompt: payload.Prompt,
		N: 1, Size: CreativeOpenAIImageSize(run.ImageSize, run.AspectRatio),
		Quality: payload.Quality, Background: payload.Background, OutputFormat: "png", ResponseFormat: "b64_json",
	}
	if run.Operation != creative.CreativeOperationGenerate {
		parsed.Endpoint = upstream.OpenAIImagesEditsEndpoint
		parsed.Multipart = true
	}
	for i, image := range payload.Sources {
		parsed.Uploads = append(parsed.Uploads, upstream.ImageUpload{FileName: fmt.Sprintf("source_%d.%s", i, CreativeFileExtension(image.Mime)), ContentType: image.Mime, Data: image.Bytes})
	}
	if payload.Mask != nil {
		parsed.HasMask = true
		parsed.MaskUpload = &upstream.ImageUpload{FileName: "mask.png", ContentType: payload.Mask.Mime, Data: payload.Mask.Bytes}
	}
	return parsed
}

// parseOAuthImageOutputs 接受原生 Images JSON 或 Responses 图片事件，按现有上游错误分类返回。
func (e *Target) parseOAuthImageOutputs(ctx context.Context, body []byte, direct bool) ([]creative.CreativeOutput, error) {
	var results []openai.OpenAIResponsesImageResult
	var err error
	if direct && gjson.ValidBytes(body) {
		results, err = openai.ParseDirectImagesResponse(body)
		// 部分兼容上游交付 URL，交由同一受限下载器读取。
		if err != nil && !gjson.GetBytes(body, "error").Exists() && gjson.GetBytes(body, "data.0.url").String() != "" {
			return e.parseOpenAIImageOutputs(ctx, body)
		}
	} else {
		if failure := openai.ExtractOpenAIImagesUpstreamError(body); failure != nil {
			return nil, creative.CreativeHTTPStatusError(failure.ClientStatusCode(), "")
		}
		results, _, _, _, _, err = openai.CollectOpenAIImagesFromResponsesBody(body, time.Now)
	}
	if err != nil {
		var failure *openai.OpenAIImagesUpstreamError
		if errors.As(err, &failure) && gjson.GetBytes(body, "error").Exists() {
			return nil, creative.CreativeHTTPStatusError(failure.ClientStatusCode(), "")
		}
		return nil, creative.CreativeImageResultError("INVALID_IMAGE_RESPONSE", "creative platform returned invalid image response")
	}
	for _, item := range results {
		decoded, decodeErr := DecodeBase64Image(strings.TrimSpace(item.Result))
		if decodeErr == nil && len(decoded.Bytes) > 0 {
			return []creative.CreativeOutput{{Index: 0, Bytes: decoded.Bytes, Mime: decoded.Mime}}, nil
		}
	}
	return nil, creative.CreativeImageResultError("INVALID_IMAGE_RESPONSE", "creative platform returned no decodable image output")
}
