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
    <p class="break-all text-xs text-gray-600 dark:text-gray-300">
      {{ result.model }} · {{ result.reasoning_effort || t('admin.accounts.quality.effortDefault') }} · {{ result.finished_at }}
    </p>
    <p class="text-sm">{{ t('admin.accounts.quality.keyword') }}：<strong class="break-all">{{ result.keyword }}</strong></p>
    <details>
      <summary class="cursor-pointer text-sm font-medium">{{ t('admin.accounts.quality.prompt') }}</summary>
      <p class="mt-2 whitespace-pre-wrap break-words text-sm">{{ result.prompt }}</p>
    </details>
    <!-- 回答始终作为纯文本渲染，不能执行模型返回的 HTML。 -->
    <pre class="max-h-72 overflow-auto whitespace-pre-wrap break-words border-2 border-gray-300 bg-gray-50 p-3 font-sans text-sm dark:border-dark-500 dark:bg-dark-900">{{ result.response_text || t('admin.accounts.quality.noAnswer') }}</pre>
    <p v-if="result.error" class="break-words text-sm text-red-600 dark:text-red-400">{{ result.error }}</p>
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
import { qualityStatusClass } from './codexQualityPresentation'
defineProps<{ result: CodexQualityResult }>()
const { t } = useI18n()
</script>

<style scoped>
.quality-result { border: 2px solid var(--bh-ink); background: var(--bh-surface, #fff); box-shadow: var(--bh-shadow-sm); }
.quality-label { border: 2px solid currentColor; padding: .25rem .5rem; font-weight: 800; font-size: .75rem; }
</style>
