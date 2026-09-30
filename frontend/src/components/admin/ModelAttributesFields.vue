<template>
  <div class="grid gap-4 sm:grid-cols-2">
    <label class="sm:col-span-2">
      <span class="input-label">{{ t('admin.modelAttributes.fields.display_name') }}</span>
      <input class="input" :value="modelValue.display_name ?? ''" :placeholder="t('admin.modelAttributes.inherit')" @input="set('display_name', ($event.target as HTMLInputElement).value || undefined)" />
    </label>
    <label v-for="field in attributeLimits" :key="field">
      <span class="input-label">{{ t(`admin.modelAttributes.fields.${field}`) }}</span>
      <input type="number" min="1" step="1" class="input" :value="modelValue[field] ?? ''" :placeholder="t('admin.modelAttributes.inherit')" @input="set(field, ($event.target as HTMLInputElement).value === '' ? undefined : Number(($event.target as HTMLInputElement).value))" />
    </label>
    <div v-for="field in attributeCapabilities" :key="field">
      <label class="input-label">{{ t(`admin.modelAttributes.fields.${field}`) }}</label>
      <Select :model-value="modelValue[field] === undefined ? 'inherit' : String(modelValue[field])" :options="booleanOptions" @update:model-value="set(field, $event === 'inherit' ? undefined : $event === 'true')" />
    </div>
    <fieldset v-for="field in ['input_modalities', 'output_modalities'] as const" :key="field" class="space-y-2 rounded-control border border-gray-200 p-3 dark:border-dark-600">
      <legend class="px-1 text-sm font-medium">{{ t(`admin.modelAttributes.fields.${field}`) }}</legend>
      <label class="flex items-center gap-2 text-sm">
        <input type="checkbox" class="rounded-compact" :checked="modelValue[field] === undefined" @change="set(field, ($event.target as HTMLInputElement).checked ? undefined : [])" />
        {{ t('admin.modelAttributes.inherit') }}
      </label>
      <div v-if="modelValue[field] !== undefined" class="flex flex-wrap gap-3">
        <label v-for="modality in attributeModalities" :key="modality" class="flex items-center gap-2 text-sm">
          <input type="checkbox" class="rounded-compact" :checked="modelValue[field]?.includes(modality)" @change="toggleModality(field, modality)" />
          {{ t(`admin.modelAttributes.modalities.${modality}`) }}
        </label>
      </div>
    </fieldset>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import { attributeCapabilities, attributeLimits, attributeModalities, type ModelAttributes } from '@/types/modelAttributes'

const props = defineProps<{ modelValue: ModelAttributes }>()
const emit = defineEmits<{ 'update:modelValue': [ModelAttributes] }>()
const { t } = useI18n()
const booleanOptions = computed(() => [
  { value: 'inherit', label: t('admin.modelAttributes.inherit') },
  { value: 'true', label: t('admin.modelAttributes.supported') },
  { value: 'false', label: t('admin.modelAttributes.unsupported') },
])

// 删除属性键代表继承，显式 false 和空数组必须保留。
function set<K extends keyof ModelAttributes>(key: K, value: ModelAttributes[K]) {
  const next = { ...props.modelValue, [key]: value }
  if (value === undefined) delete next[key]
  emit('update:modelValue', next)
}
function toggleModality(field: 'input_modalities' | 'output_modalities', modality: string) {
  const values = props.modelValue[field] ?? []
  set(field, values.includes(modality) ? values.filter(value => value !== modality) : [...values, modality])
}
</script>
