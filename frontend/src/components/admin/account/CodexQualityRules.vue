<template>
  <!-- 三元素配合文字，说明既有调度规则；不触发操作，也不只依靠颜色。 -->
  <div class="quality-rules" data-testid="quality-rules">
    <div v-for="(status, index) in statuses" :key="status" class="quality-rule" :class="qualityStatusClass(status)">
      <span aria-hidden="true" class="h-5 w-5 shrink-0 bg-current sm:h-6 sm:w-6" :class="index === 0 ? '[clip-path:polygon(50%_0,100%_100%,0_100%)]' : index === 2 ? 'rounded-full' : ''" />
      <div class="min-w-0 flex-1">
        <p class="text-lg font-extrabold leading-tight sm:text-xl" data-testid="quality-rule-status">{{ t(`admin.accounts.quality.status.${status}`) }}</p>
        <p class="mt-1.5 text-sm font-semibold leading-snug sm:text-base" data-testid="quality-rule-action"><span class="mr-2" aria-hidden="true">→</span>{{ t(`admin.accounts.quality.rule.${status}`) }}</p>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { qualityStatusClass } from './codexQualityPresentation'
const { t } = useI18n()
const statuses = ['full', 'degraded', 'failed'] as const
</script>
<style scoped>
/* 以一个连续面板承载三种结果，不把规则拆成零散小提示或可点击卡片。 */
.quality-rules { display: grid; border: 2px solid var(--bh-ink); background: var(--bh-surface); box-shadow: var(--bh-shadow-sm); }
.quality-rule { display: flex; align-items: center; gap: 1rem; min-width: 0; padding: 1rem 1.25rem; }
.quality-rule + .quality-rule { border-top: 1px solid var(--bh-ink); }
@media (min-width: 640px) {
  .quality-rules { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .quality-rule { padding: 1.5rem; }
  .quality-rule + .quality-rule { border-top: 0; border-left: 1px solid var(--bh-ink); }
}
</style>
