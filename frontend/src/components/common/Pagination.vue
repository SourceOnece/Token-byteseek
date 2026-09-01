<template>
  <div
    class="pagination-root flex items-center justify-between border-t-2 border-gray-950 bg-white px-4 py-3 dark:border-dark-200/60 dark:bg-dark-800 lg:px-6 sm:px-6"
  >
    <div class="pagination-mobile flex flex-1 items-center justify-between lg:hidden">
      <!-- Mobile pagination -->
      <button
        @click="goToPage(page - 1)"
        :disabled="page === 1"
        class="pagination-control bh-page-btn px-4"
      >
        {{ t('pagination.previous') }}
      </button>
      <span class="font-mono text-sm font-bold text-gray-800 dark:text-dark-100">
        {{ t('pagination.pageOf', { page, total: totalPages }) }}
      </span>
      <button
        @click="goToPage(page + 1)"
        :disabled="page === totalPages"
        class="pagination-control bh-page-btn ml-3 px-4"
      >
        {{ t('pagination.next') }}
      </button>
    </div>

    <div class="pagination-desktop hidden lg:flex lg:flex-1 lg:items-center lg:justify-between">
      <!-- Desktop pagination info -->
      <div class="flex items-center space-x-4">
        <p class="pagination-summary text-sm font-semibold text-gray-700 dark:text-dark-200">
          {{ t('pagination.showing') }}
          <span class="font-mono font-bold text-gray-950 dark:text-white">{{ fromItem }}</span>
          {{ t('pagination.to') }}
          <span class="font-mono font-bold text-gray-950 dark:text-white">{{ toItem }}</span>
          {{ t('pagination.of') }}
          <span class="bh-total-chip">{{ total }}</span>
          {{ t('pagination.results') }}
        </p>

        <!-- Page size selector -->
        <div v-if="showPageSizeSelector" class="flex items-center space-x-2">
          <span class="text-sm font-semibold text-gray-700 dark:text-dark-200"
            >{{ t('pagination.perPage') }}:</span
          >
          <div class="page-size-select w-20">
            <Select
              :model-value="pageSize"
              :options="pageSizeSelectOptions"
              @update:model-value="handlePageSizeChange"
            />
          </div>
        </div>

        <div v-if="showJump" class="flex items-center space-x-2">
          <span class="text-sm font-semibold text-gray-700 dark:text-dark-200">{{ t('pagination.jumpTo') }}</span>
          <input
            v-model="jumpPage"
            type="number"
            min="1"
            :max="totalPages"
            class="pagination-jump-input input w-20 text-sm"
            :placeholder="t('pagination.jumpPlaceholder')"
            @keyup.enter="submitJump"
          />
          <button type="button" class="pagination-jump-button btn btn-ghost btn-sm" @click="submitJump">
            {{ t('pagination.jumpAction') }}
          </button>
        </div>
      </div>

      <!-- Desktop pagination buttons：方块页码组 -->
      <nav
        class="pagination-nav relative z-0 inline-flex"
        aria-label="Pagination"
      >
        <!-- Previous button -->
        <button
          @click="goToPage(page - 1)"
          :disabled="page === 1"
          class="pagination-control bh-page-btn px-2"
          :aria-label="t('pagination.previous')"
        >
          <Icon name="chevronLeft" size="md" :stroke-width="2.5" />
        </button>

        <!-- Page numbers -->
        <button
          v-for="(pageNum, index) in visiblePages"
          :key="`${pageNum}-${index}`"
          @click="typeof pageNum === 'number' && goToPage(pageNum)"
          :disabled="typeof pageNum !== 'number'"
          :class="[
            'pagination-control pagination-page-button bh-page-btn px-4 font-mono',
            pageNum === page && 'bh-page-btn-active',
            typeof pageNum !== 'number' && 'cursor-default'
          ]"
          :aria-label="
            typeof pageNum === 'number' ? t('pagination.goToPage', { page: pageNum }) : undefined
          "
          :aria-current="pageNum === page ? 'page' : undefined"
        >
          {{ pageNum }}
        </button>

        <!-- Next button -->
        <button
          @click="goToPage(page + 1)"
          :disabled="page === totalPages"
          class="pagination-control bh-page-btn px-2"
          :aria-label="t('pagination.next')"
        >
          <Icon name="chevronRight" size="md" :stroke-width="2.5" />
        </button>
      </nav>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import Select from './Select.vue'
import { getConfiguredTablePageSizeOptions, normalizeTablePageSize } from '@/utils/tablePreferences'
import { setPersistedPageSize } from '@/composables/usePersistedPageSize'

const { t } = useI18n()

interface Props {
  total: number
  page: number
  pageSize: number
  pageSizeOptions?: number[]
  showPageSizeSelector?: boolean
  showJump?: boolean
}

interface Emits {
  (e: 'update:page', page: number): void
  (e: 'update:pageSize', pageSize: number): void
}

const props = withDefaults(defineProps<Props>(), {
  pageSizeOptions: () => getConfiguredTablePageSizeOptions(),
  showPageSizeSelector: true,
  showJump: false
})

const emit = defineEmits<Emits>()

const totalPages = computed(() => Math.ceil(props.total / props.pageSize))

const fromItem = computed(() => {
  if (props.total === 0) return 0
  return (props.page - 1) * props.pageSize + 1
})

const toItem = computed(() => {
  const to = props.page * props.pageSize
  return to > props.total ? props.total : to
})

const pageSizeSelectOptions = computed(() => {
  const options = Array.from(
    new Set([
      ...getConfiguredTablePageSizeOptions(),
      normalizeTablePageSize(props.pageSize)
    ])
  ).sort((a, b) => a - b)

  return options.map((size) => ({
    value: size,
    label: String(size)
  }))
})

const jumpPage = ref('')

const visiblePages = computed(() => {
  const pages: (number | string)[] = []
  const maxVisible = 7
  const total = totalPages.value

  if (total <= maxVisible) {
    // Show all pages if total is small
    for (let i = 1; i <= total; i++) {
      pages.push(i)
    }
  } else {
    // Always show first page
    pages.push(1)

    const start = Math.max(2, props.page - 2)
    const end = Math.min(total - 1, props.page + 2)

    // Add ellipsis before if needed
    if (start > 2) {
      pages.push('...')
    }

    // Add middle pages
    for (let i = start; i <= end; i++) {
      pages.push(i)
    }

    // Add ellipsis after if needed
    if (end < total - 1) {
      pages.push('...')
    }

    // Always show last page
    pages.push(total)
  }

  return pages
})

const goToPage = (newPage: number) => {
  if (newPage >= 1 && newPage <= totalPages.value && newPage !== props.page) {
    emit('update:page', newPage)
  }
}

const handlePageSizeChange = (value: string | number | boolean | null) => {
  if (value === null || typeof value === 'boolean') return
  const newPageSize = normalizeTablePageSize(typeof value === 'string' ? parseInt(value, 10) : value)
  setPersistedPageSize(newPageSize)
  emit('update:pageSize', newPageSize)
}

const submitJump = () => {
  const value = jumpPage.value.trim()
  if (!value) return
  const pageNum = Number.parseInt(value, 10)
  if (Number.isNaN(pageNum)) return
  const nextPage = Math.min(Math.max(pageNum, 1), totalPages.value)
  jumpPage.value = ''
  goToPage(nextPage)
}
</script>

<style scoped>
.pagination-control,
.pagination-jump-input,
.pagination-jump-button {
  height: var(--pagination-control-height, 2.25rem);
  min-height: var(--pagination-control-height, 2.25rem);
}

.page-size-select :deep(.select-trigger) {
  height: 2.25rem;
  height: var(--pagination-control-height, 2.25rem);
  min-height: 0;
  @apply px-3 py-1.5 text-sm;
}

/* 方块页码：相邻共享 2px 边框，当前页黄底红杠 */
.bh-page-btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  padding-top: 0.5rem;
  padding-bottom: 0.5rem;
  margin-left: -2px;
  border: 2px solid var(--bh-ink);
  background: var(--bh-surface);
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--bh-ink);
  transition: background 0.12s ease;
}

.bh-page-btn:first-child {
  margin-left: 0;
}

.bh-page-btn:hover:not(:disabled):not(.bh-page-btn-active) {
  background: rgba(255, 204, 0, 0.35);
  z-index: 1;
}

.bh-page-btn:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}

.bh-page-btn-active {
  z-index: 2;
  background: var(--bh-yellow) !important;
  color: #141414 !important;
  box-shadow: inset 0 -4px 0 0 var(--bh-red);
  font-weight: 800;
}

.dark .bh-page-btn-active {
  border-color: rgba(244, 240, 230, 0.85);
}

/* 总数芯片 */
.bh-total-chip {
  display: inline-block;
  padding: 0 6px;
  border: 2px solid var(--bh-ink);
  background: var(--bh-yellow);
  color: #141414;
  font-family: 'Geist Mono Variable', ui-monospace, monospace;
  font-weight: 800;
}
</style>
