package provider

import (
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
)

// DecodeArchiveIDToken 沿用原 OpenAI 导入的非认证解码，不新增过期或签名验证。
func DecodeArchiveIDToken(token string) (*provider.ArchiveIdentityHints, error) {
	claims, err := openai.DecodeIDToken(token)
	if err != nil {
		return nil, err
	}
	info := claims.GetUserInfo()
	if info == nil {
		return nil, nil
	}
	return &provider.ArchiveIdentityHints{Email: info.Email, PlanType: info.PlanType, ChatGPTAccountID: info.ChatGPTAccountID, ChatGPTUserID: info.ChatGPTUserID, OrganizationID: info.OrganizationID}, nil
}
