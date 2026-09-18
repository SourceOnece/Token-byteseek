<template>
  <div class="mt-2 space-y-1 border-t border-gray-200 pt-1.5 dark:border-dark-600" data-testid="codex-ticket-status" :aria-label="t('admin.accounts.tickets.title')">
    <div v-for="row in rows" :key="row.model" class="grid grid-cols-[3rem_minmax(0,1fr)] items-baseline gap-2 text-xs" :title="description(row)">
      <span class="font-bold text-bh-blue dark:text-blue-300">{{ row.model === 'gpt-6-astra' ? 'Astra' : 'Sol' }}</span>
      <span class="break-words font-semibold tabular-nums" :class="color(row)">{{ label(row) }}</span>
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
  if (state(row) !== 'ready') return t(`admin.accounts.tickets.state.${state(row)}`)
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
  const reasons = ['network', 'upstream', 'invalid_ticket', 'credential', 'storage', 'cancelled']
  if (row.reason && reasons.includes(row.reason) && !props.failed) info.push(t(`admin.accounts.tickets.reason.${row.reason}`))
  if (props.status?.collection_paused) info.push(t('admin.accounts.tickets.pausedHint'))
  return info.join('\n')
}
</script>
