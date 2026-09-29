package egress

import (
	"strconv"
)

// TLSSelection 只投影提供商资格与本次 Router 命中，不接收提供商或平台服务。
type TLSSelection struct {
	Enabled                   bool
	DirectProfileID           int64
	RouterMatched             bool
	RouterID, RouterProfileID int64
}

// ResolveRequestPolicy 保留 Router -> 提供商绑定 -> 内置默认的选择顺序。
func (s *TLSFingerprintProfileService) ResolveRequestPolicy(input TLSSelection) EgressPolicy {
	if input.RouterMatched {
		if p, ok := s.ResolveRoutableTLSProfileByID(input.Enabled, input.RouterProfileID); ok {
			return RequestPolicy(RequestPolicyInput{TLSProfile: p})
		}
	}
	return RequestPolicy(RequestPolicyInput{TLSProfile: s.ResolveTLSProfileByID(input.Enabled, input.DirectProfileID)})
}

// WebSocketTLSIdentity 保留原稳定配置键；随机模板不能令 continuation 每轮换池。
// Router 命中但模板缺失时仍保留原 Router 键语义，不按实际回退结果改键。
func WebSocketTLSIdentity(input TLSSelection, hasProfile bool, profileCacheKey string) string {
	if !hasProfile {
		return ""
	}
	if input.RouterMatched {
		if input.RouterProfileID == -1 {
			return "tls-router-random"
		}
		return "tls-router-" + strconv.FormatInt(input.RouterID, 10) + "-" + strconv.FormatInt(input.RouterProfileID, 10)
	}
	if input.DirectProfileID == -1 {
		return "tls-random"
	}
	return profileCacheKey
}
