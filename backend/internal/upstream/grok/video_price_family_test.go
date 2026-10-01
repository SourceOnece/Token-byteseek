package grok

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanonicalGrokImagineVideoPriceFamily(t *testing.T) {
	t.Parallel()
	require.Equal(t, "grok-imagine-video", CanonicalImagineVideoModel("grok-imagine-video"))
	require.Equal(t, "grok-imagine-video-1.5", CanonicalImagineVideoModel("grok-imagine-video-1.5"))
	require.Equal(t, "grok-imagine-video-1.5-preview", CanonicalImagineVideoModel("grok-imagine-video-1.5-preview"))
	require.Equal(t, "xai/grok-video-1.5", CanonicalImagineVideoModel("xai/grok-video-1.5"))
	require.Equal(t, "grok-imagine-video-2", CanonicalImagineVideoModel("grok-imagine-video-2"))
	require.Equal(t, "xai/grok-imagine-video-2", CanonicalImagineVideoModel("xai/grok-imagine-video-2"))
}
