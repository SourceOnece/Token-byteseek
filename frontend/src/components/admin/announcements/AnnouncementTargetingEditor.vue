<template>
  <div class="rounded-surface border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/50">
    <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <div class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.announcements.form.targetingMode') }}
        </div>
        <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">
          {{ mode === 'all' ? t('admin.announcements.form.targetingAll') : t('admin.announcements.form.targetingCustom') }}
        </div>
      </div>

      <div class="flex items-center gap-3">
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input
            type="radio"
            name="announcement-targeting-mode"
            value="all"
            :checked="mode === 'all'"
            @change="setMode('all')"
            class="h-4 w-4"
          />
          {{ t('admin.announcements.form.targetingAll') }}
        </label>
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input
            type="radio"
            name="announcement-targeting-mode"
            value="custom"
            :checked="mode === 'custom'"
            @change="setMode('custom')"
            class="h-4 w-4"
          />
          {{ t('admin.announcements.form.targetingCustom') }}
        </label>
      </div>
    </div>

    <RuleListEditor
      v-if="mode === 'custom'"
      class="mt-4"
      :items="anyOf"
      :item-key="targetingRowKey"
      :title="`OR (${anyOf.length}/50)`"
      :add-label="t('admin.announcements.form.addOrGroup')"
      :empty-text="`${t('admin.announcements.form.targetingCustom')}: ${t('admin.announcements.form.addOrGroup')}`"
      :max="50"
      :error="validationError"
      variant="card"
      :item-label="(index) => `${t('admin.announcements.form.targetingCustom')} #${index + 1}`"
      test-id="announcement-groups"
      @add="addOrGroup"
      @remove="removeOrGroup"
    >
      <template #row="{ item: group, index: groupIndex }">
        <RuleListEditor
          :items="group.all_of || []"
          :item-key="targetingRowKey"
          :title="`AND (${group.all_of?.length || 0}/50)`"
          :add-label="t('admin.announcements.form.addAndCondition')"
          :max="50"
          variant="card"
          :item-label="(index) => t('common.ruleIndex', { index: index + 1 })"
          :test-id="`announcement-conditions-${groupIndex}`"
          @add="addAndCondition(groupIndex)"
          @remove="removeAndCondition(groupIndex, $event)"
        >
          <template #row="{ item: cond, index: condIndex }">
            <div class="flex flex-col gap-3 md:flex-row md:items-end">
              <div class="w-full md:w-52">
                <label class="input-label">{{ t('admin.announcements.form.conditionType') }}</label>
                <Select
                  :model-value="cond.type"
                  :options="conditionTypeOptions"
                  @update:model-value="(v) => setConditionType(groupIndex, condIndex, v as any)"
                />
              </div>
              <div v-if="cond.type === 'subscription'" class="min-w-0 flex-1">
                <label class="input-label">{{ t('admin.announcements.form.selectPackages') }}</label>
                <div class="grid max-h-40 grid-cols-1 gap-2 overflow-y-auto rounded-surface border border-gray-200 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-800 sm:grid-cols-2">
                  <label
                    v-for="plan in plans"
                    :key="plan.id"
                    class="flex cursor-pointer items-start gap-2 rounded-control px-2 py-1.5 transition-colors hover:bg-white dark:hover:bg-dark-700"
                  >
                    <input
                      type="checkbox"
                      :checked="subscriptionSelections[groupIndex]?.[condIndex]?.includes(plan.id)"
                      class="mt-0.5 h-4 w-4 rounded-compact border-gray-300 text-primary-500 focus:ring-primary-500 dark:border-dark-500"
                      @change="togglePlanSelection(groupIndex, condIndex, plan.id, ($event.target as HTMLInputElement).checked)"
                    />
                    <div class="min-w-0">
                      <div class="text-sm font-medium text-gray-900 dark:text-white">
                        {{ plan.name }}
                      </div>
                      <div class="text-xs text-gray-500 dark:text-dark-400">
                        {{ plan.validity_days }} {{ t('payment.days') }}
                      </div>
                    </div>
                  </label>
                  <div
                    v-if="plans.length === 0"
                    class="sm:col-span-2 py-2 text-center text-sm text-gray-500 dark:text-dark-400"
                  >
                    {{ t('admin.announcements.form.selectPackages') }}
                  </div>
                </div>
              </div>
              <div v-else class="flex min-w-0 flex-1 flex-col gap-3 sm:flex-row">
                <div class="w-full sm:w-44">
                  <label class="input-label">{{ t('admin.announcements.form.operator') }}</label>
                  <Select
                    :model-value="cond.operator"
                    :options="balanceOperatorOptions"
                    @update:model-value="(v) => setOperator(groupIndex, condIndex, v as any)"
                  />
                </div>
                <div class="w-full sm:flex-1">
                  <label class="input-label">{{ t('admin.announcements.form.balanceValue') }}</label>
                  <input
                    :value="String(cond.value ?? '')"
                    type="number"
                    step="any"
                    class="input"
                    @input="(e) => setBalanceValue(groupIndex, condIndex, (e.target as HTMLInputElement).value)"
                  />
                </div>
              </div>
            </div>
          </template>
        </RuleListEditor>
      </template>
    </RuleListEditor>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, toRaw, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type {
  AnnouncementTargeting,
  AnnouncementCondition,
  AnnouncementConditionGroup,
  AnnouncementConditionType,
  AnnouncementOperator,
  SubscriptionPlan
} from '@/types'

import Select from '@/components/common/Select.vue'
import RuleListEditor from '@/components/common/RuleListEditor.vue'
import { createStableObjectKeyResolver } from '@/utils/stableObjectKey'

const { t } = useI18n()

const props = defineProps<{
  modelValue: AnnouncementTargeting
  plans: SubscriptionPlan[]
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: AnnouncementTargeting): void
}>()

const anyOf = computed(() => props.modelValue?.any_of ?? [])

type Mode = 'all' | 'custom'
const mode = computed<Mode>(() => (anyOf.value.length === 0 ? 'all' : 'custom'))

const conditionTypeOptions = computed(() => [
  { value: 'subscription', label: t('admin.announcements.form.conditionSubscription') },
  { value: 'balance', label: t('admin.announcements.form.conditionBalance') }
])

const balanceOperatorOptions = computed(() => [
  { value: 'gt', label: t('admin.announcements.operators.gt') },
  { value: 'gte', label: t('admin.announcements.operators.gte') },
  { value: 'lt', label: t('admin.announcements.operators.lt') },
  { value: 'lte', label: t('admin.announcements.operators.lte') },
  { value: 'eq', label: t('admin.announcements.operators.eq') }
])

function setMode(next: Mode) {
  if (next === 'all') {
    emit('update:modelValue', { any_of: [] })
    return
  }
  if (anyOf.value.length === 0) {
    emit('update:modelValue', { any_of: [{ all_of: [defaultSubscriptionCondition()] }] })
  }
}

function defaultSubscriptionCondition(): AnnouncementCondition {
  return {
    type: 'subscription' as AnnouncementConditionType,
    operator: 'in' as AnnouncementOperator,
    plan_ids: []
  }
}

function defaultBalanceCondition(): AnnouncementCondition {
  return {
    type: 'balance' as AnnouncementConditionType,
    operator: 'gte' as AnnouncementOperator,
    value: 0
  }
}

type TargetingDraft = {
  any_of: AnnouncementConditionGroup[]
}

// 展示 key 保存在 WeakMap 中，不写入公告的提交数据。
const inheritedRowKeys = new WeakMap<object, string>()
const newRowKey = createStableObjectKeyResolver<object>('announcement-targeting')

function targetingRowKey(row: object): string {
  const raw = toRaw(row)
  return inheritedRowKeys.get(raw) ?? newRowKey(raw)
}

// 克隆后的组和条件继承原行身份，编辑字段时保留输入焦点。
function cloneTargeting(): TargetingDraft {
  const current = props.modelValue ?? { any_of: [] }
  const draft: TargetingDraft = JSON.parse(JSON.stringify(current))
  draft.any_of ??= []
  draft.any_of.forEach((group, groupIndex) => {
    const previous = current.any_of?.[groupIndex]
    if (!previous) return
    inheritedRowKeys.set(group, targetingRowKey(previous))
    group.all_of?.forEach((condition, condIndex) => {
      const previousCondition = previous.all_of?.[condIndex]
      if (previousCondition) inheritedRowKeys.set(condition, targetingRowKey(previousCondition))
    })
  })
  return draft
}

function updateTargeting(mutator: (draft: TargetingDraft) => void) {
  const draft = cloneTargeting()
  mutator(draft)
  emit('update:modelValue', draft)
}

function addOrGroup() {
  updateTargeting((draft) => {
    if (draft.any_of.length >= 50) return
    draft.any_of.push({ all_of: [defaultSubscriptionCondition()] })
  })
}

function removeOrGroup(groupIndex: number) {
  updateTargeting((draft) => {
    draft.any_of.splice(groupIndex, 1)
  })
}

function addAndCondition(groupIndex: number) {
  updateTargeting((draft) => {
    const group = draft.any_of[groupIndex]
    if (!group.all_of) group.all_of = []
    if (group.all_of.length >= 50) return
    group.all_of.push(defaultSubscriptionCondition())
  })
}

function removeAndCondition(groupIndex: number, condIndex: number) {
  updateTargeting((draft) => {
    const group = draft.any_of[groupIndex]
    if (!group?.all_of) return
    group.all_of.splice(condIndex, 1)
  })
}

function setConditionType(groupIndex: number, condIndex: number, nextType: AnnouncementConditionType) {
  updateTargeting((draft) => {
    const group = draft.any_of[groupIndex]
    if (!group?.all_of) return

    if (nextType === 'subscription') {
      group.all_of[condIndex] = defaultSubscriptionCondition()
    } else {
      group.all_of[condIndex] = defaultBalanceCondition()
    }
  })
}

function setOperator(groupIndex: number, condIndex: number, op: AnnouncementOperator) {
  updateTargeting((draft) => {
    const group = draft.any_of[groupIndex]
    if (!group?.all_of) return

    const cond = group.all_of[condIndex]
    if (!cond) return

    cond.operator = op
  })
}

function setBalanceValue(groupIndex: number, condIndex: number, raw: string) {
  const n = raw === '' ? 0 : Number(raw)
  updateTargeting((draft) => {
    const group = draft.any_of[groupIndex]
    if (!group?.all_of) return

    const cond = group.all_of[condIndex]
    if (!cond) return

    cond.value = Number.isFinite(n) ? n : 0
  })
}

// We keep plan_ids selection in a parallel reactive map and mirror it back to targeting.plan_ids.
const subscriptionSelections = reactive<Record<number, Record<number, number[]>>>({})

function ensureSelectionPath(groupIndex: number, condIndex: number) {
  if (!subscriptionSelections[groupIndex]) subscriptionSelections[groupIndex] = {}
  if (!subscriptionSelections[groupIndex][condIndex]) subscriptionSelections[groupIndex][condIndex] = []
}

function togglePlanSelection(groupIndex: number, condIndex: number, planID: number, checked: boolean) {
  ensureSelectionPath(groupIndex, condIndex)
  const current = subscriptionSelections[groupIndex][condIndex] ?? []
  subscriptionSelections[groupIndex][condIndex] = checked
    ? [...current, planID]
    : current.filter((id) => id !== planID)
}

// Sync from modelValue to subscriptionSelections (one-way: model -> local state)
watch(
  () => props.modelValue,
  (v) => {
    const groups = v?.any_of ?? []
    for (let gi = 0; gi < groups.length; gi++) {
      const allOf = groups[gi]?.all_of ?? []
      for (let ci = 0; ci < allOf.length; ci++) {
        const c = allOf[ci]
        if (c?.type === 'subscription') {
          ensureSelectionPath(gi, ci)
          const newIds = (c.plan_ids ?? []).slice()
          const currentIds = subscriptionSelections[gi]?.[ci] ?? []
          if (JSON.stringify(newIds.sort()) !== JSON.stringify(currentIds.sort())) {
            subscriptionSelections[gi][ci] = newIds
          }
        }
      }
    }
  },
  { immediate: true }
)

// Sync from subscriptionSelections to modelValue (one-way: local state -> model)
// Use a debounced approach to avoid infinite loops
let syncTimeout: ReturnType<typeof setTimeout> | null = null
watch(
  () => subscriptionSelections,
  () => {
    // Debounce the sync to avoid rapid fire updates
    if (syncTimeout) clearTimeout(syncTimeout)

    syncTimeout = setTimeout(() => {
      // Build the new targeting state
      const newTargeting = cloneTargeting()

      const groups = newTargeting.any_of ?? []
      for (let gi = 0; gi < groups.length; gi++) {
        const allOf = groups[gi]?.all_of ?? []
        for (let ci = 0; ci < allOf.length; ci++) {
          const c = allOf[ci]
          if (c?.type === 'subscription') {
            ensureSelectionPath(gi, ci)
            c.operator = 'in' as AnnouncementOperator
            c.plan_ids = (subscriptionSelections[gi]?.[ci] ?? []).slice()
          }
        }
      }

      // Only emit if there's an actual change (deep comparison)
      if (JSON.stringify(props.modelValue) !== JSON.stringify(newTargeting)) {
        emit('update:modelValue', newTargeting)
      }
    }, 0)
  },
  { deep: true }
)

const validationError = computed(() => {
  if (mode.value !== 'custom') return ''

  const groups = anyOf.value
  if (groups.length === 0) return t('admin.announcements.form.addOrGroup')

  if (groups.length > 50) return 'any_of > 50'

  for (const g of groups) {
    const allOf = g?.all_of ?? []
    if (allOf.length === 0) return t('admin.announcements.form.addAndCondition')
    if (allOf.length > 50) return 'all_of > 50'

    for (const c of allOf) {
      if (c.type === 'subscription') {
        if (!c.plan_ids || c.plan_ids.length === 0) return t('admin.announcements.form.selectPackages')
      }
    }
  }

  return ''
})
</script>
