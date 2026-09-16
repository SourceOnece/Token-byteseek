package service

import (
	"context"
	"testing"
)

func TestWithHTTPUpstreamProfile_DefaultKeepsContext(t *testing.T) {
	ctx := context.Background()
	got := WithHTTPUpstreamProfile(ctx, HTTPUpstreamProfileDefault)
	if got != ctx {
		t.Fatal("default profile should not wrap context")
	}
}

func TestWithHTTPUpstreamProfile_OpenAI(t *testing.T) {
	ctx := WithHTTPUpstreamProfile(context.TODO(), HTTPUpstreamProfileOpenAI)
	if profile := HTTPUpstreamProfileFromContext(ctx); profile != HTTPUpstreamProfileOpenAI {
		t.Fatalf("expected profile %q, got %q", HTTPUpstreamProfileOpenAI, profile)
	}
}

// 长流 profile 必须在上下文读取时保留，否则路由到默认 transport 后保活失效。
func TestWithHTTPUpstreamProfile_LongStream(t *testing.T) {
	ctx := WithHTTPUpstreamProfile(context.Background(), HTTPUpstreamProfileLongStream)
	if got := HTTPUpstreamProfileFromContext(ctx); got != HTTPUpstreamProfileLongStream {
		t.Fatalf("long stream profile lost: %s", got)
	}
}

func TestWithHTTPUpstreamRedirectsDisabled(t *testing.T) {
	//nolint:staticcheck // 验证 nil context 的防御性回退逻辑。
	ctx := WithHTTPUpstreamRedirectsDisabled(nil)
	if !HTTPUpstreamRedirectsDisabled(ctx) {
		t.Fatal("expected redirects to be disabled")
	}
	if HTTPUpstreamRedirectsDisabled(context.Background()) {
		t.Fatal("redirects should remain enabled by default")
	}
}
