<template>
  <div class="border-t pt-4">
    <div class="mb-3 flex items-center justify-between gap-3">
      <label :for="fieldId" class="text-sm font-bold text-bh-blue dark:text-blue-300">{{ t('admin.groups.modelAllowlist.title') }}</label>
      <Toggle :model-value="modelValue.enabled" :aria-label="t('admin.groups.modelAllowlist.title')" @update:model-value="setEnabled" />
    </div>
    <textarea v-if="modelValue.enabled" :id="fieldId" v-model="modelText" class="input min-h-28 w-full font-mono text-sm" spellcheck="false" :aria-label="t('admin.groups.modelAllowlist.models')" />
  </div>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import type { ModelsListConfig } from '@/types'

const props = defineProps<{ modelValue: ModelsListConfig }>()
const emit = defineEmits<{ 'update:modelValue': [value: ModelsListConfig] }>()
const { t } = useI18n()
const fieldId = useId()
// 编辑时保留空行与尾部空白，提交后由后端统一规范化，避免输入换行时跳光标。
const modelText = computed({
  get: () => props.modelValue.models.join('\n'),
  set: value => emit('update:modelValue', { ...props.modelValue, models: value.split('\n') })
})
const setEnabled = (enabled: boolean) => emit('update:modelValue', { ...props.modelValue, enabled })
</script>
