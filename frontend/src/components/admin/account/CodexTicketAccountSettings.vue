<template>
  <section class="space-y-5 border-2 border-[color:var(--bh-ink)] bg-[var(--bh-surface)] p-4 sm:p-5" style="box-shadow:var(--bh-shadow-sm)" data-testid="ticket-account-settings">
    <div class="flex flex-wrap items-baseline justify-between gap-2">
      <h3 class="text-lg font-extrabold text-bh-blue dark:text-blue-300">{{ t('admin.accounts.ticketPolicy.title') }}</h3>
      <span class="text-sm font-semibold">{{ bulk ? t('admin.accounts.ticketPolicy.selected', { count: ids.length }) : t(`admin.accounts.ticketPolicy.source.${source}`) }}</span>
    </div>
    <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.accounts.ticketPolicy.hint') }}</p>
    <div class="grid gap-5 sm:grid-cols-2">
      <div>
        <label :for="uid + '-mode'" class="mb-2 flex items-center gap-2 font-bold">
          <input v-if="bulk" v-model="fields.mode" type="checkbox" :disabled="locked" data-testid="ticket-edit-mode" />
          {{ t('admin.accounts.ticketPolicy.mode') }}
        </label>
        <Select :id="uid + '-mode'" v-model="mode" :options="modeOptions" :disabled="locked || !fields.mode" />
      </div>
      <div>
        <label :for="uid + '-guard'" class="mb-2 flex items-center gap-2 font-bold">
          <input v-if="bulk" v-model="fields.guard" type="checkbox" :disabled="locked" data-testid="ticket-edit-guard" />
          {{ t('admin.accounts.ticketPolicy.guard') }}
        </label>
        <Select :id="uid + '-guard'" v-model="guard" :options="guardOptions" :disabled="locked || !fields.guard" />
      </div>
    </div>
    <div class="space-y-3 border-t-2 border-[color:var(--bh-ink)] pt-4">
      <label :for="uid + '-proxy-action'" class="flex items-center gap-2 font-bold">
        <input v-model="fields.proxy" type="checkbox" :disabled="locked" data-testid="ticket-edit-proxy" />
        {{ t('admin.accounts.ticketPolicy.proxy') }}
      </label>
      <Select :id="uid + '-proxy-action'" v-model="proxyAction" :options="proxyOptions" :disabled="locked || !fields.proxy" />
      <input v-if="proxyAction === 'custom'" :id="uid + '-proxy'" v-model="proxyURL" type="password" class="input w-full font-mono" autocomplete="new-password" :aria-label="t('admin.accounts.ticketPolicy.proxy')" :disabled="locked || !fields.proxy" placeholder="socks5h://user-{sid}:password@proxy.example:1080" data-testid="ticket-account-proxy" />
      <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.accounts.ticketPolicy.proxyHint', { sid: '{sid}', random: '{random}' }) }}</p>
    </div>
    <p v-if="guard.startsWith('recover') && fields.guard" class="border-l-4 border-bh-yellow pl-3 text-sm font-semibold text-yellow-800 dark:text-bh-yellow">{{ t('admin.accounts.ticketPolicy.guardRisk') }}</p>
    <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.accounts.ticketPolicy.saveHint') }}</p>
    <p v-if="error" class="break-words text-sm text-bh-red dark:text-red-400" role="alert">{{ error }}</p>
    <p v-if="saved" class="text-sm font-semibold text-emerald-700 dark:text-emerald-400" role="status">{{ t('admin.accounts.ticketPolicy.saved') }}</p>
    <div class="flex justify-end gap-2">
      <button v-if="loadFailed" type="button" class="btn btn-secondary" :disabled="loading" @click="load">{{ t('common.retry') }}</button>
      <button type="button" class="btn btn-primary" :disabled="locked || (!fields.mode && !fields.guard && !fields.proxy) || ids.length > 500" data-testid="ticket-account-save" @click="save">{{ t('admin.accounts.ticketPolicy.save') }}</button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import { ticketAccountAPI, type TicketAccountPatch, type TicketAccountSettings } from '@/api/admin/codexTickets'
import { extractApiErrorMessage } from '@/utils/apiError'

// 专用保存不混入通用账号凭据；批量每项必须主动勾选，代理永远不回显。
const props = defineProps<{ ids: number[]; bulk?: boolean }>()
const emit = defineEmits<{ saved: [] }>()
const { t } = useI18n(), uid = useId()
const fields = reactive({ mode: !props.bulk, guard: !props.bulk, proxy: false })
const mode = ref<TicketAccountSettings['mode']>('inherit'), guard = ref<TicketAccountSettings['watchdog_mode']>('inherit')
const proxyAction = ref('custom'), proxyURL = ref(''), revision = ref(''), source = ref('gateway')
const loading = ref(false), saving = ref(false), loadFailed = ref(false), error = ref(''), saved = ref(false)
const locked = computed(() => loading.value || saving.value || loadFailed.value)
const modeOptions = computed(() => ['inherit', 'on', 'off'].map(value => ({ value, label: t(`admin.accounts.ticketPolicy.modes.${value}`) })))
const guardOptions = computed(() => ['inherit', 'off', 'observe', 'recover_length', 'recover_model', 'recover'].map(value => ({ value, label: t(`admin.accounts.ticketPolicy.guards.${value}`) })))
const proxyOptions = computed(() => ['custom', 'inherit'].map(value => ({ value, label: t(`admin.accounts.ticketPolicy.proxyActions.${value}`) })))
let controller: AbortController | undefined, sequence = 0
async function load() {
  controller?.abort(); controller = new AbortController(); const current = ++sequence
  fields.mode = !props.bulk; fields.guard = !props.bulk; fields.proxy = false
  mode.value = 'inherit'; guard.value = 'inherit'; proxyURL.value = ''; revision.value = ''; source.value = 'gateway'
  saved.value = false; loadFailed.value = false; error.value = ''; loading.value = !props.bulk
  if (props.bulk) return
  try {
    const data = await ticketAccountAPI.get(props.ids[0], controller.signal)
    if (current !== sequence) return
    mode.value = data.mode; guard.value = data.watchdog_mode; revision.value = data.revision; source.value = data.proxy_source
  } catch (err) { if (current === sequence && !controller.signal.aborted) { error.value = extractApiErrorMessage(err, t('common.error')); loadFailed.value = true } }
  finally { if (current === sequence) loading.value = false }
}
async function save() {
  if (locked.value) return
  const patch: TicketAccountPatch = {}
  if (fields.mode) patch.mode = mode.value
  if (fields.guard) patch.watchdog_mode = guard.value
  if (fields.proxy) {
    if (proxyAction.value === 'custom' && !proxyURL.value.trim()) { error.value = t('admin.accounts.ticketPolicy.proxyRequired'); return }
    patch.harvest_proxy_url = proxyAction.value === 'inherit' ? '' : proxyURL.value.trim()
  }
  if (!Object.keys(patch).length) return
  const current = sequence
  saving.value = true; error.value = ''; saved.value = false
  try {
    const result = await ticketAccountAPI.update([...props.ids], patch, props.bulk ? undefined : revision.value)
    if (current !== sequence) return
    proxyURL.value = ''; fields.proxy = false
    if (props.bulk) { fields.mode = false; fields.guard = false }
    else if (result[0]) { revision.value = result[0].revision; source.value = result[0].proxy_source }
    saved.value = true; emit('saved')
  } catch (err) { if (current === sequence) error.value = extractApiErrorMessage(err, t('common.error')) }
  finally { saving.value = false }
}
watch(() => props.ids.join(',') + '/' + !!props.bulk, load, { immediate: true })
onBeforeUnmount(() => { sequence++; controller?.abort(); proxyURL.value = '' })
</script>
