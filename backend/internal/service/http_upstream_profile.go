package service

import "github.com/TokenFlux/TokenRouter/internal/egress"

// 旧 service 名称保留为别名，已有调用方不改接口；实际所有权已移到 egress。
type HTTPUpstreamProfile = egress.HTTPUpstreamProfile

const (
	HTTPUpstreamProfileDefault       = egress.HTTPUpstreamProfileDefault
	HTTPUpstreamProfileOpenAI        = egress.HTTPUpstreamProfileOpenAI
	HTTPUpstreamProfileOpenAIHarvest = egress.HTTPUpstreamProfileOpenAIHarvest
	HTTPUpstreamProfileGrok          = egress.HTTPUpstreamProfileGrok
	HTTPUpstreamProfileLongStream    = egress.HTTPUpstreamProfileLongStream
)

var WithHTTPUpstreamProfile = egress.WithHTTPUpstreamProfile
var HTTPUpstreamProfileFromContext = egress.HTTPUpstreamProfileFromContext
var WithHTTPUpstreamRedirectsDisabled = egress.WithHTTPUpstreamRedirectsDisabled
var HTTPUpstreamRedirectsDisabled = egress.HTTPUpstreamRedirectsDisabled
var WithHTTPUpstreamPublicHostsOnly = egress.WithHTTPUpstreamPublicHostsOnly
var HTTPUpstreamPublicHostsOnly = egress.HTTPUpstreamPublicHostsOnly
