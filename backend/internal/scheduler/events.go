// 本文件定义调度事件契约。
package scheduler

const (
	SchedulerOutboxEventProviderChanged       = "provider_changed"
	SchedulerOutboxEventProviderGroupsChanged = "provider_groups_changed"
	SchedulerOutboxEventProviderBulkChanged   = "provider_bulk_changed"
	SchedulerOutboxEventProviderLastUsed      = "provider_last_used"
	SchedulerOutboxEventGroupChanged          = "group_changed"
	SchedulerOutboxEventFullRebuild           = "full_rebuild"
)

// GroupPayload 保留空分组的 untyped nil，避免改变持久化去重指纹。
func GroupPayload(groupIDs []int64) any {
	if len(groupIDs) == 0 {
		return nil
	}
	return map[string]any{"group_ids": groupIDs}
}
