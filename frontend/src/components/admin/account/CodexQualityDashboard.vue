<template>
  <section class="card p-4 space-y-3" data-testid="quality-dashboard">
    <div class="flex items-center justify-between gap-3"><h3 class="font-extrabold text-lg">{{ t('admin.accounts.quality.dashboard') }}</h3><button class="btn btn-secondary btn-sm" :disabled="loading" @click="load">{{ t('common.refresh') }}</button></div>
    <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.quality.dashboardHint') }}</p>
    <p v-if="error" role="alert" class="text-yellow-700 dark:text-bh-yellow">{{ t('admin.accounts.quality.loadFailed') }}</p>
    <template v-if="counts">
      <CodexQualitySummary :counts="counts" :rate-label="t('admin.accounts.quality.poolRate')" @select="openAccounts" />
      <div class="flex flex-wrap gap-3"><button v-for="status in ['untested', 'stale', 'cancelled']" :key="status" class="btn btn-secondary btn-sm" @click="openAccounts(status)">{{ t(status === 'untested' ? 'admin.accounts.quality.untested' : `admin.accounts.quality.status.${status}`) }} <strong>{{ counts[status] || 0 }}</strong></button></div>
    </template>
  </section>
</template>
<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { qualitySchedulesAPI } from '@/api/admin/codexQuality'
import CodexQualitySummary from './CodexQualitySummary.vue'
const { t } = useI18n(), router = useRouter()
const counts = ref<Record<string, number> | null>(null), loading = ref(false), error = ref(false)
let active = true
async function load() { if (loading.value) return; loading.value = true; try { const value = await qualitySchedulesAPI.stats(); if (active) { counts.value = value; error.value = false } } catch { if (active) error.value = true } finally { if (active) loading.value = false } }
function openAccounts(status: string) { void router.push({ path: '/admin/accounts', query: { platform: 'openai', type: 'oauth', ...(status ? { quality_status: status } : {}) } }) }
onMounted(load)
onBeforeUnmount(() => { active = false })
defineExpose({ refresh: load })
</script>
