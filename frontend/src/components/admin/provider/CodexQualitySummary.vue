<template>
  <div class="grid grid-cols-2 gap-4 lg:grid-cols-4" aria-live="polite">
    <button type="button" class="quality-stat" @click="$emit('select', '')"><span class="flex items-center justify-between gap-2 text-sm font-semibold"><span>{{ rateLabel || t('admin.accounts.quality.rate') }}</span><span aria-hidden="true">↗</span></span><strong class="mt-3 block text-3xl font-extrabold tabular-nums tracking-tight text-emerald-700 dark:text-emerald-400 sm:text-4xl">{{ stats.rate === null ? '—' : `${stats.rate}%` }}</strong></button>
    <button v-for="status in statuses" :key="status" type="button" class="quality-stat" :data-testid="`quality-summary-${status}`" @click="$emit('select', status)">
      <span class="flex items-center justify-between gap-2 text-sm font-bold" :class="qualityStatusClass(status)"><span>{{ t(`admin.accounts.quality.status.${status}`) }}</span><span aria-hidden="true">↗</span></span>
      <strong class="mt-3 block text-3xl font-extrabold tabular-nums tracking-tight sm:text-4xl" :class="qualityStatusClass(status)">{{ counts[status] || 0 }}</strong>
    </button>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { qualityStatusClass } from './codexQualityPresentation'
const props = defineProps<{ counts: Record<string, number>; rateLabel?: string }>()
defineEmits<{ select: [status: string] }>()
const { t } = useI18n()
const statuses = ['full', 'degraded', 'failed'] as const
const stats = computed(() => {
  const total = statuses.reduce((sum, status) => sum + (props.counts[status] || 0), 0)
  return { rate: total ? ((props.counts.full || 0) / total * 100).toFixed(1) : null }
})
</script>
<style scoped>
.quality-stat { min-width: 0; padding: 1rem; text-align: left; border: 2px solid var(--bh-ink); background: var(--bh-surface); box-shadow: var(--bh-shadow-sm); overflow-wrap: anywhere; }
@media (min-width: 640px) { .quality-stat { padding: 1.25rem; } }
.quality-stat:active { transform: translate(2px, 2px); box-shadow: 1px 1px 0 var(--bh-shadow-ink); }
.quality-stat:focus-visible { outline: 3px solid var(--bh-blue); outline-offset: 3px; }
</style>
