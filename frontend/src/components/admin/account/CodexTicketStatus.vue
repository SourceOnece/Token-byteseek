<template>
  <div class="mt-2 space-y-1 border-t border-gray-200 pt-1.5 dark:border-dark-600" data-testid="codex-ticket-status" :aria-label="t('admin.accounts.tickets.title')">
    <div v-for="row in rows" :key="row.model" class="text-xs" :title="description(row)">
      <div class="grid grid-cols-[minmax(3rem,5rem)_minmax(0,1fr)] items-baseline gap-2">
        <button type="button" class="min-w-0 break-all border-2 border-current px-1 py-0.5 text-left font-bold text-bh-blue [box-shadow:var(--bh-shadow-sm)] active:translate-x-0.5 active:translate-y-0.5 active:shadow-none focus-visible:outline focus-visible:outline-2 dark:text-blue-300" :aria-label="t('admin.accounts.tickets.openDetails', { model: row.model })" data-testid="ticket-model-detail" @click="selectedModel = row.model">{{ modelLabel(row.model) }}</button>
        <span class="break-words font-semibold tabular-nums" :class="row.latest && !failed ? latestColor(row) : color(row)" :data-testid="row.latest && !failed ? 'ticket-latest' : undefined">{{ row.latest && !failed ? latestLabel(row) : label(row) }}</span>
      </div>
      <p v-if="row.latest && !failed" class="mt-0.5 break-words text-[10px] tabular-nums" :class="color(row)" data-testid="ticket-current">{{ t('admin.accounts.tickets.currentTicket') }}：{{ label(row) }}</p>
      <p v-if="!failed && displayDiagnostic(row)?.http_status" class="mt-1 break-words text-[10px] text-gray-500 dark:text-gray-400" data-testid="ticket-diagnostic">
        HTTP {{ displayDiagnostic(row)?.http_status }} ·
        <strong v-if="displayDiagnostic(row)?.header_present" class="border border-current px-1 text-xs tabular-nums" :class="ratioColor(row)" data-testid="ticket-length-ratio">{{ displayDiagnostic(row)?.header_length }}/{{ row.target_length || 292 }}</strong>
        <span v-else>{{ t('admin.accounts.tickets.noHeader') }}</span>
        <span v-if="diagnosticExtra(row)"> · {{ diagnosticExtra(row) }}</span>
      </p>
      <p v-if="!failed && displayDiagnostic(row)?.degraded_signal" class="mt-1 text-xs font-bold text-bh-red dark:text-red-400" data-testid="ticket-degraded-signal">{{ t('admin.accounts.tickets.degradedSignal') }}</p>
    </div>
  </div>
  <BaseDialog v-if="selectedRow" :show="true" :title="selectedRow.model" width="normal" @close="selectedModel = ''">
    <div class="space-y-3" data-testid="ticket-detail-content">
      <p class="font-bold" :class="selectedRow.latest && !failed ? latestColor(selectedRow) : color(selectedRow)">{{ selectedRow.latest && !failed ? latestLabel(selectedRow) : label(selectedRow) }}</p>
      <p :class="color(selectedRow)">{{ t('admin.accounts.tickets.currentTicket') }}：{{ label(selectedRow) }}</p>
      <p v-if="!failed && displayDiagnostic(selectedRow)?.header_present"><strong class="border-2 border-current px-2 py-1 font-mono" :class="ratioColor(selectedRow)">{{ displayDiagnostic(selectedRow)?.header_length }}/{{ selectedRow.target_length || 292 }}</strong></p>
      <p v-if="!failed && displayDiagnostic(selectedRow)?.degraded_signal" class="font-bold text-bh-red dark:text-red-400">{{ t('admin.accounts.tickets.degradedSignal') }}</p>
      <p class="whitespace-pre-wrap break-words text-sm leading-relaxed text-gray-700 dark:text-gray-200">{{ description(selectedRow) }}</p>
    </div>
    <template #footer><button type="button" class="btn btn-secondary" @click="selectedModel = ''">{{ t('common.close') }}</button></template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { TicketAccountStatus, TicketModelStatus, TicketState } from '@/api/admin/codexTickets'

const props = defineProps<{ status?: TicketAccountStatus; now: number; failed?: boolean }>()
const { t } = useI18n()
const rows = computed<TicketModelStatus[]>(() => props.status?.models || ['gpt-6-astra', 'gpt-5.6-sol'].map(model => ({ model, state: 'pending' })))
const selectedModel = ref('')
const selectedRow = computed(() => rows.value.find(row => row.model === selectedModel.value))
// 账号切换/模型移除时关闭旧详情，详情内容随当前状态更新，不保存跨账号快照。
watch(() => props.status?.account_id, () => { selectedModel.value = '' })
watch(() => rows.value.map(row => row.model).join('\n'), () => { if (!rows.value.some(row => row.model === selectedModel.value)) selectedModel.value = '' })
function modelLabel(model: string) { return model === 'gpt-6-astra' ? 'Astra' : model === 'gpt-5.6-sol' ? 'Sol' : model }
function ratioColor(row: TicketModelStatus) { return displayDiagnostic(row)?.header_length === (row.target_length || 292) ? 'text-emerald-700 dark:text-emerald-400' : 'text-yellow-700 dark:text-bh-yellow' }
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
function latestLabel(row: TicketModelStatus) {
  const latest = row.latest!
  const source = latest.source === 'manual' ? 'manual' : 'auto'
  const status = ['ready', 'missing', 'failed', 'cancelled', 'skipped'].includes(latest.state) ? latest.state : 'skipped'
  return `${t('admin.accounts.tickets.source.' + source)} · ${t('admin.accounts.ticketCollect.status.' + status)}`
}
function latestColor(row: TicketModelStatus) {
  const state = row.latest?.state
  if (state === 'ready') return 'text-emerald-700 dark:text-emerald-400'
  if (state === 'failed') return 'text-bh-red dark:text-red-400'
  if (state === 'missing') return 'text-yellow-700 dark:text-bh-yellow'
  return 'text-gray-500 dark:text-gray-400'
}
function displayDiagnostic(row: TicketModelStatus) { return row.latest ? row.latest.diagnostic : row.diagnostic }
// 票据状态与质量测试分开，使用固定文案，不把不透明上游错误或票据正文放入 DOM。
function description(row: TicketModelStatus) {
  const info = [row.model]
  if (!props.failed && displayDiagnostic(row)?.degraded_signal) info.push(t('admin.accounts.tickets.degradedSignal'))
  if (row.latest && !props.failed) {
    info.push(latestLabel(row), `${t('admin.accounts.tickets.latestAt')}: ${new Date(row.latest.checked_at).toLocaleString()}`)
    if (row.latest.reference_ip) info.push(`${t('admin.accounts.ticketCollect.ip')}: ${row.latest.reference_ip}`)
    else if (row.latest.ip_status) {
      const status = ['unavailable', 'timeout', 'network', 'tls', 'http_error', 'invalid_response', 'proxy_config', 'cancelled', 'not_attempted'].includes(row.latest.ip_status) ? row.latest.ip_status : 'unavailable'
      info.push(t('admin.accounts.ticketCollect.ipStatus.' + status))
      if (row.latest.ip_http_status) info.push('IP HTTP ' + row.latest.ip_http_status)
    }
  }
  if (!row.latest && row.checked_at && !props.failed) info.push(`${t('admin.accounts.tickets.checkedAt')}: ${new Date(row.checked_at).toLocaleString()}`)
  const reasons = ['network', 'upstream', 'invalid_ticket', 'credential', 'storage', 'cancelled', 'proxy_config']
  const reason = row.latest ? row.latest.reason : row.reason
  if (reason && reasons.includes(reason) && !props.failed) info.push(t(`admin.accounts.tickets.reason.${reason}`))
  else if (reason && ['ineligible', 'account_changed', 'concurrency_busy', 'backoff'].includes(reason) && !props.failed) info.push(t(`admin.accounts.ticketCollect.reason.${reason}`))
  if (props.status?.collection_paused) info.push(t('admin.accounts.tickets.pausedHint'))
  const diagnostic = displayDiagnostic(row)
  if (diagnostic && !props.failed) {
    info.push(t('admin.accounts.tickets.attempt', { proxy: diagnostic.proxy_name || '—', count: diagnostic.attempt }))
    if (diagnostic.http_status) info.push(diagnosticSummary(row))
    if (diagnostic.retry_not_before) info.push(t('admin.accounts.tickets.retryAfter', { time: new Date(diagnostic.retry_not_before).toLocaleString() }))
  }
  return info.join('\n')
}
function diagnosticSummary(row: TicketModelStatus) {
  const d = displayDiagnostic(row)
  if (!d?.http_status) return ''
  const parts = ['HTTP ' + d.http_status, d.header_present ? d.header_length + '/' + (row.target_length || 292) : t('admin.accounts.tickets.noHeader')]
  const extra = diagnosticExtra(row)
  if (extra) parts.push(extra)
  return parts.join(' · ')
}
function diagnosticExtra(row: TicketModelStatus) {
  const d = displayDiagnostic(row)
  if (!d) return ''
  const parts: string[] = []
  if (d.header_present && !d.prefix_valid) parts.push(t('admin.accounts.tickets.badPrefix'))
  if (d.response_kind === 'html') parts.push('HTML')
  if (d.error_kind && ['overloaded', 'rate_limit', 'quota', 'auth', 'invalid_request'].includes(d.error_kind)) parts.push(t('admin.accounts.tickets.errorKind.' + d.error_kind))
  else if (d.completion_seen) parts.push(t('admin.accounts.tickets.completedNoTicket'))
  return parts.join(' · ')
}
</script>
