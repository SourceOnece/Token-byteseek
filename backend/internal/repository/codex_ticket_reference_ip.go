package repository

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/pkg/httpclient"
	"github.com/TokenFlux/TokenRouter/internal/service"
)

// 手动采集用 HTTPS 参考探测，不改全站代理测试默认值，不带 OAuth 头且不直连兜底。
func (s *proxyProbeService) ProbeTicketReferenceIP(ctx context.Context, proxyURL string) service.CodexTicketReferenceIP {
	if strings.TrimSpace(proxyURL) == "" {
		return service.CodexTicketReferenceIP{Status: "proxy_config"}
	}
	base, err := httpclient.GetClient(httpclient.Options{ProxyURL: proxyURL, Timeout: 5 * time.Second, InsecureSkipVerify: s.insecureSkipVerify, ValidateResolvedIP: s.validateResolvedIP, AllowPrivateHosts: s.allowPrivateHosts})
	if err != nil {
		return service.CodexTicketReferenceIP{Status: "proxy_config"}
	}
	client := *base
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return s.probeTicketReferenceTargets(ctx, &client)
}

func (s *proxyProbeService) probeTicketReferenceTargets(ctx context.Context, client *http.Client) service.CodexTicketReferenceIP {
	targets := s.configuredProbeURLs
	custom := len(targets) > 0
	if !custom {
		targets = []configuredProbeTarget{{"https://chatgpt.com/cdn-cgi/trace", "chatgpt-trace"}, {"https://api64.ipify.org?format=json", "ipify"}}
	}
	result := service.CodexTicketReferenceIP{Status: "unavailable"}
	for _, target := range targets {
		if ctx.Err() != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				result.Status = "cancelled"
			} else {
				result.Status = "timeout"
			}
			return result
		}
		attemptCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		result = s.probeTicketReferenceTarget(attemptCtx, client, target)
		cancel()
		if custom {
			result.Source = "configured"
		}
		if result.Status == "reference" {
			return result
		}
	}
	return result
}

func (s *proxyProbeService) probeTicketReferenceTarget(ctx context.Context, client *http.Client, target configuredProbeTarget) service.CodexTicketReferenceIP {
	result := service.CodexTicketReferenceIP{Status: "network", Source: "ipify"}
	if target.parser == "chatgpt-trace" {
		result.Source = "chatgpt_trace"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.url, nil)
	if err != nil {
		result.Status = "proxy_config"
		return result
	}
	req.Close = true
	resp, err := client.Do(req)
	if err != nil {
		var network net.Error
		var certificate *tls.CertificateVerificationError
		var authority x509.UnknownAuthorityError
		switch {
		case errors.Is(err, context.Canceled):
			result.Status = "cancelled"
		case errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &network) && network.Timeout()):
			result.Status = "timeout"
		case errors.As(err, &certificate) || errors.As(err, &authority):
			result.Status = "tls"
		}
		return result
	}
	defer resp.Body.Close()
	result.HTTPStatus = resp.StatusCode
	if resp.StatusCode != http.StatusOK {
		result.Status = "http_error"
		return result
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8193))
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result.Status = "timeout"
		}
		return result
	}
	result.Status = "invalid_response"
	if len(body) > 8192 {
		return result
	}
	var info *service.ProxyExitInfo
	switch target.parser {
	case "chatgpt-trace":
		info, _, err = s.parseChatGPTTrace(body, 0)
	case "ipify":
		info, _, err = s.parseIPify(body, 0)
	case "ip-api":
		info, _, err = s.parseIPAPI(body, 0)
	default:
		return result
	}
	if err != nil || info == nil {
		return result
	}
	ip, err := netip.ParseAddr(strings.TrimSpace(info.IP))
	if err != nil {
		return result
	}
	result.IP = ip.String()
	result.Status = "reference"
	return result
}
