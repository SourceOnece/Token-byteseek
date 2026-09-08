<template>
  <BaseDialog :show="show" :title="t('admin.accounts.quality.schedule.title')" width="extra-wide" :close-on-escape="!historyPlan && !detailRun" @close="close">
    <div class="space-y-5">
      <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.accounts.quality.schedule.description') }}</p>
      <div class="flex flex-wrap gap-2">
        <button class="btn btn-primary" data-testid="quality-plan-new" @click="newPlan">{{ t('admin.accounts.quality.schedule.create') }}</button>
        <button class="btn btn-secondary" @click="load">{{ t('common.refresh') }}</button>
      </div>
      <p v-if="error" role="alert" class="text-bh-red dark:text-red-400">{{ error }}</p>
      <form v-if="draft" class="quality-panel space-y-4 p-4" @submit.prevent="save">
        <div class="grid gap-4 sm:grid-cols-3">
          <div><label class="input-label" for="quality-plan-name">{{ t('admin.accounts.quality.schedule.name') }}</label><input id="quality-plan-name" v-model="draft.name" class="input w-full" maxlength="100" required /></div>
          <div><label class="input-label" for="quality-plan-interval">{{ t('admin.accounts.quality.schedule.interval') }}</label><input id="quality-plan-interval" v-model.number="draft.interval_minutes" type="number" min="1" max="43200" class="input w-full font-bold text-bh-blue dark:text-blue-300" required /></div>
          <div><label class="input-label" for="quality-plan-keep">{{ t('admin.accounts.quality.schedule.keep') }}</label><input id="quality-plan-keep" v-model.number="draft.keep_runs" type="number" min="1" max="100" class="input w-full" required /></div>
        </div>
        <div class="grid gap-4 sm:grid-cols-3">
          <div><label class="input-label text-bh-blue dark:text-blue-300" for="quality-plan-model">{{ t('admin.accounts.quality.model') }}</label><Select id="quality-plan-model" v-model="draft.config.model" :options="models" creatable searchable /></div>
          <div><label class="input-label text-yellow-700 dark:text-bh-yellow" for="quality-plan-effort">{{ t('admin.accounts.quality.effort') }}</label><Select id="quality-plan-effort" v-model="draft.config.reasoning_effort" :options="efforts" /></div>
          <div><label class="input-label">{{ t('admin.accounts.quality.concurrency') }}</label><Select v-model="draft.config.concurrency" :options="concurrencyOptions" /></div>
        </div>
        <div><label class="input-label" for="quality-plan-timeout">{{ t('admin.accounts.quality.timeout') }}</label><input id="quality-plan-timeout" v-model.number="draft.config.timeout_seconds" type="number" min="10" max="3600" class="input w-full sm:w-48" required /></div>
        <div><label class="input-label" for="quality-plan-prompt">{{ t('admin.accounts.quality.prompt') }}</label><textarea id="quality-plan-prompt" v-model="draft.config.prompt" class="input w-full" rows="3" maxlength="16000" required /></div>
        <div><label class="input-label" for="quality-plan-keyword">{{ t('admin.accounts.quality.keyword') }}</label><input id="quality-plan-keyword" v-model="draft.config.keyword" class="input w-full" maxlength="200" required /></div>
        <div class="border-2 border-gray-400 p-3 dark:border-dark-400">
          <div class="flex flex-wrap items-center gap-2">
            <strong class="text-bh-blue dark:text-blue-300">{{ t('admin.accounts.quality.schedule.selected', { count: draft.config.account_ids.length }) }}</strong>
            <input v-model="search" class="input min-w-0 flex-1" :placeholder="t('admin.accounts.searchAccounts')" @keydown.enter.prevent="loadAccounts(1)" />
            <button type="button" class="btn btn-secondary btn-sm" @click="loadAccounts(1)">{{ t('common.search') }}</button>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="selecting" data-testid="quality-plan-select-all" @click="selectAll">{{ t('admin.accounts.quality.schedule.selectAll') }}</button>
            <button type="button" class="btn btn-secondary btn-sm" @click="draft.config.account_ids = []">{{ t('admin.accounts.bulkActions.clear') }}</button>
          </div>
          <p class="my-2 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.quality.schedule.fixedSelection') }}</p>
          <div class="max-h-56 space-y-1 overflow-auto">
            <label v-for="account in accountRows" :key="account.id" class="flex items-center gap-2 border-b border-gray-200 py-2 dark:border-dark-600">
              <input type="checkbox" :checked="draft.config.account_ids.includes(account.id)" :disabled="!eligible(account)" @change="toggleAccount(account.id)" />
              <span class="min-w-0 break-all text-sm"><strong class="text-bh-blue dark:text-blue-300">{{ account.credentials?.email || account.extra?.email || account.name }}</strong> · #{{ account.id }}</span>
            </label>
          </div>
          <Pagination v-if="accountTotal > 50" :page="accountPage" :page-size="50" :total="accountTotal" :show-page-size-selector="false" @update:page="loadAccounts" />
        </div>
        <label class="flex items-center gap-2"><input v-model="draft.enabled" type="checkbox" /><span>{{ t('admin.accounts.quality.schedule.enabled') }}</span></label>
        <label class="flex items-start gap-2 text-sm font-bold"><input v-model="draft.config.confirm_scheduling" type="checkbox" class="mt-1" data-testid="quality-plan-confirm" /><span>{{ t('admin.accounts.quality.schedule.confirm') }}</span></label>
        <div class="flex gap-2"><button type="submit" class="btn btn-primary" :disabled="saving || selecting || !draft.config.confirm_scheduling || !draft.config.account_ids.length" data-testid="quality-plan-save">{{ t('common.save') }}</button><button type="button" class="btn btn-secondary" @click="draft = null">{{ t('common.cancel') }}</button></div>
      </form>
      <p v-if="!plans.length && !draft" class="text-sm text-gray-500">{{ t('admin.accounts.quality.schedule.empty') }}</p>
      <article v-for="plan in plans" :key="plan.id" class="quality-panel space-y-3 p-4">
        <div class="flex flex-wrap items-center justify-between gap-2"><h4 class="text-lg font-bold">{{ plan.name }}</h4><strong :class="plan.enabled ? 'text-bh-blue dark:text-blue-300' : 'text-gray-500'">{{ t(plan.enabled ? 'admin.accounts.quality.schedule.enabled' : 'admin.accounts.quality.schedule.paused') }}</strong></div>
        <div class="flex flex-wrap gap-x-5 gap-y-2"><strong class="break-all text-bh-blue dark:text-blue-300">{{ plan.config.model }}</strong><strong class="text-yellow-700 dark:text-bh-yellow">{{ t('admin.accounts.quality.effort') }}：{{ plan.config.reasoning_effort || t('admin.accounts.quality.effortDefault') }}</strong></div>
        <p class="text-sm">{{ t('admin.accounts.quality.schedule.intervalValue', { minutes: plan.interval_minutes, count: plan.config.account_ids.length }) }}</p>
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.quality.schedule.next') }}：{{ plan.active_run_id ? t('admin.accounts.quality.schedule.running') : formatTime(plan.next_run_at) }}</p>
        <div class="flex flex-wrap gap-2"><button class="btn btn-primary btn-sm" @click="viewHistory(plan)">{{ t('admin.accounts.quality.schedule.history') }}</button><button class="btn btn-secondary btn-sm" @click="edit(plan)">{{ t('common.edit') }}</button><button v-if="plan.enabled" class="btn btn-secondary btn-sm" @click="pause(plan)">{{ t('admin.accounts.quality.schedule.pause') }}</button></div>
      </article>
    </div>
  </BaseDialog>
  <BaseDialog :show="!!historyPlan && show" :title="`${t('admin.accounts.quality.schedule.history')} · ${historyPlan?.name || ''}`" width="wide" :z-index="65" :close-on-escape="!detailRun" @close="historyPlan = null">
    <button class="btn btn-secondary btn-sm mb-4" @click="refreshHistory">{{ t('common.refresh') }}</button>
    <p v-if="historyError" class="text-bh-red">{{ historyError }}</p>
    <p v-if="!runs.length" class="text-sm text-gray-500">{{ t('admin.accounts.quality.schedule.noRuns') }}</p>
    <article v-for="run in runs" :key="run.id" class="quality-panel mb-4 space-y-3 p-4">
      <p class="font-bold">#{{ run.id }} · {{ formatTime(run.started_at) }} · {{ t(`admin.accounts.quality.schedule.runStatus.${run.status}`) }}</p>
      <p class="flex flex-wrap gap-3"><strong class="break-all text-bh-blue dark:text-blue-300">{{ run.config.model }}</strong><strong class="text-yellow-700 dark:text-bh-yellow">{{ run.config.reasoning_effort || t('admin.accounts.quality.effortDefault') }}</strong></p>
      <CodexQualitySummary :counts="run.counts" @select="status => openRun(run, status)" />
      <CodexQualityProgress :done="Object.values(run.counts).reduce((sum, count) => sum + count, 0)" :total="run.config.account_ids.length" :label="t(`admin.accounts.quality.schedule.runStatus.${run.status}`)" />
    </article>
  </BaseDialog>
  <BaseDialog :show="!!detailRun && show" :title="detailTitle" width="wide" :z-index="80" @close="detailRun = null">
    <p v-if="detailError" class="text-bh-red">{{ detailError }}</p>
    <p v-if="!detailResults.length" class="text-gray-500">{{ t('admin.accounts.quality.emptyCategory') }}</p>
    <div class="space-y-4"><CodexQualityResultCard v-for="result in detailResults" :key="result.account_id" :result="result" /></div>
    <Pagination v-if="detailTotal > 20" :page="detailPage" :page-size="20" :total="detailTotal" :show-page-size-selector="false" @update:page="loadDetail" />
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Pagination from '@/components/common/Pagination.vue'
import CodexQualitySummary from './CodexQualitySummary.vue'
import CodexQualityProgress from './CodexQualityProgress.vue'
import CodexQualityResultCard from './CodexQualityResult.vue'
import { getModelsByPlatform } from '@/composables/useModelWhitelist'
import { adminAPI } from '@/api/admin'
import { qualitySchedulesAPI, type QualitySchedule, type QualityRun, type CodexQualityResult } from '@/api/admin/codexQuality'
import type { Account } from '@/types'

const props = defineProps<{ show: boolean; accountIds: number[] }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const plans = ref<QualitySchedule[]>([]), draft = ref<QualitySchedule | null>(null)
const error = ref(''), saving = ref(false), selecting = ref(false)
const search = ref(''), accountRows = ref<Account[]>([]), accountPage = ref(1), accountTotal = ref(0)
const historyPlan = ref<QualitySchedule | null>(null), runs = ref<QualityRun[]>([]), historyError = ref('')
const detailRun = ref<QualityRun | null>(null), detailStatus = ref(''), detailPage = ref(1), detailTotal = ref(0), detailResults = ref<CodexQualityResult[]>([]), detailError = ref('')
const detailTitle = computed(() => `#${detailRun.value?.id || ''} · ${detailStatus.value ? t(`admin.accounts.quality.status.${detailStatus.value}`) : t('admin.accounts.quality.details')}`)
const models = getModelsByPlatform('openai').filter(id => !id.includes('image')).map(value => ({ value, label: value }))
const efforts = computed(() => [{ value: '', label: t('admin.accounts.quality.effortDefault') }, ...['none', 'minimal', 'low', 'medium', 'high', 'xhigh', 'max'].map(value => ({ value, label: value }))])
const concurrencyOptions = [1, 2, 3, 4, 5].map(value => ({ value, label: String(value) }))
let generation = 0, accountRequest = 0, detailRequest = 0, timer: ReturnType<typeof setInterval> | null = null
const message = (e: unknown) => e instanceof Error ? e.message : (e as { message?: string })?.message || t('common.error')
const formatTime = (value?: string) => value ? new Date(value).toLocaleString() : '—'
const eligible = (account: Account) => account.platform === 'openai' && account.type === 'oauth' && !account.parent_account_id && account.credentials?.auth_mode !== 'agentIdentity'

async function load() { const v = generation; try { const value = await qualitySchedulesAPI.list(); if (v === generation) plans.value = value } catch (e) { if (v === generation) error.value = message(e) } }
function newPlan() {
  draft.value = { id: 0, name: '', interval_minutes: 60, keep_runs: 30, enabled: true, config: { account_ids: [...props.accountIds], model: 'gpt-6-astra', reasoning_effort: '', prompt: '', keyword: '', concurrency: 3, timeout_seconds: 120, confirm_scheduling: false } }
  error.value = ''; search.value = ''; void loadAccounts(1)
}
function edit(plan: QualitySchedule) { draft.value = JSON.parse(JSON.stringify(plan)); draft.value!.config.confirm_scheduling = false; draft.value!.config.timeout_seconds ||= 120; error.value = ''; search.value = ''; void loadAccounts(1) }
async function loadAccounts(page = 1) {
  const v = ++accountRequest
  try { const value = await adminAPI.accounts.list(page, 50, { platform: 'openai', type: 'oauth', search: search.value }); if (v !== accountRequest) return; accountRows.value = value.items; accountTotal.value = value.total; accountPage.value = page } catch (e) { if (v === accountRequest) error.value = message(e) }
}
function toggleAccount(id: number) { if (!draft.value) return; const ids = draft.value.config.account_ids; draft.value.config.account_ids = ids.includes(id) ? ids.filter(value => value !== id) : [...ids, id] }
async function selectAll() {
  if (!draft.value || selecting.value) return
  selecting.value = true; const target = draft.value, query = search.value; const v = generation
  try {
    const ids: number[] = []
    for (let page = 1; ; page++) {
      const value = await adminAPI.accounts.list(page, 100, { platform: 'openai', type: 'oauth', search: query })
      if (v !== generation || draft.value !== target) return
      ids.push(...value.items.filter(eligible).map(account => account.id))
      if (ids.length > 500) throw new Error(t('admin.accounts.quality.tooMany'))
      if (!value.items.length || page * 100 >= value.total) break
    }
    target.config.account_ids = [...new Set(ids)]
  } catch (e) { error.value = message(e) } finally { selecting.value = false }
}
async function save() {
  if (!draft.value || saving.value || selecting.value || !draft.value.config.confirm_scheduling) return
  if (draft.value.config.account_ids.length > 500) { error.value = t('admin.accounts.quality.tooMany'); return }
  saving.value = true; const v = generation
  try { await qualitySchedulesAPI.save(JSON.parse(JSON.stringify(draft.value))); if (v !== generation) return; draft.value = null; await load() } catch (e) { if (v === generation) error.value = message(e) } finally { saving.value = false }
}
async function pause(plan: QualitySchedule) { try { await qualitySchedulesAPI.pause(plan.id); await load(); await refreshHistory() } catch (e) { error.value = message(e) } }
async function viewHistory(plan: QualitySchedule) { historyPlan.value = plan; runs.value = []; historyError.value = ''; await refreshHistory() }
async function refreshHistory() { const id = historyPlan.value?.id; if (!id) return; try { const value = await qualitySchedulesAPI.runs(id); if (historyPlan.value?.id === id) runs.value = value } catch (e) { historyError.value = message(e) } }
function openRun(run: QualityRun, status: string) { detailRun.value = run; detailStatus.value = status; detailResults.value = []; void loadDetail(1) }
async function loadDetail(page = 1) { const id = detailRun.value?.id, v = ++detailRequest; if (!id) return; detailError.value = ''; try { const value = await qualitySchedulesAPI.detail(id, detailStatus.value, page); if (v !== detailRequest) return; detailResults.value = value.items; detailTotal.value = value.total; detailPage.value = page } catch (e) { if (v === detailRequest) detailError.value = message(e) } }
function close() { emit('close') }
watch(() => props.show, show => {
  generation++; accountRequest++; detailRequest++
  if (timer) clearInterval(timer); timer = null
  if (show) { draft.value = null; error.value = ''; historyPlan.value = null; detailRun.value = null; void load(); timer = setInterval(() => { void load(); void refreshHistory() }, 15000) }
})
onBeforeUnmount(() => { generation++; accountRequest++; detailRequest++; if (timer) clearInterval(timer) })
</script>
<style scoped>
.quality-panel { border: 2px solid var(--bh-ink); background: var(--bh-surface); box-shadow: var(--bh-shadow-sm); }
</style>
