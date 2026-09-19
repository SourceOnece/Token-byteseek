<template>
  <section class="space-y-5 border-2 border-[color:var(--bh-ink)] bg-[var(--bh-surface)] p-5 sm:p-6" style="box-shadow:var(--bh-shadow-sm)" data-testid="codex-ticket-settings">
    <div class="flex items-start justify-between gap-4"><div><h3 class="text-xl font-extrabold text-bh-blue dark:text-blue-300">{{ t('admin.settings.codexTicket.title') }}</h3><p class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ t('admin.accounts.ticketWorkbench.gatewayHint') }}</p></div><Toggle v-model="enabled" :disabled="locked" :aria-label="t('admin.settings.codexTicket.title')" data-testid="codex-ticket-toggle" /></div>
    <p class="border-l-4 border-bh-yellow pl-3 text-sm text-yellow-800 dark:text-bh-yellow">{{ t('admin.settings.codexTicket.warning') }}</p>
    <div class="space-y-4 border-t-2 border-[color:var(--bh-ink)] pt-5">
      <label class="flex items-center gap-2 text-lg font-extrabold"><input v-model="editProxy" type="checkbox" :disabled="locked" />{{ t('admin.accounts.ticketWorkbench.defaultProxy') }}</label>
      <CodexTicketProxyEditor ref="editor" :value="policy" :locked="locked || !editProxy" :test-disabled="locked" />
    </div>
    <p v-if="error" role="alert" class="text-sm text-bh-red dark:text-red-400">{{ error }}</p>
    <div class="flex justify-end gap-3"><button v-if="loadFailed" type="button" class="btn btn-secondary" :disabled="loading" @click="load">{{ t('common.retry') }}</button><button type="button" class="btn btn-primary" data-testid="codex-ticket-save" :disabled="locked" @click="save">{{ t('admin.settings.codexTicket.save') }}</button></div>
  </section>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import CodexTicketProxyEditor from '@/components/admin/account/CodexTicketProxyEditor.vue'
import { apiClient } from '@/api/client'
import type { TicketSettings, TicketProxyPolicy } from '@/api/admin/codexTickets'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
// 网关只编辑总闸与缺省代理，采集规则在账号工作台；保留旧API响应兼容读取。
const { t } = useI18n(), app = useAppStore()
const editor = ref<InstanceType<typeof CodexTicketProxyEditor>>(), policy = ref<TicketProxyPolicy>()
const enabled = ref(false), editProxy = ref(false), loading = ref(true), saving = ref(false), loadFailed = ref(false), error = ref(''), revision = ref('')
const locked = computed(() => loading.value || saving.value || loadFailed.value)
function apply(data: TicketSettings) {
  if (!data.proxy_policy || !Array.isArray(data.proxy_policy.proxies)) throw new Error(t('admin.settings.codexTicket.versionMismatch'))
  enabled.value = data.enabled; revision.value = data.revision; editProxy.value = false
  policy.value = data.proxy_policy || { mode: data.selection_mode, dynamic_source: 'template', proxy_protocol: 'http', extraction_configured: false, proxies: data.proxies, fixed_proxy_id: data.fixed_proxy_id }
}
async function load() { loading.value = true; error.value = ''; loadFailed.value = false
  try { apply((await apiClient.get<TicketSettings>('/admin/settings/codex-ticket')).data) } catch (e) { error.value = extractApiErrorMessage(e, t('common.error')); loadFailed.value = true } finally { loading.value = false }
}
async function save() { if (locked.value) return
  try { const proxy_policy = editProxy.value ? editor.value?.patch() : undefined; saving.value = true; error.value = ''
    apply((await apiClient.put<TicketSettings>('/admin/settings/codex-ticket', { enabled: enabled.value, revision: revision.value, proxy_policy })).data); app.showSuccess(t('admin.settings.codexTicket.saved'))
  } catch (e) { error.value = extractApiErrorMessage(e, t('common.error')) } finally { saving.value = false }
}
onMounted(load)
</script>
