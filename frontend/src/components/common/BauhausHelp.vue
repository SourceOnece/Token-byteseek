<template>
  <!-- 辅助规则按需展开，配置字段与重要风险提示不放入此容器。 -->
  <details class="border-l-2 border-[color:var(--bh-ink)] pl-3 text-xs text-gray-600 dark:text-gray-300" data-testid="bauhaus-help">
    <summary class="w-fit cursor-pointer py-1 font-bold text-bh-blue focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 dark:text-blue-300">{{ title }}</summary>
    <div class="mt-2 space-y-2 leading-relaxed"><slot /></div>
  </details>
</template>
<script setup lang="ts">
defineProps<{ title: string }>()
</script>
