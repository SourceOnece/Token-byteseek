<template>
  <div v-if="stages?.length" class="grid gap-4 sm:grid-cols-2" data-testid="ticket-validation-stages">
    <section v-for="stage in stages" :key="stage.name" class="min-w-0 space-y-3 border-2 border-[color:var(--bh-ink)] bg-[var(--bh-surface)] p-4" style="box-shadow:var(--bh-shadow-sm)">
      <h4 class="text-base font-extrabold text-bh-blue dark:text-blue-300">{{ t('admin.accounts.ticketWorkbench.stage.' + stage.name) }}</h4>
      <p class="break-all text-sm"><span class="mr-2 text-gray-500 dark:text-gray-400">{{ t('admin.accounts.ticketWorkbench.requestModel') }}</span><strong class="text-bh-blue dark:text-blue-300">{{ stage.request_model }}</strong></p>
      <p class="break-all text-sm"><span class="mr-2 text-gray-500 dark:text-gray-400">{{ t('admin.accounts.ticketWorkbench.responseModel') }}</span><strong :class="!stage.response_model ? 'text-gray-500 dark:text-gray-400' : stage.response_model === stage.request_model ? 'text-emerald-700 dark:text-emerald-400' : 'text-bh-red dark:text-red-400'">{{ stage.response_model || t('admin.accounts.ticketWorkbench.responseMissing') }}</strong></p>
      <p class="border-t border-[color:var(--bh-ink)] pt-2 text-sm font-semibold">HTTP {{ stage.http_status || '—' }} · <strong :class="lengthColor(stage)" :data-testid="'ticket-stage-length-' + stage.name">{{ stage.state_length >= 0 ? stage.state_length + ' B' : '—' }}</strong></p>
      <p v-if="stage.reason" class="text-sm font-bold" :class="stage.reason === 'length_signal' || stage.reason === 'model_mismatch' ? 'text-bh-red dark:text-red-400' : 'text-yellow-800 dark:text-bh-yellow'">{{ t('admin.accounts.ticketWorkbench.validationReason.' + stage.reason) }}</p>
    </section>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { TicketValidationStage } from '@/api/admin/codexTickets'
import { codexTicketLengthColor } from '@/utils/codexTicketLength'
// 管理员详情共用两阶段摘要，长模型换行；没有完整响应不伪造匹配结果。
const props = defineProps<{ stages?: TicketValidationStage[]; target?: number; degradedSignalLength?: number }>()
const { t } = useI18n()
function lengthColor(stage: TicketValidationStage) {
  const signal = stage.degraded_signal_length ?? props.degradedSignalLength ?? 0
  return codexTicketLengthColor(stage.state_length, stage.target_length || props.target || 292, stage.reason === 'length_signal' || (signal > 0 && stage.state_length === signal), stage.name === 'verify')
}
</script>
