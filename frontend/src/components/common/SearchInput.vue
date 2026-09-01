<template>
  <!-- 构成式搜索框：黄色图标格 + 输入格，共享 2px 墨线 -->
  <div class="flex w-full">
    <span
      class="flex h-11 w-11 flex-shrink-0 items-center justify-center border-2 border-gray-950 bg-bh-yellow text-gray-950 dark:border-dark-100"
      aria-hidden="true"
    >
      <Icon name="search" size="md" :stroke-width="2.5" />
    </span>
    <input
      :value="modelValue"
      type="text"
      class="input -ml-[2px] flex-1"
      :placeholder="placeholder"
      @input="handleInput"
    />
  </div>
</template>

<script setup lang="ts">
import { useDebounceFn } from '@vueuse/core'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  modelValue: string
  placeholder?: string
  debounceMs?: number
}>(), {
  placeholder: 'Search...',
  debounceMs: 300
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'search', value: string): void
}>()

const debouncedEmitSearch = useDebounceFn((value: string) => {
  emit('search', value)
}, props.debounceMs)

const handleInput = (event: Event) => {
  const value = (event.target as HTMLInputElement).value
  emit('update:modelValue', value)
  debouncedEmitSearch(value)
}
</script>
