<template>
  <!-- 只表达诊断长度，不将颜色作为获票或模型质量判断。 -->
  <span class="inline-flex items-baseline font-mono text-xs font-bold tabular-nums" data-testid="ticket-length-ratio" :aria-label="t('admin.accounts.tickets.lengthLabel', { actual, target })">
    <span :class="actualColor" data-testid="ticket-length-actual">{{ actual }}</span><span class="text-gray-500 dark:text-gray-400" aria-hidden="true">/</span><span class="text-emerald-700 dark:text-emerald-400" data-testid="ticket-length-target">{{ target }}</span>
  </span>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
const props = defineProps<{ actual: number; target: number; signal?: boolean }>()
const { t } = useI18n()
// 信号优先于相等色；目标数值始终绿色，避免两侧被同一个状态染色。
const actualColor = computed(() => props.signal ? 'text-bh-red dark:text-red-400' : props.actual === props.target ? 'text-emerald-700 dark:text-emerald-400' : 'text-yellow-700 dark:text-bh-yellow')
</script>
