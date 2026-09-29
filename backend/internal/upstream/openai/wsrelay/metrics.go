package openai_ws_v2

import (
	"sync/atomic"
)

var passthroughUsageParseFailureTotal atomic.Int64

func recordUsageParseFailure() {
	passthroughUsageParseFailureTotal.Add(1)
}
