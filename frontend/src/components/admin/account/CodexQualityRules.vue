<template>
  <!-- 三元素配合文字，说明既有调度规则；不触发操作，也不只依靠颜色。 -->
  <div class="grid gap-2 sm:grid-cols-3" data-testid="quality-rules">
    <div v-for="(status, index) in statuses" :key="status" class="flex min-w-0 items-center gap-3 border-l-4 border-current bg-[var(--bh-surface)] px-3 py-2" :class="qualityStatusClass(status)">
      <span aria-hidden="true" class="h-3 w-3 shrink-0 bg-current" :class="index === 0 ? '[clip-path:polygon(50%_0,100%_100%,0_100%)]' : index === 2 ? 'rounded-full' : ''" />
      <div class="min-w-0 text-xs font-bold"><span>{{ t(`admin.accounts.quality.status.${status}`) }}</span><span class="mx-2" aria-hidden="true">→</span><span>{{ t(`admin.accounts.quality.rule.${status}`) }}</span></div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { qualityStatusClass } from './codexQualityPresentation'
const { t } = useI18n()
const statuses = ['full', 'degraded', 'failed'] as const
</script>
