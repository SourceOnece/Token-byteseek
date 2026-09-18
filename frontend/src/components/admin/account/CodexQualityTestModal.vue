<template>
  <BaseDialog :show="show" :title="t('admin.accounts.quality.title')" width="extra-wide" :close-on-escape="category === null" @close="close">
    <div class="space-y-6 sm:space-y-8">
      <p class="border-l-4 border-bh-yellow pl-3 text-sm font-bold text-yellow-800 dark:text-bh-yellow">
        {{ t('admin.accounts.quality.warning', { count: targetIds.length, seconds: timeoutSeconds }) }}
      </p>
      <CodexQualityRules />
      <BauhausHelp :title="t('admin.accounts.quality.rules')">
        <p>{{ t('admin.accounts.quality.disclaimer') }}</p>
        <p>{{ t('admin.accounts.quality.rememberHint') }}</p>
        <p>{{ t('admin.accounts.quality.keywordHint') }}</p>
        <p>{{ t('admin.accounts.quality.rateHint') }}</p>
      </BauhausHelp>
      <div class="quality-fields grid items-end gap-5 sm:grid-cols-2">
        <div>
          <label class="input-label text-bh-blue dark:text-blue-300" for="quality-model">{{ t('admin.accounts.quality.model') }}</label>
          <Select id="quality-model" v-model="model" :options="models" :creatable="true" :searchable="true" :disabled="running" />
        </div>
        <div>
          <label class="input-label text-yellow-700 dark:text-bh-yellow" for="quality-effort">{{ t('admin.accounts.quality.effort') }}</label>
          <Select id="quality-effort" v-model="effort" :options="efforts" :disabled="running" />
        </div>
      </div>
      <div class="quality-fields grid items-end gap-5 sm:grid-cols-3">
        <div><label class="input-label" for="quality-protocol">{{ t('admin.accounts.quality.protocol') }}</label><Select id="quality-protocol" v-model="protocol" :options="protocols" :disabled="running" /></div>
        <div>
          <label class="input-label" for="quality-concurrency">{{ t('admin.accounts.quality.concurrency') }}</label>
          <Select id="quality-concurrency" v-model="concurrency" :options="concurrencyOptions" :disabled="running" />
        </div>
        <div>
          <label class="input-label" for="quality-timeout">{{ t('admin.accounts.quality.timeout') }}</label>
          <input id="quality-timeout" v-model.number="timeoutSeconds" type="number" min="10" max="3600" class="input w-full font-bold text-bh-blue dark:text-blue-300" :disabled="running" />
        </div>
      </div>
      <div>
        <label class="input-label" for="quality-prompt">{{ t('admin.accounts.quality.prompt') }}</label>
        <textarea id="quality-prompt" v-model="prompt" class="input w-full" rows="4" maxlength="16000" :disabled="running" />
      </div>
      <div>
        <label class="input-label" for="quality-keyword">{{ t('admin.accounts.quality.keyword') }}</label>
        <input id="quality-keyword" v-model="keyword" class="input w-full" maxlength="200" :disabled="running" />
        <p class="input-hint">{{ t('admin.accounts.quality.keywordShort') }}</p>
      </div>
      <label class="flex items-start gap-3 border-y-2 border-[color:var(--bh-ink)] py-4 text-sm font-semibold leading-relaxed">
        <input v-model="confirmed" type="checkbox" class="mt-1 h-4 w-4 shrink-0" :disabled="running" data-testid="quality-confirm" />
        <span>{{ t('admin.accounts.quality.confirm') }}</span>
      </label>
      <p v-if="error" role="alert" class="break-words text-sm text-red-600 dark:text-red-400">{{ error }}</p>
      <CodexQualitySummary :counts="stats" @select="openCategory" />
      <div v-if="running || results.length" class="flex flex-wrap gap-4 text-sm font-bold">
        <span class="text-bh-blue dark:text-blue-300">{{ t('admin.accounts.quality.model') }}：{{ model }}</span>
        <span class="text-yellow-700 dark:text-bh-yellow">{{ t('admin.accounts.quality.effort') }}：{{ effort || t('admin.accounts.quality.effortDefault') }}</span>
      </div>
      <CodexQualityProgress v-if="running || results.length" :done="results.length" :total="targetIds.length" :label="t(running ? 'admin.accounts.quality.progress' : completed ? 'admin.accounts.quality.completed' : 'admin.accounts.quality.stopped', { done: results.length, total: targetIds.length })" />
    </div>
    <template #footer>
      <button v-if="running" class="btn btn-danger" @click="stop">{{ t('admin.accounts.quality.stop') }}</button>
      <button v-else class="btn btn-primary" :disabled="!canStart" data-testid="quality-start" @click="start">{{ t('admin.accounts.quality.start') }}</button>
      <button class="btn btn-secondary" @click="close">{{ t('common.close') }}</button>
    </template>
  </BaseDialog>
  <CodexQualityResultsDialog :show="category !== null && show" :title="categoryTitle" :results="categoryResults" @close="category = null" />
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import BauhausHelp from '@/components/common/BauhausHelp.vue'
import CodexQualityRules from './CodexQualityRules.vue'
import { adminAPI } from '@/api/admin'
import { useAuthStore } from '@/stores/auth'
import { getModelsByPlatform } from '@/composables/useModelWhitelist'
import { qualityStats, runCodexQualityBatch, type CodexQualityResult } from '@/api/admin/codexQuality'
import CodexQualitySummary from './CodexQualitySummary.vue'
import CodexQualityProgress from './CodexQualityProgress.vue'
import CodexQualityResultsDialog from './CodexQualityResultsDialog.vue'
import { qualityProtocolOptions } from './codexQualityPresentation'
import { loadQualityPreferences, saveQualityPreferences } from './codexQualityPreferences'

const props = defineProps<{ show: boolean; accountIds: number[] }>()
const emit = defineEmits<{ close: []; result: [result: CodexQualityResult]; finished: [] }>()
const { t } = useI18n()
const authStore = useAuthStore()
const targetIds = ref<number[]>([])
const model = ref('gpt-6-astra')
const effort = ref('')
const protocol = ref('responses')
const protocols = computed(() => qualityProtocolOptions(t('admin.accounts.quality.protocolDefault')))
const prompt = ref('')
const keyword = ref('')
const concurrency = ref(3)
const timeoutSeconds = ref(120)
const confirmed = ref(false)
const running = ref(false)
const completed = ref(false)
const error = ref('')
const results = ref<CodexQualityResult[]>([])
const category = ref<string | null>(null)
const categoryTitle = computed(() => category.value ? t(`admin.accounts.quality.status.${category.value}`) : t('admin.accounts.quality.details'))
const categoryResults = computed(() => category.value ? results.value.filter(result => result.status === category.value) : results.value)
function openCategory(status: string) { category.value = status }
const models = ref(getModelsByPlatform('openai').filter(id => !id.includes('image')).map(value => ({ value, label: value })))
const efforts = computed(() => [
  { value: '', label: t('admin.accounts.quality.effortDefault') },
  ...['none', 'minimal', 'low', 'medium', 'high', 'xhigh', 'max'].map(value => ({ value, label: value }))
])
const concurrencyOptions = [1, 2, 3, 4, 5].map(value => ({ value, label: String(value) }))
const stats = computed(() => { const value = qualityStats(results.value); return { full: value.full, degraded: value.degraded, failed: value.failed } })
const canStart = computed(() => confirmed.value && model.value.trim() && prompt.value.trim() && keyword.value.trim() && targetIds.value.length > 0 && targetIds.value.length <= 500 && Number.isInteger(timeoutSeconds.value) && timeoutSeconds.value >= 10 && timeoutSeconds.value <= 3600)
let controller: AbortController | null = null
let generation = 0
let preferencesReady = false
let preferencesOwner: number | undefined

// 显式打开/换管理员时恢复配置；确认框和本轮账号集合始终重新初始化。
watch([() => props.show, () => authStore.user?.id], async ([show, adminId]) => {
  const version = ++generation
  controller?.abort()
  preferencesReady = false
  if (!show) return
  preferencesOwner = adminId
  const previous = loadQualityPreferences(adminId)
  model.value = previous.model; effort.value = previous.effort; protocol.value = previous.protocol
  prompt.value = previous.prompt; keyword.value = previous.keyword
  concurrency.value = previous.concurrency; timeoutSeconds.value = previous.timeoutSeconds
  if (!models.value.some(item => item.value === model.value)) models.value = [{ value: model.value, label: model.value }, ...models.value]
  preferencesReady = true
  // 打开时冻结选中集合，测试中勾选其它账号不会扩大操作范围。
  targetIds.value = [...new Set(props.accountIds)]
  results.value = []; category.value = null; error.value = ''; completed.value = false; confirmed.value = false
  if (targetIds.value.length > 500) error.value = t('admin.accounts.quality.tooMany')
  try {
    if (!targetIds.value.length) return
    const options = await adminAPI.accounts.getAvailableModels(targetIds.value[0])
    if (version !== generation) return
    models.value = [...new Set([model.value, 'gpt-6-astra', ...options.map(item => item.id)])]
      .filter(id => !id.includes('image')).map(value => ({ value, label: value }))
  } catch { /* 模型目录失败仍允许自研 Select 输入模型 ID。 */ }
}, { immediate: true })

// 同步保存避免紧接着关闭/跳页时遗漏最后一次输入；恢复期间不把默认值写回。
watch([model, effort, protocol, prompt, keyword, concurrency, timeoutSeconds], () => {
  if (!preferencesReady || !props.show || preferencesOwner !== authStore.user?.id) return
  saveQualityPreferences(preferencesOwner, { model: model.value, effort: effort.value, protocol: protocol.value,
    prompt: prompt.value, keyword: keyword.value, concurrency: concurrency.value, timeoutSeconds: timeoutSeconds.value })
}, { flush: 'sync' })

async function start() {
  if (!canStart.value || running.value) return
  const version = generation
  running.value = true; completed.value = false; results.value = []; category.value = null; error.value = ''
  const current = new AbortController(); controller = current
  try {
    await runCodexQualityBatch({ account_ids: targetIds.value, model: model.value.trim(), reasoning_effort: effort.value,
      prompt: prompt.value.trim(), keyword: keyword.value.trim(), api_protocol: protocol.value, concurrency: concurrency.value, timeout_seconds: timeoutSeconds.value, confirm_scheduling: confirmed.value }, current.signal, result => {
      if (version !== generation) return
      results.value = [...results.value.filter(item => item.account_id !== result.account_id), result]
      emit('result', result)
    })
    if (version === generation) completed.value = true
  } catch (e) {
    if (version === generation && !current.signal.aborted) error.value = e instanceof Error ? e.message : t('admin.accounts.quality.requestFailed')
  } finally {
    if (controller === current) { controller = null; running.value = false }
    emit('finished')
  }
}
function stop() { controller?.abort() }
function close() { stop(); emit('close') }
onBeforeUnmount(() => { generation++; stop() })
</script>

<style scoped>
.quality-fields :deep(.input-label) { min-height: 1.5rem; display: flex; align-items: flex-end; font-size: .875rem; }
.quality-fields :deep(.input), .quality-fields :deep(.select-trigger) { min-height: 42px; }
</style>
