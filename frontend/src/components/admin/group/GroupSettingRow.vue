<template>
  <div
    class="flex items-start justify-between gap-4"
    :data-group-setting-row="setting"
  >
    <div class="min-w-0">
      <div class="flex items-center gap-2">
        <span class="text-sm font-medium text-primary-900 dark:text-dark-50">{{
          label
        }}</span>
        <HelpTooltip
          v-if="help"
          :content="help"
          :tooltip-id="`${id}-help`"
          trigger="both"
          :closable="false"
        >
          <template #trigger>
            <button
              type="button"
              class="inline-flex rounded-compact text-gray-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500"
              :aria-label="label"
              :aria-describedby="`${id}-help`"
            >
              <Icon name="questionCircle" size="sm" />
            </button>
          </template>
        </HelpTooltip>
      </div>
      <p v-if="hint" :id="`${id}-hint`" class="input-hint">{{ hint }}</p>
    </div>
    <Toggle
      :id="id"
      :model-value="modelValue"
      :aria-label="label"
      :aria-describedby="hint ? `${id}-hint` : undefined"
      :data-group-setting="setting"
      size="md"
      class="shrink-0"
      @update:model-value="emit('update:modelValue', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'

// 所有布尔设置共用标签、说明和开关的位置，展开内容由调用方紧随其后放置。
defineProps<{
  id: string
  label: string
  modelValue: boolean
  hint?: string
  help?: string
  setting?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
</script>
