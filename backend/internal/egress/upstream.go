package egress

import (
	"context"
	"net/http"

	"github.com/TokenFlux/TokenRouter/internal/pkg/tlsfingerprint"
)

// HTTPUpstream 是所有协议适配器共用的出站端口；账号、票据和网关只提交
// 代理、账号隔离和 TLS 指纹投影，不直接持有连接池或传输实现。
type HTTPUpstream interface {
	Do(*http.Request, string, int64, int) (*http.Response, error)
	DoWithTLS(*http.Request, string, int64, int, *tlsfingerprint.Profile) (*http.Response, error)
}

// HTTPUpstreamProfile 描述一次请求的传输用途，而不是业务协议。
type HTTPUpstreamProfile string

const (
	HTTPUpstreamProfileDefault       HTTPUpstreamProfile = ""
	HTTPUpstreamProfileOpenAI        HTTPUpstreamProfile = "openai"
	HTTPUpstreamProfileOpenAIHarvest HTTPUpstreamProfile = "openai_harvest"
	HTTPUpstreamProfileGrok          HTTPUpstreamProfile = "grok"
	HTTPUpstreamProfileLongStream    HTTPUpstreamProfile = "long_stream"
)

type upstreamProfileKey struct{}
type upstreamDisableRedirectKey struct{}
type upstreamPublicHostsKey struct{}

// WithHTTPUpstreamProfile 将出站用途写入本次请求上下文。
func WithHTTPUpstreamProfile(ctx context.Context, profile HTTPUpstreamProfile) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if profile == HTTPUpstreamProfileDefault {
		return ctx
	}
	return context.WithValue(ctx, upstreamProfileKey{}, profile)
}

// HTTPUpstreamProfileFromContext 只接受已知 profile，未知值回退普通连接策略。
func HTTPUpstreamProfileFromContext(ctx context.Context) HTTPUpstreamProfile {
	if ctx == nil {
		return HTTPUpstreamProfileDefault
	}
	profile, ok := ctx.Value(upstreamProfileKey{}).(HTTPUpstreamProfile)
	if !ok {
		return HTTPUpstreamProfileDefault
	}
	switch profile {
	case HTTPUpstreamProfileOpenAI, HTTPUpstreamProfileOpenAIHarvest, HTTPUpstreamProfileGrok, HTTPUpstreamProfileLongStream:
		return profile
	default:
		return HTTPUpstreamProfileDefault
	}
}

// WithHTTPUpstreamRedirectsDisabled 禁止携带凭据的探测请求跟随重定向。
func WithHTTPUpstreamRedirectsDisabled(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, upstreamDisableRedirectKey{}, true)
}

func HTTPUpstreamRedirectsDisabled(ctx context.Context) bool {
	return ctx != nil && ctx.Value(upstreamDisableRedirectKey{}) == true
}

// WithHTTPUpstreamPublicHostsOnly 标记不可信下载必须逐跳验证公网地址。
func WithHTTPUpstreamPublicHostsOnly(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, upstreamPublicHostsKey{}, true)
}

func HTTPUpstreamPublicHostsOnly(ctx context.Context) bool {
	return ctx != nil && ctx.Value(upstreamPublicHostsKey{}) == true
}
