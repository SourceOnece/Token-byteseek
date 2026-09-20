import type { TicketModelStatus } from '@/api/admin/codexTickets'

type Translate = (key: string, values?: Record<string, unknown>) => string
const reasons = ['network', 'upstream', 'invalid_ticket', 'credential', 'storage', 'cancelled', 'proxy_config', 'proxy_provider', 'cooldown', 'ineligible', 'account_changed', 'concurrency_busy', 'backoff', 'business_proxy', 'incomplete_response', 'model_mismatch', 'length_signal', 'timeout', 'round_timeout', 'task_timeout', 'client_disconnected', 'lease_lost', 'config_unavailable', 'service_stopped', 'config_disabled', 'account_missing', 'unsupported_account', 'account_inactive', 'account_expired', 'account_overloaded', 'account_rate_limited', 'account_cooldown', 'model_unsupported', 'attempt_limit']

// 账号悬停/点击、手动实时摘要和历史共用固定文案；未知原因不伪造为账号变化。
export function ticketReasonText(t: Translate, reason?: string): string {
  if (!reason) return ''
  return t('admin.accounts.ticketDiagnostic.reasons.' + ([...reasons, 'proxy_timeout', 'transport_timeout', 'concurrency_config'].includes(reason) ? reason : 'unknown'))
}

export function ticketDiagnosticText(t: Translate, diagnostic?: TicketModelStatus['diagnostic']): string {
  if (!diagnostic) return ''
  const parts: string[] = []
  if (diagnostic.previous_attempt) parts.push(t('admin.accounts.ticketDiagnostic.previousAttempt'))
  if (diagnostic.phase && ['eligibility', 'queue', 'proxy', 'harvest', 'verify', 'publish'].includes(diagnostic.phase)) parts.push(t('admin.accounts.ticketDiagnostic.phases.' + diagnostic.phase))
  if (diagnostic.timeout_seconds) parts.push(t('admin.accounts.ticketDiagnostic.budget', { seconds: diagnostic.timeout_seconds }))
  if (diagnostic.elapsed_ms !== undefined) parts.push(t('admin.accounts.ticketDiagnostic.elapsed', { seconds: (diagnostic.elapsed_ms / 1000).toFixed(1) }))
  if (diagnostic.network_kind && ['timeout', 'dns', 'tls', 'connection'].includes(diagnostic.network_kind)) parts.push(t('admin.accounts.ticketDiagnostic.network.' + diagnostic.network_kind))
  if (diagnostic.provider_status) parts.push(t('admin.accounts.ticketDiagnostic.providerHTTP', { status: diagnostic.provider_status }))
  if (diagnostic.error_kind && ['overloaded', 'rate_limit', 'quota', 'auth', 'invalid_request'].includes(diagnostic.error_kind)) parts.push(t('admin.accounts.tickets.errorKind.' + diagnostic.error_kind))
  return parts.join(' · ')
}
