package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/payment"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
)

func TestRedactAuditBody_JSONRedactsSecrets(t *testing.T) {
	redactor := provideAuditRedactor()
	raw := []byte(`{
		"name": "acc1",
		"base_url": "https://evil.example.com",
		"credentials": {"api_key": "sk-secret-123", "base_url": "https://evil.example.com"},
		"credential": "google-id-token-canary",
		"new_password": "hunter2",
		"totp_code": "123456",
		"nested": [{"access_token": "tok_abc"}]
	}`)
	out := redactor.RedactBody(raw, "application/json")

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}

	// 敏感字段被擦除。
	for _, secret := range []string{"sk-secret-123", "google-id-token-canary", "hunter2", "123456", "tok_abc"} {
		if strings.Contains(out, secret) {
			t.Fatalf("redacted body still contains secret %q: %s", secret, out)
		}
	}
	// 非敏感字段（base_url、name）保留以便追责。
	if !strings.Contains(out, "evil.example.com") {
		t.Fatalf("base_url should be preserved for providerability: %s", out)
	}
	if !strings.Contains(out, "acc1") {
		t.Fatalf("name should be preserved: %s", out)
	}
}

// 打票代理使用独立的只写字段，用户名和密码均不可落到管理审计正文。
func TestRedactAuditBody_CodexHarvestProxy(t *testing.T) {
	out := provideAuditRedactor().RedactBody([]byte(`{"enabled":true,"harvest_proxy_url":"socks5h://test-user:proxy-secret@host:1080"}`), "application/json")
	if strings.Contains(out, "test-user") || strings.Contains(out, "proxy-secret") || strings.Contains(out, "host:1080") {
		t.Fatalf("代理凭据未被完整脱敏: %s", out)
	}
	if !strings.Contains(out, `"enabled":true`) {
		t.Fatalf("开关的非敏感状态应保留: %s", out)
	}
}

// 多代理的只写 URL 仍以同一键递归脱敏，不因放进数组而漏出密码。
func TestRedactAuditBody_CodexHarvestProxyList(t *testing.T) {
	out := provideAuditRedactor().RedactBody([]byte(`{"proxies":[{"id":"x","name":"A","harvest_proxy_url":"http://user:secret-a@host"},{"name":"B","harvest_proxy_url":"http://user:secret-b@host"}]}`), "application/json")
	if strings.Contains(out, "secret-a") || strings.Contains(out, "secret-b") || strings.Contains(out, "@host") {
		t.Fatalf("代理列表未脱敏: %s", out)
	}
}

// 账号批量编辑的嵌套patch同样整字段脱敏，不留下动态代理用户名/地址。
func TestRedactAuditBody_CodexAccountProxyPatch(t *testing.T) {
	out := provideAuditRedactor().RedactBody([]byte(`{"account_ids":[1,2],"patch":{"harvest_proxy_url":"socks5h://private-user:secret-canary@proxy.invalid:1080","mode":"on"}}`), "application/json")
	for _, v := range []string{"private-user", "secret-canary", "proxy.invalid"} {
		if strings.Contains(out, v) {
			t.Fatalf("账号代理未脱敏")
		}
	}
}

func TestRedactAuditBody_CodexProxyExtractionURL(t *testing.T) {
	out := provideAuditRedactor().RedactBody([]byte(`{"policy":{"extraction_url":"https://provider.invalid/gen?user=private&pass=private-canary"}}`), "application/json")
	if strings.Contains(out, "private-canary") || strings.Contains(out, "provider.invalid") {
		t.Fatal("取号URL不得泄漏到审计")
	}
}

// 裸键 "session"（Ollama Cloud 会话保存的请求体字段）值整体就是浏览器 Cookie 明文，
// 必须命中键级脱敏；session_id 等运行态标识不受影响，保留以便追责。
func TestRedactAuditBody_BareSessionKeyRedacted(t *testing.T) {
	redactor := provideAuditRedactor()
	raw := []byte(`{"session": "wos-session=cookie-canary", "session_id": "sid-visible"}`)
	out := redactor.RedactBody(raw, "application/json")

	if strings.Contains(out, "cookie-canary") {
		t.Fatalf("redacted body still contains the session cookie: %s", out)
	}
	if !strings.Contains(out, "sid-visible") {
		t.Fatalf("session_id should be preserved for providerability: %s", out)
	}
}

// TestRedactAuditBody_AuthoritativeTablesSynced 覆盖曾经漏网的凭证字段：
// 提供商 credentials 敏感子键、支付渠道无分隔符密钥、字符串值内嵌凭证的 proxy_key / custom_key，
// 以及 camelCase 等命名变体（归一化比对）。
func TestRedactAuditBody_AuthoritativeTablesSynced(t *testing.T) {
	redactor := provideAuditRedactor()
	raw := []byte(`{
		"credentials": {
			"session_key": "sk-session-aaa",
			"service_account_json": "{\"private_key\":\"pem-body-bbb\"}",
			"service_account": "sa-blob-ccc"
		},
		"proxy_key": "socks5|1.2.3.4|1080|proxyuser|proxypass-ddd",
		"custom_key": "sk-custom-eee",
		"config": {
			"pkey": "easypay-merchant-fff",
			"privateKey": "alipay-pem-ggg",
			"apiv3key": "wxpay-v3-hhh",
			"SecretKey": "stripe-sk-iii",
			"webhookSecret": "whsec-jjj"
		},
		"provider_key": "stripe",
		"name": "instance-1"
	}`)
	out := redactor.RedactBody(raw, "application/json")

	for _, secret := range []string{
		"sk-session-aaa", "pem-body-bbb", "sa-blob-ccc",
		"proxypass-ddd", "sk-custom-eee",
		"easypay-merchant-fff", "alipay-pem-ggg", "wxpay-v3-hhh",
		"stripe-sk-iii", "whsec-jjj",
	} {
		if strings.Contains(out, secret) {
			t.Fatalf("redacted body still contains secret %q: %s", secret, out)
		}
	}
	// provider_key 是渠道标识而非密钥，必须保留以便追责。
	if !strings.Contains(out, `"provider_key":"stripe"`) {
		t.Fatalf("provider_key should be preserved for providerability: %s", out)
	}
	if !strings.Contains(out, "instance-1") {
		t.Fatalf("name should be preserved: %s", out)
	}
}

// SensitiveCredentialKeys 中的每个键都必须被审计脱敏判定命中（防两表漂移的守卫）。
func TestAuditSensitiveKeys_CoverCredentialTable(t *testing.T) {
	redactor := provideAuditRedactor()
	for _, k := range providercore.SensitiveCredentialKeys {
		if !redactor.IsSensitiveKey(k) {
			t.Fatalf("credential key %q is not covered by audit redaction", k)
		}
	}
	for provider, fields := range payment.ConfigProviderSensitiveConfigFields {
		for k := range fields {
			if !redactor.IsSensitiveKey(k) {
				t.Fatalf("payment provider %q sensitive field %q is not covered by audit redaction", provider, k)
			}
		}
	}
}

func TestRedactAuditBody_NonJSONOmitted(t *testing.T) {
	redactor := provideAuditRedactor()
	out := redactor.RedactBody([]byte("username=admin&password=secret"), "application/x-www-form-urlencoded")
	if strings.Contains(out, "secret") {
		t.Fatalf("non-json body must not leak content: %s", out)
	}
	if !strings.Contains(out, "omitted") {
		t.Fatalf("expected omission marker, got: %s", out)
	}
}

func TestRedactAuditBody_Empty(t *testing.T) {
	redactor := provideAuditRedactor()
	if got := redactor.RedactBody(nil, "application/json"); got != "" {
		t.Fatalf("expected empty for nil body, got %q", got)
	}
}
