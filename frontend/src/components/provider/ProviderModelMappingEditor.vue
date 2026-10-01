<template>
  <ModelMappingEditor
    :model-value="modelValue"
    :title="title"
    :hint="hint ?? t('admin.providers.mapRequestModels')"
    :add-label="t('admin.providers.addMapping')"
    :empty-text="t('admin.providers.modelMappingEmpty')"
    :source-label="sourcePlaceholder ?? t('admin.providers.requestModel')"
    :target-label="targetPlaceholder ?? t('admin.providers.actualModel')"
    :source-placeholder="sourcePlaceholder ?? t('admin.providers.requestModel')"
    :target-placeholder="targetPlaceholder ?? t('admin.providers.actualModel')"
    :field-errors="fieldErrors"
    :test-id="testId"
    @update:model-value="emit('update:modelValue', $event)"
    @add="emit('add', $event)"
    @remove="(row, index) => emit('remove', row, index)"
  >
    <template v-if="$slots['header-actions']" #header-actions>
      <slot name="header-actions" />
    </template>
    <template v-if="presets?.length" #footer>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="preset in presets"
          :key="preset.label"
          type="button"
          :class="['rounded-control px-3 py-1 text-xs transition-colors', preset.color]"
          @click="emit('preset', preset.from, preset.to)"
        >
          + {{ preset.label }}
        </button>
      </div>
    </template>
  </ModelMappingEditor>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ModelMappingEditor from '@/components/common/ModelMappingEditor.vue'
import {
  WILDCARD_ONLY_RULES,
  validateModelMappingRows,
  type ModelMappingRow,
} from '@/utils/modelMappingRules'

/** ProviderMappingPreset 是映射区下方的快捷添加项。 */
export interface ProviderMappingPreset {
  label: string
  from: string
  to: string
  color: string
}

const props = defineProps<{
  modelValue: ModelMappingRow[]
  presets?: ProviderMappingPreset[]
  title?: string
  hint?: string
  sourcePlaceholder?: string
  targetPlaceholder?: string
  wildcardValidation?: boolean
  testId?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [rows: ModelMappingRow[]]
  add: [row: ModelMappingRow]
  remove: [row: ModelMappingRow, index: number]
  preset: [from: string, to: string]
}>()

const { t } = useI18n()

// 通配符提示只做实时展示，提交时仍由 buildModelMappingObject 跳过无效规则。
const fieldErrors = computed(() => {
  if (!props.wildcardValidation) return undefined
  return validateModelMappingRows(props.modelValue, WILDCARD_ONLY_RULES).map((issue) => ({
    from: issue.from ? t('admin.providers.wildcardOnlyAtEnd') : undefined,
    to: issue.to ? t('admin.providers.targetNoWildcard') : undefined,
  }))
})
</script>
