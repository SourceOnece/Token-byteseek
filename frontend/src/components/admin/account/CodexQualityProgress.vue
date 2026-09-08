<template>
  <div class="space-y-2" data-testid="quality-progress">
    <div class="flex items-center justify-between gap-2 text-sm font-bold">
      <span>{{ label }}</span><span class="font-mono text-bh-blue dark:text-blue-300">{{ processed }} / {{ total }} · {{ percent }}%</span>
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
.quality-progress-track { height: 16px; border: 2px solid var(--bh-ink); background: var(--bh-surface); box-shadow: 2px 2px 0 var(--bh-shadow-ink); overflow: hidden; }
.quality-progress-fill { height: 100%; background: var(--bh-blue); transition: width .2s ease; }
@media (prefers-reduced-motion: reduce) { .quality-progress-fill { transition: none; } }
</style>
