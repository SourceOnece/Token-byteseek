package postgres

const (
	OllamaCloudBaseURLRegexSQL       = `^[hH][tT][tT][pP][sS]://([wW][wW][wW]\.)?[oO][lL][lL][aA][mM][aA]\.[cC][oO][mM](:443)?(/v1)?$`
	OllamaCloudBaseURLMatchSQLPrefix = "btrim("
	OllamaCloudBaseURLMatchSQLSuffix = ") ~ '" + OllamaCloudBaseURLRegexSQL + "'"
	OllamaCloudUsageEligibleSQL      = `
	platform IN ('openai', 'anthropic')
	AND type = 'apikey'
	AND ` + OllamaCloudBaseURLMatchSQLPrefix + `credentials ->> 'base_url'` + OllamaCloudBaseURLMatchSQLSuffix + `
	AND jsonb_typeof(credentials -> 'api_key') = 'string'
`
)

func OllamaCloudBaseURLMatchesSQL(expression string) string {
	return OllamaCloudBaseURLMatchSQLPrefix + expression + OllamaCloudBaseURLMatchSQLSuffix
}
