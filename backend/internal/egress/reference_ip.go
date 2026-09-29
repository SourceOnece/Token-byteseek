package egress

// ReferenceIP 是脱敏的出站诊断结果，不携带代理 URL 或网络错误原文。
type ReferenceIP struct {
	CountryCode string `json:"country_code,omitempty"`
	Country     string `json:"country,omitempty"`
	Region      string `json:"region,omitempty"`
	City        string `json:"city,omitempty"`
	IP          string `json:"ip,omitempty"`
	Status      string `json:"status"`
	Source      string `json:"source,omitempty"`
	HTTPStatus  int    `json:"http_status,omitempty"`
}
