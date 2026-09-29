<template>
  <span v-if="usage === 'reused' || usage === 'new'" class="inline-flex border-2 border-current px-2 py-0.5 text-xs font-bold" :class="usage === 'reused' ? 'text-emerald-700 dark:text-emerald-400' : 'text-bh-blue dark:text-blue-300'" data-testid="ticket-ip-usage">{{ t('admin.accounts.ticketWorkbench.ipUsage.' + usage) }}</span>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
// 只渲染后端允许的枚举，不把任意缓存文案或代理地址放进标签。
defineProps<{ usage?: string }>()
const { t } = useI18n()
</script>
