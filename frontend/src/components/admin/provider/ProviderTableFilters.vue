<template>
  <div class="flex min-w-0 flex-nowrap items-center gap-3">
    <SearchInput
      :model-value="searchQuery"
      :placeholder="t('admin.providers.searchProviders')"
      class="min-w-0 flex-1 sm:flex-none sm:w-56 lg:w-52"
      @update:model-value="$emit('update:searchQuery', $event)"
      @search="$emit('change')"
    />

    <FilterDropdown :active-count="activeFilterCount" :columns="2" :description="t('admin.providers.filterHint')" trigger-test-id="provider-filters-toggle" @reset="clearFilters">
<FilterField :label="t('admin.accounts.quality.allResults')">

            <Select :model-value="filters.quality_status" :options="qualityOptions" @update:model-value="value => $emit('update:filters', { ...filters, quality_status: value })" @change="$emit('change')" />
          </FilterField>
<FilterField :label="t('admin.providers.columns.platform')">

            <Select :model-value="filters.platform" class="w-full" :options="pOpts" @update:model-value="updatePlatform" @change="$emit('change')" />
          </FilterField>
<FilterField :label="t('admin.providers.columns.type')">

            <Select :model-value="filters.type" class="w-full" :options="tOpts" @update:model-value="updateType" @change="$emit('change')" />
          </FilterField>
<FilterField :label="t('admin.providers.columns.status')">

            <Select :model-value="filters.status" class="w-full" :options="sOpts" @update:model-value="updateStatus" @change="$emit('change')" />
          </FilterField>
<FilterField :label="t('admin.providers.privacyFilter')">

            <Select :model-value="filters.privacy_mode" class="w-full" :options="privacyOpts" @update:model-value="updatePrivacyMode" @change="$emit('change')" />
          </FilterField>
<FilterField :label="t('admin.providers.columns.groups')">

            <Select :model-value="filters.group" class="w-full" :options="gOpts" searchable @update:model-value="updateGroup" @change="$emit('change')" />
          </FilterField>
<FilterField :label="t('admin.accounts.ticketWorkbench.filter')">

            <Select v-model="ticketModeDraft" :options="ticketOptions" data-testid="ticket-type-filter" @change="applyTicketFilter" />
          </FilterField>
<FilterField :label="t('admin.accounts.ticketWorkbench.actualLength')" :value-text="String(ticketLengthDraft || '')" @clear="ticketLengthDraft = ''; applyTicketFilter()">

            <input id="ticket-actual-length-filter" v-model="ticketLengthDraft" type="number" min="6" max="8192" step="1" class="input w-full" :placeholder="t('admin.accounts.ticketWorkbench.filterLength')" :aria-invalid="ticketLengthInvalid" data-testid="ticket-actual-length-filter" @input="scheduleTicketFilter" @keydown.enter.prevent="applyTicketFilter" />
            <p v-if="ticketLengthInvalid" role="alert" class="input-hint text-bh-red dark:text-red-400">{{ t('admin.accounts.ticketWorkbench.invalidActualLength') }}</p>
            <p v-else class="input-hint">{{ t('admin.accounts.ticketWorkbench.actualLengthHint') }}</p>
          </FilterField>
</FilterDropdown>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import FilterDropdown from '@/components/common/FilterDropdown.vue'
import FilterField from '@/components/common/FilterField.vue'
import type { AdminGroup } from '@/types'
import { CONCRETE_PLATFORM_OPTIONS } from '@/constants/platforms'

const props = defineProps<{ searchQuery: string; filters: Record<string, any>; groups?: AdminGroup[] }>()
const emit = defineEmits(['update:searchQuery', 'update:filters', 'change'])
const { t } = useI18n()

const filterKeys = ['platform', 'type', 'status', 'privacy_mode', 'group', 'quality_status', 'ticket_filter'] as const
const ticketModes = ['', 'on', 'off', 'configured', 'proxy_account', 'proxy_gateway', 'fixed', 'rotate', 'dynamic']
const ticketOptions = computed(() => ticketModes.map(value => ({ value, label: t('admin.accounts.ticketWorkbench.filters.' + (value || 'all')) })))
const ticketModeDraft = ref(''), ticketLengthDraft = ref<string | number>('')
let ticketTimer: ReturnType<typeof setTimeout> | undefined
const ticketLengthInvalid = computed(() => {
  const raw = String(ticketLengthDraft.value).trim(), value = Number(raw)
  return raw !== '' && (!/^\d+$/.test(raw) || !Number.isInteger(value) || value < 6 || value > 8192)
})
// 两列合并为兼容的原ticket_filter参数；只新增实际长度语法，不把旧目标值改解读成实际值。
watch(() => props.filters.ticket_filter, raw => {
  clearTimeout(ticketTimer)
  const parts = String(raw || '').split(',')
  ticketModeDraft.value = parts.find(value => ticketModes.includes(value)) || ''
  ticketLengthDraft.value = parts.find(value => value.startsWith('actual_length:'))?.slice(14) || ''
  if (String(raw || '').startsWith('length:')) {
    emit('update:filters', { ...props.filters, ticket_filter: '' })
    emit('change')
  }
}, { immediate: true })
function applyTicketFilter() {
  clearTimeout(ticketTimer)
  if (ticketLengthInvalid.value) return
  const raw = String(ticketLengthDraft.value).trim()
  const value = [ticketModeDraft.value, raw ? 'actual_length:' + Number(raw) : ''].filter(Boolean).join(',')
  if (value === String(props.filters.ticket_filter || '')) return
  emit('update:filters', { ...props.filters, ticket_filter: value })
  emit('change')
}
function scheduleTicketFilter() { clearTimeout(ticketTimer); ticketTimer = setTimeout(applyTicketFilter, 450) }
const qualityOptions = computed(() => [
  { value: '', label: t('admin.accounts.quality.allResults') },
  ...['full', 'degraded', 'failed', 'untested', 'cancelled', 'stale'].map(value => ({ value, label: value === 'untested' ? t('admin.accounts.quality.untested') : t(`admin.accounts.quality.status.${value}`) }))
])

const activeFilterCount = computed(() => filterKeys.filter((key) => String(props.filters?.[key] ?? '').trim() !== '').length)

const updatePlatform = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, platform: value }) }
const updateType = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, type: value }) }
const updateStatus = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, status: value }) }
const updatePrivacyMode = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, privacy_mode: value }) }
const updateGroup = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, group: value }) }

const clearFilters = () => {
  clearTimeout(ticketTimer)
  ticketModeDraft.value = ''; ticketLengthDraft.value = ''
  const nextFilters = { ...props.filters }
  for (const key of filterKeys) nextFilters[key] = ''
  emit('update:filters', nextFilters)
  emit('change')
}

// 票据输入延迟任务归本字段所有，浮层监听由公共组件统一管理。
onBeforeUnmount(() => { clearTimeout(ticketTimer) })

const pOpts = computed(() => [{ value: '', label: t('admin.providers.allPlatforms') }, ...CONCRETE_PLATFORM_OPTIONS])
const tOpts = computed(() => [
  { value: '', label: t('admin.providers.allTypes') },
  { value: 'oauth', label: t('admin.providers.oauthType') },
  { value: 'setup-token', label: t('admin.providers.setupToken') },
  { value: 'apikey', label: t('admin.providers.apiKey') },
  { value: 'service_account', label: t('admin.providers.serviceAccount') },
  { value: 'bedrock', label: 'AWS Bedrock' },
  { value: 'cosy', label: t('admin.providers.types.qoderCosy') }
])
const sOpts = computed(() => [
  { value: '', label: t('admin.providers.allStatus') },
  { value: 'active', label: t('admin.providers.status.active') },
  { value: 'inactive', label: t('admin.providers.status.inactive') },
  { value: 'error', label: t('admin.providers.status.error') },
  { value: 'rate_limited', label: t('admin.providers.status.rateLimited') },
  { value: 'temp_unschedulable', label: t('admin.providers.status.tempUnschedulable') },
  { value: 'unschedulable', label: t('admin.providers.status.unschedulable') }
])
const privacyOpts = computed(() => [
  { value: '', label: t('admin.providers.allPrivacyModes') },
  { value: '__unset__', label: t('admin.providers.privacyUnset') },
  { value: 'training_off', label: 'Privacy' },
  { value: 'training_set_cf_blocked', label: 'CF' },
  { value: 'training_set_failed', label: 'Fail' }
])
const gOpts = computed(() => [
  { value: '', label: t('admin.providers.allGroups') },
  { value: 'ungrouped', label: t('admin.providers.ungroupedGroup') },
  ...(props.groups || []).map(g => ({
    value: String(g.id),
    // 管理端保留禁用分组可见性，后缀只提示状态，不阻止筛选。
    label: g.status === 'active' ? g.name : `${g.name} (${t('common.inactive')})`
  }))
])
</script>
