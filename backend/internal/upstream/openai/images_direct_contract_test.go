package openai

import (
	"bytes"
	"fmt"
	"github.com/TokenFlux/TokenRouter/internal/upstream"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

// 多图编辑、遮罩和压缩参数必须到达原生接口。
func TestDirectImagesMultipart(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range map[string]string{"model": "gpt-image-2.5-sunburst", "prompt": "编辑", "n": "2", "output_compression": "75"} {
		require.NoError(t, writer.WriteField(key, value))
	}
	for _, field := range []string{"image[]", "image[]", "mask"} {
		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename="image.png"`, field))
		header.Set("Content-Type", "image/png")
		p, err := writer.CreatePart(header)
		require.NoError(t, err)
		_, err = p.Write([]byte("png"))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	parsed, err := upstream.ParseImageRequest("/v1/images/edits", writer.FormDataContentType(), body.Bytes())
	require.NoError(t, err)
	payload, endpoint, err := BuildImagesOAuthPayload(parsed, parsed.Model)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(endpoint, "/images/edits"))
	require.Len(t, gjson.GetBytes(payload, "images").Array(), 2)
	require.Equal(t, "data:image/png;base64,cG5n", gjson.GetBytes(payload, "mask.image_url").String())
	require.EqualValues(t, 75, gjson.GetBytes(payload, "output_compression").Int())
	require.EqualValues(t, 2, gjson.GetBytes(payload, "n").Int())
}

// 原生 SSE 可先交付部分图片，再输出完成帧和计费数据。
func TestDirectImagesStreaming(t *testing.T) {
	sink := &imagesContractSink{}
	output := upstream.NewOutputContext(sink)
	response := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("data: {\"type\":\"image_generation.partial_image\",\"b64_json\":\"cGFydA==\"}\n\ndata: {\"type\":\"image_generation.completed\",\"b64_json\":\"ZmluYWw=\",\"output_format\":\"png\",\"usage\":{\"input_tokens\":2,\"output_tokens\":8}}\n\n"))}
	options := ImageResponseOptions{DirectModel: "gpt-image-2", ExpectedImages: 1, ResponseHeaders: func(http.Header, http.Header) {}, AdjustedWrittenSize: func() int { return 0 }, WrittenSize: func() int { return 0 }, StreamInterval: func() time.Duration { return 0 }, KeepaliveInterval: func() time.Duration { return 0 }, ClassifyReadError: func(err error) error { return err }, Logf: func(string, ...any) {}}
	usage, count, _, _, err := ReadImagesOAuthStreaming(response, output, options, time.Now(), "url", "image_generation", "gpt-image-2")
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Equal(t, 8, usage.ImageOutputTokens)
	text := strings.Join(sink.chunks, "")
	require.Contains(t, text, "image_generation.partial_image")
	require.Contains(t, text, "image_generation.completed")
	require.Contains(t, text, "data:image/png;base64,ZmluYWw=")
}
