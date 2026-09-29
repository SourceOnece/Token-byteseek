package app

import (
	"context"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/usage"
)

// projectGeminiModelUsage 明确选择提供商成本，不能使用用户实际扣费字段。
func projectGeminiModelUsage(rows []usage.ModelStat) []provider.GeminiModelUsage {
	if rows == nil {
		return nil
	}
	values := make([]provider.GeminiModelUsage, len(rows))
	for i, row := range rows {
		values[i] = provider.GeminiModelUsage{Model: row.Model, Requests: row.Requests, TotalTokens: row.TotalTokens, ProviderCost: row.ProviderCost}
	}
	return values
}

type providerGeminiUsageReader struct{ source usage.UsageLogRepository }

func (s providerGeminiUsageReader) GetModelUsage(ctx context.Context, id int64, start, end time.Time) ([]provider.GeminiModelUsage, error) {
	rows, err := s.source.GetModelStatsWithFilters(ctx, start, end, 0, 0, id, 0, nil, nil, nil)
	return projectGeminiModelUsage(rows), err
}

type providerGeminiUsageBatchReader struct {
	providerGeminiUsageReader
	provider.GeminiUsageTotalsBatchReader
}

// newProviderGeminiUsageReader 保留批量读取能力和原查询参数，不复制统计缓存。
func newProviderGeminiUsageReader(source usage.UsageLogRepository) provider.GeminiQuotaUsageReader {
	if source == nil {
		return nil
	}
	reader := providerGeminiUsageReader{source: source}
	if batch, ok := source.(provider.GeminiUsageTotalsBatchReader); ok {
		return providerGeminiUsageBatchReader{reader, batch}
	}
	return reader
}
