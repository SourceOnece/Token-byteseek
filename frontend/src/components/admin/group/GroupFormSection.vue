<template>
  <section class="group-settings-section space-y-4">
    <div v-if="title" class="flex flex-wrap items-start justify-between gap-3">
      <div class="min-w-0 flex-1">
        <h4 class="text-sm font-semibold text-primary-900 dark:text-dark-50">
          {{ title }}
        </h4>
        <p v-if="hint" class="input-hint">{{ hint }}</p>
      </div>
      <slot name="actions" />
    </div>
    <slot />
  </section>
</template>

<script setup lang="ts">
// 分区仅提供标题和间距，复杂规则的边框与背景由规则编辑器拥有。
defineProps<{ title?: string; hint?: string }>()
</script>

<style scoped>
.group-settings-section + .group-settings-section {
  @apply border-t border-gray-200 pt-6 dark:border-dark-600;
}
</style>
