package app

import (
	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/gateway/admission"
	"github.com/TokenFlux/TokenRouter/internal/infra/telemetry/logging"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
)

// provideFundingAdmission 复用唯一权益缓存和现有 RPM 存储，资金检查通过后才累计 RPM。
func provideFundingAdmission(funds *billing.Eligibility, rpm scheduler.UserRPMCache, rates billing.UserGroupRateRepository) *admission.FundingAdmission {
	limiter := scheduler.NewRPMAdmission(rpm, rates, scheduler.Diagnostics{Logf: logging.LegacyPrintf})
	return admission.NewFundingAdmission(funds, limiter)
}
