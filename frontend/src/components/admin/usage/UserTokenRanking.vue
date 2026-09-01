<template>
  <!-- 用量页"用户排行"tab 内容：无卡片外观，依赖父级统一卡片；筛选/时间范围复用页面级筛选栏 -->
  <div>
    <!-- 排行说明与条数控制 -->
    <div class="usage-ranking-admin-header flex flex-wrap items-center justify-between gap-3 px-4 py-3 sm:px-6">
      <p class="text-xs font-semibold text-gray-600 dark:text-dark-300">{{ t('admin.usage.tokenRanking.subtitle') }}</p>
      <div class="flex items-center gap-3">
        <span v-if="!loading && items.length > 0" class="text-xs font-mono font-bold text-gray-700 dark:text-dark-200">
          {{ t('admin.usage.tokenRanking.userCount', { count: items.length }) }}
        </span>
        <div class="w-28">
          <Select v-model="limit" :options="limitOptions" @change="load" />
        </div>
      </div>
    </div>

    <!-- 用户 Token 排行表 -->
    <div class="overflow-x-auto">
      <table class="usage-ranking-admin-table w-full min-w-max">
        <thead>
          <tr>
            <th class="w-16 px-4 py-3 text-left text-xs font-extrabold tracking-wider text-gray-950 sm:px-6">#</th>
            <th class="px-4 py-3 text-left text-xs font-extrabold tracking-wider text-gray-950">
              {{ t('admin.usage.tokenRanking.columns.user') }}
            </th>
            <th
              v-for="col in sortableColumns"
              :key="col.key"
              class="usage-ranking-sort cursor-pointer select-none whitespace-nowrap px-4 py-3 text-right text-xs font-extrabold uppercase tracking-wider transition-colors"
              :class="sortBy === col.key ? 'usage-ranking-sort-active' : ''"
              @click="setSort(col.key)"
            >
              {{ t(col.label) }}
              <span v-if="sortBy === col.key" aria-hidden="true">↓</span>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td :colspan="sortableColumns.length + 2" class="py-12 text-center">
              <LoadingSpinner />
            </td>
          </tr>
          <tr v-else-if="items.length === 0">
            <td :colspan="sortableColumns.length + 2" class="py-12 text-center text-sm font-semibold text-gray-600">
              {{ t('admin.dashboard.noDataAvailable') }}
            </td>
          </tr>
          <tr
            v-for="(item, index) in items"
            v-else
            :key="item.user_id"
            class="usage-ranking-admin-row cursor-pointer transition-colors"
            :class="index < 3 ? RANK_ROW_CLASSES[index] : ''"
            :title="t('admin.usage.tokenRanking.rowHint')"
            @click="$emit('select-user', item.user_id, item.email)"
          >
            <td class="px-4 py-3 sm:px-6">
              <span
                v-if="index < 3"
                class="inline-flex h-7 w-7 items-center justify-center border-2 border-gray-950 text-xs font-extrabold"
                :class="RANK_BADGE_CLASSES[index]"
              >{{ index + 1 }}</span>
              <span v-else class="inline-block w-6 text-center font-mono text-sm font-bold tabular-nums text-gray-600">{{ index + 1 }}</span>
            </td>
            <td class="max-w-[260px] truncate px-4 py-3 text-sm font-bold text-gray-800 dark:text-dark-100" :title="item.email">
              {{ item.email || `User #${item.user_id}` }}
              <span class="ml-1 font-normal text-gray-400 dark:text-gray-500">#{{ item.user_id }}</span>
            </td>
            <td :class="['whitespace-nowrap px-4 py-3 text-right font-mono text-sm font-semibold tabular-nums text-gray-600 dark:text-dark-300', index < 3 ? RANK_VALUE_CLASSES[index] : '']">{{ item.requests.toLocaleString() }}</td>
            <td :class="['whitespace-nowrap px-4 py-3 text-right font-mono text-sm font-semibold tabular-nums text-gray-600 dark:text-dark-300', index < 3 ? RANK_VALUE_CLASSES[index] : '']">{{ fmtTokens(item.input_tokens) }}</td>
            <td :class="['whitespace-nowrap px-4 py-3 text-right font-mono text-sm font-semibold tabular-nums text-gray-600 dark:text-dark-300', index < 3 ? RANK_VALUE_CLASSES[index] : '']">{{ fmtTokens(item.output_tokens) }}</td>
            <td :class="['whitespace-nowrap px-4 py-3 text-right font-mono text-sm font-semibold tabular-nums text-gray-600 dark:text-dark-300', index < 3 ? RANK_VALUE_CLASSES[index] : '']">{{ fmtTokens(item.cache_tokens) }}</td>
            <td :class="['usage-ranking-total whitespace-nowrap px-4 py-3 text-right font-mono text-sm font-extrabold tabular-nums', index < 3 ? RANK_VALUE_CLASSES[index] : '']">{{ fmtTokens(item.total_tokens) }}</td>
            <td :class="['usage-ranking-cost whitespace-nowrap px-4 py-3 text-right font-mono text-sm font-extrabold tabular-nums', index < 3 ? RANK_VALUE_CLASSES[index] : '']">${{ fmtCost(item.actual_cost) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getUserBreakdown, type UserBreakdownParams } from '@/api/admin/dashboard'
import { formatCompactNumber, formatCostFixed } from '@/utils/format'
import type { UserBreakdownItem } from '@/types'
import Select from '@/components/common/Select.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'

const props = defineProps<{
  startDate: string
  endDate: string
  filters: Record<string, unknown>
  model?: string
}>()

defineEmits<{ (e: 'select-user', userId: number, email: string): void }>()

const { t } = useI18n()

type SortKey = NonNullable<UserBreakdownParams['sort_by']>
const sortableColumns: { key: SortKey; label: string }[] = [
  { key: 'requests', label: 'admin.usage.tokenRanking.columns.requests' },
  { key: 'input_tokens', label: 'admin.usage.tokenRanking.columns.inputTokens' },
  { key: 'output_tokens', label: 'admin.usage.tokenRanking.columns.outputTokens' },
  { key: 'cache_tokens', label: 'admin.usage.tokenRanking.columns.cacheTokens' },
  { key: 'total_tokens', label: 'admin.usage.tokenRanking.columns.totalTokens' },
  { key: 'actual_cost', label: 'admin.usage.tokenRanking.columns.cost' },
]

const limitOptions = [
  { value: 20, label: 'Top 20' },
  { value: 50, label: 'Top 50' },
  { value: 100, label: 'Top 100' },
  { value: 200, label: 'Top 200' },
]

// 前三名固定使用包豪斯红、黄、蓝，数值与徽章保持同色。
const RANK_BADGE_CLASSES = [
  'rank-admin-badge-red',
  'rank-admin-badge-gold',
  'rank-admin-badge-blue',
]
const RANK_ROW_CLASSES = ['rank-admin-row-red', 'rank-admin-row-gold', 'rank-admin-row-blue']
const RANK_VALUE_CLASSES = ['rank-value-red', 'rank-value-yellow', 'rank-value-blue']

const items = ref<UserBreakdownItem[]>([])
const loading = ref(false)
const sortBy = ref<SortKey>('total_tokens')
const limit = ref(50)
let reqSeq = 0

const fmtTokens = (v: number) => formatCompactNumber(v)
const fmtCost = (v: number) => formatCostFixed(v, 4)

const setSort = (key: SortKey) => {
  if (sortBy.value === key) return
  sortBy.value = key
  load()
}

const load = async () => {
  const seq = ++reqSeq
  loading.value = true
  try {
    const params: UserBreakdownParams = {
      ...props.filters,
      start_date: props.startDate,
      end_date: props.endDate,
      sort_by: sortBy.value,
      limit: limit.value,
    }
    if (props.model) params.model = props.model
    const res = await getUserBreakdown(params)
    if (seq !== reqSeq) return
    items.value = res.users || []
  } catch {
    if (seq !== reqSeq) return
    items.value = []
  } finally {
    if (seq === reqSeq) loading.value = false
  }
}

// 共享筛选、日期范围或模型变化时重新加载排行。
watch(
  () => [props.startDate, props.endDate, props.model, JSON.stringify(props.filters)],
  () => load(),
  { immediate: true }
)

defineExpose({ reload: load })
</script>
