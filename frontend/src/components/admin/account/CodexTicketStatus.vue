<template>
  <div class="mt-2 space-y-1 border-t border-gray-200 pt-1.5 dark:border-dark-600" data-testid="codex-ticket-status" :aria-label="t('admin.accounts.tickets.title')">
    <div v-for="row in rows" :key="row.model" class="text-xs" :title="description(row)">
      <div class="grid grid-cols-[3rem_minmax(0,1fr)] items-baseline gap-2">
        <span class="font-bold text-bh-blue dark:text-blue-300">{{ row.model === 'gpt-6-astra' ? 'Astra' : 'Sol' }}</span>
        <span class="break-words font-semibold tabular-nums" :class="color(row)">{{ label(row) }}</span>
      </div>
      <p v-if="!failed && row.diagnostic?.http_status && state(row) !== 'ready'" class="mt-0.5 break-words text-[10px] text-gray-500 dark:text-gray-400" data-testid="ticket-diagnostic">{{ diagnosticSummary(row) }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TicketAccountStatus, TicketModelStatus, TicketState } from '@/api/admin/codexTickets'

const props = defineProps<{ status?: TicketAccountStatus; now: number; failed?: boolean }>()
const { t } = useI18n()
const rows = computed<TicketModelStatus[]>(() => ['gpt-6-astra', 'gpt-5.6-sol'].map(model => props.status?.models.find(row => row.model === model) || { model, state: 'pending' }))
function remaining(row: TicketModelStatus) { return Math.max(0, Math.floor((Date.parse(row.expires_at || '') - props.now) / 1000)) }
function state(row: TicketModelStatus): TicketState | 'loading' {
  if (props.failed) return 'unavailable'
  if (!props.status) return 'loading'
  if (row.state === 'ready' && (!Number.isFinite(remaining(row)) || remaining(row) <= 0)) return 'expired'
  return row.state
}
function label(row: TicketModelStatus) {
  if (state(row) !== 'ready') {
    const text = t(`admin.accounts.tickets.state.${state(row)}`)
    return row.blocked && !props.failed ? `${text} · ${t('admin.accounts.tickets.modelBlocked')}` : text
  }
  const seconds = remaining(row)
  return `${Math.floor(seconds / 60)}m${String(seconds % 60).padStart(2, '0')}s`
}
function color(row: TicketModelStatus) {
  const current = state(row)
  if (current === 'ready') return 'text-emerald-700 dark:text-emerald-400'
  if (current === 'failed') return 'text-bh-red dark:text-red-400'
  if (current === 'missing' || current === 'expired') return 'text-yellow-700 dark:text-bh-yellow'
  if (current === 'collecting') return 'text-bh-blue dark:text-blue-300'
  return 'text-gray-500 dark:text-gray-400'
}
// 票据状态与质量测试分开，使用固定文案，不把不透明上游错误或票据正文放入 DOM。
function description(row: TicketModelStatus) {
  const info = [row.model, t('admin.accounts.tickets.notQuality')]
  if (row.checked_at && !props.failed) info.push(`${t('admin.accounts.tickets.checkedAt')}: ${new Date(row.checked_at).toLocaleString()}`)
  const reasons = ['network', 'upstream', 'invalid_ticket', 'credential', 'storage', 'cancelled', 'proxy_config']
  if (row.reason && reasons.includes(row.reason) && !props.failed) info.push(t(`admin.accounts.tickets.reason.${row.reason}`))
  if (props.status?.collection_paused) info.push(t('admin.accounts.tickets.pausedHint'))
  if (row.diagnostic && !props.failed) {
    info.push(t('admin.accounts.tickets.attempt', { proxy: row.diagnostic.proxy_name || '—', count: row.diagnostic.attempt }))
    if (row.diagnostic.http_status) info.push(diagnosticSummary(row))
    if (row.diagnostic.retry_not_before) info.push(t('admin.accounts.tickets.retryAfter', { time: new Date(row.diagnostic.retry_not_before).toLocaleString() }))
  }
  return info.join('\n')
}
function diagnosticSummary(row: TicketModelStatus) {
  const d = row.diagnostic
  if (!d?.http_status) return ''
  const parts = ['HTTP ' + d.http_status, d.header_present ? d.header_length + '/' + (row.target_length || 292) : t('admin.accounts.tickets.noHeader')]
  if (d.header_present && !d.prefix_valid) parts.push(t('admin.accounts.tickets.badPrefix'))
  if (d.response_kind === 'html') parts.push('HTML')
  if (d.error_kind && ['overloaded', 'rate_limit', 'quota', 'auth', 'invalid_request'].includes(d.error_kind)) parts.push(t('admin.accounts.tickets.errorKind.' + d.error_kind))
  else if (d.completion_seen) parts.push(t('admin.accounts.tickets.completedNoTicket'))
  return parts.join(' · ')
}
</script>
