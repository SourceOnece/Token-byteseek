<template>
  <button
    type="button"
    @click="toggle"
    class="toggle-control relative inline-flex flex-shrink-0 cursor-pointer rounded-none border-0 p-0 focus:outline-none"
    :class="[
      props.modelValue ? 'toggle-active' : 'bg-gray-300 dark:bg-dark-600',
      props.size === 'sm' ? 'h-5 w-9' : 'h-6 w-11',
      props.disabled && 'cursor-not-allowed opacity-50'
    ]"
    role="switch"
    :aria-checked="props.modelValue"
    :aria-disabled="props.disabled"
    :disabled="props.disabled"
  >
    <!-- 普通规格采用 16px 滑块和 4px 留白；小规格保留现有的 2px 留白。 -->
    <span
      class="pointer-events-none absolute h-4 w-4 transform rounded-none border border-bh-ink bg-white transition-transform duration-150 ease-out"
      :class="[
        props.size === 'sm' ? 'left-0.5 top-0.5' : 'left-1 top-1',
        props.modelValue ? (props.size === 'sm' ? 'translate-x-4' : 'translate-x-5') : 'translate-x-0'
      ]"
    />
  </button>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  modelValue: boolean
  size?: 'sm' | 'md'
  disabled?: boolean
}>(), {
  size: 'md',
  disabled: false
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

// 开关保持受控，异步保存或确认期间由调用方决定何时更新值。
function toggle() {
  if (props.disabled) return
  emit('update:modelValue', !props.modelValue)
}
</script>

<style scoped>
/* 内描边不占布局宽度，保留两种规格滑块的间距和行程。 */
.toggle-control {
  box-shadow: inset 0 0 0 2px var(--bh-ink), var(--bh-shadow);
  transition: translate 150ms ease, box-shadow 150ms ease, background-color 150ms ease;
}

@media (hover: hover) {
  .toggle-control:hover:not(:disabled) { translate: -1px -1px; }
}

.toggle-control:active:not(:disabled) {
  translate: 2px 2px;
  box-shadow: inset 0 0 0 2px var(--bh-ink), 2px 2px 0 var(--bh-shadow-ink);
}

.toggle-control:focus-visible {
  outline: 2px solid var(--bh-blue);
  outline-offset: 4px;
}

.toggle-control:disabled {
  box-shadow: inset 0 0 0 2px var(--bh-ink);
}

@media (prefers-reduced-motion: reduce) {
  .toggle-control, .toggle-control > span { transition: none; }
}
</style>
