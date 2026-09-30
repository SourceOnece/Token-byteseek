<template>
  <div class="space-y-3">
    <p v-if="attributes.route_differences" class="text-sm text-amber-700 dark:text-amber-300">{{ t('admin.modelAttributes.routeDifferences') }}</p>
    <dl class="grid grid-cols-2 gap-x-4 gap-y-3 text-sm">
      <template v-for="field in fields" :key="field">
        <dt class="text-gray-500 dark:text-dark-400">{{ t(`admin.modelAttributes.fields.${field}`) }}</dt>
        <dd class="break-words text-gray-900 dark:text-dark-50">{{ format(attributes[field]) }}</dd>
      </template>
    </dl>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { attributeCapabilities, attributeLimits, type ModelAttributes } from '@/types/modelAttributes'

defineProps<{ attributes: ModelAttributes }>()
const { t } = useI18n()
const fields = ['display_name', ...attributeLimits, 'input_modalities', 'output_modalities', ...attributeCapabilities] as const
function format(value: ModelAttributes[keyof ModelAttributes]) {
  if (value === undefined || value === null) return t('admin.modelAttributes.unknown')
  if (typeof value === 'boolean') return t(value ? 'admin.modelAttributes.supported' : 'admin.modelAttributes.unsupported')
  if (Array.isArray(value)) return value.length ? value.map(item => t(`admin.modelAttributes.modalities.${item}`)).join(', ') : t('admin.modelAttributes.none')
  return typeof value === 'number' ? value.toLocaleString() : value
}
</script>
