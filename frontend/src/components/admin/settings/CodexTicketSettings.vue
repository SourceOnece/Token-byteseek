<template>
  <section class="space-y-4 border-2 border-[color:var(--bh-ink)] bg-[var(--bh-surface)] p-4" style="box-shadow: var(--bh-shadow-sm)" data-testid="codex-ticket-settings">
    <div class="flex items-start justify-between gap-4">
      <div><h3 class="font-bold text-bh-blue dark:text-blue-300">{{ t('admin.settings.codexTicket.title') }}</h3>
        <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">{{ t('admin.settings.codexTicket.description') }}</p></div>
      <Toggle v-model="enabled" :disabled="locked" :aria-label="t('admin.settings.codexTicket.title')" data-testid="codex-ticket-toggle" />
    </div>
    <p class="border-l-4 border-bh-yellow pl-3 text-sm font-semibold text-yellow-800 dark:text-bh-yellow">{{ t('admin.settings.codexTicket.warning') }}</p>
    <!-- 固定运行参数用数字层级表达；详细限制按需展开，保存风险保持可见。 -->
    <dl class="grid grid-cols-3 gap-3 text-sm" data-testid="ticket-rule-metrics">
      <div class="border-l-4 border-bh-blue pl-3"><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.codexTicket.cacheTime') }}</dt><dd class="font-mono text-xl font-bold text-bh-blue dark:text-blue-300">60 <small>min</small></dd></div>
      <div class="border-l-4 border-bh-yellow pl-3"><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.codexTicket.renewEarly') }}</dt><dd class="font-mono text-xl font-bold text-yellow-700 dark:text-bh-yellow">10 <small>min</small></dd></div>
      <div class="border-l-4 border-bh-red pl-3"><dt class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.codexTicket.workers') }}</dt><dd class="font-mono text-xl font-bold text-bh-red dark:text-red-400">4</dd></div>
    </dl>
    <BauhausHelp :title="t('admin.accounts.quality.rules')"><p>{{ t('admin.settings.codexTicket.rules') }}</p><p>{{ t('admin.settings.codexTicket.limits') }}</p></BauhausHelp>
    <div>
      <label for="ticket-models" class="input-label text-bh-blue dark:text-blue-300">{{ t('admin.settings.codexTicket.models') }}</label>
      <textarea id="ticket-models" v-model="modelText" rows="3" class="input w-full font-mono" :disabled="locked" :aria-describedby="'ticket-models-hint'" />
      <p id="ticket-models-hint" class="input-hint">{{ t('admin.settings.codexTicket.modelsHint') }}</p>
    </div>
    <div class="grid gap-4 sm:grid-cols-2">
      <div><label for="ticket-mode" class="input-label">{{ t('admin.settings.codexTicket.mode') }}</label>
        <Select id="ticket-mode" :model-value="mode" :options="modeOptions" :disabled="locked" @update:model-value="setMode" /></div>
      <div v-if="mode === 'fixed'"><label for="ticket-fixed" class="input-label">{{ t('admin.settings.codexTicket.fixedProxy') }}</label>
        <Select id="ticket-fixed" v-model="fixedID" :options="proxyOptions" :disabled="locked || !proxies.length" /></div>
    </div>
    <p class="input-hint">{{ t(mode === 'rotate' ? 'admin.settings.codexTicket.rotateHint' : 'admin.settings.codexTicket.fixedHint') }}</p>
    <p class="input-hint">{{ t('admin.settings.codexTicket.ipHint') }}</p>
    <div class="space-y-3">
      <div v-for="(proxy, index) in proxies" :key="proxy.id" class="border-2 border-[color:var(--bh-ink)] p-3" data-testid="ticket-proxy-row">
        <div class="mb-3 flex items-center justify-between gap-3">
          <span class="font-mono text-sm font-bold text-bh-blue dark:text-blue-300">{{ t('admin.settings.codexTicket.proxyLabel', { index: String(index + 1).padStart(2, '0') }) }}</span>
          <button type="button" class="btn btn-danger btn-sm" :disabled="locked" data-testid="ticket-remove" @click="removing = proxy.id">{{ t('common.delete') }}</button>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <div><label :for="'ticket-name-' + proxy.id" class="input-label">{{ t('admin.settings.codexTicket.proxyName') }}</label>
            <input :id="'ticket-name-' + proxy.id" v-model="proxy.name" class="input w-full" maxlength="64" :disabled="locked" data-testid="ticket-proxy-name" /></div>
          <div><label :for="'ticket-url-' + proxy.id" class="input-label">{{ t('admin.settings.codexTicket.proxy') }}</label>
            <input :id="'ticket-url-' + proxy.id" v-model="proxy.url" type="password" class="input w-full font-mono" autocomplete="new-password" :disabled="locked" :placeholder="proxy.configured ? t('admin.settings.codexTicket.keepAddress') : 'socks5h://user:password@proxy.example:1080'" />
          </div>
        </div>
        <p class="mt-2 input-hint">{{ t(proxy.configured ? 'admin.settings.codexTicket.configured' : 'admin.settings.codexTicket.notConfigured') }}</p>
      </div>
      <button type="button" class="btn btn-primary" :disabled="locked || proxies.length >= 20" data-testid="ticket-add" @click="addProxy">{{ t('admin.settings.codexTicket.addProxy') }}</button>
    </div>
    <div class="grid gap-3 border-t-2 border-[color:var(--bh-ink)] pt-4 sm:grid-cols-2 lg:grid-cols-4">
      <div><label for="ticket-length" class="input-label">{{ t('admin.settings.codexTicket.targetLength') }}</label>
        <input id="ticket-length" v-model.number="targetLength" type="number" min="6" max="8192" step="1" class="input w-full font-bold text-bh-blue dark:text-blue-300" :disabled="locked" /></div>
      <div><label for="ticket-signal" class="input-label">{{ t('admin.settings.codexTicket.signalLength') }}</label>
        <input id="ticket-signal" v-model.number="signalLength" type="number" min="0" max="8192" step="1" class="input w-full font-bold text-bh-red dark:text-red-400" :disabled="locked" />
        <p class="input-hint">{{ t('admin.settings.codexTicket.signalHint') }}</p></div>
      <div><label for="ticket-attempts" class="input-label">{{ t('admin.settings.codexTicket.attempts') }}</label>
        <input id="ticket-attempts" v-model.number="attempts" type="number" min="1" step="1" class="input w-full font-bold text-bh-blue dark:text-blue-300" :disabled="locked" /></div>
      <div><label for="ticket-retry" class="input-label">{{ t('admin.settings.codexTicket.retryInterval') }}</label>
        <input id="ticket-retry" v-model.number="retryInterval" type="number" min="1" max="30" step="1" class="input w-full" :disabled="locked" /></div>
      <div><label for="ticket-interval" class="input-label">{{ t('admin.settings.codexTicket.interval') }}</label>
        <input id="ticket-interval" v-model.number="interval" type="number" min="6" max="3600" step="1" class="input w-full" :disabled="locked" /></div>
    </div>
    <p v-if="error || loadError" class="break-words text-sm text-bh-red dark:text-red-400" role="alert">{{ error || loadError }}</p>
    <div class="flex justify-end gap-2">
      <button v-if="loadError" type="button" class="btn btn-secondary" :disabled="loading" @click="load">{{ t('common.retry') }}</button>
      <button type="button" class="btn btn-primary" :disabled="locked" data-testid="codex-ticket-save" @click="save">{{ t('admin.settings.codexTicket.save') }}</button>
    </div>
    <ConfirmDialog :show="!!removing" :title="t('admin.settings.codexTicket.removeTitle')" :message="t('admin.settings.codexTicket.removeHint')" danger :loading="saving" @cancel="removing = ''" @confirm="removeProxy" />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import Select from '@/components/common/Select.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import BauhausHelp from '@/components/common/BauhausHelp.vue'
import { apiClient } from '@/api/client'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

interface ProxyRow { id: string; name: string; configured: boolean; url: string }
interface Settings {
  models?: string[]; degraded_signal_length?: number
  enabled: boolean; proxy_configured: boolean; proxies: Omit<ProxyRow, 'url'>[]
  selection_mode: 'fixed' | 'rotate'; fixed_proxy_id: string; revision: string
  max_attempts: number; retry_interval_seconds: number; probe_interval_seconds: number; target_length?: number
}
const { t } = useI18n()
const app = useAppStore()
const enabled = ref(false), proxies = ref<ProxyRow[]>([]), fixedID = ref(''), revision = ref('')
const mode = ref<'fixed' | 'rotate'>('fixed')
const attempts = ref(3), retryInterval = ref(1), interval = ref(6), removing = ref('')
const targetLength = ref(292)
const signalLength = ref(0), modelText = ref('gpt-6-astra\ngpt-5.6-sol')
const loading = ref(true), saving = ref(false), error = ref(''), loadError = ref('')
const locked = computed(() => loading.value || saving.value || !!loadError.value)
const proxyOptions = computed(() => proxies.value.map((p, i) => ({ value: p.id, label: p.name || `Proxy ${i + 1}` })))
const modeOptions = computed(() => [{ value: 'rotate', label: t('admin.settings.codexTicket.rotate') }, { value: 'fixed', label: t('admin.settings.codexTicket.fixed') }])
function setMode(value: unknown) {
  if (value !== 'fixed' && value !== 'rotate') return
  if (value === 'rotate' && mode.value !== value && attempts.value < 2) attempts.value = 3
  mode.value = value
}
function newID() {
  // 兼容非 HTTPS 管理入口，仅用作代理行标识，不参与认证或签名。
  const bytes = crypto.getRandomValues(new Uint8Array(16))
  bytes[6] = (bytes[6] & 15) | 64; bytes[8] = (bytes[8] & 63) | 128
  const hex = Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('')
  return [hex.slice(0, 8), hex.slice(8, 12), hex.slice(12, 16), hex.slice(16, 20), hex.slice(20)].join('-')
}
function addProxy() {
  if (locked.value || proxies.value.length >= 20) return
  const id = newID()
  proxies.value.push({ id, name: `Proxy ${proxies.value.length + 1}`, configured: false, url: '' })
  if (!fixedID.value) fixedID.value = id
}
function removeProxy() {
  if (locked.value) return
  proxies.value = proxies.value.filter(p => p.id !== removing.value)
  if (fixedID.value === removing.value) fixedID.value = proxies.value[0]?.id || ''
  removing.value = ''
}
function apply(data: Settings) {
  if (!Array.isArray(data.proxies) || !['fixed', 'rotate'].includes(data.selection_mode)) throw new Error(t('admin.settings.codexTicket.versionMismatch'))
  enabled.value = data.enabled; mode.value = data.selection_mode; revision.value = data.revision
  proxies.value = data.proxies.map(p => ({ id: p.id, name: p.name, configured: p.configured, url: '' }))
  fixedID.value = data.fixed_proxy_id; attempts.value = data.max_attempts
  targetLength.value = data.target_length || 292
  signalLength.value = data.degraded_signal_length || 0
  modelText.value = (data.models?.length ? data.models : ['gpt-6-astra', 'gpt-5.6-sol']).join('\n')
  retryInterval.value = data.retry_interval_seconds; interval.value = data.probe_interval_seconds
  removing.value = ''
}
// 独立保存且版本冲突时拒绝覆盖；密码留空表示按 ID 保留，不写本地存储。
async function load() {
  loading.value = true; loadError.value = ''; error.value = ''
  try { apply((await apiClient.get<Settings>('/admin/settings/codex-ticket')).data) }
  catch (err) { loadError.value = extractApiErrorMessage(err, t('common.error')) }
  finally { loading.value = false }
}
async function save() {
  if (locked.value) return
  const models = [...new Set(modelText.value.split(/\r?\n/).map(model => model.trim()).filter(Boolean))]
  if (!models.length || models.length > 100 || models.some(model => new TextEncoder().encode(model).length > 256 || /[\s\p{Cc}]/u.test(model)) ||
    !Number.isInteger(signalLength.value) || (signalLength.value !== 0 && (signalLength.value < 6 || signalLength.value > 8192))) {
    error.value = t('admin.settings.codexTicket.invalidModelsSignal'); return
  }
  const validNumber = (v: number, lo: number, hi: number) => Number.isInteger(v) && v >= lo && v <= hi
  if ((enabled.value && !proxies.value.length) || proxies.value.some(p => !p.name.trim() || (!p.configured && !p.url.trim())) ||
    !Number.isSafeInteger(attempts.value) || attempts.value < 1 || !validNumber(targetLength.value, 6, 8192) || !validNumber(retryInterval.value, 1, 30) || !validNumber(interval.value, 6, 3600)) {
    error.value = t('admin.settings.codexTicket.invalidForm'); return
  }
  saving.value = true; error.value = ''
  try {
    const payload = { enabled: enabled.value, revision: revision.value, selection_mode: mode.value, fixed_proxy_id: fixedID.value,
      max_attempts: attempts.value, target_length: targetLength.value, models, degraded_signal_length: signalLength.value, retry_interval_seconds: retryInterval.value, probe_interval_seconds: interval.value,
      proxies: proxies.value.map(p => ({ id: p.id, name: p.name.trim(), harvest_proxy_url: p.url.trim() })) }
    apply((await apiClient.put<Settings>('/admin/settings/codex-ticket', payload)).data)
    app.showSuccess(t('admin.settings.codexTicket.saved'))
  } catch (err) { error.value = extractApiErrorMessage(err, t('common.error')) }
  finally { saving.value = false }
}
onMounted(load)
</script>
