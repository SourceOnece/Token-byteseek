package ops

import "time"

const opsCleanupDefaultSchedule = "0 3 * * *"

const (
	opsCleanupDefaultBatchSize  = 1000
	OpsCleanupDefaultBatchSize  = opsCleanupDefaultBatchSize
	opsCleanupDefaultBatchPause = 200 * time.Millisecond
	OpsCleanupDefaultBatchPause = opsCleanupDefaultBatchPause
)
