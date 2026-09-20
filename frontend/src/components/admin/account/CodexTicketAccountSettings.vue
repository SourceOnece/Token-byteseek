<template>
  <section
    :class="bulk ? 'space-y-5 border-t border-gray-200 pt-4 dark:border-dark-600' : 'space-y-5 border-2 border-[color:var(--bh-ink)] bg-[var(--bh-surface)] p-4 sm:p-6'"
    :style="bulk ? undefined : { boxShadow: 'var(--bh-shadow-sm)' }"
    data-testid="ticket-account-settings"
  >
    <!-- 批量入口沿原表单：标题左、勾选右；编辑区虚化且真实禁用，勾选后恢复。 -->
    <template v-if="bulk">
      <div>
        <h3 class="input-label mb-0">{{ t('admin.accounts.ticketWorkbench.title') }}</h3>
        <p class="input-hint">{{ t('admin.accounts.ticketPolicy.selected', { count: ids.length }) }}</p>
      </div>
      <div v-for="item in bulkChoices" :key="item.key" class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="mb-3 flex items-center justify-between gap-4">
          <label :id="uid + '-bulk-' + item.key + '-label'" :for="uid + '-bulk-' + item.key + '-enabled'" class="input-label mb-0">{{ t(item.label) }}</label>
          <input :id="uid + '-bulk-' + item.key + '-enabled'" v-model="fields[item.key]" type="checkbox" :disabled="locked || (item.key === 'guard' && dualFlowInForm)" :aria-controls="uid + '-bulk-' + item.key" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" :data-testid="'ticket-edit-' + item.key" />
        </div>
        <div :id="uid + '-bulk-' + item.key" role="group" :aria-labelledby="uid + '-bulk-' + item.key + '-label'" :class="!fields[item.key] && 'pointer-events-none opacity-50'" :data-testid="'ticket-bulk-body-' + item.key">
          <Select v-if="item.key === 'flow'" v-model="bulkFlow" :options="flowOptions" :disabled="locked || !fields.flow" :placeholder="' '" />
          <Select v-else-if="item.key === 'mode'" v-model="mode" :options="modeOptions" :disabled="locked || !fields.mode" :placeholder="' '" />
          <Select v-else v-model="displayGuard" :options="guardOptions" :disabled="locked || !fields.guard || dualFlowInForm" :placeholder="' '" />
        </div>
        <p v-if="item.key === 'flow' && dualFlowInForm" class="input-hint text-yellow-800 dark:text-bh-yellow">{{ t('admin.accounts.ticketWorkbench.verifiedRisk') }}</p>
      </div>
      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="mb-3 flex items-center justify-between gap-4">
          <label :id="uid + '-bulk-models-label'" :for="uid + '-bulk-models-enabled'" class="input-label mb-0">{{ t('admin.settings.codexTicket.models') }}</label>
          <input :id="uid + '-bulk-models-enabled'" v-model="ruleFields.models" type="checkbox" :disabled="locked" :aria-controls="uid + '-models'" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" data-testid="ticket-edit-models" />
        </div>
        <textarea :id="uid + '-models'" v-model="modelText" rows="3" class="input w-full font-mono" :disabled="locked || !ruleFields.models" :class="!ruleFields.models && 'cursor-not-allowed opacity-50'" :aria-labelledby="uid + '-bulk-models-label'" data-testid="ticket-bulk-models" />
      </div>
      <div v-for="field in bulkNumberFields" :key="field.key" class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="mb-3 flex items-center justify-between gap-4">
          <label :id="uid + '-bulk-' + field.key + '-label'" :for="uid + '-bulk-' + field.key + '-enabled'" class="input-label mb-0">{{ t('admin.accounts.ticketWorkbench.fields.' + field.key) }}</label>
          <input :id="uid + '-bulk-' + field.key + '-enabled'" v-model="ruleFields[field.key]" type="checkbox" :disabled="locked" :aria-controls="uid + '-' + field.key" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" :data-testid="'ticket-edit-' + field.key" />
        </div>
        <input :id="uid + '-' + field.key" v-model.number="rules[field.key]" type="number" :min="field.min" :max="field.max" step="1" class="input w-full" :disabled="locked || !ruleFields[field.key]" :class="!ruleFields[field.key] && 'cursor-not-allowed opacity-50'" :aria-labelledby="uid + '-bulk-' + field.key + '-label'" :data-testid="'ticket-rule-' + field.key" />
      </div>
      <p v-if="ruleFields.max_attempts && rules.max_attempts === 0" class="input-hint text-yellow-800 dark:text-bh-yellow">{{ t('admin.accounts.ticketWorkbench.unlimitedRisk') }}</p>
      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="mb-3 flex items-center justify-between gap-4">
          <label :id="uid + '-bulk-proxy-label'" :for="uid + '-bulk-proxy-enabled'" class="input-label mb-0">{{ t('admin.accounts.ticketPolicy.proxy') }}</label>
          <input :id="uid + '-bulk-proxy-enabled'" v-model="fields.proxy" type="checkbox" :disabled="locked" :aria-controls="uid + '-bulk-proxy'" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" data-testid="ticket-edit-proxy" />
        </div>
        <div :id="uid + '-bulk-proxy'" role="group" :aria-labelledby="uid + '-bulk-proxy-label'" :class="!fields.proxy && 'pointer-events-none opacity-50'" data-testid="ticket-bulk-body-proxy">
          <CodexTicketProxyEditor ref="proxyEditor" :value="policy" :blank="proxyMixed" :account-id="ids[0]" allow-inherit :inherited="source === 'gateway'" :locked="locked || !fields.proxy" :test-disabled="locked || !fields.proxy" />
        </div>
      </div>
    </template>
    <template v-else>
    <div class="flex flex-wrap items-baseline justify-between gap-2"><h3 class="text-xl font-extrabold text-bh-blue dark:text-blue-300">{{ t('admin.accounts.ticketWorkbench.title') }}</h3><span class="text-sm font-bold">{{ bulk ? t('admin.accounts.ticketPolicy.selected', { count: ids.length }) : source ? t('admin.accounts.ticketPolicy.source.' + source) : '' }}</span></div>
    <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.accounts.ticketWorkbench.accountHint') }}</p>
    <p v-if="!bulk && !globalEnabled" class="border-l-4 border-bh-yellow pl-3 text-sm font-bold text-yellow-800 dark:text-bh-yellow">{{ t('admin.accounts.ticketWorkbench.masterOff') }}</p>
    <div class="space-y-3 border-y-2 border-[color:var(--bh-ink)] py-4">
      <div class="flex items-center justify-between gap-4"><label :for="uid + '-verified-flow'" class="flex items-center gap-2 text-lg font-extrabold"><input v-if="bulk" v-model="fields.flow" type="checkbox" :disabled="locked" data-testid="ticket-edit-flow" />{{ t('admin.accounts.ticketWorkbench.verifiedFlow') }}</label><Select v-if="bulk" v-model="bulkFlow" :options="flowOptions" :disabled="locked || !fields.flow" :placeholder="' '" class="w-36" /><Toggle v-else :id="uid + '-verified-flow'" v-model="verifiedFlow" :disabled="locked || !fields.flow" :aria-label="t('admin.accounts.ticketWorkbench.verifiedFlow')" data-testid="ticket-verified-flow" /></div>
      <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.accounts.ticketWorkbench.verifiedHint') }}</p>
      <p v-if="dualFlowInForm" class="border-l-4 border-bh-yellow pl-3 text-sm text-yellow-800 dark:text-bh-yellow">{{ t('admin.accounts.ticketWorkbench.verifiedRisk') }}</p>
    </div>
    <div class="grid gap-5 sm:grid-cols-2">
      <div><label :for="uid + '-mode'" class="mb-2 flex items-center gap-2 font-bold"><input v-if="bulk" v-model="fields.mode" type="checkbox" :disabled="locked" data-testid="ticket-edit-mode" />{{ t('admin.accounts.ticketPolicy.mode') }}</label><Select :id="uid + '-mode'" v-model="mode" :options="modeOptions" :placeholder="' '" :disabled="locked || !fields.mode" /></div>
      <div><label :for="uid + '-guard'" class="mb-2 flex items-center gap-2 font-bold"><input v-if="bulk" v-model="fields.guard" type="checkbox" :disabled="locked || dualFlowInForm" data-testid="ticket-edit-guard" />{{ t('admin.accounts.ticketPolicy.guard') }}</label><Select :id="uid + '-guard'" v-model="displayGuard" :options="guardOptions" :placeholder="' '" :disabled="locked || !fields.guard || dualFlowInForm" /></div>
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
      <CodexTicketProxyEditor ref="proxyEditor" :value="policy" :blank="proxyMixed" :template-source="templateMode || draft" :account-id="ids[0]" allow-inherit :inherited="source === 'gateway'" :locked="locked || !fields.proxy" :test-disabled="locked" />
    </div>
    </template>
    <p v-if="guard.startsWith('recover') && fields.guard" class="border-l-4 border-bh-yellow pl-3 text-sm text-yellow-800 dark:text-bh-yellow">{{ t('admin.accounts.ticketPolicy.guardRisk') }}</p>
    <p class="text-sm text-gray-600 dark:text-gray-300">{{ t(draft ? 'admin.accounts.ticketWorkbench.createHint' : templateMode ? 'admin.accounts.ticketWorkbench.templateHint' : 'admin.accounts.ticketPolicy.saveHint') }}</p>
    <p v-if="error" class="break-words text-sm text-bh-red dark:text-red-400" role="alert">{{ error }}</p>
    <p v-if="saved" class="text-sm font-semibold text-emerald-700 dark:text-emerald-400" role="status">{{ t('admin.accounts.ticketPolicy.saved') }}</p>
    <div v-if="!draft || loadFailed" class="flex justify-end gap-2"><button v-if="loadFailed" type="button" class="btn btn-secondary" :disabled="loading" @click="load">{{ t('common.retry') }}</button><button v-if="!draft" type="button" class="btn btn-primary" :disabled="locked || !hasChanges || ids.length > 500" data-testid="ticket-account-save" @click="save">{{ t('admin.accounts.ticketPolicy.save') }}</button></div>
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
const props = withDefaults(defineProps<{ ids?: number[]; bulk?: boolean; templateMode?: boolean; draft?: boolean }>(), { ids: () => [] })
const emit = defineEmits<{ saved: [] }>()
const { t } = useI18n(), uid = useId()
const defaults = (): TicketRules => ({ models: ['gpt-6-astra', 'gpt-5.6-sol'], target_length: 292, degraded_signal_length: 0, max_attempts: 1, concurrency: 4, cache_minutes: 60, refresh_before_minutes: 10, retry_interval_seconds: 1, probe_interval_seconds: 6, failure_threshold: 0, cooldown_seconds: 300 })
type NumberRule = Exclude<keyof TicketRules, 'models'>
const groups: { name: string; fields: { key: NumberRule; min: number; max?: number }[] }[] = [
  { name: 'validation', fields: [{ key: 'target_length', min: 6, max: 8192 }, { key: 'degraded_signal_length', min: 0, max: 8192 }, { key: 'max_attempts', min: 0 }, { key: 'retry_interval_seconds', min: 1, max: 30 }] },
  { name: 'renewal', fields: [{ key: 'concurrency', min: 1, max: 4 }, { key: 'cache_minutes', min: 1, max: 1440 }, { key: 'refresh_before_minutes', min: 0, max: 1439 }, { key: 'probe_interval_seconds', min: 6, max: 3600 }] },
  { name: 'cooldown', fields: [{ key: 'failure_threshold', min: 0, max: 100000 }, { key: 'cooldown_seconds', min: 1, max: 86400 }] }
]
const bulkNumberFields = groups.flatMap(group => group.fields)
const bulkChoices = [
  { key: 'flow', label: 'admin.accounts.ticketWorkbench.verifiedFlow' },
  { key: 'mode', label: 'admin.accounts.ticketPolicy.mode' },
  { key: 'guard', label: 'admin.accounts.ticketPolicy.guard' }
] as const
const rules = reactive<{ [K in keyof TicketRules]: TicketRules[K] | '' }>(defaults()), modelText = ref(defaults().models.join('\n'))
const ruleFields = reactive(Object.fromEntries(Object.keys(defaults()).map(k => [k, !props.bulk])) as Record<keyof TicketRules, boolean>)
const fields = reactive({ mode: !props.bulk, guard: !props.bulk, proxy: false, flow: !props.bulk })
const verifiedFlow = ref(false)
const mixedFlow = ref(false), proxyMixed = ref(false)
const bulkFlow = computed({ get: () => mixedFlow.value ? '' : String(verifiedFlow.value), set: (v: string) => { mixedFlow.value = v === ''; verifiedFlow.value = v === 'true' } })
const flowOptions = computed(() => [{ value: 'true', label: t('common.enabled') }, { value: 'false', label: t('common.disabled') }])
const mode = ref<TicketAccountSettings['mode'] | ''>('inherit'), guard = ref<TicketAccountSettings['watchdog_mode'] | ''>('observe')
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
const guardOptions = computed(() => ['inherit', 'off', 'observe', 'recover_length', 'recover_model', 'recover'].map(value => ({ value, label: t('admin.accounts.ticketPolicy.guards.' + value) })))
let controller: AbortController | undefined, sequence = 0
let readyPromise = Promise.resolve()
function load() {
  // 每次读取绑定独立Promise，旧请求的finally不能提前放行新账号创建。
  readyPromise = loadData()
  return readyPromise
}
async function loadData() {
  controller?.abort(); controller = new AbortController(); const current = ++sequence
  fields.mode = !props.bulk; fields.guard = !props.bulk; fields.proxy = false; fields.flow = !props.bulk; verifiedFlow.value = false
  for (const key of Object.keys(ruleFields) as (keyof TicketRules)[]) ruleFields[key] = !props.bulk
  Object.assign(rules, defaults()); modelText.value = Array.isArray(rules.models) ? rules.models.join('\n') : ''; mode.value = 'on'; guard.value = 'observe'
  revision.value = ''; source.value = 'gateway'; policy.value = undefined; saved.value = false; loadFailed.value = false; error.value = ''; loading.value = true; mixedFlow.value = false; proxyMixed.value = false
  if (props.bulk) {for(const key of Object.keys(ruleFields) as (keyof TicketRules)[])rules[key]='' as never;modelText.value='';mode.value='';guard.value='';mixedFlow.value=false;proxyMixed.value=true}
  try {
    const rows: TicketAccountSettings[] = []
    if (props.bulk) {
      // 小批并发读取，避免500号同时请求；整批读完才允许修改，不用部分结果冒充共同值。
      for (let i=0; i<props.ids.length; i+=6) {
        rows.push(...await Promise.all(props.ids.slice(i,i+6).map(id=>ticketAccountAPI.get(id,controller!.signal))))
        if (current !== sequence) return
      }
    } else rows.push(props.templateMode || props.draft ? await ticketAccountAPI.defaults(controller.signal) : await ticketAccountAPI.get(props.ids[0], controller.signal))
    if (current !== sequence) return
    const data = rows[0]
    if (!data) throw new Error(t('common.error'))
    if (!data.rules || !Array.isArray(data.rules.models) || !data.proxy_policy || typeof data.verified_flow !== 'boolean') throw new Error(t('admin.settings.codexTicket.versionMismatch'))
    verifiedFlow.value = data.verified_flow
    globalEnabled.value = data.global_enabled !== false
    // 保存原选择，双链路强制守护只用于显示；关闭开关不能把继承关系写成observe。
    mode.value = data.mode === 'off' ? 'off' : 'on'; guard.value = data.watchdog_mode || 'inherit'; revision.value = data.revision; source.value = data.proxy_source
    Object.assign(rules, data.rules || defaults()); modelText.value = Array.isArray(rules.models) ? rules.models.join('\n') : ''; policy.value = data.proxy_policy
    if (props.bulk) {
      const common = (get: (row: TicketAccountSettings) => unknown) => rows.every(row=>JSON.stringify(get(row))===JSON.stringify(get(data)))
      mixedFlow.value=!common(row=>row.verified_flow)
      if (!common(row=>row.mode==='off'?'off':'on')) mode.value=''
      if (!common(row=>row.watchdog_mode)) guard.value=''
      for (const key of Object.keys(ruleFields) as (keyof TicketRules)[]) if (!common(row=>row.rules[key])) rules[key]='' as never
      modelText.value=Array.isArray(rules.models)?rules.models.join('\n'):''
      proxyMixed.value=!common(row=>[row.proxy_source,row.proxy_policy])
      if(proxyMixed.value) {policy.value=undefined;source.value=''}
    }
  } catch (err) { if (current === sequence && !controller.signal.aborted) { error.value = extractApiErrorMessage(err, t('common.error')); loadFailed.value = true } }
  finally { if (current === sequence) loading.value = false }
}
function patch(): TicketAccountPatch {
  if (locked.value) throw new Error(t('common.loading'))
  const patch: TicketAccountPatch = {}, rulePatch: Partial<TicketRules> = {}
  if (fields.mode) { if(!mode.value) throw new Error(t('admin.settings.codexTicket.invalidForm')); patch.mode = mode.value }
  if (fields.guard && !dualFlowInForm.value) { if(!guard.value) throw new Error(t('admin.settings.codexTicket.invalidForm')); patch.watchdog_mode = guard.value }
  if (fields.flow) { if(mixedFlow.value) throw new Error(t('admin.settings.codexTicket.invalidForm')); patch.verified_flow = verifiedFlow.value }
  for (const key of Object.keys(ruleFields) as (keyof TicketRules)[]) {
    if (!ruleFields[key]) continue
    if (key === 'models') rulePatch.models = [...new Set(modelText.value.split(/\r?\n/).map(v => v.trim()).filter(Boolean))]
    else { const value = rules[key]; if (typeof value!=='number' || !Number.isSafeInteger(value)) throw new Error(t('admin.settings.codexTicket.invalidForm')); rulePatch[key] = value }
  }
  if (Object.keys(rulePatch).length) patch.rules = rulePatch
  if (fields.proxy) patch.proxy_policy = proxyEditor.value?.patch()
  return patch
}
async function save() {
  if (locked.value || !hasChanges.value) return
  let value: TicketAccountPatch
  try { value=patch() } catch(err) { error.value=extractApiErrorMessage(err,t('common.error'));return }
  const current = sequence; saving.value = true; error.value = ''; saved.value = false
  try {
    const result = props.templateMode ? [await ticketAccountAPI.updateDefaults(value,revision.value)] : await ticketAccountAPI.update([...props.ids], value, props.bulk ? undefined : revision.value); if (current !== sequence) return
    fields.proxy = false
    if (props.bulk) { fields.mode = false; fields.guard = false; fields.flow = false; for (const key of Object.keys(ruleFields) as (keyof TicketRules)[]) ruleFields[key] = false }
    if (result[0]) { revision.value = result[0].revision; source.value = result[0].proxy_source; policy.value = result[0].proxy_policy }
    saved.value = true; emit('saved')
  } catch (err) { if (current === sequence) error.value = extractApiErrorMessage(err, t('common.error')) }
  finally { saving.value = false }
}
watch(() => props.ids.join(',') + '/' + !!props.bulk + '/' + !!props.templateMode + '/' + !!props.draft, load, { immediate: true })
onBeforeUnmount(() => { sequence++; controller?.abort(); loadFailed.value = true })
defineExpose({ patch, ensureReady: async () => { await readyPromise } })
</script>
