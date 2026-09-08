<template>
  <div class="grid grid-cols-2 gap-3 lg:grid-cols-4" aria-live="polite">
    <button type="button" class="quality-stat" @click="$emit('select', '')"><span class="text-xs">{{ rateLabel || t('admin.accounts.quality.rate') }}</span><strong class="block text-2xl text-bh-blue dark:text-blue-300">{{ stats.rate === null ? '—' : `${stats.rate}%` }}</strong></button>
    <button v-for="status in statuses" :key="status" type="button" class="quality-stat" :data-testid="`quality-summary-${status}`" @click="$emit('select', status)">
      <span class="text-sm font-bold" :class="qualityStatusClass(status)">{{ t(`admin.accounts.quality.status.${status}`) }}</span>
      <strong class="block text-2xl" :class="qualityStatusClass(status)">{{ counts[status] || 0 }}</strong>
    </button>
  </div>
  <p class="mt-3 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.quality.clickCategory') }}</p>
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
.quality-stat { padding: .75rem; text-align: left; border: 2px solid var(--bh-ink); background: var(--bh-surface); box-shadow: var(--bh-shadow-sm); }
.quality-stat:active { transform: translate(2px, 2px); box-shadow: 1px 1px 0 var(--bh-shadow-ink); }
.quality-stat:focus-visible { outline: 3px solid var(--bh-blue); outline-offset: 3px; }
</style>
