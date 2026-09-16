import type { SubscriptionBulkActionRequest } from '@/api/admin/subscriptions'
import type { BulkAssignSubscriptionRequest } from '@/types'

export interface BulkSubscriptionOperation<T = SubscriptionBulkActionRequest> {
  request: T
  key: string
  storageKey: string | null
  outcomeUncertain: boolean
}

const pendingKeys = new Map<string, string>()

function currentAdminId(): number | null {
  try {
    const user = JSON.parse(globalThis.localStorage?.getItem('auth_user') ?? 'null') as { id?: unknown } | null
    const id = user?.id
    return typeof id === 'number' && Number.isSafeInteger(id) && id > 0 ? id : null
  } catch {
    return null
  }
}

function readStoredKey(storageKey: string): string | null {
  try {
    return globalThis.sessionStorage?.getItem(storageKey) ?? null
  } catch {
    return null
  }
}

function storeKey(storageKey: string, key: string | null) {
  try {
    if (key) globalThis.sessionStorage?.setItem(storageKey, key)
    else globalThis.sessionStorage?.removeItem(storageKey)
  } catch {
    // 浏览器存储不可用时仍在当前会话内复用操作键。
  }
}

export function prepareBulkSubscriptionOperation(input: SubscriptionBulkActionRequest): BulkSubscriptionOperation {
  // 发送与存储一致的规范化参数，表格重新排序后仍可取回同一批结果。
  const request: SubscriptionBulkActionRequest = {
    subscription_ids: [...new Set(input.subscription_ids)].sort((a, b) => a - b),
    action: input.action
  }
  if (input.action === 'extend') request.days = input.days
  if (input.action === 'reset_quota') {
    request.daily = !!input.daily
    request.weekly = !!input.weekly
    request.monthly = !!input.monthly
  }
  return prepareOperation(request)
}

export function prepareBulkSubscriptionAssignment(input: BulkAssignSubscriptionRequest): BulkSubscriptionOperation<BulkAssignSubscriptionRequest> {
  return prepareOperation({ user_ids: [...new Set(input.user_ids)].sort((a,b) => a-b), plan_id: input.plan_id, validity_days: input.validity_days, notes: input.notes })
}

function prepareOperation<T>(request: T): BulkSubscriptionOperation<T> {
  const adminId = currentAdminId()
  const storageKey = adminId ? `sub2api:admin:subscription-bulk:${adminId}:${JSON.stringify(request)}` : null
  let key = storageKey ? pendingKeys.get(storageKey) ?? readStoredKey(storageKey) : null
  const outcomeUncertain = !!key
  if (!key) {
    const requestId = globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`
    key = `subscription-bulk-${adminId ?? 'unknown'}-${requestId}`
  }
  if (storageKey) {
    pendingKeys.set(storageKey, key)
    storeKey(storageKey, key)
  }
  return { request, key, storageKey, outcomeUncertain }
}

export function completeBulkSubscriptionOperation(operation: BulkSubscriptionOperation<unknown>) {
  if (!operation.storageKey) return
  pendingKeys.delete(operation.storageKey)
  storeKey(operation.storageKey, null)
}
