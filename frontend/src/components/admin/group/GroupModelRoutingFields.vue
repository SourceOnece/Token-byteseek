<template>
  <GroupFormSection>
    <GroupSettingRow
      :id="`${idPrefix}-model-routing`"
      :model-value="enabled"
      :label="t('admin.groups.modelRouting.title')"
      :hint="
        t(
          enabled
            ? 'admin.groups.modelRouting.noRulesHint'
            : 'admin.groups.modelRouting.disabledHint',
        )
      "
      :help="t('admin.groups.modelRouting.tooltip')"
      setting="model_routing_enabled"
      @update:model-value="emit('enabled', $event)"
    />
    <template v-if="enabled">
      <div
        v-for="rule in rules"
        :key="getKey(rule)"
        class="space-y-4 rounded-surface border border-gray-200 bg-gray-50/50 p-4 dark:border-dark-600 dark:bg-dark-800/40"
      >
        <div class="flex items-end gap-3">
          <div class="min-w-0 flex-1">
            <label :for="`${getKey(rule)}-pattern`" class="input-label">{{
              t('admin.groups.modelRouting.modelPattern')
            }}</label>
            <input
              :id="`${getKey(rule)}-pattern`"
              :value="rule.pattern"
              type="text"
              class="input"
              :placeholder="
                t('admin.groups.modelRouting.modelPatternPlaceholder')
              "
              @input="
                emit('pattern', rule, ($event.target as HTMLInputElement).value)
              "
            />
          </div>
          <button
            type="button"
            class="btn btn-ghost btn-icon shrink-0 text-red-500"
            :aria-label="t('admin.groups.modelRouting.removeRule')"
            @click="emit('remove', rule)"
          >
            <Icon name="trash" size="sm" />
          </button>
        </div>
        <div>
          <label :for="`${getKey(rule)}-providers`" class="input-label">{{
            t('admin.groups.modelRouting.providers')
          }}</label>
          <div v-if="rule.providers.length" class="mb-2 flex flex-wrap gap-2">
            <span
              v-for="provider in rule.providers"
              :key="provider.id"
              class="inline-flex max-w-full items-center gap-2 rounded-compact bg-primary-100 px-2 py-1 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
            >
              <span class="min-w-0 break-all">{{ provider.name }}</span>
              <button
                type="button"
                class="shrink-0 rounded-compact focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500"
                :aria-label="
                  t('admin.groups.settings.removeItem', { name: provider.name })
                "
                @click="emit('removeProvider', rule, provider.id)"
              >
                <Icon name="x" size="sm" />
              </button>
            </span>
          </div>
          <div class="relative provider-search-container">
            <input
              :id="`${getKey(rule)}-providers`"
              :value="search.keywords[getKey(rule)] || ''"
              type="text"
              class="input"
              :placeholder="
                t('admin.groups.modelRouting.searchProviderPlaceholder')
              "
              @input="
                emit('search', rule, ($event.target as HTMLInputElement).value)
              "
              @focus="emit('focus', rule)"
            />
            <div
              v-if="
                search.open[getKey(rule)] &&
                search.results[getKey(rule)]?.length
              "
              class="dropdown absolute left-0 right-0 z-50 mt-1 max-h-menu-sm overflow-auto"
            >
              <button
                v-for="provider in search.results[getKey(rule)]"
                :key="provider.id"
                type="button"
                class="dropdown-item-sm"
                :disabled="
                  rule.providers.some((selected) => selected.id === provider.id)
                "
                :class="{
                  'opacity-50': rule.providers.some(
                    (selected) => selected.id === provider.id,
                  ),
                }"
                @click="emit('selectProvider', rule, provider)"
              >
                <span class="min-w-0 break-all">{{ provider.name }}</span>
                <span class="shrink-0 text-xs text-gray-400"
                  >#{{ provider.id }}</span
                >
              </button>
            </div>
          </div>
          <p class="input-hint">
            {{ t('admin.groups.modelRouting.providersHint') }}
          </p>
        </div>
      </div>
      <button type="button" class="btn btn-secondary" @click="emit('add')">
        <Icon name="plus" size="sm" />{{
          t('admin.groups.modelRouting.addRule')
        }}
      </button>
    </template>
  </GroupFormSection>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import GroupFormSection from './GroupFormSection.vue'
import GroupSettingRow from './GroupSettingRow.vue'
import type {
  GroupProviderSearchState,
  GroupModelRoutingRule,
  GroupRoutingProvider,
} from './groupSettingsTypes'

// 不复制规则对象，保证页面的搜索键和取消请求逻辑始终指向同一条规则。
defineProps<{
  idPrefix: string
  enabled: boolean
  rules: GroupModelRoutingRule[]
  search: GroupProviderSearchState
  getKey: (rule: GroupModelRoutingRule) => string
}>()
const emit = defineEmits<{
  enabled: [value: boolean]
  add: []
  remove: [rule: GroupModelRoutingRule]
  pattern: [rule: GroupModelRoutingRule, value: string]
  search: [rule: GroupModelRoutingRule, keyword: string]
  focus: [rule: GroupModelRoutingRule]
  selectProvider: [rule: GroupModelRoutingRule, provider: GroupRoutingProvider]
  removeProvider: [rule: GroupModelRoutingRule, providerId: number]
}>()
const { t } = useI18n()
</script>
