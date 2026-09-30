<template>
  <AppLayout>
    <div role="tablist" :aria-label="t('admin.modelAttributes.title')" class="mb-4 flex flex-wrap gap-3">
      <button v-for="tab in ['configs', 'defaults'] as const" :key="tab" type="button" role="tab" :aria-selected="activeTab === tab" class="btn" :class="activeTab === tab ? 'btn-warning' : 'btn-secondary'" @click="activeTab = tab">{{ t(`admin.modelAttributes.tabs.${tab}`) }}</button>
    </div>
    <TablePageLayout>
      <template #filters>
        <div class="space-y-3">
          <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('admin.modelAttributes.description') }}</p>
          <div class="flex flex-wrap items-center gap-3">
            <input v-model="search" class="input min-w-0 flex-1 sm:max-w-xs" :placeholder="t('common.search')" :aria-label="t('common.search')" />
            <Select v-if="activeTab === 'configs'" v-model="status" :options="statusOptions" class="w-40" />
            <template v-else>
              <Select v-model="provider" :options="providerOptions" class="w-48" />
              <Select v-model="capability" :options="capabilityOptions" class="w-48" />
            </template>
            <div class="ml-auto flex gap-2">
              <button class="btn btn-secondary btn-icon" :disabled="loading || updating" :aria-label="t('common.refresh')" @click="load"><Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" /></button>
              <button v-if="activeTab === 'configs'" class="btn btn-primary" @click="edit()"><Icon name="plus" size="md" class="mr-2" />{{ t('admin.modelAttributes.create') }}</button>
              <button v-else class="btn btn-primary" :disabled="updating" @click="updateCatalog">{{ t(updating ? 'admin.pricing.defaults.updating' : 'admin.pricing.defaults.update') }}</button>
            </div>
          </div>
          <p v-if="activeTab === 'defaults' && version" class="text-xs text-gray-500 dark:text-dark-400">models.dev · {{ version.slice(0, 12) }} · {{ updatedAt ? new Date(updatedAt).toLocaleString() : '' }}</p>
          <p v-if="error || (activeTab === 'defaults' && catalogError)" role="alert" class="text-sm text-red-600 dark:text-red-400">{{ error || catalogError }}</p>
        </div>
      </template>
      <template #table>
        <DataTable v-if="activeTab === 'configs'" :columns="configColumns" :data="configs" :loading="loading" column-order-storage-key="model-attribute-config-columns">
          <template #cell-status="{ row }"><Toggle :model-value="row.status === 'active'" @update:model-value="toggleStatus(row)" /></template>
          <template #cell-groups="{ row }">{{ row.group_ids.length }}</template>
          <template #cell-rules="{ row }">{{ row.rules.length }}</template>
          <template #cell-actions="{ row }">
            <div class="flex gap-2">
              <button class="btn btn-secondary btn-icon" :aria-label="t('common.edit')" @click="edit(row)"><Icon name="edit" size="sm" /></button>
              <button class="btn btn-secondary btn-icon" :aria-label="t('common.delete')" @click="deleting = row"><Icon name="trash" size="sm" /></button>
            </div>
          </template>
        </DataTable>
        <DataTable v-else :columns="defaultColumns" :data="defaults" :loading="loading" column-order-storage-key="model-default-attribute-columns">
          <template #cell-name="{ row }">{{ row.attributes.display_name ?? t('admin.modelAttributes.unknown') }}</template>
          <template #cell-context="{ row }">{{ row.attributes.context?.toLocaleString() ?? t('admin.modelAttributes.unknown') }}</template>
          <template #cell-output="{ row }">{{ row.attributes.output_limit?.toLocaleString() ?? t('admin.modelAttributes.unknown') }}</template>
          <template #cell-actions="{ row }"><button class="btn btn-secondary" @click="detail = row">{{ t('admin.modelAttributes.details') }}</button></template>
        </DataTable>
      </template>
      <template #pagination>
        <Pagination v-if="total" :page="page" :total="total" :page-size="pageSize" @update:page="page = $event; load()" @update:page-size="pageSize = $event; page = 1; load()" />
      </template>
    </TablePageLayout>

    <BaseDialog :show="showEditor" :title="t(form.id ? 'admin.modelAttributes.edit' : 'admin.modelAttributes.create')" width="extra-wide" @close="showEditor = false">
      <form id="attribute-form" class="space-y-5" @submit.prevent="save">
        <p v-if="formError" role="alert" class="text-sm text-red-600">{{ formError }}</p>
        <div class="grid gap-4 sm:grid-cols-2">
          <label><span class="input-label">{{ t('common.name') }}</span><input v-model="form.name" required maxlength="100" class="input" /></label>
          <div><label class="input-label">{{ t('common.status') }}</label><Select v-model="form.status" :options="editStatusOptions" /></div>
          <label class="sm:col-span-2"><span class="input-label">{{ t('admin.modelAttributes.configDescription') }}</span><textarea v-model="form.description" class="input" rows="2" /></label>
        </div>
        <fieldset class="rounded-control border border-gray-200 p-3 dark:border-dark-600">
          <legend class="px-1 text-sm font-medium">{{ t('admin.modelAttributes.groups') }}</legend>
          <p v-if="groupsLoading" class="text-sm">{{ t('common.loading') }}</p>
          <div class="flex max-h-40 flex-wrap gap-3 overflow-auto">
            <label v-for="group in groups" :key="group.id" class="flex items-center gap-2 text-sm">
              <input v-model="form.group_ids" type="checkbox" class="rounded-compact" :value="group.id" />{{ group.name }}
            </label>
          </div>
        </fieldset>
        <RuleListEditor :items="form.rules" :title="t('admin.modelAttributes.rules')" :empty-text="t('admin.modelAttributes.emptyRules')" variant="card" @add="form.rules.push({ models: [], attributes: {} })" @remove="form.rules.splice($event, 1)" @move="moveRule">
          <template #row="{ item, index }">
            <div class="space-y-4">
              <label class="block"><span class="input-label">{{ t('admin.modelAttributes.models') }}</span><input class="input" required :value="item.models.join(', ')" :placeholder="t('admin.modelAttributes.modelHint')" @change="form.rules[index]!.models = ($event.target as HTMLInputElement).value.split(',').map(value => value.trim()).filter(Boolean)" /></label>
              <ModelAttributesFields v-model="item.attributes" />
            </div>
          </template>
        </RuleListEditor>
      </form>
      <template #footer><div class="flex justify-end gap-3"><button class="btn btn-secondary" @click="showEditor = false">{{ t('common.cancel') }}</button><button form="attribute-form" type="submit" class="btn btn-primary" :disabled="saving || groupsLoading">{{ t('common.save') }}</button></div></template>
    </BaseDialog>

    <BaseDialog :show="!!detail" :title="detail?.model ?? ''" @close="detail = null">
      <div v-if="detail" class="space-y-4">
        <p class="text-sm text-gray-500">{{ detail.source }} · {{ detail.provider }}</p>
        <p v-if="detail.canonical_model_id" class="break-all text-sm">{{ detail.canonical_model_id }}</p>
        <ModelAttributesSummary :attributes="detail.attributes" />
      </div>
    </BaseDialog>
    <ConfirmDialog :show="!!deleting" :title="t('common.delete')" :message="t('admin.modelAttributes.deleteConfirm', { name: deleting?.name })" :danger="true" @confirm="remove" @cancel="deleting = null" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import RuleListEditor from '@/components/common/RuleListEditor.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import ModelAttributesFields from '@/components/admin/ModelAttributesFields.vue'
import ModelAttributesSummary from '@/components/common/ModelAttributesSummary.vue'
import { modelAttributesAPI, type AttributeConfig, type DefaultAttributes } from '@/api/admin/modelAttributes'
import { adminAPI } from '@/api/admin'
import type { AdminGroup } from '@/types'
import { attributeCapabilities } from '@/types/modelAttributes'
import { extractApiErrorMessage } from '@/utils/apiError'
import { SEARCH_DEBOUNCE_MS } from '@/constants/ui'

const { t } = useI18n()
const activeTab = ref<'configs' | 'defaults'>('configs')
const search = ref(''), status = ref(''), provider = ref(''), capability = ref('')
const page = ref(1), pageSize = ref(20), total = ref(0)
const loading = ref(false), updating = ref(false), saving = ref(false), showEditor = ref(false), groupsLoading = ref(false)
const error = ref(''), formError = ref(''), catalogError = ref(''), version = ref(''), updatedAt = ref('')
const configs = ref<AttributeConfig[]>([]), defaults = ref<DefaultAttributes[]>([]), providers = ref<string[]>([])
const groups = ref<AdminGroup[]>([])
const detail = ref<DefaultAttributes | null>(null), deleting = ref<AttributeConfig | null>(null)
const blank = (): AttributeConfig => ({ name: '', description: '', status: 'active', group_ids: [], rules: [] })
const form = ref<AttributeConfig>(blank())
const editStatusOptions = computed(() => [{ value: 'active', label: t('common.active') }, { value: 'disabled', label: t('common.disabled') }])
const statusOptions = computed(() => [{ value: '', label: t('admin.modelAttributes.allStatuses') }, ...editStatusOptions.value])
const providerOptions = computed(() => [{ value: '', label: t('admin.modelAttributes.allProviders') }, ...providers.value.map(value => ({ value, label: value }))])
const capabilityOptions = computed(() => [{ value: '', label: t('admin.modelAttributes.allCapabilities') }, ...attributeCapabilities.map(value => ({ value, label: t(`admin.modelAttributes.fields.${value}`) }))])
const configColumns = computed(() => ['name', 'status', 'groups', 'rules', 'actions'].map(key => ({ key, label: t(`admin.modelAttributes.columns.${key}`) })))
const defaultColumns = computed(() => ['model', 'name', 'provider', 'context', 'output', 'actions'].map(key => ({ key, label: t(`admin.modelAttributes.columns.${key}`) })))
let sequence = 0
let searchTimer: ReturnType<typeof setTimeout> | undefined

// 请求编号阻止慢查询覆盖用户切换后的页签和筛选结果。
async function load() {
  const current = ++sequence
  loading.value = true
  error.value = ''
  try {
    if (activeTab.value === 'configs') {
      const result = await modelAttributesAPI.list({ page: page.value, page_size: pageSize.value, search: search.value, status: status.value })
      if (current !== sequence) return
      configs.value = result.items
      total.value = result.total
    } else {
      const result = await modelAttributesAPI.defaults({ page: page.value, page_size: pageSize.value, search: search.value, provider: provider.value, capability: capability.value })
      if (current !== sequence) return
      defaults.value = result.items
      total.value = result.total
      providers.value = result.providers
      version.value = result.version
      updatedAt.value = result.last_updated
      catalogError.value = result.last_error ?? ''
    }
  } catch (cause) {
    if (current === sequence) error.value = extractApiErrorMessage(cause, t('common.error'))
  } finally {
    if (current === sequence) loading.value = false
  }
}
async function edit(config?: AttributeConfig) {
  form.value = config ? JSON.parse(JSON.stringify(config)) : blank()
  formError.value = ''
  showEditor.value = true
  groupsLoading.value = true
  try { groups.value = await adminAPI.groups.getAll() }
  catch (cause) { formError.value = extractApiErrorMessage(cause, t('common.error')) }
  finally { groupsLoading.value = false }
}
function moveRule(from: number, to: number) {
  const item = form.value.rules.splice(from, 1)[0]
  if (item) form.value.rules.splice(to, 0, item)
}
async function save() {
  if (saving.value) return
  saving.value = true
  formError.value = ''
  try { await modelAttributesAPI.save(form.value); showEditor.value = false; await load() }
  catch (cause) { formError.value = extractApiErrorMessage(cause, t('common.error')) }
  finally { saving.value = false }
}
async function toggleStatus(config: AttributeConfig) {
  try { await modelAttributesAPI.save({ ...config, status: config.status === 'active' ? 'disabled' : 'active' }); await load() }
  catch (cause) { error.value = extractApiErrorMessage(cause, t('common.error')) }
}
async function remove() {
  const id = deleting.value?.id
  deleting.value = null
  if (!id) return
  try { await modelAttributesAPI.remove(id); await load() }
  catch (cause) { error.value = extractApiErrorMessage(cause, t('common.error')) }
}
async function updateCatalog() {
  if (updating.value) return
  updating.value = true
  error.value = ''
  try { await modelAttributesAPI.update(); await load() }
  catch (cause) { error.value = extractApiErrorMessage(cause, t('common.error')) }
  finally { updating.value = false }
}
watch([activeTab, status, provider, capability], () => { page.value = 1; load() })
watch(search, () => { clearTimeout(searchTimer); searchTimer = setTimeout(() => { page.value = 1; load() }, SEARCH_DEBOUNCE_MS) })
onMounted(load)
onUnmounted(() => { sequence++; clearTimeout(searchTimer) })
</script>
