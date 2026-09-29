// 兑换接口共享 DTO 定义，避免请求字段与展示结构分叉。
package httpapi

import (
	billingdto "github.com/TokenFlux/TokenRouter/internal/billing/httpapi/dto"
)

type RedeemCode = billingdto.RedeemCode

type AdminRedeemCode = billingdto.AdminRedeemCode

type BatchUpdateRedeemCodeFields = billingdto.BatchUpdateRedeemCodeFields

type BatchUpdateRedeemCodesRequest = billingdto.BatchUpdateRedeemCodesRequest
