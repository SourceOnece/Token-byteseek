// HTTP 协议适配返回本次已观察的字段；平台与入站请求分别拥有自己的状态。
package upstream

import (
	"time"

	wire "github.com/TokenFlux/TokenRouter/internal/protocol/openai"
)

type ResponsesObservation struct {
	// UpstreamResponseModel 是协议转换前的上游模型声明；空值表示未声明。
	UpstreamResponseModel                                               string
	HasUsage, Served, HTTPCommitted, RetryCommitted, ClientDisconnected bool
	FirstSemanticOutput                                                 *time.Duration
	Usage                                                               *wire.ForwardUsage
	FirstTokenMs                                                        *int
	ResponseID                                                          string
	SearchCount, ImageCount                                             int
	ImageOutputSizes                                                    []string
}
