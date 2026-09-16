<template>
  <div>
    <div class="mb-2 flex items-center justify-between gap-2">
      <label class="input-label mb-0">{{ t('admin.accounts.opencodeGo.protocolRules.title') }}</label>
      <button
        type="button"
        class="btn btn-secondary h-8 w-8 p-0"
        :title="t('admin.accounts.opencodeGo.protocolRules.restoreDefaults')"
        :aria-label="t('admin.accounts.opencodeGo.protocolRules.restoreDefaults')"
        @click="restoreDefaults"
      >
        <Icon name="refresh" size="sm" />
      </button>
    </div>
    <p class="input-hint mb-2">{{ t('admin.accounts.opencodeGo.protocolRules.hint') }}</p>
    <div v-if="rows.length > 0" class="mb-2 space-y-2">
      <div
        v-for="(row, index) in rows"
        :key="getRowKey(row)"
        class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]"
      >
        <input
          v-model="row.pattern"
          type="text"
          class="input col-span-2 min-w-0 font-mono text-sm sm:col-span-1"
          maxlength="128"
          :placeholder="t('admin.accounts.opencodeGo.protocolRules.patternPlaceholder')"
          :data-testid="`opencode-go-protocol-pattern-${index}`"
        />
        <Select v-model="row.protocol" class="min-w-0" :options="protocolOptions" :data-testid="`opencode-go-protocol-select-${index}`" :aria-label="t('admin.accounts.cnProviders.apiProtocol.title')" />
        <button
          type="button"
          class="btn btn-danger h-9 w-9 p-0"
          :title="t('admin.accounts.opencodeGo.protocolRules.remove')"
          :aria-label="t('admin.accounts.opencodeGo.protocolRules.remove')"
          @click="removeRow(index)"
        >
          <Icon name="trash" size="sm" />
        </button>
      </div>
    </div>
    <div
      class="mb-2 flex items-center gap-2 rounded-lg border border-dashed border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-500 dark:border-dark-600 dark:bg-dark-800/60 dark:text-gray-400"
      data-testid="opencode-go-protocol-fallback"
    >
      <span class="flex-1 font-mono">*</span>
      <span>{{ t('admin.accounts.opencodeGo.protocolRules.fallback') }}</span>
    </div>
    <button
      type="button"
      class="btn btn-secondary w-full gap-2"
      :disabled="rows.length >= 64"
      data-testid="opencode-go-protocol-add-rule"
      @click="addRow"
    >
      <Icon name="plus" size="sm" />
      {{ t('admin.accounts.opencodeGo.protocolRules.add') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Select from '@/components/common/Select.vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { createStableObjectKeyResolver } from '@/utils/stableObjectKey'
import {
  cloneOpenCodeGoProtocolRules,
  defaultOpenCodeProtocolRules,
  type OpenCodeAccountMode,
  type OpenCodeGoProtocolRule
} from '@/components/account/credentialsBuilder'

const props = withDefaults(defineProps<{
  rows: OpenCodeGoProtocolRule[]
  plan?: OpenCodeAccountMode
}>(), {
  plan: 'go'
})

const emit = defineEmits<{
  (e: 'update:rows', rows: OpenCodeGoProtocolRule[]): void
}>()

const { t } = useI18n()
const protocolOptions = computed(() => [
  { value: 'chat_completions', label: t('admin.accounts.cnProviders.apiProtocol.chatCompletions') },
  { value: 'responses', label: t('admin.accounts.cnProviders.apiProtocol.responses') },
  { value: 'anthropic', label: t('admin.accounts.cnProviders.apiProtocol.anthropic') }
])
const getRowKey = createStableObjectKeyResolver<OpenCodeGoProtocolRule>('opencode-go-protocol-rule')

const addRow = () => {
  emit('update:rows', [...props.rows, { pattern: '', protocol: 'chat_completions' }])
}

const removeRow = (index: number) => {
  emit('update:rows', props.rows.filter((_, i) => i !== index))
}

const restoreDefaults = () => {
  emit('update:rows', cloneOpenCodeGoProtocolRules(defaultOpenCodeProtocolRules(props.plan)))
}
</script>

