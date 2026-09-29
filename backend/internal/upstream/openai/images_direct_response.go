package openai

import (
	"fmt"
	wire "github.com/TokenFlux/TokenRouter/internal/protocol/openai"
	"github.com/TokenFlux/TokenRouter/internal/upstream"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/http"
	"strings"
)

// DirectImagesUsage 只采信上游明细，不从总缓存量猜测图片缓存占比。
func DirectImagesUsage(body []byte) (wire.ForwardUsage, bool) {
	v := gjson.GetBytes(body, "usage")
	u, ok := wire.OpenAIUsageFromGJSON(v)
	if !ok {
		return u, false
	}
	if !v.Get("output_tokens_details.image_tokens").Exists() {
		u.ImageOutputTokens = u.OutputTokens
	}
	if cached := v.Get("input_tokens_details.cached_tokens_details"); !v.Get("input_tokens_details.cached_tokens").Exists() && cached.IsObject() {
		images, _ := wire.BoundedJSONNonNegativeInt(cached.Get("image_tokens"))
		texts, _ := wire.BoundedJSONNonNegativeInt(cached.Get("text_tokens"))
		u.CacheReadInputTokens = min(images, max(u.InputTokens, 0))
		u.CacheReadInputTokens += min(texts, max(u.InputTokens-u.CacheReadInputTokens, 0))
	}
	imageCached, _ := wire.BoundedJSONNonNegativeInt(v.Get("input_tokens_details.cached_tokens_details.image_tokens"))
	u.ImageCacheReadTokens = min(imageCached, max(u.ImageInputTokens, 0), max(u.CacheReadInputTokens, 0))
	return u, true
}

// ParseDirectImagesResponse 与后台单号测试共用原生响应校验。
func ParseDirectImagesResponse(body []byte) ([]OpenAIResponsesImageResult, error) {
	if !gjson.ValidBytes(body) {
		return nil, fmt.Errorf("invalid Images API JSON response")
	}
	root := gjson.ParseBytes(body)
	if e := OpenAIImagesUpstreamErrorFromGJSON(root.Get("error"), ""); e != nil {
		return nil, e
	}
	var images []OpenAIResponsesImageResult
	for _, item := range root.Get("data").Array() {
		image := directImageMeta(item, root, item.Get("model").String())
		if strings.TrimSpace(image.Result) != "" {
			images = append(images, image)
		}
	}
	if len(images) == 0 {
		return nil, &OpenAIImagesUpstreamError{StatusCode: 502, ErrorType: "upstream_error", Message: "Images API returned no image output"}
	}
	return images, nil
}

func directImageMeta(item, root gjson.Result, model string) OpenAIResponsesImageResult {
	meta := func(key string) string {
		if v := item.Get(key).String(); v != "" {
			return v
		}
		return root.Get(key).String()
	}
	result := OpenAIResponsesImageResult{Result: item.Get("b64_json").String(), RevisedPrompt: item.Get("revised_prompt").String(), Model: model, OutputFormat: meta("output_format"), Size: meta("size"), Quality: meta("quality"), Background: meta("background")}
	if size := DetectOpenAIImageResultSize(result.Result); size != "" {
		result.Size = size
	}
	return result
}

// WriteDirectImagesJSON 在交付前校验真实图片输出，健康与重试仍由统一执行器决定。
func WriteDirectImagesJSON(resp *http.Response, sink upstream.OutputSink, options ImageResponseOptions, body []byte) (wire.ForwardUsage, int, []string, error) {
	root := gjson.ParseBytes(body)
	if options.ObserveResponseModel != nil {
		options.ObserveResponseModel(root.Get("model").String())
	}
	if e := OpenAIImagesUpstreamErrorFromGJSON(root.Get("error"), resp.Header.Get("x-request-id")); e != nil {
		return wire.ForwardUsage{}, 0, nil, e
	}
	usage, _ := DirectImagesUsage(body)
	var sizes []string
	count := 0
	for i, item := range root.Get("data").Array() {
		if options.ObserveResponseModel != nil {
			options.ObserveResponseModel(item.Get("model").String())
		}
		img := directImageMeta(item, root, options.DirectModel)
		if strings.TrimSpace(img.Result) == "" {
			continue
		}
		count++
		if img.Size != "" {
			sizes = append(sizes, img.Size)
			body, _ = sjson.SetBytes(body, fmt.Sprintf("data.%d.size", i), img.Size)
			if count == 1 {
				body, _ = sjson.SetBytes(body, "size", img.Size)
			}
		}
		body, _ = sjson.SetBytes(body, fmt.Sprintf("data.%d.model", i), options.DirectModel)
		if options.DirectFormat == "url" {
			format := img.OutputFormat
			if format == "" {
				format = options.DirectOutputFormat
			}
			body, _ = sjson.SetBytes(body, fmt.Sprintf("data.%d.url", i), "data:"+OpenAIImageOutputMIMEType(format)+";base64,"+img.Result)
			body, _ = sjson.DeleteBytes(body, fmt.Sprintf("data.%d.b64_json", i))
		}
	}
	if count == 0 {
		return usage, 0, nil, options.EmptyOutput(body)
	}
	body, _ = sjson.SetBytes(body, "model", options.DirectModel)
	c := upstream.NewOutputContext(sink)
	options.ResponseHeaders(c.Writer.Header(), resp.Header)
	c.Data(resp.StatusCode, "application/json", body)
	return usage, count, sizes, nil
}
