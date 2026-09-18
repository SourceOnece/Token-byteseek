import { apiClient } from '../client'

export type TicketState = 'ready' | 'pending' | 'collecting' | 'missing' | 'expired' | 'failed' | 'disabled' | 'unsupported' | 'unavailable' | 'paused'
export interface TicketModelStatus {
  model: string
  state: TicketState
  reason?: string
  checked_at?: string
  expires_at?: string
}
export interface TicketAccountStatus {
  account_id: number
  eligible: boolean
  collection_paused: boolean
  models: TicketModelStatus[]
}
export interface TicketStatusResponse {
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
