package provider

import (
	"errors"
	"fmt"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
)

// GrokStoredAccessToken 仅读取原搜索所用凭据；刷新仍由提供商协调器按原路径执行。
func GrokStoredAccessToken(value *Record) (string, error) {
	switch value.Type {
	case capability.ProviderTypeOAuth:
		token := value.GetGrokAccessToken()
		if token == "" {
			return "", errors.New("grok access_token not found in credentials")
		}
		return token, nil
	case capability.ProviderTypeSetupToken:
		token := value.GetCredential("access_token")
		if token == "" {
			return "", errors.New("access_token not found in credentials")
		}
		return token, nil
	case capability.ProviderTypeAPIKey:
		token := value.GetCredential("api_key")
		if token == "" {
			return "", errors.New("api_key not found in credentials")
		}
		return token, nil
	case capability.ProviderTypeBedrock:
		return "", nil
	case capability.ProviderTypeServiceAccount:
		return "", fmt.Errorf("unsupported service account platform: %s", value.Platform)
	default:
		return "", fmt.Errorf("unsupported provider type: %s", value.Type)
	}
}
