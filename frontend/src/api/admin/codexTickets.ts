import { apiClient, buildApiUrl } from '../client'
import { ADMIN_UI_REQUEST_HEADER } from '../adminUIRequest'

export type TicketState = 'ready' | 'pending' | 'collecting' | 'missing' | 'expired' | 'failed' | 'disabled' | 'unsupported' | 'unavailable' | 'paused'
export interface TicketModelStatus {
  model: string
  state: TicketState
  blocked?: boolean
  target_length?: number
  degraded_signal_length?: number
  latest?: TicketLatest
  reason?: string
  checked_at?: string
  expires_at?: string
  diagnostic?: {
    proxy_id: string; proxy_name: string; attempt: number; http_status?: number; degraded_signal?: boolean
    header_length: number; header_present: boolean; prefix_valid: boolean
    response_kind?: string; error_kind?: string; completion_seen?: boolean; retry_not_before?: string
  }
}

export interface TicketLatest {
  source: 'manual' | 'auto'; state: string; reason?: string; checked_at: string
  diagnostic?: TicketModelStatus['diagnostic']; reference_ip?: string; ip_status?: string; ip_source?: string; ip_http_status?: number
}

export interface TicketSettings {
  models?: string[]; degraded_signal_length?: number
  enabled: boolean; proxy_configured: boolean; target_length: number; revision: string
  selection_mode: 'fixed' | 'rotate'; fixed_proxy_id: string; max_attempts: number
  retry_interval_seconds: number; probe_interval_seconds: number
  proxies: { id: string; name: string; configured: boolean }[]
}
export interface TicketCollectionEvent {
  id?: number; kind: 'attempt' | 'result'; account_id: number; account_name: string; email: string
  model: string; target_length: number; status: string; reason?: string; attempt: number
  started_at: string; finished_at: string; duration_ms: number; expires_at?: string
  diagnostic?: TicketModelStatus['diagnostic']; reference_ip?: string; ip_checked_at?: string; ip_status?: string; ip_source?: string; ip_http_status?: number
}
export interface TicketCollectionRun {
  id: string; status: string; config: TicketSettings; total: number; started_at: string; finished_at?: string; counts: Record<string, number>
}
export const ticketCollectionAPI = {
  // 仅在显式点击时发送 DELETE，不使用读取接口触发删除或自动重试。
  async clearHistory() { return (await apiClient.delete<{ deleted: number }>('/admin/accounts/codex-ticket-runs')).data },
  async deleteRun(id: string) { return (await apiClient.delete<{ deleted: number }>(`/admin/accounts/codex-ticket-runs/${id}`)).data },
  async deleteEvent(id: string, eventID: number) { return (await apiClient.delete<{ deleted: number }>(`/admin/accounts/codex-ticket-runs/${id}/events/${eventID}`)).data },
  async settings() { return (await apiClient.get<TicketSettings>('/admin/settings/codex-ticket')).data },
  async runs(page = 1) { return (await apiClient.get<TicketCollectionRun[]>('/admin/accounts/codex-ticket-runs', { params: { page } })).data },
  async detail(id: string, params: { page?: number; kind?: string; status?: string; account_id?: number; model?: string } = {}) {
    return (await apiClient.get<{ run: TicketCollectionRun; items: TicketCollectionEvent[]; total: number; page: number }>(`/admin/accounts/codex-ticket-runs/${id}`, { params })).data
  }
}

// 不自动重试消耗额度的 POST；逐事件处理且不把无限轮数的日志留在浏览器内存中。
export async function runTicketCollection(ids: number[], revision: string, signal: AbortSignal, onEvent: (kind: string, data: unknown) => void) {
  const response = await fetch(buildApiUrl('/admin/accounts/codex-ticket-collect'), {
    method: 'POST', signal, credentials: 'include',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${localStorage.getItem('auth_token') || ''}`, [ADMIN_UI_REQUEST_HEADER]: '1' },
    body: JSON.stringify({ account_ids: ids, confirmed: true, revision })
  })
  if (!response.ok) { const body = await response.json().catch(() => null); throw new Error(body?.message || `HTTP ${response.status}`) }
  if (!response.body || !response.headers.get('content-type')?.includes('text/event-stream')) throw new Error('Invalid collection stream')
  const reader = response.body.getReader(), decoder = new TextDecoder()
  let buffer = '', completed = false
  const parse = (raw: string) => {
    const value = raw.split('\n').filter(line => line.startsWith('data:')).map(line => line.slice(5).trimStart()).join('\n')
    if (!value) return
    const event = JSON.parse(value)
    if (event.type === 'complete') completed = true
    onEvent(event.type, event.data)
  }
  try {
    for (;;) {
      const { done, value } = await reader.read()
      buffer += done ? decoder.decode() : decoder.decode(value, { stream: true })
      buffer = buffer.replace(/\r\n/g, '\n')
      if (buffer.length > 1024 * 1024) throw new Error('Collection event is too large')
      let end: number
      while ((end = buffer.indexOf('\n\n')) >= 0) { parse(buffer.slice(0, end)); buffer = buffer.slice(end + 2) }
      if (done) break
    }
    if (buffer.trim()) parse(buffer)
    if (!completed) throw new Error('Collection stream ended before completion')
  } finally { await reader.cancel().catch(() => {}); reader.releaseLock() }
}
export interface TicketAccountStatus {
  account_id: number
  eligible: boolean
  collection_paused: boolean
  models: TicketModelStatus[]
}
export interface TicketStatusResponse {
  models?: string[]
  enabled: boolean
  server_time: string
  items: TicketAccountStatus[]
}

// 只读状态接口，不会因为翻页、刷新或轮询而增加上游采集请求。
export async function getCodexTicketStatus(ids: number[], signal?: AbortSignal) {
  const { data } = await apiClient.get<TicketStatusResponse>('/admin/settings/codex-ticket/status', {
    params: { account_ids: ids.join(',') }, signal
  })
  return data
}
