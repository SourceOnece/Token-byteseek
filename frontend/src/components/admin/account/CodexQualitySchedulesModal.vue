<template>
  <BaseDialog :show="show" :title="t('admin.accounts.quality.schedule.title')" width="extra-wide" :close-on-escape="!historyPlan && !detailRun" @close="close">
    <div class="space-y-5">
      <div v-if="!draft" class="flex flex-wrap items-center justify-between gap-3">
        <button class="btn btn-primary gap-2" data-testid="quality-plan-new" @click="newPlan"><Icon name="plus" size="sm" />{{ t('admin.accounts.quality.schedule.create') }}</button>
        <button class="btn btn-secondary quality-icon" :title="t('common.refresh')" :aria-label="t('common.refresh')" @click="load"><Icon name="refresh" size="sm" /></button>
      </div>
      <p v-if="error" role="alert" class="text-bh-red dark:text-red-400">{{ error }}</p>
      <form v-if="draft" class="quality-form space-y-5" @submit.prevent="save">
        <fieldset :disabled="saving || selecting" class="min-w-0 space-y-5">
        <div class="grid items-end gap-4 sm:grid-cols-3">
          <div><label class="input-label" for="quality-plan-name">{{ t('admin.accounts.quality.schedule.name') }}</label><input id="quality-plan-name" v-model="draft.name" class="input w-full" maxlength="100" required /></div>
          <div><label class="input-label" for="quality-plan-interval">{{ t('admin.accounts.quality.schedule.interval') }}</label><input id="quality-plan-interval" v-model.number="draft.interval_minutes" type="number" min="1" max="43200" class="input w-full font-bold text-bh-blue dark:text-blue-300" required /></div>
          <div><label class="input-label" for="quality-plan-keep">{{ t('admin.accounts.quality.schedule.keep') }}</label><input id="quality-plan-keep" v-model.number="draft.keep_runs" type="number" min="1" max="100" class="input w-full" required /></div>
        </div>
        <div class="grid items-end gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div><label class="input-label text-bh-blue dark:text-blue-300" for="quality-plan-model">{{ t('admin.accounts.quality.model') }}</label><Select id="quality-plan-model" v-model="draft.config.model" :options="models" creatable searchable /></div>
          <div><label class="input-label text-yellow-700 dark:text-bh-yellow" for="quality-plan-effort">{{ t('admin.accounts.quality.effort') }}</label><Select id="quality-plan-effort" v-model="draft.config.reasoning_effort" :options="efforts" /></div>
          <div><label class="input-label" for="quality-plan-concurrency">{{ t('admin.accounts.quality.concurrency') }}</label><Select id="quality-plan-concurrency" v-model="draft.config.concurrency" :options="concurrencyOptions" /></div>
          <div><label class="input-label" for="quality-plan-timeout">{{ t('admin.accounts.quality.timeout') }}</label><input id="quality-plan-timeout" v-model.number="draft.config.timeout_seconds" type="number" min="10" max="3600" class="input w-full" required /></div>
        </div>
        <div><label class="input-label" for="quality-plan-prompt">{{ t('admin.accounts.quality.prompt') }}</label><textarea id="quality-plan-prompt" v-model="draft.config.prompt" class="input w-full" rows="3" maxlength="16000" required /></div>
        <div><label class="input-label" for="quality-plan-keyword">{{ t('admin.accounts.quality.keyword') }}</label><input id="quality-plan-keyword" v-model="draft.config.keyword" class="input w-full" maxlength="200" required /></div>
        <div class="sm:max-w-sm"><label class="input-label text-bh-blue dark:text-blue-300" for="quality-plan-protocol">{{ t('admin.accounts.quality.protocol') }}</label><Select id="quality-plan-protocol" v-model="draft.config.api_protocol" :options="protocols" /></div>
        <div class="border-y-2 border-bh-ink py-4">
          <div class="flex flex-wrap items-center gap-2">
            <strong class="text-bh-blue dark:text-blue-300">{{ t('admin.accounts.quality.schedule.selected', { count: draft.config.account_ids.length }) }}</strong>
            <input v-model="search" class="input min-w-0 flex-1 basis-40" :aria-label="t('admin.accounts.searchAccounts')" :placeholder="t('admin.accounts.searchAccounts')" @keydown.enter.prevent="loadAccounts(1)" />
            <button type="button" class="btn btn-secondary quality-icon" :title="t('common.search')" :aria-label="t('common.search')" @click="loadAccounts(1)"><Icon name="search" size="sm" /></button>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="selecting" data-testid="quality-plan-select-all" @click="selectAll">{{ t('admin.accounts.quality.schedule.selectAll') }}</button>
            <button type="button" class="btn btn-secondary btn-sm" @click="draft.config.account_ids = []">{{ t('admin.accounts.bulkActions.clear') }}</button>
          </div>
          <p class="my-2 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.quality.schedule.fixedSelection') }}</p>
          <div class="max-h-56 space-y-1 overflow-auto">
            <label v-for="account in accountRows" :key="account.id" class="flex items-center gap-2 border-b border-gray-200 py-2 dark:border-dark-600">
              <input type="checkbox" :checked="draft.config.account_ids.includes(account.id)" :disabled="!eligible(account)" @change="toggleAccount(account.id)" />
              <span class="min-w-0 flex-1 break-all text-sm"><strong class="text-bh-blue dark:text-blue-300">{{ account.credentials?.email || account.extra?.email || account.name }}</strong> · #{{ account.id }}</span>
              <span class="shrink-0 text-xs font-bold text-gray-500 dark:text-gray-400">{{ account.type === 'apikey' ? 'API Key' : 'OAuth' }}</span>
            </label>
          </div>
          <Pagination v-if="accountTotal > 50" :page="accountPage" :page-size="50" :total="accountTotal" :show-page-size-selector="false" @update:page="loadAccounts" />
        </div>
        <div class="flex items-center justify-between gap-3 py-2"><span class="text-sm font-bold">{{ t('admin.accounts.quality.schedule.periodic') }}</span><Toggle v-model="draft.enabled" class="quality-enabled" :aria-label="t('admin.accounts.quality.schedule.periodic')" /></div>
        <label class="flex items-start gap-2 text-sm font-bold"><input v-model="draft.config.confirm_scheduling" type="checkbox" class="mt-1" data-testid="quality-plan-confirm" /><span>{{ t('admin.accounts.quality.schedule.confirm') }}</span></label>
        <div class="flex gap-2"><button type="submit" class="btn btn-primary" :disabled="saving || selecting || !draft.config.confirm_scheduling || !draft.config.account_ids.length" data-testid="quality-plan-save">{{ t('common.save') }}</button><button type="button" class="btn btn-secondary" @click="draft = null">{{ t('common.cancel') }}</button></div>
        </fieldset>
      </form>
      <p v-if="!plans.length && !draft" class="text-sm text-gray-500">{{ t('admin.accounts.quality.schedule.empty') }}</p>
      <div v-if="!draft" class="space-y-5">
      <article v-for="plan in plans" :key="plan.id" class="quality-panel space-y-4 p-4 sm:p-5" :data-testid="`quality-plan-${plan.id}`">
        <div class="flex flex-wrap items-center justify-between gap-4 border-b-2 border-bh-ink pb-4">
          <h4 class="min-w-0 flex-1 break-words text-lg font-bold">{{ plan.name }}</h4>
          <div class="flex shrink-0 items-center gap-3"><span class="text-xs font-bold" :class="plan.enabled ? 'text-green-700 dark:text-green-400' : 'text-gray-500'">{{ t(plan.enabled ? 'admin.accounts.quality.schedule.enabled' : 'admin.accounts.quality.schedule.paused') }}</span><Toggle :model-value="plan.enabled" class="quality-enabled" :disabled="busyPlan !== null" :aria-label="`${t('admin.accounts.quality.schedule.periodic')} · ${plan.name}`" @update:model-value="toggleEnabled(plan)" /></div>
        </div>
        <div class="grid min-w-0 gap-4 sm:grid-cols-2">
          <div class="min-w-0 border-l-4 border-bh-blue pl-3"><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.quality.model') }}</p><strong class="break-all text-bh-blue dark:text-blue-300">{{ plan.config.model }}</strong></div>
          <div class="min-w-0 border-l-4 border-bh-yellow pl-3"><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.quality.effort') }}</p><strong class="text-yellow-700 dark:text-bh-yellow">{{ plan.config.reasoning_effort || t('admin.accounts.quality.effortDefault') }}</strong></div>
        </div>
        <p class="text-sm">{{ t('admin.accounts.quality.schedule.intervalValue', { minutes: plan.interval_minutes, count: plan.config.account_ids.length }) }}</p>
        <p class="text-sm font-bold text-bh-blue dark:text-blue-300">{{ t('admin.accounts.quality.protocol') }}：{{ qualityProtocolLabel(plan.config.api_protocol) || t('admin.accounts.quality.protocolDefault') }}</p>
        <p class="break-words text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.quality.schedule.next') }}：{{ plan.active_run_id ? t('admin.accounts.quality.schedule.running') : plan.manual_requested_at ? t('admin.accounts.quality.schedule.queued') : plan.enabled ? formatTime(plan.next_run_at) : t('admin.accounts.quality.schedule.paused') }}</p>
        <div class="flex flex-wrap items-center gap-3 border-t-2 border-bh-ink pt-4">
          <button class="btn btn-primary btn-sm gap-2" data-testid="quality-plan-run" :disabled="busyPlan !== null || !!plan.active_run_id || !!plan.manual_requested_at" @click="trigger(plan)"><Icon name="play" size="sm" />{{ t('admin.accounts.quality.schedule.runNow') }}</button>
          <button class="btn btn-secondary btn-sm gap-2" @click="viewHistory(plan)"><Icon name="clock" size="sm" />{{ t('admin.accounts.quality.schedule.history') }}</button>
          <button class="btn btn-secondary quality-icon sm:ml-auto" :title="t('common.edit')" :aria-label="t('common.edit')" :disabled="busyPlan !== null" @click="edit(plan)"><Icon name="edit" size="sm" /></button>
          <button v-if="plan.active_run_id || plan.manual_requested_at" class="btn btn-danger quality-icon" :title="t('admin.accounts.quality.schedule.stop')" :aria-label="t('admin.accounts.quality.schedule.stop')" :disabled="busyPlan !== null" @click="stopPlan(plan)"><Icon name="x" size="sm" /></button>
        </div>
      </article>
      </div>
    </div>
  </BaseDialog>
  <BaseDialog :show="!!historyPlan && show" :title="`${t('admin.accounts.quality.schedule.history')} · ${historyPlan?.name || ''}`" width="wide" :z-index="65" :close-on-escape="!detailRun" @close="historyPlan = null">
    <button class="btn btn-secondary btn-sm mb-4" @click="refreshHistory">{{ t('common.refresh') }}</button>
    <p v-if="historyError" class="text-bh-red">{{ historyError }}</p>
    <p v-if="!runs.length" class="text-sm text-gray-500">{{ t('admin.accounts.quality.schedule.noRuns') }}</p>
    <article v-for="run in runs" :key="run.id" class="mb-5 space-y-4 border-b-2 border-bh-ink pb-5">
      <p class="font-bold">#{{ run.id }} · {{ formatTime(run.started_at) }} · {{ t(`admin.accounts.quality.schedule.runStatus.${run.status}`) }}</p>
      <p class="text-xs text-gray-500 dark:text-gray-400">{{ t(run.trigger_source === 'manual' ? 'admin.accounts.quality.schedule.manualRun' : 'admin.accounts.quality.schedule.scheduledRun') }}</p>
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
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import CodexQualitySummary from './CodexQualitySummary.vue'
import CodexQualityProgress from './CodexQualityProgress.vue'
import CodexQualityResultCard from './CodexQualityResult.vue'
import { getModelsByPlatform } from '@/composables/useModelWhitelist'
import { adminAPI } from '@/api/admin'
import { qualitySchedulesAPI, type QualitySchedule, type QualityRun, type CodexQualityResult } from '@/api/admin/codexQuality'
import type { Account } from '@/types'
import { isQualityTestable as eligible, qualityProtocolOptions, qualityProtocolLabel } from './codexQualityPresentation'

const props = defineProps<{ show: boolean; accountIds: number[] }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const plans = ref<QualitySchedule[]>([]), draft = ref<QualitySchedule | null>(null)
const error = ref(''), saving = ref(false), selecting = ref(false)
const busyPlan = ref<number | null>(null)
const search = ref(''), accountRows = ref<Account[]>([]), accountPage = ref(1), accountTotal = ref(0)
const historyPlan = ref<QualitySchedule | null>(null), runs = ref<QualityRun[]>([]), historyError = ref('')
const detailRun = ref<QualityRun | null>(null), detailStatus = ref(''), detailPage = ref(1), detailTotal = ref(0), detailResults = ref<CodexQualityResult[]>([]), detailError = ref('')
const detailTitle = computed(() => `#${detailRun.value?.id || ''} · ${detailStatus.value ? t(`admin.accounts.quality.status.${detailStatus.value}`) : t('admin.accounts.quality.details')}`)
const models = getModelsByPlatform('openai').filter(id => !id.includes('image')).map(value => ({ value, label: value }))
const efforts = computed(() => [{ value: '', label: t('admin.accounts.quality.effortDefault') }, ...['none', 'minimal', 'low', 'medium', 'high', 'xhigh', 'max'].map(value => ({ value, label: value }))])
const concurrencyOptions = [1, 2, 3, 4, 5].map(value => ({ value, label: String(value) }))
const protocols = computed(() => qualityProtocolOptions(t('admin.accounts.quality.protocolDefault')))
let generation = 0, accountRequest = 0, detailRequest = 0, timer: ReturnType<typeof setInterval> | null = null
const message = (e: unknown) => e instanceof Error ? e.message : (e as { message?: string })?.message || t('common.error')
const formatTime = (value?: string) => value ? new Date(value).toLocaleString() : '—'

async function load() { const v = generation; try { const value = await qualitySchedulesAPI.list(); if (v === generation) plans.value = value } catch (e) { if (v === generation) error.value = message(e) } }
function newPlan() {
  draft.value = { id: 0, name: '', interval_minutes: 60, keep_runs: 30, enabled: true, config: { account_ids: [...props.accountIds], model: 'gpt-6-astra', reasoning_effort: '', api_protocol: 'responses', prompt: '', keyword: '', concurrency: 3, timeout_seconds: 120, confirm_scheduling: false } }
  error.value = ''; search.value = ''; void loadAccounts(1)
}
function edit(plan: QualitySchedule) { draft.value = JSON.parse(JSON.stringify(plan)); draft.value!.config.confirm_scheduling = false; draft.value!.config.timeout_seconds ||= 120; draft.value!.config.api_protocol ||= ''; error.value = ''; search.value = ''; void loadAccounts(1) }
async function loadAccounts(page = 1) {
  const v = ++accountRequest
  try { const value = await adminAPI.accounts.list(page, 50, { platform: 'openai', search: search.value }); if (v !== accountRequest) return; accountRows.value = value.items; accountTotal.value = value.total; accountPage.value = page } catch (e) { if (v === accountRequest) error.value = message(e) }
}
function toggleAccount(id: number) { if (!draft.value) return; const ids = draft.value.config.account_ids; draft.value.config.account_ids = ids.includes(id) ? ids.filter(value => value !== id) : [...ids, id] }
async function selectAll() {
  if (!draft.value || selecting.value) return
  selecting.value = true; const target = draft.value, query = search.value; const v = generation
  try {
    const ids: number[] = []
    for (let page = 1; ; page++) {
      const value = await adminAPI.accounts.list(page, 100, { platform: 'openai', search: query })
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
// 保存成功后再更新受控开关，重复点击不会并发提交或虚显成功。
async function actOnPlan(plan: QualitySchedule, action: () => Promise<unknown>) {
  if (busyPlan.value !== null) return
  busyPlan.value = plan.id; error.value = ''; const v = generation
  try { await action(); if (v === generation) { await load(); await refreshHistory() } }
  catch (e) { if (v === generation) error.value = message(e) }
  finally { busyPlan.value = null }
}
function toggleEnabled(plan: QualitySchedule) { return actOnPlan(plan, () => qualitySchedulesAPI.setEnabled(plan.id, !plan.enabled)) }
function stopPlan(plan: QualitySchedule) { return actOnPlan(plan, () => qualitySchedulesAPI.setEnabled(plan.id, false)) }
function trigger(plan: QualitySchedule) { return actOnPlan(plan, () => qualitySchedulesAPI.trigger(plan.id)) }
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
.quality-enabled.toggle-active { background-color: theme('colors.green.600'); }
.quality-icon { width: 36px; height: 36px; min-height: 36px; padding: 0; flex-shrink: 0; }
.quality-form :deep(.input-label) { min-height: 2.5rem; display: flex; align-items: flex-end; }
.quality-form :deep(.input), .quality-form :deep(.select-trigger) { min-height: 42px; }
</style>
