package admin

import (
	"time"

	"github.com/TokenFlux/TokenRouter/internal/server/httpx"
)

type snapshotCache = httpx.SnapshotCache

// newSnapshotCache 委托所属模块的唯一实现。
func newSnapshotCache(ttl time.Duration) *snapshotCache { return httpx.NewSnapshotCache(ttl) }

// parseBoolQueryWithDefault 委托所属模块的唯一实现。
func parseBoolQueryWithDefault(raw string, def bool) bool {
	return httpx.ParseBoolQueryWithDefault(raw, def)
}

func normalizeInt64IDList(v []int64) []int64  { return httpx.NormalizeInt64IDList(v) }
func ifNoneMatchMatched(raw, tag string) bool { return httpx.IfNoneMatchMatched(raw, tag) }
