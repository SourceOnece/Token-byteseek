<template>
  <section class="space-y-5 border-2 border-[color:var(--bh-ink)] bg-[var(--bh-surface)] p-4 sm:p-6" style="box-shadow:var(--bh-shadow-sm)" data-testid="ticket-account-settings">
    <div class="flex flex-wrap items-baseline justify-between gap-2"><h3 class="text-xl font-extrabold text-bh-blue dark:text-blue-300">{{ t('admin.accounts.ticketWorkbench.title') }}</h3><span class="text-sm font-bold">{{ bulk ? t('admin.accounts.ticketPolicy.selected', { count: ids.length }) : t('admin.accounts.ticketPolicy.source.' + source) }}</span></div>
    <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.accounts.ticketWorkbench.accountHint') }}</p>
    <p v-if="!bulk && !globalEnabled" class="border-l-4 border-bh-yellow pl-3 text-sm font-bold text-yellow-800 dark:text-bh-yellow">{{ t('admin.accounts.ticketWorkbench.masterOff') }}</p>
    <div class="space-y-3 border-y-2 border-[color:var(--bh-ink)] py-4">
      <div class="flex items-center justify-between gap-4"><label :for="uid + '-verified-flow'" class="flex items-center gap-2 text-lg font-extrabold"><input v-if="bulk" v-model="fields.flow" type="checkbox" :disabled="locked" data-testid="ticket-edit-flow" />{{ t('admin.accounts.ticketWorkbench.verifiedFlow') }}</label><Toggle :id="uid + '-verified-flow'" v-model="verifiedFlow" :disabled="locked || !fields.flow" :aria-label="t('admin.accounts.ticketWorkbench.verifiedFlow')" data-testid="ticket-verified-flow" /></div>
      <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.accounts.ticketWorkbench.verifiedHint') }}</p>
      <p v-if="dualFlowInForm" class="border-l-4 border-bh-yellow pl-3 text-sm text-yellow-800 dark:text-bh-yellow">{{ t('admin.accounts.ticketWorkbench.verifiedRisk') }}</p>
    </div>
    <div class="grid gap-5 sm:grid-cols-2">
      <div><label :for="uid + '-mode'" class="mb-2 flex items-center gap-2 font-bold"><input v-if="bulk" v-model="fields.mode" type="checkbox" :disabled="locked" data-testid="ticket-edit-mode" />{{ t('admin.accounts.ticketPolicy.mode') }}</label><Select :id="uid + '-mode'" v-model="mode" :options="modeOptions" :disabled="locked || !fields.mode" /></div>
      <div><label :for="uid + '-guard'" class="mb-2 flex items-center gap-2 font-bold"><input v-if="bulk" v-model="fields.guard" type="checkbox" :disabled="locked || dualFlowInForm" data-testid="ticket-edit-guard" />{{ t('admin.accounts.ticketPolicy.guard') }}</label><Select :id="uid + '-guard'" v-model="displayGuard" :options="guardOptions" :disabled="locked || !fields.guard || dualFlowInForm" /></div>
    </div>
    <div class="grid grid-cols-3 gap-3 border-y-2 border-[color:var(--bh-ink)] py-4 text-center">
      <div><p class="text-xs sm:text-sm">{{ t('admin.accounts.ticketWorkbench.target') }}</p><strong class="mt-1 block text-2xl text-emerald-700 dark:text-emerald-400">{{ rules.target_length }}</strong></div>
      <div><p class="text-xs sm:text-sm">{{ t('admin.accounts.ticketWorkbench.attempts') }}</p><strong class="mt-1 block text-2xl text-bh-blue dark:text-blue-300">{{ rules.max_attempts === 0 ? '∞' : rules.max_attempts }}</strong></div>
      <div><p class="text-xs sm:text-sm">{{ t('admin.accounts.ticketWorkbench.cache') }}</p><strong class="mt-1 block text-2xl">{{ rules.cache_minutes }}<small class="ml-1 text-xs">min</small></strong></div>
    </div>
    <div>
      <label :for="uid + '-models'" class="mb-2 flex items-center gap-2 font-bold"><input v-if="bulk" v-model="ruleFields.models" type="checkbox" :disabled="locked" />{{ t('admin.settings.codexTicket.models') }}</label>
      <textarea :id="uid + '-models'" v-model="modelText" rows="3" class="input w-full font-mono" :disabled="locked || !ruleFields.models" />
    </div>
    <div v-for="group in groups" :key="group.name" class="space-y-4 border-t-2 border-[color:var(--bh-ink)] pt-4">
      <h4 class="text-lg font-extrabold text-bh-blue dark:text-blue-300">{{ t('admin.accounts.ticketWorkbench.groups.' + group.name) }}</h4>
      <div class="grid items-start gap-4 sm:grid-cols-2">
        <div v-for="field in group.fields" :key="field.key">
          <label :for="uid + '-' + field.key" class="input-label flex items-center gap-2"><input v-if="bulk" v-model="ruleFields[field.key]" type="checkbox" :disabled="locked" />{{ t('admin.accounts.ticketWorkbench.fields.' + field.key) }}</label>
          <input :id="uid + '-' + field.key" v-model.number="rules[field.key]" type="number" :min="field.min" :max="field.max" step="1" class="input w-full font-bold" :disabled="locked || !ruleFields[field.key]" :data-testid="'ticket-rule-' + field.key" />
        </div>
      </div>
    </div>
    <p v-if="rules.max_attempts === 0" class="border-l-4 border-bh-yellow pl-3 text-sm text-yellow-800 dark:text-bh-yellow">{{ t('admin.accounts.ticketWorkbench.unlimitedRisk') }}</p>
    <div class="space-y-4 border-t-2 border-[color:var(--bh-ink)] pt-4">
      <label class="flex items-center gap-2 text-lg font-extrabold"><input v-model="fields.proxy" type="checkbox" :disabled="locked" data-testid="ticket-edit-proxy" />{{ t('admin.accounts.ticketPolicy.proxy') }}</label>
      <CodexTicketProxyEditor ref="proxyEditor" :value="policy" :account-id="ids[0]" allow-inherit :inherited="source === 'gateway'" :locked="locked || !fields.proxy" :test-disabled="locked" />
    </div>
    <p v-if="guard.startsWith('recover') && fields.guard" class="border-l-4 border-bh-yellow pl-3 text-sm text-yellow-800 dark:text-bh-yellow">{{ t('admin.accounts.ticketPolicy.guardRisk') }}</p>
    <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.accounts.ticketPolicy.saveHint') }}</p>
    <p v-if="error" class="break-words text-sm text-bh-red dark:text-red-400" role="alert">{{ error }}</p>
    <p v-if="saved" class="text-sm font-semibold text-emerald-700 dark:text-emerald-400" role="status">{{ t('admin.accounts.ticketPolicy.saved') }}</p>
    <div class="flex justify-end gap-2"><button v-if="loadFailed" type="button" class="btn btn-secondary" :disabled="loading" @click="load">{{ t('common.retry') }}</button><button type="button" class="btn btn-primary" :disabled="locked || !hasChanges || ids.length > 500" data-testid="ticket-account-save" @click="save">{{ t('admin.accounts.ticketPolicy.save') }}</button></div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import CodexTicketProxyEditor from './CodexTicketProxyEditor.vue'
import { ticketAccountAPI, type TicketAccountPatch, type TicketAccountSettings, type TicketRules, type TicketProxyPolicy } from '@/api/admin/codexTickets'
import { extractApiErrorMessage } from '@/utils/apiError'
// 单号显示有效规则完整快照；批量每一项单独勾选，未勾选不提交。
const props = defineProps<{ ids: number[]; bulk?: boolean }>()
const emit = defineEmits<{ saved: [] }>()
const { t } = useI18n(), uid = useId()
const defaults = (): TicketRules => ({ models: ['gpt-6-astra', 'gpt-5.6-sol'], target_length: 292, degraded_signal_length: 0, max_attempts: 1, concurrency: 4, cache_minutes: 60, refresh_before_minutes: 10, retry_interval_seconds: 1, probe_interval_seconds: 6, failure_threshold: 0, cooldown_seconds: 300 })
type NumberRule = Exclude<keyof TicketRules, 'models'>
const groups: { name: string; fields: { key: NumberRule; min: number; max?: number }[] }[] = [
  { name: 'validation', fields: [{ key: 'target_length', min: 6, max: 8192 }, { key: 'degraded_signal_length', min: 0, max: 8192 }, { key: 'max_attempts', min: 0 }, { key: 'retry_interval_seconds', min: 1, max: 30 }] },
  { name: 'renewal', fields: [{ key: 'concurrency', min: 1, max: 4 }, { key: 'cache_minutes', min: 1, max: 1440 }, { key: 'refresh_before_minutes', min: 0, max: 1439 }, { key: 'probe_interval_seconds', min: 6, max: 3600 }] },
  { name: 'cooldown', fields: [{ key: 'failure_threshold', min: 0, max: 100000 }, { key: 'cooldown_seconds', min: 1, max: 86400 }] }
]
const rules = reactive<TicketRules>(defaults()), modelText = ref(defaults().models.join('\n'))
const ruleFields = reactive(Object.fromEntries(Object.keys(defaults()).map(k => [k, !props.bulk])) as Record<keyof TicketRules, boolean>)
const fields = reactive({ mode: !props.bulk, guard: !props.bulk, proxy: false, flow: !props.bulk })
const verifiedFlow = ref(false)
const mode = ref<TicketAccountSettings['mode']>('inherit'), guard = ref<TicketAccountSettings['watchdog_mode']>('observe')
const dualFlowInForm = computed(() => verifiedFlow.value && (!props.bulk || fields.flow))
// 显示新模式的有效守护，但不覆盖关闭开关后要恢复的旧选择。
const displayGuard = computed({ get: () => dualFlowInForm.value ? 'recover' : guard.value, set: (value: string) => { if (!dualFlowInForm.value) guard.value = value as TicketAccountSettings['watchdog_mode'] } })
const revision = ref(''), source = ref('gateway'), policy = ref<TicketProxyPolicy>()
const globalEnabled = ref(true)
const proxyEditor = ref<InstanceType<typeof CodexTicketProxyEditor>>()
const loading = ref(false), saving = ref(false), loadFailed = ref(false), error = ref(''), saved = ref(false)
const locked = computed(() => loading.value || saving.value || loadFailed.value)
const hasChanges = computed(() => Object.values(fields).some(Boolean) || Object.values(ruleFields).some(Boolean))
const modeOptions = computed(() => ['on', 'off'].map(value => ({ value, label: t('admin.accounts.ticketPolicy.modes.' + value) })))
const guardOptions = computed(() => ['off', 'observe', 'recover_length', 'recover_model', 'recover'].map(value => ({ value, label: t('admin.accounts.ticketPolicy.guards.' + value) })))
let controller: AbortController | undefined, sequence = 0
async function load() {
  controller?.abort(); controller = new AbortController(); const current = ++sequence
  fields.mode = !props.bulk; fields.guard = !props.bulk; fields.proxy = false; fields.flow = !props.bulk; verifiedFlow.value = false
  for (const key of Object.keys(ruleFields) as (keyof TicketRules)[]) ruleFields[key] = !props.bulk
  Object.assign(rules, defaults()); modelText.value = rules.models.join('\n'); mode.value = 'on'; guard.value = 'observe'
  revision.value = ''; source.value = 'gateway'; policy.value = undefined; saved.value = false; loadFailed.value = false; error.value = ''; loading.value = !props.bulk
  if (props.bulk) return
  try {
    const data = await ticketAccountAPI.get(props.ids[0], controller.signal); if (current !== sequence) return
    if (!data.rules || !Array.isArray(data.rules.models) || !data.proxy_policy || typeof data.verified_flow !== 'boolean') throw new Error(t('admin.settings.codexTicket.versionMismatch'))
    verifiedFlow.value = data.verified_flow
    globalEnabled.value = data.global_enabled !== false
    mode.value = data.mode === 'off' ? 'off' : 'on'; guard.value = verifiedFlow.value ? (data.watchdog_mode === 'inherit' ? 'observe' : data.watchdog_mode) : data.effective_watchdog_mode || 'observe'; revision.value = data.revision; source.value = data.proxy_source
    Object.assign(rules, data.rules || defaults()); modelText.value = rules.models.join('\n'); policy.value = data.proxy_policy
  } catch (err) { if (current === sequence && !controller.signal.aborted) { error.value = extractApiErrorMessage(err, t('common.error')); loadFailed.value = true } }
  finally { if (current === sequence) loading.value = false }
}
async function save() {
  if (locked.value || !hasChanges.value) return
  const patch: TicketAccountPatch = {}, rulePatch: Partial<TicketRules> = {}
  if (fields.mode) patch.mode = mode.value
  if (fields.guard && !dualFlowInForm.value) patch.watchdog_mode = guard.value
  if (fields.flow) patch.verified_flow = verifiedFlow.value
  for (const key of Object.keys(ruleFields) as (keyof TicketRules)[]) {
    if (!ruleFields[key]) continue
    if (key === 'models') rulePatch.models = [...new Set(modelText.value.split(/\r?\n/).map(v => v.trim()).filter(Boolean))]
    else { const value = rules[key]; if (!Number.isSafeInteger(value)) { error.value = t('admin.settings.codexTicket.invalidForm'); return } rulePatch[key] = value }
  }
  if (Object.keys(rulePatch).length) patch.rules = rulePatch
  try { if (fields.proxy) patch.proxy_policy = proxyEditor.value?.patch() } catch (err) { error.value = extractApiErrorMessage(err, t('common.error')); return }
  const current = sequence; saving.value = true; error.value = ''; saved.value = false
  try {
    const result = await ticketAccountAPI.update([...props.ids], patch, props.bulk ? undefined : revision.value); if (current !== sequence) return
    fields.proxy = false
    if (props.bulk) { fields.mode = false; fields.guard = false; fields.flow = false; for (const key of Object.keys(ruleFields) as (keyof TicketRules)[]) ruleFields[key] = false }
    if (result[0]) { revision.value = result[0].revision; source.value = result[0].proxy_source; policy.value = result[0].proxy_policy }
    saved.value = true; emit('saved')
  } catch (err) { if (current === sequence) error.value = extractApiErrorMessage(err, t('common.error')) }
  finally { saving.value = false }
}
watch(() => props.ids.join(',') + '/' + !!props.bulk, load, { immediate: true })
onBeforeUnmount(() => { sequence++; controller?.abort() })
</script>
