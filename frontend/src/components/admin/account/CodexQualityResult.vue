<template>
  <article class="quality-result min-w-0 space-y-3 p-4">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="min-w-0">
        <p class="break-all font-semibold">{{ result.email || result.account_name || `#${result.account_id}` }}</p>
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ result.account_name }} · #{{ result.account_id }}</p>
      </div>
      <span :class="['quality-label', qualityStatusClass(result.status)]">
        {{ t(`admin.accounts.quality.status.${result.status}`) }}
      </span>
    </div>
    <div class="grid gap-3 sm:grid-cols-2">
      <div class="border-l-4 border-bh-blue pl-3"><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.quality.model') }}</p><strong class="break-all text-lg text-bh-blue dark:text-blue-300">{{ result.model }}</strong></div>
      <div class="border-l-4 border-bh-yellow pl-3"><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.quality.effort') }}</p><strong class="text-lg text-yellow-700 dark:text-bh-yellow">{{ result.reasoning_effort || t('admin.accounts.quality.effortDefault') }}</strong></div>
    </div>
    <p class="text-xs text-gray-500 dark:text-gray-400">{{ result.finished_at }}</p>
    <p v-if="result.api_protocol" class="text-sm font-bold text-bh-blue dark:text-blue-300">{{ t('admin.accounts.quality.actualProtocol') }}：{{ qualityProtocolLabel(result.api_protocol) }}</p>
    <p class="text-sm font-bold text-bh-blue dark:text-blue-300">{{ t('admin.accounts.quality.timeout') }}：{{ result.timeout_seconds || 120 }}s</p>
    <p class="text-sm">{{ t('admin.accounts.quality.keyword') }}：<strong class="break-all">{{ result.keyword }}</strong></p>
    <details>
      <summary class="cursor-pointer text-sm font-medium">{{ t('admin.accounts.quality.prompt') }}</summary>
      <p class="mt-2 whitespace-pre-wrap break-words text-sm">{{ result.prompt }}</p>
    </details>
    <!-- 回答始终作为纯文本渲染，不能执行模型返回的 HTML。 -->
    <details class="border-2 border-gray-300 dark:border-dark-500" data-testid="quality-answer">
      <summary class="cursor-pointer p-3 font-bold focus-visible:outline focus-visible:outline-2">{{ t('admin.accounts.quality.viewAnswer') }}</summary>
      <pre class="max-h-72 overflow-auto whitespace-pre-wrap break-words bg-gray-50 p-3 font-sans text-sm dark:bg-dark-900">{{ result.response_text || t('admin.accounts.quality.noAnswer') }}</pre>
    </details>
    <p v-if="result.error" class="break-words text-sm font-semibold" :class="qualityStatusClass(result.status)">{{ result.error }}</p>
    <p class="text-xs font-semibold">
      {{ result.scheduling_applied
        ? t(result.schedulable ? 'admin.accounts.quality.schedulingOn' : 'admin.accounts.quality.schedulingOff')
        : t('admin.accounts.quality.schedulingUnchanged') }}
    </p>
  </article>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { CodexQualityResult } from '@/api/admin/codexQuality'
import { qualityStatusClass, qualityProtocolLabel } from './codexQualityPresentation'
defineProps<{ result: CodexQualityResult }>()
const { t } = useI18n()
</script>

<style scoped>
.quality-result { border: 2px solid var(--bh-ink); background: var(--bh-surface, #fff); box-shadow: var(--bh-shadow-sm); }
.quality-label { border: 2px solid currentColor; padding: .25rem .5rem; font-weight: 800; font-size: .75rem; }
</style>
