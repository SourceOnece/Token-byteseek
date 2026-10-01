package site

// 主题是站点级展示设置，不参与鉴权、模型路由和账号调度。
const (
	SettingKeySiteTheme = "site_theme"
	SiteThemeTokenFlux  = "tokenflux"
	SiteThemeBauhaus    = "bauhaus"
)

// NormalizeSiteTheme 为缺失或历史无效值提供稳定默认；写入仍需严格校验。
func NormalizeSiteTheme(value string) string {
	if value == SiteThemeBauhaus {
		return SiteThemeBauhaus
	}
	return SiteThemeTokenFlux
}
