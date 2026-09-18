<template>
  <section class="space-y-3 border-2 border-[color:var(--bh-ink)] bg-[var(--bh-surface)] p-4" style="box-shadow: var(--bh-shadow-sm)" data-testid="codex-ticket-settings">
    <div class="flex items-start justify-between gap-4">
      <div>
        <h3 class="font-bold text-bh-blue dark:text-blue-300">{{ t('admin.settings.codexTicket.title') }}</h3>
        <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">{{ t('admin.settings.codexTicket.description') }}</p>
      </div>
      <Toggle v-model="enabled" :disabled="loading || saving || !!loadError" :aria-label="t('admin.settings.codexTicket.title')" data-testid="codex-ticket-toggle" />
    </div>
    <p class="border-l-4 border-bh-yellow pl-3 text-sm font-semibold text-yellow-800 dark:text-bh-yellow">{{ t('admin.settings.codexTicket.warning') }}</p>
    <label for="codex-ticket-proxy" class="input-label">{{ t('admin.settings.codexTicket.proxy') }}</label>
    <input id="codex-ticket-proxy" v-model="proxy" type="password" class="input w-full font-mono" autocomplete="new-password" :disabled="loading || saving || !!loadError" placeholder="socks5h://user:password@proxy.example:1080" />
    <p class="input-hint">{{ t(configured ? 'admin.settings.codexTicket.configured' : 'admin.settings.codexTicket.notConfigured') }}</p>
    <label v-if="configured" class="flex items-center gap-2 text-sm">
      <input v-model="clearProxy" type="checkbox" :disabled="loading || saving || !!loadError" />{{ t('admin.settings.codexTicket.clearProxy') }}
    </label>
    <p v-if="error || loadError" class="break-words text-sm text-bh-red dark:text-red-400" role="alert">{{ error || loadError }}</p>
    <div class="flex justify-end gap-2">
      <button v-if="loadError" type="button" class="btn btn-secondary" :disabled="loading" @click="load">{{ t('common.retry') }}</button>
      <button type="button" class="btn btn-primary" :disabled="loading || saving || !!loadError" data-testid="codex-ticket-save" @click="save">{{ t('admin.settings.codexTicket.save') }}</button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import { apiClient } from '@/api/client'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

interface Settings { enabled: boolean; proxy_configured: boolean }
const { t } = useI18n()
const app = useAppStore()
const enabled = ref(false), configured = ref(false), proxy = ref(''), clearProxy = ref(false)
const loading = ref(true), saving = ref(false), error = ref(''), loadError = ref('')
function apply(data: Settings) { enabled.value = data.enabled; configured.value = data.proxy_configured; proxy.value = ''; clearProxy.value = false }
// 独立保存，不将写入用的代理密码混入系统设置大表单或公开配置。
async function load() {
  loading.value = true; loadError.value = ''
  try { apply((await apiClient.get<Settings>('/admin/settings/codex-ticket')).data) }
  catch (err) { loadError.value = extractApiErrorMessage(err, t('common.error')) }
  finally { loading.value = false }
}
async function save() {
  if (loading.value || saving.value || loadError.value) return
  saving.value = true; error.value = ''
  try {
    apply((await apiClient.put<Settings>('/admin/settings/codex-ticket', { enabled: enabled.value, harvest_proxy_url: proxy.value.trim(), clear_proxy: clearProxy.value })).data)
    app.showSuccess(t('admin.settings.codexTicket.saved'))
  } catch (err) { error.value = extractApiErrorMessage(err, t('common.error')) }
  finally { saving.value = false }
}
onMounted(load)
</script>
