<template>
  <div
    class="group-settings-section space-y-6"
    data-group-field="routing-policy"
  >
    <GroupFormSection
      :title="t('admin.groups.routingPolicy.mapping')"
      :hint="t('admin.groups.routingPolicy.mappingHint')"
    >
      <template #actions>
        <button type="button" class="btn btn-secondary" @click="addMapping">
          {{ t('common.add') }}
        </button>
      </template>
      <div
        v-for="(row, index) in mappingRows"
        :key="row.id"
        class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 rounded-surface border border-gray-200 bg-gray-50/50 p-4 dark:border-dark-600 dark:bg-dark-800/40 md:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)_auto]"
      >
        <input
          v-model="row.source"
          class="input min-w-0 flex-1"
          :aria-label="t('admin.groups.routingPolicy.source')"
          :placeholder="t('admin.groups.routingPolicy.source')"
          required
          @input="publishMappings"
        />
        <Icon
          name="arrowRight"
          size="sm"
          class="hidden md:block"
          aria-hidden="true"
        />
        <input
          v-model="row.target"
          class="input col-start-1 row-start-2 min-w-0 md:col-start-3 md:row-start-1"
          :aria-label="t('admin.groups.routingPolicy.target')"
          :placeholder="t('admin.groups.routingPolicy.target')"
          required
          @input="publishMappings"
        />
        <button
          type="button"
          class="btn btn-ghost btn-icon col-start-2 row-span-2 row-start-1 text-red-500 md:col-start-4 md:row-span-1"
          :aria-label="t('common.delete')"
          @click="removeMapping(index)"
        >
          <Icon name="trash" size="sm" />
        </button>
      </div>
      <p v-if="mappingError" role="alert" class="text-sm text-red-600">
        {{ mappingError }}
      </p>
      <input
        class="sr-only"
        tabindex="-1"
        :value="mappingError ? '' : 'valid'"
        required
        :aria-label="t('admin.groups.routingPolicy.mapping')"
      />
    </GroupFormSection>
    <GroupFormSection>
      <GroupSettingRow
        :id="`${idPrefix}-restrict-models`"
        :model-value="value.restrict_models"
        :label="t('admin.groups.routingPolicy.restrict')"
        :hint="t('admin.groups.routingPolicy.allowlistHint')"
        setting="restrict_models"
        @update:model-value="update({ restrict_models: $event })"
      />
      <template v-if="value.restrict_models">
        <label :for="`${idPrefix}-restriction-source`" class="input-label">{{
          t('admin.groups.settings.restrictionSource')
        }}</label>
        <Select
          :id="`${idPrefix}-restriction-source`"
          :aria-label="t('admin.groups.settings.restrictionSource')"
          :model-value="value.restriction_model_source || 'group_mapped'"
          :options="sourceOptions"
          @update:model-value="
            update({
              restriction_model_source: String(
                $event,
              ) as GroupRoutingPolicy['restriction_model_source'],
            })
          "
        />
        <ModelTagInput
          :aria-label="t('admin.groups.settings.allowedModels')"
          :models="value.allowed_models"
          @update:models="update({ allowed_models: $event })"
        />
      </template>
    </GroupFormSection>
  </div>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GroupRoutingPolicy } from '@/types'
import Select from '@/components/common/Select.vue'
import GroupSettingRow from './GroupSettingRow.vue'
import GroupFormSection from './GroupFormSection.vue'
import Icon from '@/components/icons/Icon.vue'
import ModelTagInput from '@/components/admin/pricing/ModelTagInput.vue'
import { findModelConflict } from '@/components/admin/pricing/types'
import { cloneRoutingPolicy } from './routingPolicy'

const props = withDefaults(
  defineProps<{ modelValue?: GroupRoutingPolicy; idPrefix?: string }>(),
  { idPrefix: 'group-model-policy' },
)
const emit = defineEmits<{ 'update:modelValue': [value: GroupRoutingPolicy] }>()
const { t } = useI18n()
const value = computed(() => cloneRoutingPolicy(props.modelValue))
const sourceOptions = computed(() =>
  ['requested', 'group_mapped', 'upstream'].map((key) => ({
    value: key,
    label: t(`admin.groups.routingPolicy.basis.${key}`),
  })),
)
let rowID = 0
const mappingRows = ref<{ id: number; source: string; target: string }[]>([])
let lastPublished = ''
watch(
  () => [props.modelValue?.model_mapping] as const,
  () => {
    const mapping = value.value.model_mapping
    const signature = JSON.stringify(mapping)
    if (signature === lastPublished) return
    mappingRows.value = Object.entries(mapping).map(([source, target]) => ({
      id: ++rowID,
      source,
      target,
    }))
  },
  { immediate: true, deep: true },
)
const mappingError = computed(() => {
  const sources = mappingRows.value.map((row) => row.source.trim())
  if (
    sources.some((source) => !source) ||
    mappingRows.value.some((row) => !row.target.trim())
  )
    return t('admin.groups.routingPolicy.incompleteMapping')
  return findModelConflict(sources)
    ? t('admin.groups.routingPolicy.conflict')
    : ''
})
function update(patch: Partial<GroupRoutingPolicy>) {
  emit('update:modelValue', { ...value.value, ...patch })
}
function publishMappings() {
  const mapping = Object.fromEntries(
    mappingRows.value.map((row) => [row.source.trim(), row.target.trim()]),
  )
  lastPublished = JSON.stringify(mapping)
  update({ model_mapping: mapping })
}
function addMapping() {
  mappingRows.value.push({ id: ++rowID, source: '', target: '' })
  publishMappings()
}
function removeMapping(index: number) {
  mappingRows.value.splice(index, 1)
  publishMappings()
}
</script>
