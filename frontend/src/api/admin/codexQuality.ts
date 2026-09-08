import { apiClient, buildApiUrl } from '../client'
import { ADMIN_UI_REQUEST_HEADER } from '../adminUIRequest'

export type QualityStatus = 'full' | 'degraded' | 'failed' | 'skipped' | 'stale' | 'cancelled'
export interface CodexQualityResult {
  account_id: number
  account_name: string
  email: string
  model: string
  reasoning_effort: string
  prompt: string
  keyword: string
  response_text: string
  status: QualityStatus
  error?: string
  started_at: string
  finished_at: string
  scheduling_applied: boolean
  schedulable: boolean
}
export interface CodexQualityRequest {
  account_ids: number[]
  model: string
  reasoning_effort: string
  prompt: string
  keyword: string
  concurrency: number
  confirm_scheduling: boolean
}

export async function listCodexQualityResults(ids: number[], signal?: AbortSignal, detail = false) {
  if (!ids.length) return []
  const { data } = await apiClient.get<CodexQualityResult[]>('/admin/accounts/codex-quality-results', {
    params: { account_ids: ids.join(','), detail }, signal
  })
  return data
}

// 保留跨网络分块的 UTF-8 和 SSE 事件边界，缺少 complete 不能显示整批完成。
export async function consumeQualityStream(
  body: ReadableStream<Uint8Array>, onResult: (result: CodexQualityResult) => void
) {
  const reader = body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let complete = false
  const parse = (raw: string) => {
    const data = raw.split('\n').filter(line => line.startsWith('data:'))
      .map(line => line.slice(5).trimStart()).join('\n')
    if (!data) return
    const event = JSON.parse(data)
    if (event.type === 'result') onResult(event.data)
    if (event.type === 'complete') complete = true
  }
  try {
    while (true) {
      const { done, value } = await reader.read()
      buffer += done ? decoder.decode() : decoder.decode(value, { stream: true })
      buffer = buffer.replace(/\r\n/g, '\n')
      if (buffer.length > 2 * 1024 * 1024) throw new Error('Test response is too large')
      let end: number
      while ((end = buffer.indexOf('\n\n')) >= 0) {
        parse(buffer.slice(0, end))
        buffer = buffer.slice(end + 2)
      }
      if (done) break
    }
    if (buffer.trim()) parse(buffer)
    if (!complete) throw new Error('Batch stream ended before completion')
  } finally {
    await reader.cancel().catch(() => {})
    reader.releaseLock()
  }
}

export async function runCodexQualityBatch(
  request: CodexQualityRequest, signal: AbortSignal,
  onResult: (result: CodexQualityResult) => void
) {
  // 不自动重试消耗上游额度的 POST；认证失败交给管理员重新登录后确认。
  const response = await fetch(buildApiUrl('/admin/accounts/codex-quality-test'), {
    method: 'POST', credentials: 'include', signal,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${localStorage.getItem('auth_token') || ''}`,
      [ADMIN_UI_REQUEST_HEADER]: '1'
    },
    body: JSON.stringify(request)
  })
  if (!response.ok) {
    const error = await response.json().catch(() => null)
    throw new Error(error?.message || `HTTP ${response.status}`)
  }
  if (!response.body || !response.headers.get('content-type')?.includes('text/event-stream')) {
    throw new Error('Invalid batch stream response')
  }
  await consumeQualityStream(response.body, onResult)
}

// 失败属于已完成测试，跳过、取消及并发配置冲突不进入满血率分母。
export function qualityStats(results: CodexQualityResult[]) {
  const full = results.filter(r => r.status === 'full').length
  const degraded = results.filter(r => r.status === 'degraded').length
  const failed = results.filter(r => r.status === 'failed').length
  const tested = full + degraded + failed
  return { full, degraded, failed, tested, rate: tested ? (full / tested * 100).toFixed(1) : null }
}
