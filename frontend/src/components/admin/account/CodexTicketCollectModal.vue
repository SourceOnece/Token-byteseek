<template>
  <BaseDialog :show="show" :title="t('admin.accounts.ticketCollect.title')" width="extra-wide" :close-on-escape="!detailID" @close="close">
    <div class="space-y-6 sm:space-y-8">
      <div class="flex flex-wrap gap-2">
        <button class="btn btn-primary btn-sm" :disabled="running" @click="tab = 'collect'">{{ t('admin.accounts.ticketCollect.collectTab') }}</button>
        <button class="btn btn-secondary btn-sm" :disabled="running" @click="tab = 'history'; loadHistory()">{{ t('admin.accounts.ticketCollect.history') }}</button>
      </div>
      <template v-if="tab === 'collect'">
        <p class="border-l-4 border-bh-yellow pl-3 text-base font-semibold text-yellow-800 dark:text-bh-yellow">{{ t('admin.accounts.ticketCollect.hint', { count: targets.length }) }}</p>
        <div v-if="settings" class="border-2 border-[color:var(--bh-ink)] bg-[var(--bh-surface)] p-5 sm:p-6" style="box-shadow:var(--bh-shadow-sm)">
          <p class="text-lg font-extrabold text-bh-blue dark:text-blue-300">{{ t('admin.accounts.ticketWorkbench.accountRules') }}</p>
          <p class="mt-3 text-sm text-gray-600 dark:text-gray-300">{{ t('admin.accounts.ticketWorkbench.unlimitedRisk') }}</p>
        </div>
        <p v-if="settings && !settings.enabled" class="font-bold text-bh-red">{{ t('admin.accounts.ticketCollect.disabled') }}</p>
        <label class="flex items-start gap-3 text-sm font-semibold leading-relaxed"><input v-model="confirmed" type="checkbox" class="mt-1 h-4 w-4 shrink-0" :disabled="running" data-testid="ticket-collect-confirm" />{{ t('admin.accounts.ticketCollect.confirm') }}</label>
        <p class="text-sm font-medium leading-relaxed text-yellow-800 dark:text-bh-yellow">{{ t('admin.accounts.ticketCollect.historyHint') }}</p>
        <BauhausHelp :title="t('admin.accounts.quality.rules')"><p>{{ t('admin.accounts.ticketCollect.eligibilityHint') }}</p><p>{{ t('admin.accounts.ticketCollect.ipHint') }}</p></BauhausHelp>
        <div v-if="run" class="space-y-4">
          <CodexQualityProgress :done="processed" :total="run.total" :label="t('admin.accounts.ticketCollect.progress')" />
          <div class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
            <button v-for="status in statuses" :key="status" class="ticket-stat border-2 border-[color:var(--bh-ink)] bg-[var(--bh-surface)] p-4 text-left focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2" :class="color(status)" @click="openResults(run.id, status)">
              <span class="block text-sm font-bold">{{ statusLabel(status) }}</span><span class="mt-3 block text-3xl font-extrabold tabular-nums">{{ run.counts[status] || 0 }}</span>
            </button>
          </div>
          <details class="border-2 border-[color:var(--bh-ink)] p-3"><summary class="cursor-pointer font-bold text-bh-blue dark:text-blue-300">{{ t('admin.accounts.ticketCollect.live') }}</summary>
            <div class="mt-3 max-h-64 space-y-2 overflow-y-auto">
              <p v-for="event in live" :key="event.id" class="break-words text-xs"><strong>{{ event.email || event.account_name || event.account_id }}</strong> · <span class="font-bold text-bh-blue dark:text-blue-300">{{ event.model }}</span> · #{{ event.attempt }} · {{ statusLabel(event.status) }} · {{ event.diagnostic?.proxy_name || '—' }} · {{ event.reference_ip || t('admin.accounts.ticketCollect.ipUnknown') }}</p>
            </div>
          </details>
          <p class="text-xs text-gray-500">{{ t('admin.accounts.ticketCollect.runStatus') }}：{{ runLabel(run.status) }}</p>
        </div>
      </template>
      <template v-else>
        <p class="text-sm leading-relaxed text-gray-500 dark:text-gray-400">{{ t('admin.accounts.ticketCollect.historyHint') }}</p>
        <p class="text-sm leading-relaxed text-yellow-800 dark:text-bh-yellow">{{ t('admin.accounts.ticketCollect.deleteHint') }}</p>
        <button class="btn btn-danger btn-sm" :disabled="deleting || busy || running || !history.length" data-testid="ticket-history-clear" @click="deleteHistory()">{{ t('admin.accounts.ticketCollect.clearHistory') }}</button>
        <div v-for="item in history" :key="item.id" class="flex items-center gap-3 border-2 border-[color:var(--bh-ink)] bg-[var(--bh-surface)] p-3" style="box-shadow:var(--bh-shadow-sm)">
          <button class="min-w-0 flex-1 text-left focus-visible:outline focus-visible:outline-2" @click="openResults(item.id)">
            <span class="block break-words font-bold text-bh-blue dark:text-blue-300">{{ new Date(item.started_at).toLocaleString() }} · {{ item.config.account_rules?.length ? t('admin.accounts.ticketWorkbench.perAccount') : (item.config.target_length || 292) + ' bytes' }}</span>
            <span class="text-sm">{{ runLabel(item.status) }} · {{ totalDone(item) }} / {{ item.total }}</span>
          </button>
          <button class="btn btn-danger btn-sm shrink-0" :disabled="deleting || item.status === 'running'" :title="item.status === 'running' ? t('admin.accounts.ticketCollect.deleteActive') : t('common.delete')" data-testid="ticket-history-delete" @click="deleteHistory(item.id)">{{ t('common.delete') }}</button>
        </div>
        <div class="flex justify-end gap-2"><button class="btn btn-secondary btn-sm" :disabled="historyPage <= 1 || busy" @click="historyPage--; loadHistory()">{{ t('admin.accounts.ticketCollect.previous') }}</button><span>{{ historyPage }}</span><button class="btn btn-secondary btn-sm" :disabled="history.length < 20 || busy" @click="historyPage++; loadHistory()">{{ t('admin.accounts.ticketCollect.next') }}</button></div>
      </template>
      <p v-if="error" role="alert" class="break-words text-sm text-bh-red dark:text-red-400">{{ error }}</p>
      <p v-if="notice" role="status" class="text-sm font-bold text-bh-blue dark:text-blue-300">{{ notice }}</p>
    </div>
    <template #footer>
      <button v-if="running" class="btn btn-danger" @click="controller?.abort()">{{ t('admin.accounts.ticketCollect.stop') }}</button>
      <button v-else-if="tab === 'collect'" class="btn btn-primary" :disabled="!canStart" data-testid="ticket-collect-start" @click="start">{{ t('admin.accounts.ticketCollect.start') }}</button>
      <button class="btn btn-secondary" @click="close">{{ t('common.close') }}</button>
    </template>
  </BaseDialog>
  <BaseDialog :show="show && !!detailID" :title="t('admin.accounts.ticketCollect.results')" width="extra-wide" :z-index="70" @close="detailID = ''">
    <div class="space-y-5">
      <p class="text-sm leading-relaxed text-gray-500 dark:text-gray-400">{{ t('admin.accounts.ticketCollect.ipHint') }}</p>
      <div class="flex flex-wrap gap-2"><button class="btn btn-secondary btn-sm" @click="detailKind = 'result'; detailAccount = 0; detailModel = ''; detailPage = 1; loadDetail()">{{ t('admin.accounts.ticketCollect.results') }}</button><button class="btn btn-secondary btn-sm" @click="detailKind = 'attempt'; detailPage = 1; loadDetail()">{{ t('admin.accounts.ticketCollect.attempts') }}</button><button class="btn btn-secondary btn-sm" @click="loadDetail()">{{ t('common.refresh') }}</button></div>
      <p v-if="detailError" role="alert" class="text-sm text-bh-red">{{ detailError }}</p>
      <article v-for="item in details" :key="item.id" class="space-y-4 border-2 border-[color:var(--bh-ink)] bg-[var(--bh-surface)] p-5 text-sm sm:p-6" style="box-shadow:var(--bh-shadow-sm)">
        <div class="flex flex-wrap justify-between gap-3 border-b-2 border-[color:var(--bh-ink)] pb-4"><strong class="break-all text-lg">{{ item.email || item.account_name || item.account_id }}</strong><strong :class="color(item.status)">{{ statusLabel(item.status) }}</strong></div>
        <p class="break-words text-lg font-bold text-bh-blue dark:text-blue-300">{{ item.model }} · {{ t('admin.accounts.ticketCollect.target') }} {{ item.target_length }} · #{{ item.attempt }}</p>
        <p>{{ new Date(item.started_at).toLocaleString() }} · {{ item.duration_ms }} ms</p>
        <p class="break-words">{{ item.diagnostic?.proxy_name || '—' }} · {{ t('admin.accounts.ticketCollect.ip') }}：<span class="font-mono font-bold text-bh-blue dark:text-blue-300">{{ item.reference_ip || t('admin.accounts.ticketCollect.ipUnknown') }}</span></p>
        <p v-if="item.ip_status && item.ip_status !== 'reference'" class="text-xs text-yellow-800 dark:text-bh-yellow" data-testid="ticket-ip-error">{{ ipStatusLabel(item.ip_status) }}<span v-if="item.ip_http_status"> · HTTP {{ item.ip_http_status }}</span></p>
        <p v-if="item.ip_source" class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.ticketCollect.ipSource') }}：{{ ipSourceLabel(item.ip_source) }}<span v-if="geoCountry(item.reference_ip)" class="ml-2 font-mono font-extrabold text-bh-blue dark:text-blue-300">[{{ geoCountry(item.reference_ip) }}]</span><span v-if="item.ip_checked_at"> · {{ new Date(item.ip_checked_at).toLocaleString() }}</span></p>
        <p v-if="item.diagnostic">HTTP {{ item.diagnostic.http_status || '—' }} · <CodexTicketLength v-if="item.diagnostic.header_present" :actual="item.diagnostic.header_length" :target="item.target_length" :signal="item.diagnostic.degraded_signal" /><span v-else>{{ t('admin.accounts.tickets.noHeader') }}</span><span v-if="responseKindLabel(item.diagnostic.response_kind)"> · {{ responseKindLabel(item.diagnostic.response_kind) }}</span><span v-if="item.diagnostic.error_kind"> · {{ errorKind(item.diagnostic.error_kind) }}</span></p>
        <p v-if="item.diagnostic?.degraded_signal" class="font-bold text-bh-red dark:text-red-400">{{ t('admin.accounts.tickets.degradedSignal') }}</p>
        <p v-if="item.reason" class="text-xs">{{ reasonLabel(item.reason) }}</p>
        <p v-if="item.diagnostic?.retry_not_before" class="text-xs text-yellow-800 dark:text-bh-yellow">{{ t('admin.accounts.tickets.retryAfter', { time: new Date(item.diagnostic.retry_not_before).toLocaleString() }) }}</p>
        <button v-if="detailKind === 'result' && item.attempt" class="btn btn-secondary btn-sm" @click="detailKind = 'attempt'; detailAccount = item.account_id; detailModel = item.model; detailStatus = ''; detailPage = 1; loadDetail()">{{ t('admin.accounts.ticketCollect.attempts') }}</button>
        <button class="btn btn-danger btn-sm" :disabled="deleting || detailRunStatus === 'running' || !item.id" :title="t('admin.accounts.ticketCollect.deleteHint')" data-testid="ticket-event-delete" @click="deleteEvent(item)">{{ t('common.delete') }}</button>
      </article>
      <p v-if="!details.length && !detailBusy" class="text-sm text-gray-500">{{ t('admin.accounts.ticketCollect.empty') }}</p>
      <Pagination :page="detailPage" :page-size="50" :total="detailTotal" :show-page-size-selector="false" @update:page="detailPage = $event; loadDetail()" />
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import CodexQualityProgress from './CodexQualityProgress.vue'
import CodexTicketLength from './CodexTicketLength.vue'
import BauhausHelp from '@/components/common/BauhausHelp.vue'
import { runTicketCollection, ticketCollectionAPI, type TicketCollectionEvent, type TicketCollectionRun, type TicketSettings } from '@/api/admin/codexTickets'
import { useAuthStore } from '@/stores/auth'
import { fetchOne, getEntry } from '@/utils/ipGeoLookup'
const props = defineProps<{ show: boolean; accountIds: number[]; historyOnly?: boolean }>()
const emit = defineEmits<{ close: []; finished: []; result: [] }>()
const { t } = useI18n(), auth = useAuthStore()
const targets = ref<number[]>([]), settings = ref<TicketSettings>(), confirmed = ref(false), running = ref(false), busy = ref(false), error = ref('')
const tab = ref('collect'), run = ref<TicketCollectionRun>(), live = ref<TicketCollectionEvent[]>([]), history = ref<TicketCollectionRun[]>([]), historyPage = ref(1)
const deleting = ref(false), notice = ref(''), detailRunStatus = ref('running')
const detailID = ref(''), detailKind = ref('result'), detailStatus = ref(''), detailAccount = ref(0), detailModel = ref(''), detailPage = ref(1), detailTotal = ref(0), details = ref<TicketCollectionEvent[]>([]), detailError = ref(''), detailBusy = ref(false)
const statuses = ['ready', 'missing', 'failed', 'skipped', 'cancelled']
let controller: AbortController | null = null, generation = 0, detailGeneration = 0, historyGeneration = 0
const canStart = computed(() => !busy.value && !deleting.value && !running.value && confirmed.value && settings.value?.enabled && targets.value.length > 0 && targets.value.length <= 500)
const totalDone = (r: TicketCollectionRun) => Object.values(r.counts).reduce((a, b) => a + b, 0)
const processed = computed(() => run.value ? totalDone(run.value) : 0)
const statusLabel = (s: string) => t('admin.accounts.ticketCollect.status.' + (statuses.includes(s) ? s : 'skipped'))
const runLabel = (s: string) => t('admin.accounts.ticketCollect.run.' + (['running', 'completed', 'cancelled', 'failed', 'interrupted'].includes(s) ? s : 'interrupted'))
const color = (s: string) => s === 'ready' ? 'text-emerald-700 dark:text-emerald-400' : s === 'failed' ? 'text-bh-red dark:text-red-400' : s === 'missing' ? 'text-yellow-800 dark:text-bh-yellow' : 'text-gray-500 dark:text-gray-400'
const reasonLabel = (s: string) => ['network', 'upstream', 'invalid_ticket', 'credential', 'storage', 'cancelled', 'proxy_config'].includes(s) ? t('admin.accounts.tickets.reason.' + s) : t('admin.accounts.ticketCollect.reason.' + (['ineligible', 'account_changed', 'concurrency_busy', 'backoff'].includes(s) ? s : 'account_changed'))
const errorKind = (s: string) => ['overloaded', 'rate_limit', 'quota', 'auth', 'invalid_request'].includes(s) ? t('admin.accounts.tickets.errorKind.' + s) : '—'
const ipStatusLabel = (s: string) => t('admin.accounts.ticketCollect.ipStatus.' + (['unavailable', 'timeout', 'network', 'tls', 'http_error', 'invalid_response', 'proxy_config', 'cancelled', 'not_attempted'].includes(s) ? s : 'unavailable'))
const ipSourceLabel = (s: string) => s === 'chatgpt_trace' ? t('admin.accounts.ticketCollect.source.chatgptTrace') : s === 'ipify' ? t('admin.accounts.ticketCollect.source.ipify') : t('admin.accounts.ticketCollect.configuredSource')
// 未知响应类型只用于服务端诊断，不在用户日志中显示无意义的“other”。
const responseKindLabel = (s?: string) => s === 'sse' ? 'SSE' : s === 'json' ? 'JSON' : s === 'html' ? 'HTML' : ''
// 复用 IP 管理的 GeoJS 缓存，只查询已展示的参考 IP，不改票据采集请求。
const geoCountry = (ip?: string) => ip ? getEntry(ip).detail?.countryCode?.toUpperCase() || '' : ''
async function loadReferenceGeo(items: TicketCollectionEvent[]) {
  const ips = [...new Set(items.map(item => item.reference_ip).filter((ip): ip is string => Boolean(ip)))]
  await Promise.all(ips.map(ip => fetchOne(ip)))
}

// 冻结选中集合，换管理员/关闭时取消，避免旧异步结果写进新弹窗。
watch([() => props.show, () => auth.user?.id], async ([show]) => {
  const version = ++generation; controller?.abort(); detailGeneration++; historyGeneration++; running.value = false; detailID.value = ''
  if (!show) return
  targets.value = [...new Set(props.accountIds)]; tab.value = props.historyOnly ? 'history' : 'collect'
  settings.value = undefined; confirmed.value = false; run.value = undefined; live.value = []; error.value = ''; notice.value = ''; historyPage.value = 1; busy.value = true
  try { const data = await ticketCollectionAPI.settings(); if (version === generation) settings.value = data }
  catch (e) { if (version === generation) error.value = e instanceof Error ? e.message : t('common.error') }
  finally { if (version === generation) busy.value = false }
  if (version === generation && tab.value === 'history') void loadHistory()
}, { immediate: true })
async function loadHistory() {
  const version = ++historyGeneration, current = generation; busy.value = true
  try {
    const data = await ticketCollectionAPI.runs(historyPage.value)
    if (version === historyGeneration && current === generation) {
      history.value = data
      if (!data.length && historyPage.value > 1) { historyPage.value--; void loadHistory() }
    }
  }
  catch (e) { if (current === generation) error.value = e instanceof Error ? e.message : t('common.error') }
  finally { if (version === historyGeneration && current === generation) busy.value = false }
}
function openResults(id: string, status = '') { details.value = []; detailTotal.value = 0; detailRunStatus.value = 'running'; detailID.value = id; detailKind.value = 'result'; detailStatus.value = status; detailAccount.value = 0; detailModel.value = ''; detailPage.value = 1; void loadDetail() }
async function loadDetail() {
  const version = ++detailGeneration, current = generation; detailBusy.value = true; detailError.value = ''
  try {
    const data = await ticketCollectionAPI.detail(detailID.value, { page: detailPage.value, kind: detailKind.value, status: detailStatus.value, account_id: detailAccount.value, model: detailModel.value })
    if (version === detailGeneration && current === generation) {
      details.value = data.items; detailTotal.value = data.total; detailRunStatus.value = data.run.status
      void loadReferenceGeo(data.items)
      if (run.value?.id === data.run.id && !running.value) run.value = data.run
      if (!data.items.length && detailPage.value > 1) { detailPage.value--; void loadDetail() }
    }
  } catch (e) { if (version === detailGeneration && current === generation) detailError.value = e instanceof Error ? e.message : t('common.error') }
  finally { if (version === detailGeneration && current === generation) detailBusy.value = false }
}
// 按用户要求直接执行，不弹第二次确认；请求失败不预先移除记录。
async function deleteHistory(id?: string) {
  if (deleting.value || running.value) return
  const current = generation; deleting.value = true; error.value = ''; notice.value = ''; historyGeneration++; detailGeneration++
  try {
    const data = id ? await ticketCollectionAPI.deleteRun(id) : await ticketCollectionAPI.clearHistory()
    if (current !== generation) return
    notice.value = t('admin.accounts.ticketCollect.deleted', { count: data.deleted })
    if (!id || detailID.value === id) { detailID.value = ''; details.value = [] }
    if (!id || run.value?.id === id) { run.value = undefined; live.value = [] }
    if (!id) historyPage.value = 1
    await loadHistory()
  } catch (e) { if (current === generation) error.value = e instanceof Error ? e.message : t('common.error') }
  finally { deleting.value = false }
}
async function deleteEvent(item: TicketCollectionEvent) {
  if (deleting.value || !item.id || detailRunStatus.value === 'running') return
  const current = generation, id = detailID.value; deleting.value = true; detailError.value = ''; detailGeneration++
  try {
    await ticketCollectionAPI.deleteEvent(id, item.id)
    if (current !== generation || id !== detailID.value) return
    live.value = live.value.filter(e => e.id !== item.id)
    await loadDetail()
    if (tab.value === 'history') await loadHistory()
  } catch (e) { if (current === generation && id === detailID.value) detailError.value = e instanceof Error ? e.message : t('common.error') }
  finally { deleting.value = false }
}
async function start() {
  if (!canStart.value || !settings.value) return
  const version = generation, current = new AbortController(); controller = current; running.value = true; run.value = undefined; live.value = []; error.value = ''
  try {
    await runTicketCollection(targets.value, settings.value.revision, current.signal, (kind, data) => {
      if (version !== generation) return
      if (kind === 'start' || kind === 'complete') run.value = data as TicketCollectionRun
      else if (kind === 'result' && run.value) { const e = data as TicketCollectionEvent; run.value.counts[e.status] = (run.value.counts[e.status] || 0) + 1; emit('result') }
      else if (kind === 'attempt') live.value = [data as TicketCollectionEvent, ...live.value].slice(0, 50)
    })
  } catch (e) { if (version === generation && !current.signal.aborted) error.value = e instanceof Error ? e.message : t('common.error') }
  finally {
    if (version === generation) { running.value = false; const activeRun = run.value as TicketCollectionRun | undefined; if (activeRun?.status === 'running') activeRun.status = current.signal.aborted ? 'cancelled' : 'interrupted' }
    if (controller === current) controller = null
    emit('finished')
  }
}
function close() { controller?.abort(); emit('close') }
onBeforeUnmount(() => { generation++; controller?.abort() })
</script>

<style scoped>
/* 分类卡片复用同源硬阴影，按压收短而非跟随卡片一起漂移。 */
.ticket-stat { min-width: 0; box-shadow: var(--bh-shadow-sm); overflow-wrap: anywhere; }
.ticket-stat:active { transform: translate(2px, 2px); box-shadow: 1px 1px 0 var(--bh-shadow-ink); }
</style>
