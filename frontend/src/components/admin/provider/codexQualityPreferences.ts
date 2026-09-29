// 仅保存测试配置，不保存账号选择、确认勾选、结果或凭据。
export interface CodexQualityPreferences {
  model: string
  effort: string
  protocol: string
  prompt: string
  keyword: string
  concurrency: number
  timeoutSeconds: number
}

export function defaultQualityPreferences(): CodexQualityPreferences {
  return { model: 'gpt-6-astra', effort: '', protocol: 'responses', prompt: '', keyword: '', concurrency: 3, timeoutSeconds: 120 }
}

function storageKey(adminId: number | undefined): string | null {
  return Number.isSafeInteger(adminId) && Number(adminId) > 0 ? `byteseek:codex-quality-preferences:v1:${adminId}` : null
}

function normalize(value: unknown): CodexQualityPreferences {
  const result = defaultQualityPreferences()
  if (!value || typeof value !== 'object' || Array.isArray(value)) return result
  const stored = value as Record<string, unknown>
  // 本地数据同样校验边界，损坏/过期字段各自回退，不阻止正常打开弹窗。
  if (typeof stored.model === 'string' && stored.model.trim() && new TextEncoder().encode(stored.model).length <= 200 && !/[\r\n\0]/.test(stored.model)) result.model = stored.model
  if (typeof stored.effort === 'string' && ['', 'none', 'minimal', 'low', 'medium', 'high', 'xhigh', 'max'].includes(stored.effort)) result.effort = stored.effort
  if (typeof stored.protocol === 'string' && ['', 'responses', 'chat_completions'].includes(stored.protocol)) result.protocol = stored.protocol
  for (const [field, max] of [['prompt', 16000], ['keyword', 200]] as const) {
    const text = stored[field]
    if (typeof text === 'string' && [...text].length <= max && !text.includes('\0')) result[field] = text
  }
  if (typeof stored.concurrency === 'number' && Number.isInteger(stored.concurrency) && stored.concurrency >= 1 && stored.concurrency <= 5) result.concurrency = stored.concurrency
  if (typeof stored.timeoutSeconds === 'number' && Number.isInteger(stored.timeoutSeconds) && stored.timeoutSeconds >= 10 && stored.timeoutSeconds <= 3600) result.timeoutSeconds = stored.timeoutSeconds
  return result
}

export function loadQualityPreferences(adminId: number | undefined): CodexQualityPreferences {
  const key = storageKey(adminId)
  if (key) {
    try { return normalize(JSON.parse(localStorage.getItem(key) ?? 'null')) } catch { /* 存储被禁用或损坏时使用默认配置。 */ }
  }
  return defaultQualityPreferences()
}

export function saveQualityPreferences(adminId: number | undefined, value: CodexQualityPreferences) {
  const key = storageKey(adminId)
  if (!key) return
  try { localStorage.setItem(key, JSON.stringify(normalize(value))) } catch { /* 配额不足或禁止存储不影响本次检测。 */ }
}
