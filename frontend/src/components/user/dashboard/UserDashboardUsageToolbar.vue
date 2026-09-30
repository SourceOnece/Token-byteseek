<template>
  <!-- 放在页面标题行；窄屏时标题行换行，宽度按页面内边距封顶，控件在内部继续换行 -->
  <div class="flex max-w-[calc(100vw-2rem)] flex-wrap items-center gap-2 md:max-w-[calc(100vw-3rem)]">
    <FilterDropdown :active-count="activeCount" wide @reset="resetFilters">
      <div class="grid grid-cols-1 gap-x-4 gap-y-3 sm:grid-cols-2">
        <div>
          <label class="input-label">{{ t('usage.apiKeyFilter') }}</label>
          <Select v-model="filters.api_key_id" :options="apiKeyOptions" searchable data-testid="usage-filter-api-key" @change="applyFilters" />
        </div>
        <div>
          <label class="input-label">{{ t('usage.model') }}</label>
          <Select v-model="filters.model" :options="modelOptions" searchable data-testid="usage-filter-model" @change="applyFilters" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.usage.group') }}</label>
          <Select v-model="filters.group_id" :options="groupOptions" searchable data-testid="usage-filter-group" @change="applyFilters" />
        </div>
        <div>
          <label class="input-label">{{ t('usage.type') }}</label>
          <Select v-model="filters.request_type" :options="requestTypeOptions" data-testid="usage-filter-request-type" @change="applyFilters" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.usage.billingType') }}</label>
          <Select v-model="filters.billing_type" :options="billingTypeOptions" data-testid="usage-filter-billing-type" @change="applyFilters" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.usage.billingMode') }}</label>
          <Select v-model="filters.billing_mode" :options="billingModeOptions" data-testid="usage-filter-billing-mode" @change="applyFilters" />
        </div>
        <div>
          <label class="input-label">{{ t('usage.compactionFilter') }}</label>
          <Select v-model="filters.native_compaction_v2" :options="compactionOptions" data-testid="usage-filter-compaction" @change="applyFilters" />
        </div>
      </div>
    </FilterDropdown>
    <button
      type="button"
      class="btn btn-secondary shrink-0 btn-icon"
      :disabled="refreshing"
      :title="t('common.refresh')"
      :aria-label="t('common.refresh')"
      data-testid="usage-refresh"
      @click="emit('refresh')"
    >
      <Icon name="refresh" size="sm" :class="refreshing ? 'animate-spin' : ''" />
    </button>
    <DateRangePicker
      :start-date="pickerStart"
      :end-date="pickerEnd"
      @change="selectCustomRange"
    />
    <div
      class="flex flex-wrap items-stretch gap-2"
      role="group"
      :aria-label="t('dashboard.usageChart.rangeLabel')"
    >
      <button
        v-for="option in rangeOptions"
        :key="option.value"
        type="button"
        class="btn btn-sm flex items-center"
        :class="rangePreset === option.value ? 'btn-warning' : 'btn-secondary'"
        :aria-pressed="rangePreset === option.value"
        :data-testid="`usage-range-${option.value}`"
        @click="selectPreset(option.value)"
      >
        {{ option.label }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import FilterDropdown from '@/components/common/FilterDropdown.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { injectUsageChartState, USAGE_RANGE_PRESETS } from './usageChartState'

// 刷新范围覆盖整个仪表盘，由页面统一处理。
defineProps<{ refreshing?: boolean }>()
const emit = defineEmits<{ (event: 'refresh'): void }>()

const { t } = useI18n()
const {
  rangePreset,
  pickerStart,
  pickerEnd,
  filterState,
  applyFilters,
  resetFilters,
  selectPreset,
  selectCustomRange,
} = injectUsageChartState()
const {
  filters,
  activeCount,
  apiKeyOptions,
  modelOptions,
  groupOptions,
  requestTypeOptions,
  billingTypeOptions,
  billingModeOptions,
  compactionOptions,
} = filterState

const rangeOptions = computed(() => USAGE_RANGE_PRESETS.map((value) => ({
  value,
  label: t(`dashboard.usageChart.ranges.${value}`),
})))
</script>
