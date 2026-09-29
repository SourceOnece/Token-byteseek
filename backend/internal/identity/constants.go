// 身份模块集中定义 API Key 数量边界和第三方登录的保留邮箱域名。
package identity

const (
	DefaultUserAPIKeyLimit              = 100
	MaxUserAPIKeyLimit                  = 2_147_483_647
	LinuxDoConnectSyntheticEmailDomain  = "@linuxdo-connect.invalid"
	OIDCConnectSyntheticEmailDomain     = "@oidc-connect.invalid"
	WeChatConnectSyntheticEmailDomain   = "@wechat-connect.invalid"
	DingTalkConnectSyntheticEmailDomain = "@dingtalk-connect.invalid"
)
