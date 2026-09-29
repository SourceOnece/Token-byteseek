package provider

// ProviderRefreshPlatformPolicy 组合提供商已有的刷新资格与错误快照，不持有平台客户端。
func ProviderRefreshPlatformPolicy() RefreshPlatformPolicy {
	return RefreshPlatformPolicy{
		Eligibility: GrokOAuthRequestProviderEligibilityError,
		MissingRefreshToken: func() error {
			return ErrGrokOAuthRefreshTokenMissing
		},
		SnapshotError: WithGrokCredentialFailureSnapshot,
		ConfigurationError: func(err error) error {
			return &ProviderConfigurationRefreshError{Cause: err}
		},
		ContainmentError: func(err error) error {
			return &ProviderCycleContainmentRefreshError{Cause: err}
		},
	}
}
