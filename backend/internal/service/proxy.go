package service

import "github.com/TokenFlux/TokenRouter/internal/egress"

// 兼容旧业务类型名，代理的唯一实现由egress拥有。
type Proxy = egress.Proxy

const (
	FallbackModeNone   = egress.FallbackModeNone
	FallbackModeProxy  = egress.FallbackModeProxy
	FallbackModeDirect = egress.FallbackModeDirect
)

type ProxyWithAccountCount struct {
	Proxy
	AccountCount   int64
	LatencyMs      *int64
	LatencyStatus  string
	LatencyMessage string
	IPAddress      string
	Country        string
	CountryCode    string
	Region         string
	City           string
	QualityStatus  string
	QualityScore   *int
	QualityGrade   string
	QualitySummary string
	QualityChecked *int64
}

type ProxyAccountSummary struct {
	ID       int64
	Name     string
	Platform string
	Type     string
	Notes    *string
}
