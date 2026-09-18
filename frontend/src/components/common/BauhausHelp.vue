<template>
  <!-- 辅助规则按需展开，配置字段与重要风险提示不放入此容器。 -->
  <details class="bauhaus-help text-sm text-gray-600 dark:text-gray-300" data-testid="bauhaus-help">
    <summary class="flex min-h-11 cursor-pointer list-none items-center justify-between gap-4 py-3 font-semibold text-bh-blue focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 dark:text-blue-300"><span>{{ title }}</span><span class="help-mark" aria-hidden="true" /></summary>
    <div class="space-y-3 pb-4 leading-relaxed"><slot /></div>
  </details>
</template>
<script setup lang="ts">
defineProps<{ title: string }>()
</script>
<style scoped>
/* 整行可展开，符号随状态变化；与表单同一字号和分隔节奏。 */
.bauhaus-help { border-block: 1px solid var(--bh-ink); }
summary::-webkit-details-marker { display: none; }
.help-mark { position: relative; width: 1rem; height: 1rem; flex: 0 0 1rem; }
.help-mark::before, .help-mark::after { content: ''; position: absolute; background: currentColor; }
.help-mark::before { width: 1rem; height: 2px; left: 0; top: 7px; }
.help-mark::after { width: 2px; height: 1rem; left: 7px; top: 0; }
details[open] .help-mark::after { display: none; }
</style>
