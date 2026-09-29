package httpapi

import (
	"github.com/TokenFlux/TokenRouter/internal/billing"
	billingdto "github.com/TokenFlux/TokenRouter/internal/billing/httpapi/dto"
)

// RedeemCodeFromService 委托所属模块的唯一实现。
func RedeemCodeFromService(rc *billing.RedeemCode) *RedeemCode {
	return billingdto.RedeemCodeFromService(rc)
}

// RedeemCodeFromServiceAdmin 委托所属模块的唯一实现。
func RedeemCodeFromServiceAdmin(rc *billing.RedeemCode) *AdminRedeemCode {
	return billingdto.RedeemCodeFromServiceAdmin(rc)
}
