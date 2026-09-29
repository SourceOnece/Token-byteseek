<template>
  <!-- 旧版黄色图标格与输入格共边；保留当前防抖和搜索事件。 -->
  <div class="flex w-full">
    <span class="flex h-11 w-11 shrink-0 items-center justify-center border-2 border-gray-950 bg-bh-yellow text-gray-950 dark:border-dark-100" aria-hidden="true">
      <Icon name="search" size="md" :stroke-width="2.5" />
    </span>
    <input
      :value="modelValue"
      type="text"
      class="input -ml-0.5 min-w-0 flex-1"
      :placeholder="placeholder"
      @input="handleInput"
    />
  </div>
</template>

<script setup lang="ts">
import { useDebounceFn } from '@vueuse/core'
import { SEARCH_DEBOUNCE_MS } from '@/constants/ui'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  modelValue: string
  placeholder?: string
  debounceMs?: number
}>(), {
  placeholder: 'Search...',
  debounceMs: SEARCH_DEBOUNCE_MS
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
