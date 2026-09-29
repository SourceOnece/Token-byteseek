<template>
  <div class="space-y-3" data-testid="quality-progress">
    <div class="flex flex-wrap items-baseline justify-between gap-x-6 gap-y-2 text-sm font-semibold sm:text-base">
      <span class="min-w-0 break-words">{{ label }}</span><span class="text-lg font-extrabold tabular-nums text-bh-blue dark:text-blue-300 sm:text-xl">{{ processed }} / {{ total }} · {{ percent }}%</span>
    </div>
    <div class="quality-progress-track" role="progressbar" :aria-label="label" :aria-valuemin="0" :aria-valuemax="total || 1" :aria-valuenow="processed">
      <div class="quality-progress-fill" :style="{ width: `${percent}%` }" />
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
const props = defineProps<{ done: number; total: number; label: string }>()
// 进度只表示实际已处理账号数，不以运行状态伪造百分比。
const processed = computed(() => Math.max(0, Math.min(props.done, props.total)))
const percent = computed(() => props.total > 0 ? Math.floor(processed.value / props.total * 100) : 0)
</script>
<style scoped>
.quality-progress-track { height: 16px; border: 2px solid var(--bh-ink); background: var(--bh-surface); box-shadow: var(--bh-shadow-sm); overflow: hidden; }
.quality-progress-fill { height: 100%; background: var(--bh-blue); transition: width .2s ease; }
@media (prefers-reduced-motion: reduce) { .quality-progress-fill { transition: none; } }
</style>
