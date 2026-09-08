<template>
  <BaseDialog :show="show" :title="t('admin.accounts.quality.title')" width="extra-wide" @close="close">
    <div class="space-y-5">
      <p class="border-l-4 border-bh-yellow bg-yellow-50 p-3 text-sm text-gray-900 dark:bg-yellow-950 dark:text-yellow-100">
        {{ t('admin.accounts.quality.warning', { count: targetIds.length }) }}
      </p>
      <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.quality.disclaimer') }}</p>
      <div class="grid gap-4 sm:grid-cols-3">
        <div>
          <label class="input-label" for="quality-model">{{ t('admin.accounts.quality.model') }}</label>
          <Select id="quality-model" v-model="model" :options="models" :creatable="true" :searchable="true" :disabled="running" />
        </div>
        <div>
          <label class="input-label" for="quality-effort">{{ t('admin.accounts.quality.effort') }}</label>
          <Select id="quality-effort" v-model="effort" :options="efforts" :disabled="running" />
        </div>
        <div>
          <label class="input-label" for="quality-concurrency">{{ t('admin.accounts.quality.concurrency') }}</label>
          <Select id="quality-concurrency" v-model="concurrency" :options="concurrencyOptions" :disabled="running" />
        </div>
      </div>
      <div>
        <label class="input-label" for="quality-prompt">{{ t('admin.accounts.quality.prompt') }}</label>
        <textarea id="quality-prompt" v-model="prompt" class="input w-full" rows="4" maxlength="16000" :disabled="running" />
      </div>
      <div>
        <label class="input-label" for="quality-keyword">{{ t('admin.accounts.quality.keyword') }}</label>
        <input id="quality-keyword" v-model="keyword" class="input w-full" maxlength="200" :disabled="running" />
        <p class="input-hint">{{ t('admin.accounts.quality.keywordHint') }}</p>
      </div>
      <label class="flex items-start gap-2 text-sm font-semibold">
        <input v-model="confirmed" type="checkbox" class="mt-1" :disabled="running" data-testid="quality-confirm" />
        <span>{{ t('admin.accounts.quality.confirm') }}</span>
      </label>
      <p v-if="error" role="alert" class="break-words text-sm text-red-600 dark:text-red-400">{{ error }}</p>
      <div class="grid grid-cols-2 gap-3 lg:grid-cols-4" aria-live="polite">
        <div class="quality-stat"><p class="text-xs">{{ t('admin.accounts.quality.rate') }}</p><strong class="text-2xl text-green-700 dark:text-green-300">{{ stats.rate === null ? '—' : `${stats.rate}%` }}</strong></div>
        <div class="quality-stat"><p class="text-xs">{{ t('admin.accounts.quality.status.full') }}</p><strong class="text-2xl">{{ stats.full }}</strong></div>
        <div class="quality-stat"><p class="text-xs">{{ t('admin.accounts.quality.status.degraded') }}</p><strong class="text-2xl">{{ stats.degraded }}</strong></div>
        <div class="quality-stat"><p class="text-xs">{{ t('admin.accounts.quality.status.failed') }}</p><strong class="text-2xl">{{ stats.failed }}</strong></div>
      </div>
      <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.quality.rateHint') }}</p>
      <p v-if="running || results.length" class="text-sm font-semibold" role="status">
        {{ t(running ? 'admin.accounts.quality.progress' : completed ? 'admin.accounts.quality.completed' : 'admin.accounts.quality.stopped', { done: results.length, total: targetIds.length }) }}
      </p>
      <div class="space-y-4"><CodexQualityResultCard v-for="result in pagedResults" :key="result.account_id" :result="result" /></div>
      <Pagination v-if="results.length > 20" :page="resultPage" :page-size="20" :total="results.length" :show-page-size-selector="false" @update:page="resultPage = $event" />
    </div>
    <template #footer>
      <button v-if="running" class="btn btn-danger" @click="stop">{{ t('admin.accounts.quality.stop') }}</button>
      <button v-else class="btn btn-primary" :disabled="!canStart" data-testid="quality-start" @click="start">{{ t('admin.accounts.quality.start') }}</button>
      <button class="btn btn-secondary" @click="close">{{ t('common.close') }}</button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Pagination from '@/components/common/Pagination.vue'
import { adminAPI } from '@/api/admin'
import { getModelsByPlatform } from '@/composables/useModelWhitelist'
import { qualityStats, runCodexQualityBatch, type CodexQualityResult } from '@/api/admin/codexQuality'
import CodexQualityResultCard from './CodexQualityResult.vue'

const props = defineProps<{ show: boolean; accountIds: number[] }>()
const emit = defineEmits<{ close: []; result: [result: CodexQualityResult]; finished: [] }>()
const { t } = useI18n()
const targetIds = ref<number[]>([])
const model = ref('gpt-6-astra')
const effort = ref('')
const prompt = ref('')
const keyword = ref('')
const concurrency = ref(3)
const confirmed = ref(false)
const running = ref(false)
const completed = ref(false)
const error = ref('')
const results = ref<CodexQualityResult[]>([])
const resultPage = ref(1)
const pagedResults = computed(() => results.value.slice((resultPage.value - 1) * 20, resultPage.value * 20))
const models = ref(getModelsByPlatform('openai').filter(id => !id.includes('image')).map(value => ({ value, label: value })))
const efforts = computed(() => [
  { value: '', label: t('admin.accounts.quality.effortDefault') },
  ...['none', 'minimal', 'low', 'medium', 'high', 'xhigh', 'max'].map(value => ({ value, label: value }))
])
const concurrencyOptions = [1, 2, 3, 4, 5].map(value => ({ value, label: String(value) }))
const stats = computed(() => qualityStats(results.value))
const canStart = computed(() => confirmed.value && model.value.trim() && prompt.value.trim() && keyword.value.trim() && targetIds.value.length > 0 && targetIds.value.length <= 500)
let controller: AbortController | null = null
let generation = 0

watch(() => props.show, async show => {
  const version = ++generation
  if (!show) { controller?.abort(); return }
  // 打开时冻结选中集合，测试中勾选其它账号不会扩大操作范围。
  targetIds.value = [...new Set(props.accountIds)]
  results.value = []; resultPage.value = 1; error.value = ''; completed.value = false; confirmed.value = false
  if (targetIds.value.length > 500) error.value = t('admin.accounts.quality.tooMany')
  try {
    if (!targetIds.value.length) return
    const options = await adminAPI.accounts.getAvailableModels(targetIds.value[0])
    if (version !== generation) return
    models.value = [...new Set(['gpt-6-astra', ...options.map(item => item.id)])]
      .filter(id => !id.includes('image')).map(value => ({ value, label: value }))
  } catch { /* 模型目录失败仍允许自研 Select 输入模型 ID。 */ }
})

async function start() {
  if (!canStart.value || running.value) return
  const version = generation
  running.value = true; completed.value = false; results.value = []; resultPage.value = 1; error.value = ''
  const current = new AbortController(); controller = current
  try {
    await runCodexQualityBatch({ account_ids: targetIds.value, model: model.value.trim(), reasoning_effort: effort.value,
      prompt: prompt.value.trim(), keyword: keyword.value.trim(), concurrency: concurrency.value, confirm_scheduling: confirmed.value }, current.signal, result => {
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
.quality-stat { padding: .75rem; border: 2px solid var(--bh-ink); background: var(--bh-surface, #fff); box-shadow: var(--bh-shadow-sm); }
</style>
