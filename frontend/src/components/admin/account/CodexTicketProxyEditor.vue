<template>
  <div class="space-y-4" data-testid="ticket-proxy-editor">
    <div class="grid gap-4 sm:grid-cols-2">
      <div><label :for="uid + '-mode'" class="input-label">{{ t('admin.accounts.ticketWorkbench.proxyMode') }}</label><Select :id="uid + '-mode'" v-model="mode" :options="modeOptions" :disabled="locked || testing" /></div>
      <div v-if="mode === 'dynamic'"><label :for="uid + '-source'" class="input-label">{{ t('admin.accounts.ticketWorkbench.dynamicSource') }}</label><Select :id="uid + '-source'" v-model="source" :options="sourceOptions" :disabled="locked || testing" /></div>
    </div>
    <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.accounts.ticketWorkbench.proxyHints.' + mode) }}</p>
    <template v-if="mode !== 'inherit'">
      <div v-if="mode === 'dynamic' && source === 'api'" class="grid gap-4 sm:grid-cols-[1fr_9rem]">
        <div><label :for="uid + '-api'" class="input-label">{{ t('admin.accounts.ticketWorkbench.extractionURL') }}</label><input :id="uid + '-api'" v-model="extractionURL" type="password" autocomplete="new-password" class="input w-full font-mono" :disabled="locked || testing" :placeholder="value?.extraction_configured ? t('admin.settings.codexTicket.keepAddress') : 'https://provider.example/v1/gen?...'" data-testid="ticket-extraction-url" /></div>
        <div><label :for="uid + '-protocol'" class="input-label">{{ t('admin.accounts.ticketWorkbench.proxyProtocol') }}</label><Select :id="uid + '-protocol'" v-model="protocol" :options="protocolOptions" :disabled="locked || testing" /></div>
        <p class="text-sm text-gray-600 dark:text-gray-300 sm:col-span-2">{{ t('admin.accounts.ticketWorkbench.apiHint') }}</p>
      </div>
      <template v-else>
        <div v-for="(row, index) in rows" :key="row.id" class="space-y-2 border-l-4 border-bh-blue pl-3">
          <div class="flex items-center gap-2"><input v-model="row.name" class="input min-w-0 flex-1" :aria-label="t('admin.settings.codexTicket.proxyName')" :disabled="locked || testing" maxlength="64" /><button type="button" class="btn btn-danger btn-sm" :disabled="locked || testing" @click="remove(index)">{{ t('common.delete') }}</button></div>
          <input v-model="row.url" type="password" autocomplete="new-password" class="input w-full font-mono" :aria-label="t('admin.accounts.ticketPolicy.proxy')" :disabled="locked || testing" :placeholder="row.configured ? t('admin.settings.codexTicket.keepAddress') : 'http://user:password@proxy.example:8080'" data-testid="ticket-proxy-url" />
        </div>
        <div class="flex flex-wrap items-center gap-3"><button type="button" class="btn btn-secondary" :disabled="locked || testing || rows.length >= 20" @click="add">{{ t('admin.settings.codexTicket.addProxy') }}</button><Select v-if="mode === 'fixed' && rows.length > 1" v-model="fixedID" :options="rows.map(p => ({ value: p.id, label: p.name }))" :disabled="locked || testing" /></div>
        <p v-if="mode === 'dynamic'" class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.accounts.ticketPolicy.proxyHint', { sid: '{sid}', random: '{random}' }) }}</p>
      </template>
    </template>
    <div class="flex flex-wrap items-center gap-3 border-t-2 border-[color:var(--bh-ink)] pt-4">
      <button type="button" class="btn btn-primary" :disabled="testDisabled || testing" data-testid="ticket-proxy-test" @click="test">{{ t(testing ? 'admin.accounts.ticketWorkbench.testing' : 'admin.accounts.ticketWorkbench.testProxy') }}</button>
      <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.accounts.ticketWorkbench.testHint') }}</span>
    </div>
    <p v-if="error" class="break-words text-sm text-bh-red dark:text-red-400" role="alert">{{ error }}</p>
    <div v-if="result" class="grid grid-cols-2 gap-3 border-2 border-current p-4" :class="result.status === 'reference' ? 'text-emerald-700 dark:text-emerald-400' : 'text-yellow-700 dark:text-bh-yellow'" data-testid="ticket-proxy-test-result">
      <div class="col-span-2 text-lg font-extrabold">{{ t(result.status === 'reference' ? 'admin.accounts.ticketWorkbench.connected' : 'admin.accounts.ticketWorkbench.connectionFailed') }}</div>
      <div><p class="text-sm">IP</p><strong class="break-all text-lg">{{ result.ip || '—' }}</strong></div>
      <div><p class="text-sm">{{ t('admin.accounts.ticketWorkbench.region') }}</p><strong>{{ region || '—' }}</strong></div>
      <div class="col-span-2 text-sm">{{ result.duration_ms }} ms · {{ result.source || result.status }}</div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import { testTicketProxy, type TicketProxyPatch, type TicketProxyPolicy, type TicketProxyTestResult } from '@/api/admin/codexTickets'
import { fetchOne, getEntry } from '@/utils/ipGeoLookup'
import { extractApiErrorMessage } from '@/utils/apiError'
const props = defineProps<{ value?: TicketProxyPolicy; locked?: boolean; accountId?: number; allowInherit?: boolean; inherited?: boolean; testDisabled?: boolean }>()
const { t } = useI18n(), uid = useId()
const mode = ref<TicketProxyPatch['mode']>('fixed'), source = ref<'api' | 'template'>('api'), protocol = ref<'http' | 'socks5h'>('http')
const rows = ref<{ id: string; name: string; url: string; configured: boolean }[]>([]), fixedID = ref(''), extractionURL = ref('')
const testing = ref(false), error = ref(''), result = ref<TicketProxyTestResult>()
let generation = 0
const modeOptions = computed(() => [...(props.allowInherit ? ['inherit'] : []), 'fixed', 'rotate', 'dynamic'].map(value => ({ value, label: t('admin.accounts.ticketWorkbench.proxyModes.' + value) })))
const sourceOptions = computed(() => ['api', 'template'].map(value => ({ value, label: t('admin.accounts.ticketWorkbench.sources.' + value) })))
const protocolOptions = [{ value: 'http', label: 'HTTP' }, { value: 'socks5h', label: 'SOCKS5' }]
const region = computed(() => { const g = result.value?.ip ? getEntry(result.value.ip) : undefined; return [g?.detail?.countryCode || result.value?.country_code, g?.detail?.region, g?.detail?.city].filter(Boolean).join(' · ') })
function add() { const bytes = crypto.getRandomValues(new Uint8Array(16)); bytes[6] = (bytes[6] & 15) | 64; bytes[8] = (bytes[8] & 63) | 128; const h = Array.from(bytes, n => n.toString(16).padStart(2, '0')).join(''); const id = [h.slice(0, 8), h.slice(8, 12), h.slice(12, 16), h.slice(16, 20), h.slice(20)].join('-'); rows.value.push({ id, name: `Proxy ${rows.value.length + 1}`, url: '', configured: false }); if (!fixedID.value) fixedID.value = id }
function remove(index: number) { rows.value.splice(index, 1); if (!rows.value.some(r => r.id === fixedID.value)) fixedID.value = rows.value[0]?.id || '' }
function patch(): TicketProxyPatch {
  if (mode.value === 'inherit') return { mode: 'inherit' }
  if (mode.value === 'dynamic' && source.value === 'api') {
    if (!extractionURL.value.trim() && !props.value?.extraction_configured) throw new Error(t('admin.accounts.ticketWorkbench.extractionRequired'))
    return { mode: 'dynamic', dynamic_source: 'api', proxy_protocol: protocol.value, extraction_url: extractionURL.value.trim() || undefined }
  }
  if (!rows.value.length || rows.value.some(r => !r.name.trim() || (!r.configured && !r.url.trim()))) throw new Error(t('admin.accounts.ticketPolicy.proxyRequired'))
  return { mode: mode.value, dynamic_source: 'template', fixed_proxy_id: fixedID.value, proxies: rows.value.map(r => ({ id: r.id, name: r.name.trim(), harvest_proxy_url: r.url.trim() })) }
}
async function test() { if (testing.value || props.testDisabled) return; const current = generation; error.value = ''; result.value = undefined
  try { const payload = patch(); testing.value = true; const data = await testTicketProxy(props.accountId, payload); if (current !== generation) return; result.value = data; if (data.ip) await fetchOne(data.ip) } catch (err) { if (current === generation) error.value = extractApiErrorMessage(err, t('common.error')) } finally { if (current === generation) testing.value = false }
}
watch(() => [props.value, props.inherited], () => { generation++; testing.value = false; error.value = ''; result.value = undefined; mode.value = props.allowInherit && props.inherited ? 'inherit' : props.value?.mode || 'fixed'; source.value = props.value?.dynamic_source || 'api'; protocol.value = props.value?.proxy_protocol || 'http'; rows.value = (props.value?.proxies || []).map(p => ({ ...p, url: '' })); fixedID.value = props.value?.fixed_proxy_id || ''; extractionURL.value = '' }, { immediate: true })
onBeforeUnmount(() => { generation++; extractionURL.value = ''; rows.value = [] })
defineExpose({ patch })
</script>
