<template>
  <GroupFormSection :title="t('admin.groups.settings.compatibility')">
    <GroupSettingRow
      v-for="feature in features"
      :id="`${idPrefix}-${feature.key}`"
      :key="feature.key"
      :label="t(`admin.groups.routingPolicy.${feature.label}`)"
      :hint="t(`admin.groups.routingPolicy.${feature.label}Hint`)"
      :model-value="featureEnabled(feature.key)"
      :setting="feature.key"
      @update:model-value="setFeature(feature.key, $event)"
    />
  </GroupFormSection>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { GroupRoutingPolicy } from '@/types'
import { cloneRoutingPolicy } from './routingPolicy'
import GroupFormSection from './GroupFormSection.vue'
import GroupSettingRow from './GroupSettingRow.vue'

const props = defineProps<{
  idPrefix: string
  modelValue: GroupRoutingPolicy
}>()
const emit = defineEmits<{ 'update:modelValue': [value: GroupRoutingPolicy] }>()
const { t } = useI18n()
const features = [
  { key: 'web_search_emulation', label: 'webSearch' },
  { key: 'bedrock_cc_compat', label: 'bedrock' },
]

function featureEnabled(key: string) {
  const raw = props.modelValue.features_config[key]
  if (typeof raw === 'boolean') return raw
  return !!(
    raw &&
    typeof raw === 'object' &&
    (raw as Record<string, unknown>).anthropic === true
  )
}

function setFeature(key: string, enabled: boolean) {
  // 从当前完整策略合并，避免跨页编辑功能时覆盖模型草稿或其他平台的历史设置。
  const policy = cloneRoutingPolicy(props.modelValue)
  const raw = policy.features_config[key]
  const values =
    raw && typeof raw === 'object'
      ? { ...(raw as Record<string, boolean>) }
      : {}
  policy.features_config[key] = { ...values, anthropic: enabled }
  emit('update:modelValue', policy)
}
</script>
