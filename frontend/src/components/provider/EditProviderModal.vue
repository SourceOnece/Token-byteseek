<template>
  <BaseDialog
    :show="show"
    :title="t('admin.providers.editProvider')"
    width="wide"
    @close="handleClose"
  >
    <form
      v-if="provider"
      id="edit-provider-form"
      @submit.prevent="handleSubmit"
      class="space-y-5"
    >
      <fieldset :disabled="submitting" class="min-w-0 space-y-5">
      <div>
        <label class="input-label">{{ t('common.name') }}</label>
        <input v-model="form.name" type="text" required class="input" data-tour="edit-provider-form-name" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.providers.notes') }}</label>
        <textarea
          v-model="form.notes"
          rows="3"
          class="input"
          :placeholder="t('admin.providers.notesPlaceholder')"
        ></textarea>
        <p class="input-hint">{{ t('admin.providers.notesHint') }}</p>
      </div>

      <!-- 编辑站点只改路由上下文，不擅自改写令牌来源。 -->
      <div v-if="isQoderCosyProvider" class="space-y-2">
        <label class="input-label">{{ t('admin.providers.qoder.site.label') }}</label>
        <div class="grid grid-cols-2 gap-2" role="group" :aria-label="t('admin.providers.qoder.site.label')">
          <button
            type="button"
            data-testid="edit-qoder-site-global"
            :aria-pressed="qoderSite === 'global'"
            @click="qoderSite = 'global'"
            :class="[
              'rounded-control border px-4 py-2 text-sm font-medium transition-colors',
              qoderSite === 'global'
                ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-500/15 dark:bg-primary-500/8 dark:text-primary-500'
                : 'border-gray-200 bg-white text-gray-600 hover:bg-gray-50 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-300'
            ]"
          >
            {{ t('admin.providers.qoder.site.global') }}
          </button>
          <button
            type="button"
            data-testid="edit-qoder-site-cn"
            :aria-pressed="qoderSite === 'cn'"
            @click="qoderSite = 'cn'"
            :class="[
              'rounded-control border px-4 py-2 text-sm font-medium transition-colors',
              qoderSite === 'cn'
                ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-500/15 dark:bg-primary-500/8 dark:text-primary-500'
                : 'border-gray-200 bg-white text-gray-600 hover:bg-gray-50 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-300'
            ]"
          >
            {{ t('admin.providers.qoder.site.cn') }}
          </button>
        </div>
        <p v-if="qoderSiteChanged" class="text-xs text-amber-600 dark:text-amber-400">
          {{ t('admin.providers.qoder.site.changeWarning') }}
        </p>
      </div>

      <!-- API Key fields (only for apikey type) -->
      <div v-if="provider.type === 'apikey'" class="space-y-4">
        <div v-if="!isCNApiKeyProvider || editApiProtocol !== 'adaptive'">
          <label class="input-label">{{ t('admin.providers.baseUrl') }}</label>
          <input
            v-model="editBaseUrl"
            type="text"
            class="input"
            data-testid="edit-provider-base-url"
            :placeholder="
              provider.platform === 'openai'
                ? 'https://api.openai.com'
                : provider.platform === 'gemini'
                  ? 'https://generativelanguage.googleapis.com'
                  : provider.platform === 'antigravity'
                    ? 'https://cloudcode-pa.googleapis.com'
                    : provider.platform === 'grok'
                      ? 'https://api.x.ai/v1'
                      : 'https://api.anthropic.com'
            "
          />
          <p v-if="baseUrlHint" class="input-hint">{{ baseUrlHint }}</p>
          <GrokBaseUrlPresets
            v-if="provider.platform === 'grok'"
            class="mt-2"
            @select="editBaseUrl = $event"
          />
          <CnBaseUrlPresets
            v-if="isCNApiKeyProvider"
            class="mt-2"
            :platform="cnPresetPlatform"
            :mode="editProviderMode"
            :protocol="editApiProtocol"
            :current-url="editBaseUrl"
            @select="onCnPresetSelect"
          />
        </div>
        <div v-else>
          <label class="input-label">{{ t('admin.providers.cnProviders.apiProtocol.endpoints') }}</label>
          <div class="mt-2 space-y-3">
            <div v-for="item in editAdaptiveProtocolOptions" :key="item.value">
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                {{ t(`admin.providers.cnProviders.apiProtocol.${item.labelKey}`) }}
              </label>
              <input v-model="editAdaptiveBaseUrls[item.value]" type="text" class="input" />
            </div>
          </div>
          <p v-if="!cnSupportsNativeResponses(provider.platform)" class="input-hint">
            {{ t('admin.providers.cnProviders.apiProtocol.responsesFallbackDesc') }}
          </p>
        </div>
        <!-- 国产供应商提供商模式选择 -->
        <div v-if="isCNApiKeyProvider">
          <label class="input-label">{{ t('admin.providers.cnProviders.providerMode.title') }}</label>
          <div class="mt-2 flex flex-wrap gap-2">
            <button
              v-for="opt in cnProviderModeOptions"
              :key="opt.value"
              type="button"
              :class="[
                'rounded-control border-2 px-3 py-1.5 text-xs transition',
                editProviderMode === opt.value
                  ? 'border-primary-500 bg-primary-50 font-medium text-primary-700 dark:border-primary-500/15 dark:bg-primary-500/8 dark:text-primary-500'
                  : 'border-gray-200 text-gray-700 hover:border-gray-400 dark:border-dark-600 dark:text-gray-300 dark:hover:border-dark-500'
              ]"
              @click="editProviderMode = opt.value"
            >
              {{ t(`admin.accounts.cnProviders.accountMode.${opt.labelKey}`) }}
            </button>
          </div>
          <p class="input-hint">{{ t(`admin.accounts.cnProviders.accountMode.${editProviderMode}Desc`) }}</p>
          <OpenCodeGoProtocolRulesEditor v-if="provider.platform === 'opencode_go' && editApiProtocol === 'adaptive'" v-model:rows="editOpenCodeRules" :plan="editProviderMode === 'zen' ? 'zen' : 'go'" class="mt-4" />
        </div>
        <!-- 智谱团队版 Coding Plan：组织/项目 ID 可选，清空组织 ID 即回到个人额度端点。 -->
        <div v-if="provider.platform === 'zhipu' && editProviderMode === 'coding'">
          <div class="flex items-center gap-1">
            <label class="input-label">{{ t('admin.providers.cnProviders.zhipuTeam.title') }}</label>
            <HelpTooltip trigger="click" width-class="w-80">
              <p class="mb-1 font-medium">{{ t('admin.providers.cnProviders.zhipuTeam.help.title') }}</p>
              <ol class="list-decimal space-y-1 pl-4">
                <li>{{ t('admin.providers.cnProviders.zhipuTeam.help.step1') }}</li>
                <li>{{ t('admin.providers.cnProviders.zhipuTeam.help.step2') }}</li>
                <li>{{ t('admin.providers.cnProviders.zhipuTeam.help.step3') }}</li>
                <li>{{ t('admin.providers.cnProviders.zhipuTeam.help.step4') }}</li>
              </ol>
              <p class="mt-2 break-all rounded-compact bg-black/20 p-1.5 font-mono text-xs leading-relaxed">
                {{ t('admin.providers.cnProviders.zhipuTeam.help.example') }}
              </p>
            </HelpTooltip>
          </div>
          <div class="mt-2 grid gap-4 sm:grid-cols-2">
            <div>
              <label class="input-label">{{ t('admin.providers.cnProviders.zhipuTeam.organization') }}</label>
              <input
                v-model="editZhipuOrganization"
                type="text"
                class="input"
                :placeholder="t('admin.providers.cnProviders.zhipuTeam.organizationPlaceholder')"
              />
            </div>
            <div>
              <label class="input-label">{{ t('admin.providers.cnProviders.zhipuTeam.project') }}</label>
              <input
                v-model="editZhipuProject"
                type="text"
                class="input"
                :placeholder="t('admin.providers.cnProviders.zhipuTeam.projectPlaceholder')"
              />
            </div>
          </div>
          <p class="input-hint mt-2">{{ t('admin.providers.cnProviders.zhipuTeam.hint') }}</p>
        </div>
        <div v-if="provider.platform === 'gemini'">
          <label class="input-label">{{ t('admin.providers.gemini.connectionSource.label') }}</label>
          <Select
            v-model="geminiProviderType"
            :options="geminiProviderTypeOptions"
            data-testid="edit-gemini-provider-type"
          />
          <p class="input-hint">{{ geminiProviderTypeHint }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.providers.apiKey') }}</label>
          <input
            v-model="editApiKey"
            type="password"
            class="input font-mono"
            autocomplete="new-password"
            data-1p-ignore
            data-lpignore="true"
            data-bwignore="true"
            :placeholder="
              provider.platform === 'openai'
                ? 'sk-proj-...'
                : provider.platform === 'gemini'
                  ? geminiProviderType === 'third_party'
                    ? 'api-key-...'
                    : 'AIza...'
                  : provider.platform === 'antigravity'
                    ? 'sk-...'
                    : provider.platform === 'grok'
                      ? 'xai-...'
                      : 'sk-ant-...'
            "
          />
          <p class="input-hint">{{ t('admin.providers.leaveEmptyToKeep') }}</p>
        </div>

        <div v-if="provider.platform === 'gemini' && geminiProviderType === 'official'" data-testid="edit-gemini-tier">
          <label class="input-label">{{ t('admin.providers.gemini.tier.label') }}</label>
          <Select
            v-model="geminiAIStudioTier"
            :options="geminiAIStudioTierOptions"
            data-testid="edit-gemini-tier-select"
          />
          <p class="input-hint">{{ t('admin.providers.gemini.tier.aiStudioHint') }}</p>
        </div>

        <!-- Model Restriction Section (不适用于 Antigravity) -->
        <div v-if="provider.platform !== 'antigravity'" class="border-t border-gray-200 pt-4 dark:border-dark-600">
          <label class="input-label">{{ t('admin.providers.modelRestriction') }}</label>

            <!-- Mode Toggle -->
            <div class="mb-4 flex gap-2">
              <button
                type="button"
                @click="modelRestrictionMode = 'whitelist'"
                :class="[
                  'flex-1 rounded-control px-4 py-2 text-sm font-medium transition',
                  modelRestrictionMode === 'whitelist'
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-500/8 dark:text-primary-500'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
                ]"
              >
                <Icon name="checkCircle" size="sm" :animate-on-hover="false" class="mr-1.5 inline h-4 w-4" />
                {{ t('admin.providers.modelWhitelist') }}
              </button>
              <button
                type="button"
                @click="modelRestrictionMode = 'mapping'"
                :class="[
                  'flex-1 rounded-control px-4 py-2 text-sm font-medium transition',
                  modelRestrictionMode === 'mapping'
                    ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
                ]"
              >
                <Icon name="swap" size="sm" class="mr-1.5 inline h-4 w-4" />
                {{ t('admin.providers.modelMapping') }}
              </button>
            </div>
            <p class="mb-3 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.modelRestrictionCombinedHint') }}
            </p>

            <!-- Whitelist Mode -->
            <div v-if="modelRestrictionMode === 'whitelist'" v-content-reveal>
              <ModelWhitelistSelector :model-value="allowedModels" :platform="provider?.platform || 'anthropic'" :provider-id="provider?.id" @update:model-value="setAllowedModels" />
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.selectedModels', { count: allowedModels.length }) }}
                <span v-if="allowedModels.length === 0">{{
                  t('admin.providers.supportsAllModels')
                }}</span>
              </p>
            </div>

            <!-- Mapping Mode -->
            <ProviderModelMappingEditor
              v-else
              v-model="modelMappings"
              :presets="presetMappings"
              @add="touchQoderModelRestriction"
              @remove="touchQoderModelRestriction"
              @preset="addPresetMapping"
            />
        </div>

        <!-- Pool Mode Section -->
        <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
          <div class="mb-3 flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.providers.poolMode') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.poolModeHint') }}
              </p>
            </div>
            <Toggle v-model="poolModeEnabled" variant="flush" off-tone="soft" />
          </div>
          <Collapse :open="poolModeEnabled" unmount-on-hide>
            <div class="rounded-control bg-blue-50 p-3 dark:bg-blue-900/20">
              <p class="text-xs text-blue-700 dark:text-blue-400">
                <Icon name="exclamationCircle" size="sm" class="mr-1 inline" :stroke-width="2" />
                {{ t('admin.providers.poolModeInfo') }}
              </p>
            </div>
          </Collapse>
          <Collapse :open="poolModeEnabled" unmount-on-hide>
            <div class="mt-3">
              <label class="input-label">{{ t('admin.providers.poolModeRetryCount') }}</label>
              <input
                v-model.number="poolModeRetryCount"
                type="number"
                min="0"
                :max="MAX_POOL_MODE_RETRY_COUNT"
                step="1"
                class="input"
              />
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{
                  t('admin.providers.poolModeRetryCountHint', {
                    default: DEFAULT_POOL_MODE_RETRY_COUNT,
                    max: MAX_POOL_MODE_RETRY_COUNT
                  })
                }}
              </p>
            </div>
          </Collapse>
          <Collapse :open="poolModeEnabled" unmount-on-hide>
            <div class="mt-3">
              <label class="input-label">{{ t('admin.providers.poolModeRetryStatusCodes') }}</label>
              <input
                v-model="poolModeRetryStatusCodesInput"
                type="text"
                class="input"
                :placeholder="DEFAULT_POOL_MODE_RETRY_STATUS_CODES.join(', ')"
              />
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.poolModeRetryStatusCodesHint', { default: DEFAULT_POOL_MODE_RETRY_STATUS_CODES.join(', ') }) }}
              </p>
            </div>
          </Collapse>
        </div>

        <!-- 自定义错误码区域 -->
        <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
          <div class="mb-3 flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.providers.customErrorCodes') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.customErrorCodesHint') }}
              </p>
            </div>
            <Toggle v-model="customErrorCodesEnabled" variant="flush" off-tone="soft" />
          </div>

          <Collapse :open="customErrorCodesEnabled" unmount-on-hide>
            <div class="space-y-3">
              <div class="rounded-control bg-amber-50 p-3 dark:bg-amber-900/20">
                <p class="text-xs text-amber-700 dark:text-amber-400">
                  <Icon name="exclamationTriangle" size="sm" class="mr-1 inline" :stroke-width="2" />
                  {{ t('admin.providers.customErrorCodesWarning') }}
                </p>
              </div>

              <!-- 错误码快捷按钮 -->
              <div class="flex flex-wrap gap-2">
                <button
                  v-for="code in commonErrorCodes"
                  :key="code.value"
                  type="button"
                  @click="toggleErrorCode(code.value)"
                  :class="[
                    'rounded-control px-3 py-1.5 text-sm font-medium transition-colors',
                    selectedErrorCodes.includes(code.value)
                      ? 'bg-red-100 text-red-700 ring-1 ring-red-500 dark:bg-red-900/30 dark:text-red-400'
                      : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
                  ]"
                >
                  {{ code.value }} {{ code.label }}
                </button>
              </div>

              <!-- 手动输入 -->
              <div class="flex items-center gap-2">
                <input
                  v-model.number="customErrorCodeInput"
                  type="number"
                  min="100"
                  max="599"
                  class="input flex-1"
                  :placeholder="t('admin.providers.enterErrorCode')"
                  @keyup.enter="addCustomErrorCode"
                />
                <button type="button" @click="addCustomErrorCode" class="btn btn-secondary px-3">
                  <Icon name="plus" size="sm" class="h-4 w-4" />
                </button>
              </div>

              <!-- 已选错误码汇总 -->
              <div class="flex flex-wrap gap-1.5">
                <span
                  v-for="code in selectedErrorCodes.sort((a, b) => a - b)"
                  :key="code"
                  class="inline-flex items-center gap-1 rounded-full bg-red-100 px-2.5 py-0.5 text-sm font-medium text-red-700 dark:bg-red-900/30 dark:text-red-400"
                >
                  {{ code }}
                  <button
                    type="button"
                    @click="removeErrorCode(code)"
                    class="hover:text-red-900 dark:hover:text-red-300"
                  >
                    <Icon name="x" size="sm" :stroke-width="2" />
                  </button>
                </span>
                <span v-if="selectedErrorCodes.length === 0" class="text-xs text-gray-400">
                  {{ t('admin.providers.noneSelectedUsesDefault') }}
                </span>
              </div>
            </div>
          </Collapse>
        </div>

      </div>

      <!-- Grok OAuth 客户端工具提示缓存开关。 -->
      <div
        v-if="provider.platform === 'grok' && provider.type === 'oauth'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between gap-4">
          <div class="min-w-0">
            <label class="input-label mb-0">{{ t('admin.providers.grokClientToolCache.title') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.grokClientToolCache.hint') }}
            </p>
          </div>
          <Toggle
            v-model="grokClientToolCacheEnabled"
            data-testid="grok-client-tool-cache-toggle"
            :aria-label="t('admin.providers.grokClientToolCache.title')"
          />
        </div>
      </div>

      <!-- Grok OAuth 自定义上游地址（仅改写转发端点，OAuth 授权与刷新不受影响）。 -->
      <div
        v-if="provider.platform === 'grok' && provider.type === 'oauth'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="mb-3 flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{ t('admin.providers.grokCustomBaseUrl.title') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.grokCustomBaseUrl.hint') }}
            </p>
          </div>
          <Toggle v-model="grokOAuthCustomBaseUrlEnabled" variant="flush" off-tone="soft" data-testid="grok-custom-base-url-toggle" />
        </div>
        <Collapse :open="grokOAuthCustomBaseUrlEnabled" unmount-on-hide>
          <div class="space-y-2">
            <input
              v-model="grokOAuthBaseUrl"
              type="text"
              class="input"
              data-testid="grok-custom-base-url-input"
              :placeholder="t('admin.providers.grokCustomBaseUrl.placeholder')"
            />
            <GrokBaseUrlPresets @select="grokOAuthBaseUrl = $event" />
          </div>
        </Collapse>
      </div>

      <!-- 请求头覆写区域（支持的平台 API Key 与 Grok OAuth） -->
      <div v-if="headerOverrideCapable" class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="mb-3 flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{ t('admin.providers.headerOverride.title') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.headerOverride.hint') }}
            </p>
          </div>
          <Toggle v-model="headerOverrideEnabled" variant="flush" off-tone="soft" />
        </div>

        <Collapse :open="headerOverrideEnabled" unmount-on-hide>
          <div class="space-y-3">
            <div class="rounded-control bg-blue-50 p-3 dark:bg-blue-900/20">
              <p class="text-xs text-blue-700 dark:text-blue-400">
                <Icon name="exclamationCircle" size="sm" class="mr-1 inline" :stroke-width="2" />
                {{ t('admin.providers.headerOverride.info') }}
              </p>
            </div>

            <HeaderOverrideEditor
              :rows="headerOverrideRows"
              @update:rows="headerOverrideRows = $event"
            />
          </div>
        </Collapse>
      </div>

      <!-- OAuth/COSY 模型映射：这类提供商没有 apikey 容器，需要独立的模型映射区域 -->
      <div
        v-if="supportsOAuthLikeModelRestriction"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <label class="input-label">{{ t('admin.providers.modelRestriction') }}</label>

          <!-- Mode Toggle -->
          <div class="mb-4 flex gap-2">
            <button
              type="button"
              @click="modelRestrictionMode = 'whitelist'"
              :class="[
                'flex-1 rounded-control px-4 py-2 text-sm font-medium transition',
                modelRestrictionMode === 'whitelist'
                  ? 'bg-primary-100 text-primary-700 dark:bg-primary-500/8 dark:text-primary-500'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
              ]"
            >
              {{ t('admin.providers.modelWhitelist') }}
            </button>
            <button
              type="button"
              @click="modelRestrictionMode = 'mapping'"
              :class="[
                'flex-1 rounded-control px-4 py-2 text-sm font-medium transition',
                modelRestrictionMode === 'mapping'
                  ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
              ]"
            >
              {{ t('admin.providers.modelMapping') }}
            </button>
          </div>
          <p class="mb-3 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.providers.modelRestrictionCombinedHint') }}
          </p>

          <!-- Whitelist Mode -->
          <div v-if="modelRestrictionMode === 'whitelist'" v-content-reveal>
            <ModelWhitelistSelector
              :model-value="allowedModels"
              :platform="provider?.platform || 'anthropic'"
              :provider-id="provider?.id"
              :models="isQoderCosyProvider ? qoderAvailableModels : undefined"
              @update:modelValue="setAllowedModels"
            />
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.selectedModels', { count: allowedModels.length }) }}
              <span v-if="allowedModels.length === 0">{{
                t('admin.providers.supportsAllModels')
              }}</span>
            </p>
          </div>

          <!-- Mapping Mode -->
          <ProviderModelMappingEditor
            v-else
            v-model="modelMappings"
            :presets="presetMappings"
            @add="touchQoderModelRestriction"
            @remove="touchQoderModelRestriction"
            @preset="addPresetMapping"
          />
      </div>

      <!-- Upstream fields (only for upstream type) -->
      <div v-if="provider.type === 'upstream'" class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.providers.upstream.baseUrl') }}</label>
          <input
            v-model="editBaseUrl"
            type="text"
            class="input"
            placeholder="https://cloudcode-pa.googleapis.com"
          />
          <p class="input-hint">{{ t('admin.providers.upstream.baseUrlHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.providers.upstream.apiKey') }}</label>
          <input
            v-model="editApiKey"
            type="password"
            class="input font-mono"
            placeholder="sk-..."
          />
          <p class="input-hint">{{ t('admin.providers.leaveEmptyToKeep') }}</p>
        </div>
      </div>

      <!-- Vertex Service Account -->
      <div v-if="(provider.platform === 'gemini' || provider.platform === 'anthropic') && provider.type === 'service_account'" class="space-y-4">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">Project ID</label>
            <input
              v-model="editVertexProjectId"
              type="text"
              class="input font-mono"
              readonly
              :placeholder="t('admin.providers.vertexProjectIdPlaceholder')"
            />
            <p class="input-hint">{{ t('admin.providers.vertexSaJsonEditHint') }}</p>
          </div>
          <div>
            <label class="input-label">Location</label>
            <Select
              v-model="editVertexLocation"
              :options="vertexLocationOptions"
              class="font-mono"
              searchable
            />
            <p class="input-hint">{{ t('admin.providers.vertexLocationHint') }}</p>
          </div>
        </div>

        <!-- Model Restriction Section for Service Account -->
        <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
          <label class="input-label">{{ t('admin.providers.modelRestriction') }}</label>

          <!-- Mode Toggle -->
          <div class="mb-4 flex gap-2">
            <button
              type="button"
              @click="modelRestrictionMode = 'whitelist'"
              :class="[
                'flex-1 rounded-control px-4 py-2 text-sm font-medium transition',
                modelRestrictionMode === 'whitelist'
                  ? 'bg-primary-100 text-primary-700 dark:bg-primary-500/8 dark:text-primary-500'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
              ]"
            >
              <Icon name="checkCircle" size="sm" :animate-on-hover="false" class="mr-1.5 inline h-4 w-4" />
              {{ t('admin.providers.modelWhitelist') }}
            </button>
            <button
              type="button"
              @click="modelRestrictionMode = 'mapping'"
              :class="[
                'flex-1 rounded-control px-4 py-2 text-sm font-medium transition',
                modelRestrictionMode === 'mapping'
                  ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
              ]"
            >
              <Icon name="swap" size="sm" class="mr-1.5 inline h-4 w-4" />
              {{ t('admin.providers.modelMapping') }}
            </button>
          </div>

          <!-- Whitelist Mode -->
          <div v-if="modelRestrictionMode === 'whitelist'" v-content-reveal>
            <ModelWhitelistSelector :model-value="allowedModels" :platform="provider?.platform || 'anthropic'" :provider-id="provider?.id" @update:model-value="setAllowedModels" />
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.selectedModels', { count: allowedModels.length }) }}
              <span v-if="allowedModels.length === 0">{{
                t('admin.providers.supportsAllModels')
              }}</span>
            </p>
          </div>

          <!-- Mapping Mode -->
          <ProviderModelMappingEditor
            v-else
            v-model="modelMappings"
            :presets="presetMappings"
            @add="touchQoderModelRestriction"
            @remove="touchQoderModelRestriction"
            @preset="addPresetMapping"
          />
        </div>
      </div>

      <!-- Bedrock fields (for bedrock type, both SigV4 and API Key modes) -->
      <div v-if="provider.type === 'bedrock'" class="space-y-4">
        <!-- SigV4 fields -->
        <template v-if="!isBedrockAPIKeyMode">
          <div>
            <label class="input-label">{{ t('admin.providers.bedrockAccessKeyId') }}</label>
            <input
              v-model="editBedrockAccessKeyId"
              type="text"
              class="input font-mono"
              placeholder="AKIA..."
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.providers.bedrockSecretAccessKey') }}</label>
            <input
              v-model="editBedrockSecretAccessKey"
              type="password"
              class="input font-mono"
              :placeholder="t('admin.providers.bedrockSecretKeyLeaveEmpty')"
            />
            <p class="input-hint">{{ t('admin.providers.bedrockSecretKeyLeaveEmpty') }}</p>
          </div>
          <div>
            <label class="input-label">{{ t('admin.providers.bedrockSessionToken') }}</label>
            <input
              v-model="editBedrockSessionToken"
              type="password"
              class="input font-mono"
              :placeholder="t('admin.providers.bedrockSecretKeyLeaveEmpty')"
            />
            <p class="input-hint">{{ t('admin.providers.bedrockSessionTokenHint') }}</p>
          </div>
        </template>

        <!-- API Key field -->
        <div v-if="isBedrockAPIKeyMode">
          <label class="input-label">{{ t('admin.providers.bedrockApiKeyInput') }}</label>
          <input
            v-model="editBedrockApiKeyValue"
            type="password"
            class="input font-mono"
            :placeholder="t('admin.providers.bedrockApiKeyLeaveEmpty')"
          />
          <p class="input-hint">{{ t('admin.providers.bedrockApiKeyLeaveEmpty') }}</p>
        </div>

        <!-- Shared: Region -->
        <div>
          <label class="input-label">{{ t('admin.providers.bedrockRegion') }}</label>
          <input
            v-model="editBedrockRegion"
            type="text"
            class="input"
            placeholder="us-east-1"
          />
          <p class="input-hint">{{ t('admin.providers.bedrockRegionHint') }}</p>
        </div>

        <!-- Shared: Force Global -->
        <div>
          <label class="flex items-center gap-2 cursor-pointer">
            <input
              v-model="editBedrockForceGlobal"
              type="checkbox"
              class="rounded-compact border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-500"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.providers.bedrockForceGlobal') }}</span>
          </label>
          <p class="input-hint mt-1">{{ t('admin.providers.bedrockForceGlobalHint') }}</p>
        </div>

        <!-- Model Restriction for Bedrock -->
        <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
          <label class="input-label">{{ t('admin.providers.modelRestriction') }}</label>

          <!-- Mode Toggle -->
          <div class="mb-4 flex gap-2">
            <button
              type="button"
              @click="modelRestrictionMode = 'whitelist'"
              :class="[
                'flex-1 rounded-control px-4 py-2 text-sm font-medium transition',
                modelRestrictionMode === 'whitelist'
                  ? 'bg-primary-100 text-primary-700 dark:bg-primary-500/8 dark:text-primary-500'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
              ]"
            >
              {{ t('admin.providers.modelWhitelist') }}
            </button>
            <button
              type="button"
              @click="modelRestrictionMode = 'mapping'"
              :class="[
                'flex-1 rounded-control px-4 py-2 text-sm font-medium transition',
                modelRestrictionMode === 'mapping'
                  ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
              ]"
            >
              {{ t('admin.providers.modelMapping') }}
            </button>
          </div>
          <p class="mb-3 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.providers.modelRestrictionCombinedHint') }}
          </p>

          <!-- Whitelist Mode -->
          <div v-if="modelRestrictionMode === 'whitelist'" v-content-reveal>
            <ModelWhitelistSelector :model-value="allowedModels" platform="anthropic" @update:model-value="setAllowedModels" />
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.selectedModels', { count: allowedModels.length }) }}
              <span v-if="allowedModels.length === 0">{{ t('admin.providers.supportsAllModels') }}</span>
            </p>
          </div>

          <!-- Mapping Mode -->
          <ProviderModelMappingEditor
            v-else
            v-model="modelMappings"
            :presets="bedrockPresets"
            :source-placeholder="t('admin.providers.fromModel')"
            :target-placeholder="t('admin.providers.toModel')"
            @preset="(from, to) => modelMappings.push({ from, to })"
          />
        </div>

        <!-- Pool Mode Section for Bedrock -->
        <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
          <div class="mb-3 flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.providers.poolMode') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.poolModeHint') }}
              </p>
            </div>
            <Toggle v-model="poolModeEnabled" variant="flush" off-tone="soft" />
          </div>
          <Collapse :open="poolModeEnabled" unmount-on-hide>
            <div class="rounded-control bg-blue-50 p-3 dark:bg-blue-900/20">
              <p class="text-xs text-blue-700 dark:text-blue-400">
                <Icon name="exclamationCircle" size="sm" class="mr-1 inline" :stroke-width="2" />
                {{ t('admin.providers.poolModeInfo') }}
              </p>
            </div>
          </Collapse>
          <Collapse :open="poolModeEnabled" unmount-on-hide>
            <div class="mt-3">
              <label class="input-label">{{ t('admin.providers.poolModeRetryCount') }}</label>
              <input
                v-model.number="poolModeRetryCount"
                type="number"
                min="0"
                :max="MAX_POOL_MODE_RETRY_COUNT"
                step="1"
                class="input"
              />
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{
                  t('admin.providers.poolModeRetryCountHint', {
                    default: DEFAULT_POOL_MODE_RETRY_COUNT,
                    max: MAX_POOL_MODE_RETRY_COUNT
                  })
                }}
              </p>
            </div>
          </Collapse>
          <Collapse :open="poolModeEnabled" unmount-on-hide>
            <div class="mt-3">
              <label class="input-label">{{ t('admin.providers.poolModeRetryStatusCodes') }}</label>
              <input
                v-model="poolModeRetryStatusCodesInput"
                type="text"
                class="input"
                :placeholder="DEFAULT_POOL_MODE_RETRY_STATUS_CODES.join(', ')"
              />
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.poolModeRetryStatusCodesHint', { default: DEFAULT_POOL_MODE_RETRY_STATUS_CODES.join(', ') }) }}
              </p>
            </div>
          </Collapse>
        </div>
      </div>

      <div
        v-if="provider.platform === 'antigravity' && provider.type === 'oauth'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <label class="input-label">{{ t('admin.providers.antigravityProjectIdLabel') }}</label>
        <input
          v-model="antigravityProjectId"
          data-testid="antigravity-project-id-input"
          type="text"
          class="input font-mono"
          :placeholder="t('admin.providers.antigravityProjectIdPlaceholder')"
        />
        <p class="input-hint">{{ t('admin.providers.antigravityProjectIdHint') }}</p>
      </div>

      <!-- Antigravity model restriction (applies to all antigravity types) -->
      <!-- 白名单与映射分别控制最终范围和请求改写。 -->
      <div v-if="provider.platform === 'antigravity'" class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <label class="input-label">{{ t('admin.providers.modelRestriction') }}</label>
        <p class="input-hint">{{ t('admin.providers.selectAllowedModels') }}</p>
        <ModelWhitelistSelector v-model="antigravityWhitelistModels" platform="antigravity" />

        <!-- Mapping Mode Only (no toggle for Antigravity) -->
        <ProviderModelMappingEditor
          v-model="antigravityModelMappings"
          :presets="antigravityPresetMappings"
          wildcard-validation
          @preset="addAntigravityPresetMapping"
        >
          <template #header-actions>
            <button
              type="button"
              @click="syncAntigravityUpstreamModels"
              :disabled="isSyncingAntigravityUpstream || !provider?.id"
              class="rounded-control border border-emerald-200 px-3 py-1.5 text-sm text-emerald-600 hover:bg-emerald-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-emerald-800 dark:text-emerald-400 dark:hover:bg-emerald-900/30"
            >
              {{ isSyncingAntigravityUpstream ? t('admin.providers.syncUpstreamModelsLoading') : t('admin.providers.syncUpstreamModels') }}
            </button>
          </template>
        </ProviderModelMappingEditor>
      </div>

      <!-- Temp Unschedulable Rules -->
      <div class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4">
        <div class="mb-3 flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{ t('admin.providers.tempUnschedulable.title') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.tempUnschedulable.hint') }}
            </p>
          </div>
          <Toggle v-model="tempUnschedEnabled" variant="flush" off-tone="soft" />
        </div>

        <Collapse :open="tempUnschedEnabled" unmount-on-hide>
          <TempUnschedRulesEditor v-model="tempUnschedRules" />
        </Collapse>
      </div>

      <div
        v-if="supportsProviderSchedulingThresholdOverride"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
        data-testid="provider-scheduling-threshold-section"
      >
        <div class="mb-3 flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{ t('admin.providers.providerSchedulingThresholdOverride') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.providerSchedulingThresholdOverrideHint') }}
            </p>
          </div>
          <Toggle
            v-model="providerSchedulingThresholdOverrideEnabled"
            data-testid="provider-scheduling-threshold-override-enabled"
            :aria-label="t('admin.providers.providerSchedulingThresholdOverride')"
          />
        </div>
        <Collapse :open="providerSchedulingThresholdOverrideEnabled" unmount-on-hide>
          <div >
            <label class="input-label">{{ t('admin.providers.providerSchedulingThresholdOverrideValue') }}</label>
            <input
              v-model.number="providerSchedulingThresholdOverrideValue"
              data-testid="provider-scheduling-threshold-override-value"
              type="number"
              min="1"
              max="100"
              class="input"
            />
            <p class="input-hint">{{ t('admin.providers.providerSchedulingThresholdOverrideDisabledHint') }}</p>
          </div>
        </Collapse>
      </div>

      <!-- Intercept Warmup Requests (Anthropic/Antigravity) -->
      <div
        v-if="provider?.platform === 'anthropic' || provider?.platform === 'antigravity'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{
              t('admin.providers.interceptWarmupRequests')
            }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.interceptWarmupRequestsDesc') }}
            </p>
          </div>
          <Toggle v-model="interceptWarmupRequests" variant="flush" off-tone="soft" />
        </div>
      </div>

      <div v-if="!isSparkShadow">
        <div class="mb-1 flex items-center gap-2">
          <label class="input-label mb-0">{{ t('admin.providers.proxy') }}</label>
        </div>
        <ProxySelector v-model="form.proxy_id" :proxies="proxies" />
      </div>

      <ProviderProtocolSelector v-if="provider && !provider.parent_provider_id" v-model="upstreamProtocols" :platform="provider.platform" :type="provider.type" :auth-mode="String(provider.credentials?.auth_mode ?? provider.credentials?.openai_auth_mode ?? '')" />

      <UpstreamRequestIdHeaderField
        v-model="upstreamRequestIdHeader"
        :platform="provider.platform"
        :type="provider.type"
      />

      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <div>
          <label class="input-label">{{ t('admin.providers.concurrency') }}</label>
          <!-- 输入过程中允许先清空再录入新值，提交时由后端继续校验。 -->
          <input
            v-model.number="form.concurrency"
            type="number"
            min="1"
            class="input"
            data-testid="edit-provider-concurrency"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.providers.loadFactor') }}</label>
          <input
            v-model.number="form.load_factor"
            type="number"
            min="1"
            class="input"
            :placeholder="String(form.concurrency || 1)"
            data-testid="edit-provider-load-factor"
          />
          <p class="input-hint">{{ t('admin.providers.loadFactorHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.providers.priority') }}</label>
          <input
            v-model.number="form.priority"
            type="number"
            min="1"
            class="input"
            data-tour="provider-form-priority"
          />
          <p class="input-hint">{{ t('admin.providers.priorityHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.providers.billingRateMultiplier') }}</label>
          <input v-model.number="form.rate_multiplier" type="number" min="0" step="0.001" class="input" />
          <p class="input-hint">{{ t('admin.providers.billingRateMultiplierHint') }}</p>
        </div>
      </div>
      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <label class="input-label">{{ t('admin.providers.expiresAt') }}</label>
        <input v-model="expiresAtInput" type="datetime-local" class="input" />
        <p class="input-hint">{{ t('admin.providers.expiresAtHint') }}</p>
      </div>

      <!-- OpenAI 自动透传开关（OAuth/API Key） -->
      <div
        v-if="provider?.platform === 'openai' && (provider?.type === 'oauth' || provider?.type === 'apikey')"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{ t('admin.providers.openai.oauthPassthrough') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.openai.oauthPassthroughDesc') }}
            </p>
          </div>
          <Toggle v-model="openaiPassthroughEnabled" variant="flush" off-tone="soft" />
        </div>
      </div>

      <!-- OpenAI Codex namespace 工具摊平兼容开关，仅 OAuth 可用 -->
      <div
        v-if="provider?.platform === 'openai' && provider?.type === 'oauth' && !isSparkShadow"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between gap-4">
          <div>
            <label class="input-label mb-0">{{ t('admin.providers.openai.flattenNamespaces') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.openai.flattenNamespacesDesc') }}
            </p>
          </div>
          <Toggle v-model="openaiFlattenNamespacesEnabled" variant="flush" off-tone="soft" data-testid="edit-openai-flatten-namespaces-toggle" />
        </div>
      </div>

      <!-- OpenAI Codex hosted image_generation 桥接策略 -->
      <div
        v-if="provider?.platform === 'openai' && (provider?.type === 'oauth' || provider?.type === 'apikey')"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <CodexImageToolModeSelector v-model="codexImageToolMode" />
      </div>

      <!-- OpenAI WS Mode 三态（off/ctx_pool/passthrough） -->
      <div
        v-if="provider?.platform === 'openai' && (provider?.type === 'oauth' || provider?.type === 'apikey')"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="min-w-0">
            <label class="input-label mb-0">{{ t('admin.providers.openai.wsMode') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.openai.wsModeDesc') }}
            </p>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t(openAIWSModeConcurrencyHintKey) }}
            </p>
          </div>
          <div class="w-full sm:w-52 sm:flex-shrink-0">
            <Select v-model="openaiResponsesWebSocketV2Mode" :options="openAIWSModeOptions" />
          </div>
        </div>
      </div>

      <!-- OpenAI APIKey 文本工作负载与管理员协议路由 -->
      <div
        v-if="provider?.platform === 'openai' && provider?.type === 'apikey'"
        class="space-y-5 border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex flex-col gap-3 border-t border-gray-200 pt-4 dark:border-dark-600 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <label class="input-label mb-0" for="edit-openai-continuation-supported">
              {{ t('admin.providers.openai.responsesContinuationSupported') }}
            </label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.openai.responsesContinuationSupportedDesc') }}
            </p>
          </div>
          <Toggle
            id="edit-openai-continuation-supported"
            v-model="openAIResponsesContinuationSupported"
            data-testid="edit-openai-continuation-supported"
            :aria-label="t('admin.providers.openai.responsesContinuationSupported')"
          />
        </div>
        <div class="flex flex-col gap-3 border-t border-gray-200 pt-4 dark:border-dark-600 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <label class="input-label mb-0" for="edit-openai-images-url-to-b64-json">
              {{ t('admin.providers.openai.imagesURLToB64JSON') }}
            </label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.openai.imagesURLToB64JSONDesc') }}
            </p>
          </div>
          <Toggle
            id="edit-openai-images-url-to-b64-json"
            v-model="openAIImagesURLToB64JSON"
            data-testid="edit-openai-images-url-to-b64-json"
            :aria-label="t('admin.providers.openai.imagesURLToB64JSON')"
          />
        </div>

      </div>

      <OllamaCloudUsageSettings
        v-if="provider?.ollama_cloud_usage?.eligible"
        :provider="provider"
        @updated="handleOllamaCloudUsageUpdated"
      />

      <!-- Anthropic API Key 自动透传开关 -->
      <div
        v-if="provider?.platform === 'anthropic' && provider?.type === 'apikey'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{ t('admin.providers.anthropic.apiKeyPassthrough') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.anthropic.apiKeyPassthroughDesc') }}
            </p>
          </div>
          <Toggle v-model="anthropicPassthroughEnabled" variant="flush" off-tone="soft" />
        </div>
      </div>

      <!-- Anthropic API Key 上游认证方式 -->
      <div
        v-if="provider?.platform === 'anthropic' && provider?.type === 'apikey'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between gap-4">
          <div>
            <label class="input-label mb-0">{{ t('admin.providers.anthropic.apiKeyAuthScheme') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.anthropic.apiKeyAuthSchemeDesc') }}
            </p>
          </div>
          <div class="w-56">
            <Select v-model="anthropicAPIKeyAuthScheme" :options="anthropicAPIKeyAuthSchemeOptions" />
          </div>
        </div>
      </div>

      <!-- Anthropic API Key: Web Search Emulation (hidden when global disabled) -->
      <div
        v-if="provider?.platform === 'anthropic' && provider?.type === 'apikey' && webSearchGlobalEnabled"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{ t('admin.providers.anthropic.webSearchEmulation') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.anthropic.webSearchEmulationDesc') }}
            </p>
          </div>
          <Select v-model="webSearchEmulationMode" :options="webSearchEmulationOptions" class="w-32 text-sm" />
        </div>
      </div>

      <UpstreamUsageConfigEditor
        v-if="provider?.type === 'apikey'"
        :enabled="upstreamUsageEnabled"
        :adapter="upstreamUsageAdapter"
        :base-url="upstreamUsageBaseUrl"
        :wallet-access-token="upstreamUsageWalletAccessToken"
        :wallet-user-id="upstreamUsageWalletUserId"
        :automatic-adapter="isCNApiKeyProvider"
        @update:enabled="upstreamUsageEnabled = $event"
        @update:adapter="upstreamUsageAdapter = $event"
        @update:base-url="upstreamUsageBaseUrl = $event"
        @update:wallet-access-token="upstreamUsageWalletAccessToken = $event"
        @update:wallet-user-id="upstreamUsageWalletUserId = $event"
      />

      <!-- 配额控制 (Anthropic apikey/bedrock: 配额限制 + 亲和) -->
      <div
        v-if="provider?.platform === 'anthropic' && (provider?.type === 'apikey' || provider?.type === 'bedrock')"
        class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4"
      >
        <div class="mb-3">
          <h3 class="input-label mb-0 text-base font-semibold">{{ t('admin.providers.quotaControl.title') }}</h3>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.providers.quotaControl.hint') }}
          </p>
        </div>
        <QuotaLimitCard
          :totalLimit="editQuotaLimit"
          :dailyLimit="editQuotaDailyLimit"
          :weeklyLimit="editQuotaWeeklyLimit"
          :dailyResetMode="editDailyResetMode"
          :dailyResetHour="editDailyResetHour"
          :weeklyResetMode="editWeeklyResetMode"
          :weeklyResetDay="editWeeklyResetDay"
          :weeklyResetHour="editWeeklyResetHour"
          :resetTimezone="editResetTimezone"
          :quotaNotifyGlobalEnabled="quotaNotifyGlobalEnabled"
          :quotaNotifyDailyEnabled="quotaNotifyState.daily.enabled"
          :quotaNotifyDailyThreshold="quotaNotifyState.daily.threshold"
          :quotaNotifyDailyThresholdType="quotaNotifyState.daily.thresholdType"
          :quotaNotifyWeeklyEnabled="quotaNotifyState.weekly.enabled"
          :quotaNotifyWeeklyThreshold="quotaNotifyState.weekly.threshold"
          :quotaNotifyWeeklyThresholdType="quotaNotifyState.weekly.thresholdType"
          :quotaNotifyTotalEnabled="quotaNotifyState.total.enabled"
          :quotaNotifyTotalThreshold="quotaNotifyState.total.threshold"
          :quotaNotifyTotalThresholdType="quotaNotifyState.total.thresholdType"
          @update:totalLimit="editQuotaLimit = $event"
          @update:dailyLimit="editQuotaDailyLimit = $event"
          @update:weeklyLimit="editQuotaWeeklyLimit = $event"
          @update:dailyResetMode="editDailyResetMode = $event"
          @update:dailyResetHour="editDailyResetHour = $event"
          @update:weeklyResetMode="editWeeklyResetMode = $event"
          @update:weeklyResetDay="editWeeklyResetDay = $event"
          @update:weeklyResetHour="editWeeklyResetHour = $event"
          @update:resetTimezone="editResetTimezone = $event"
          @update:quotaNotifyDailyEnabled="quotaNotifyState.daily.enabled = $event"
          @update:quotaNotifyDailyThreshold="quotaNotifyState.daily.threshold = $event"
          @update:quotaNotifyDailyThresholdType="quotaNotifyState.daily.thresholdType = $event"
          @update:quotaNotifyWeeklyEnabled="quotaNotifyState.weekly.enabled = $event"
          @update:quotaNotifyWeeklyThreshold="quotaNotifyState.weekly.threshold = $event"
          @update:quotaNotifyWeeklyThresholdType="quotaNotifyState.weekly.thresholdType = $event"
          @update:quotaNotifyTotalEnabled="quotaNotifyState.total.enabled = $event"
          @update:quotaNotifyTotalThreshold="quotaNotifyState.total.threshold = $event"
          @update:quotaNotifyTotalThresholdType="quotaNotifyState.total.thresholdType = $event"
        />
      </div>
      <!-- 配额控制 (非 Anthropic apikey/bedrock) -->
      <div
        v-else-if="provider?.type === 'apikey' || provider?.type === 'bedrock'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4"
      >
        <div class="mb-3">
          <h3 class="input-label mb-0 text-base font-semibold">{{ t('admin.providers.quotaControl.title') }}</h3>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.providers.quotaLimitHint') }}
          </p>
        </div>
        <QuotaLimitCard
          :totalLimit="editQuotaLimit"
          :dailyLimit="editQuotaDailyLimit"
          :weeklyLimit="editQuotaWeeklyLimit"
          :dailyResetMode="editDailyResetMode"
          :dailyResetHour="editDailyResetHour"
          :weeklyResetMode="editWeeklyResetMode"
          :weeklyResetDay="editWeeklyResetDay"
          :weeklyResetHour="editWeeklyResetHour"
          :resetTimezone="editResetTimezone"
          :quotaNotifyGlobalEnabled="quotaNotifyGlobalEnabled"
          :quotaNotifyDailyEnabled="quotaNotifyState.daily.enabled"
          :quotaNotifyDailyThreshold="quotaNotifyState.daily.threshold"
          :quotaNotifyDailyThresholdType="quotaNotifyState.daily.thresholdType"
          :quotaNotifyWeeklyEnabled="quotaNotifyState.weekly.enabled"
          :quotaNotifyWeeklyThreshold="quotaNotifyState.weekly.threshold"
          :quotaNotifyWeeklyThresholdType="quotaNotifyState.weekly.thresholdType"
          :quotaNotifyTotalEnabled="quotaNotifyState.total.enabled"
          :quotaNotifyTotalThreshold="quotaNotifyState.total.threshold"
          :quotaNotifyTotalThresholdType="quotaNotifyState.total.thresholdType"
          @update:totalLimit="editQuotaLimit = $event"
          @update:dailyLimit="editQuotaDailyLimit = $event"
          @update:weeklyLimit="editQuotaWeeklyLimit = $event"
          @update:dailyResetMode="editDailyResetMode = $event"
          @update:dailyResetHour="editDailyResetHour = $event"
          @update:weeklyResetMode="editWeeklyResetMode = $event"
          @update:weeklyResetDay="editWeeklyResetDay = $event"
          @update:weeklyResetHour="editWeeklyResetHour = $event"
          @update:resetTimezone="editResetTimezone = $event"
          @update:quotaNotifyDailyEnabled="quotaNotifyState.daily.enabled = $event"
          @update:quotaNotifyDailyThreshold="quotaNotifyState.daily.threshold = $event"
          @update:quotaNotifyDailyThresholdType="quotaNotifyState.daily.thresholdType = $event"
          @update:quotaNotifyWeeklyEnabled="quotaNotifyState.weekly.enabled = $event"
          @update:quotaNotifyWeeklyThreshold="quotaNotifyState.weekly.threshold = $event"
          @update:quotaNotifyWeeklyThresholdType="quotaNotifyState.weekly.thresholdType = $event"
          @update:quotaNotifyTotalEnabled="quotaNotifyState.total.enabled = $event"
          @update:quotaNotifyTotalThreshold="quotaNotifyState.total.threshold = $event"
          @update:quotaNotifyTotalThresholdType="quotaNotifyState.total.thresholdType = $event"
        />
      </div>

      <!-- OpenAI OAuth 客户端访问策略 -->
      <div
        v-if="provider?.platform === 'openai' && provider?.type === 'oauth'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between gap-4">
          <div>
            <label class="input-label mb-0">{{ t('admin.providers.openai.clientPolicy') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.openai.clientPolicyDesc') }}
            </p>
          </div>
          <div class="w-64">
            <Select v-model="openAIOAuthClientPolicy" :options="openAIOAuthClientPolicyOptions" />
          </div>
        </div>
        <div
          v-if="openAIOAuthClientPolicy === 'codex_only'"
          class="mt-4 flex items-center justify-between border-l-2 border-gray-200 pl-4 dark:border-dark-600"
        >
          <div>
            <label class="input-label mb-0">{{ t('admin.providers.openai.codexCLIOnlyAllowClaudeCode') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.openai.codexCLIOnlyAllowClaudeCodeDesc') }}
            </p>
          </div>
          <Toggle v-model="codexCLIOnlyAllowClaudeCodeEnabled" variant="flush" off-tone="soft" />
        </div>
      </div>

      <!-- Codex 指纹收敛模式（仅 OpenAI OAuth） -->
      <div
        v-if="provider?.platform === 'openai' && provider?.type === 'oauth' && !isSparkShadow"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between gap-4">
          <div class="min-w-0">
            <label class="input-label mb-0">{{ t('admin.providers.openai.codexFingerprintMode') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.openai.codexFingerprintModeDesc') }}
            </p>
          </div>
          <div class="w-52 flex-shrink-0">
            <Select
              v-model="codexFingerprintMode"
              data-testid="edit-codex-fingerprint-mode-select"
              :options="codexFingerprintModeOptions"
            />
          </div>
        </div>
      </div>

      <!-- OpenAI 订阅档位手动覆盖（Plus/Pro/Free），仅 OAuth 非影子提供商 -->
      <div
        v-if="provider?.platform === 'openai' && provider?.type === 'oauth' && !isSparkShadow"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between gap-4">
          <div class="min-w-0">
            <label class="input-label mb-0">{{ t('admin.providers.openai.planType') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.openai.planTypeDesc') }}
            </p>
          </div>
          <div class="w-44 flex-shrink-0">
            <Select
              v-model="editPlanType"
              data-testid="openai-plan-type-select"
              :options="planTypeOptions"
            />
          </div>
        </div>
      </div>

      <!-- OAuth/COSY TLS 指纹伪装 -->
      <div
        v-if="showStandaloneTLSFingerprint"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{ t('admin.providers.quotaControl.tlsFingerprint.label') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.quotaControl.tlsFingerprint.hint') }}
            </p>
          </div>
          <Toggle v-model="tlsFingerprintEnabled" variant="flush" off-tone="soft" data-testid="edit-openai-tls-fingerprint-toggle" />
        </div>
        <Collapse :open="tlsFingerprintEnabled" unmount-on-hide>
          <div class="mt-3 space-y-3">
            <Select
              v-model="tlsFingerprintProfileId"
              data-testid="edit-openai-tls-fingerprint-profile"
              :options="tlsFingerprintProfileOptions"
            />
            <div v-if="supportsTLSFingerprintRouter">
              <Select
                v-model="tlsFingerprintRouterId"
                data-testid="edit-openai-tls-fingerprint-router"
                :options="tlsFingerprintRouterOptions"
              />
              <p class="input-hint">{{ t('admin.providers.quotaControl.tlsFingerprint.routerHint') }}</p>
            </div>
          </div>
        </Collapse>
      </div>

      <div
        v-if="provider?.platform === 'openai' && (provider?.type === 'oauth' || provider?.type === 'apikey')"
        class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4"
      >
        <OpenAICompactionCheckbox v-model="openAINativeCompactionV2Mode" test-id="edit-openai-native-compaction-v2-mode"
          :label="t('admin.providers.openai.nativeCompactV2Mode')" :hint="t('admin.providers.openai.nativeCompactV2ModeDesc')" />
        <OpenAICompactionCheckbox v-model="openAICompactMode" test-id="edit-openai-compact-mode"
          :label="t('admin.providers.openai.compactMode')" :hint="t('admin.providers.openai.compactModeDesc')" />
        <ProviderModelMappingEditor
          v-if="openAICompactMode !== 'force_off'"
          v-model="openAICompactModelMappings"
          :title="t('admin.providers.openai.compactModelMapping')"
          :hint="t('admin.providers.openai.compactModelMappingDesc')"
          :source-placeholder="t('admin.providers.fromModel')"
          :target-placeholder="t('admin.providers.toModel')"
        />
      </div>

      <div>
        <div class="flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{
              t('admin.providers.autoPauseOnExpired')
            }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.autoPauseOnExpiredDesc') }}
            </p>
          </div>
          <Toggle v-model="autoPauseOnExpired" variant="flush" off-tone="soft" />
        </div>
      </div>

      <div
        v-if="provider?.platform === 'openai'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4"
      >
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('admin.providers.autoPause5hDisabled') }}</label>
            <Toggle v-model="autoPause5hDisabled" variant="flush" off-tone="soft" data-testid="auto-pause-5h-disabled" />
          </div>
          <p class="input-hint">{{ t('admin.providers.autoPauseDisabledHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.providers.autoPause5hThreshold') }}</label>
          <input
            v-model.number="autoPause5hThreshold"
            type="number"
            min="0"
            max="100"
            step="0.1"
            class="input"
            :disabled="autoPause5hDisabled"
            data-testid="auto-pause-5h-threshold"
          />
          <p class="input-hint">{{ t('admin.providers.autoPauseThresholdHint') }}</p>
        </div>
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('admin.providers.autoPause7dDisabled') }}</label>
            <Toggle v-model="autoPause7dDisabled" variant="flush" off-tone="soft" data-testid="auto-pause-7d-disabled" />
          </div>
          <p class="input-hint">{{ t('admin.providers.autoPauseDisabledHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.providers.autoPause7dThreshold') }}</label>
          <input
            v-model.number="autoPause7dThreshold"
            type="number"
            min="0"
            max="100"
            step="0.1"
            class="input"
            :disabled="autoPause7dDisabled"
            data-testid="auto-pause-7d-threshold"
          />
          <p class="input-hint">{{ t('admin.providers.autoPauseThresholdHint') }}</p>
        </div>
      </div>

      <!-- 配额控制 (Anthropic OAuth/SetupToken: 亲和 + 窗口费用 + 会话 + RPM 等) -->
      <div
        v-if="provider?.platform === 'anthropic' && (provider?.type === 'oauth' || provider?.type === 'setup-token')"
        class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4"
      >
        <div class="mb-3">
          <h3 class="input-label mb-0 text-base font-semibold">{{ t('admin.providers.quotaControl.title') }}</h3>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.providers.quotaControl.hint') }}
          </p>
        </div>

        <!-- Window Cost Limit -->
        <div class="rounded-control border border-gray-200 p-4 dark:border-dark-600">
          <div class="mb-3 flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.providers.quotaControl.windowCost.label') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.quotaControl.windowCost.hint') }}
              </p>
            </div>
            <Toggle v-model="windowCostEnabled" variant="flush" off-tone="soft" />
          </div>

          <Collapse :open="windowCostEnabled" unmount-on-hide>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="input-label">{{ t('admin.providers.quotaControl.windowCost.limit') }}</label>
                <div class="relative">
                  <span class="input-icon text-gray-500 dark:text-gray-400">$</span>
                  <input
                    v-model.number="windowCostLimit"
                    type="number"
                    min="0"
                    step="1"
                    class="input input-has-icon input-icon-text"
                    :placeholder="t('admin.providers.quotaControl.windowCost.limitPlaceholder')"
                  />
                </div>
                <p class="input-hint">{{ t('admin.providers.quotaControl.windowCost.limitHint') }}</p>
              </div>
              <div>
                <label class="input-label">{{ t('admin.providers.quotaControl.windowCost.stickyReserve') }}</label>
                <div class="relative">
                  <span class="input-icon text-gray-500 dark:text-gray-400">$</span>
                  <input
                    v-model.number="windowCostStickyReserve"
                    type="number"
                    min="0"
                    step="1"
                    class="input input-has-icon input-icon-text"
                    :placeholder="t('admin.providers.quotaControl.windowCost.stickyReservePlaceholder')"
                  />
                </div>
                <p class="input-hint">{{ t('admin.providers.quotaControl.windowCost.stickyReserveHint') }}</p>
              </div>
            </div>
          </Collapse>
        </div>

        <!-- Session Limit -->
        <div class="rounded-control border border-gray-200 p-4 dark:border-dark-600">
          <div class="mb-3 flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.providers.quotaControl.sessionLimit.label') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.quotaControl.sessionLimit.hint') }}
              </p>
            </div>
            <Toggle v-model="sessionLimitEnabled" variant="flush" off-tone="soft" />
          </div>

          <Collapse :open="sessionLimitEnabled" unmount-on-hide>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="input-label">{{ t('admin.providers.quotaControl.sessionLimit.maxSessions') }}</label>
                <input
                  v-model.number="maxSessions"
                  type="number"
                  min="1"
                  step="1"
                  class="input"
                  :placeholder="t('admin.providers.quotaControl.sessionLimit.maxSessionsPlaceholder')"
                />
                <p class="input-hint">{{ t('admin.providers.quotaControl.sessionLimit.maxSessionsHint') }}</p>
              </div>
              <div>
                <label class="input-label">{{ t('admin.providers.quotaControl.sessionLimit.idleTimeout') }}</label>
                <div class="relative">
                  <input
                    v-model.number="sessionIdleTimeout"
                    type="number"
                    min="1"
                    step="1"
                    class="input pr-12"
                    :placeholder="t('admin.providers.quotaControl.sessionLimit.idleTimeoutPlaceholder')"
                  />
                  <span class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 dark:text-gray-400">{{ t('common.minutes') }}</span>
                </div>
                <p class="input-hint">{{ t('admin.providers.quotaControl.sessionLimit.idleTimeoutHint') }}</p>
              </div>
            </div>
          </Collapse>
        </div>

        <!-- RPM Limit -->
        <div class="rounded-control border border-gray-200 p-4 dark:border-dark-600">
          <div class="mb-3 flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.providers.quotaControl.rpmLimit.label') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.quotaControl.rpmLimit.hint') }}
              </p>
            </div>
            <Toggle v-model="rpmLimitEnabled" variant="flush" off-tone="soft" />
          </div>

          <Collapse :open="rpmLimitEnabled" unmount-on-hide>
            <div class="space-y-4">
              <div>
                <label class="input-label">{{ t('admin.providers.quotaControl.rpmLimit.baseRpm') }}</label>
                <input
                  v-model.number="baseRpm"
                  type="number"
                  min="1"
                  max="1000"
                  step="1"
                  class="input"
                  :placeholder="t('admin.providers.quotaControl.rpmLimit.baseRpmPlaceholder')"
                />
                <p class="input-hint">{{ t('admin.providers.quotaControl.rpmLimit.baseRpmHint') }}</p>
              </div>

              <div>
                <label class="input-label">{{ t('admin.providers.quotaControl.rpmLimit.strategy') }}</label>
                <div class="flex gap-2">
                  <button
                    type="button"
                    @click="rpmStrategy = 'tiered'"
                    :class="[
                      'flex-1 rounded-control px-3 py-2 text-sm font-medium transition',
                      rpmStrategy === 'tiered'
                        ? 'bg-primary-100 text-primary-700 dark:bg-primary-500/8 dark:text-primary-500'
                        : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
                    ]"
                  >
                    <div class="text-center">
                      <div>{{ t('admin.providers.quotaControl.rpmLimit.strategyTiered') }}</div>
                      <div class="mt-0.5 text-xs opacity-70">{{ t('admin.providers.quotaControl.rpmLimit.strategyTieredHint') }}</div>
                    </div>
                  </button>
                  <button
                    type="button"
                    @click="rpmStrategy = 'sticky_exempt'"
                    :class="[
                      'flex-1 rounded-control px-3 py-2 text-sm font-medium transition',
                      rpmStrategy === 'sticky_exempt'
                        ? 'bg-primary-100 text-primary-700 dark:bg-primary-500/8 dark:text-primary-500'
                        : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
                    ]"
                  >
                    <div class="text-center">
                      <div>{{ t('admin.providers.quotaControl.rpmLimit.strategyStickyExempt') }}</div>
                      <div class="mt-0.5 text-xs opacity-70">{{ t('admin.providers.quotaControl.rpmLimit.strategyStickyExemptHint') }}</div>
                    </div>
                  </button>
                </div>
              </div>

              <div v-if="rpmStrategy === 'tiered'" v-content-reveal>
                <label class="input-label">{{ t('admin.providers.quotaControl.rpmLimit.stickyBuffer') }}</label>
                <input
                  v-model.number="rpmStickyBuffer"
                  type="number"
                  min="1"
                  step="1"
                  class="input"
                  :placeholder="t('admin.providers.quotaControl.rpmLimit.stickyBufferPlaceholder')"
                />
                <p class="input-hint">{{ t('admin.providers.quotaControl.rpmLimit.stickyBufferHint') }}</p>
              </div>

            </div>
          </Collapse>

          <!-- 用户消息限速模式（独立于 RPM 开关，始终可见） -->
          <div class="mt-4">
            <label class="input-label">{{ t('admin.providers.quotaControl.rpmLimit.userMsgQueue') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400 mb-2">
              {{ t('admin.providers.quotaControl.rpmLimit.userMsgQueueHint') }}
            </p>
            <div class="flex space-x-2">
              <button type="button" v-for="opt in umqModeOptions" :key="opt.value"
                @click="userMsgQueueMode = opt.value"
                :class="[
                  'px-3 py-1.5 text-sm rounded-control border transition-colors',
                  userMsgQueueMode === opt.value
                    ? 'bg-primary-600 text-white border-primary-600'
                    : 'bg-white dark:bg-dark-700 text-gray-700 dark:text-gray-300 border-gray-300 dark:border-dark-500 hover:bg-gray-50 dark:hover:bg-dark-600'
                ]">
                {{ opt.label }}
              </button>
            </div>
          </div>
        </div>

        <!-- TLS Fingerprint -->
        <div class="rounded-control border border-gray-200 p-4 dark:border-dark-600">
          <div class="flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.providers.quotaControl.tlsFingerprint.label') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.quotaControl.tlsFingerprint.hint') }}
              </p>
            </div>
            <Toggle v-model="tlsFingerprintEnabled" variant="flush" off-tone="soft" />
          </div>
          <!-- Profile selector -->
          <div v-if="tlsFingerprintEnabled" class="mt-3 space-y-3">
            <Select v-model="tlsFingerprintProfileId" :options="tlsFingerprintProfileOptions" />
          </div>
        </div>

        <!-- Session ID Masking -->
        <div class="rounded-control border border-gray-200 p-4 dark:border-dark-600">
          <div class="flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.providers.quotaControl.sessionIdMasking.label') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.quotaControl.sessionIdMasking.hint') }}
              </p>
            </div>
            <Toggle v-model="sessionIdMaskingEnabled" variant="flush" off-tone="soft" />
          </div>
        </div>

        <!-- Cache TTL Override -->
        <div class="rounded-control border border-gray-200 p-4 dark:border-dark-600">
          <div class="flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.providers.quotaControl.cacheTTLOverride.label') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.quotaControl.cacheTTLOverride.hint') }}
              </p>
            </div>
            <Toggle v-model="cacheTTLOverrideEnabled" variant="flush" off-tone="soft" />
          </div>
          <Collapse :open="cacheTTLOverrideEnabled" unmount-on-hide>
            <div class="mt-3">
              <label class="input-label text-xs">{{ t('admin.providers.quotaControl.cacheTTLOverride.target') }}</label>
              <Select
                v-model="cacheTTLOverrideTarget"
                :options="cacheTTLOverrideTargetOptions"
                class="mt-1"
              />
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.quotaControl.cacheTTLOverride.targetHint') }}
              </p>
            </div>
          </Collapse>
        </div>

        <!-- Custom Base URL Relay -->
        <div class="rounded-control border border-gray-200 p-4 dark:border-dark-600">
          <div class="flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.providers.quotaControl.customBaseUrl.label') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.quotaControl.customBaseUrl.hint') }}
              </p>
            </div>
            <Toggle v-model="customBaseUrlEnabled" variant="flush" off-tone="soft" />
          </div>
          <Collapse :open="customBaseUrlEnabled" unmount-on-hide>
            <div class="mt-3">
              <input
                v-model="customBaseUrl"
                type="text"
                class="input"
                :placeholder="t('admin.providers.quotaControl.customBaseUrl.urlHint')"
              />
            </div>
          </Collapse>
        </div>
      </div>

      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div>
          <label class="input-label">{{ t('common.status') }}</label>
          <Select v-model="form.status" :options="statusOptions" />
        </div>

        <div v-if="provider?.platform === 'antigravity'" class="mt-3 flex items-center gap-2">
          <label class="flex cursor-pointer items-center gap-2">
            <input
              type="checkbox"
              v-model="allowOverages"
              class="h-4 w-4 rounded-compact border-gray-300 text-primary-500 focus:ring-primary-500 dark:border-dark-500"
            />
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.providers.allowOverages') }}
            </span>
          </label>
          <div class="group relative">
            <span
              class="inline-flex h-4 w-4 cursor-help items-center justify-center rounded-full bg-gray-200 text-xs text-gray-500 hover:bg-gray-300 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500"
            >
              ?
            </span>
            <div
              class="pointer-events-none absolute left-0 top-full z-tooltip mt-1.5 w-72 tooltip-panel rounded-compact px-3 py-2 text-xs opacity-0 transition-opacity group-hover:opacity-100"
            >
              {{ t('admin.providers.allowOveragesTooltip') }}
              <div
                class="tooltip-caret -top-1 left-3 border-l border-t"
              ></div>
            </div>
          </div>
        </div>
      </div>

      <!-- 分组选择 -->
      <GroupSelector
        v-model="form.group_ids"
        :groups="selectableGroups"
        data-tour="provider-form-groups"
      />

      <CodexTicketAccountSettings v-if="show && provider?.platform === 'openai' && provider?.type === 'oauth' && !isSparkShadow && provider.credentials?.auth_mode !== 'agentIdentity'" ref="ticketSettings" :ids="[provider.id]" :busy="submitting" />
      </fieldset>
    </form>

    <template #footer>
      <div v-if="provider" class="flex justify-end gap-3">
        <button @click="handleClose" type="button" class="btn btn-secondary">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="edit-provider-form"
          :disabled="submitting"
          class="btn btn-primary"
          data-tour="provider-form-submit"
        >
          <Icon
            name="loader"
            size="sm"
            :animate-on-hover="false"
            v-if="submitting"
            class="-ml-1 mr-2 h-4 w-4 animate-spin"
          />
          {{ submitting ? t('admin.providers.updating') : t('common.update') }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <!-- Mixed Channel Warning Dialog -->

</template>

<script setup lang="ts">
import OpenCodeGoProtocolRulesEditor from "./OpenCodeGoProtocolRulesEditor.vue"
import CodexTicketAccountSettings from '@/components/admin/provider/CodexTicketAccountSettings.vue'
import { applyOpenCodeGoProtocolRules, cloneOpenCodeGoProtocolRules, parseOpenCodeGoProtocolRules, defaultOpenCodeProtocolRules } from './credentialsBuilder'
import { vContentReveal } from '@/directives/contentReveal'
import Collapse from '@/components/common/Collapse.vue'

// 统一协议选择只保存原生集合，不在提供商侧配置转换。
const upstreamProtocols = ref<ProtocolID[] | undefined>(undefined)

import { normalizeLegacyOpenAIExtra, normalizeOpenAICompactMode } from '@/utils/openaiLegacyConfiguration'
import ProviderProtocolSelector from './ProviderProtocolSelector.vue'
import { loadProtocolCatalog, nativeProtocolOptions } from '@/api/admin/protocolCapabilities'
import type { ProtocolID } from '@/types'
import OpenAICompactionCheckbox from './OpenAICompactionCheckbox.vue'
import { ref, reactive, computed, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import { useQuotaNotifyState } from '@/composables/useQuotaNotifyState'
import type {
  Provider,
  Proxy,
  AdminGroup,
  Group,
  OpenAICompactMode,
  OpenAIOAuthClientPolicy,
  OllamaCloudUsageState,
  UpstreamUsageAdapter
} from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import UpstreamRequestIdHeaderField from '@/components/provider/UpstreamRequestIdHeaderField.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import ProviderModelMappingEditor from '@/components/provider/ProviderModelMappingEditor.vue'
import TempUnschedRulesEditor, { type TempUnschedRuleForm } from '@/components/provider/TempUnschedRulesEditor.vue'
import type { ModelMappingRow } from '@/utils/modelMappingRules'
import ProxySelector from '@/components/common/ProxySelector.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import CodexImageToolModeSelector from '@/components/provider/CodexImageToolModeSelector.vue'
import ModelWhitelistSelector from '@/components/provider/ModelWhitelistSelector.vue'
import QuotaLimitCard from '@/components/provider/QuotaLimitCard.vue'
import GrokBaseUrlPresets from '@/components/provider/GrokBaseUrlPresets.vue'
import CnBaseUrlPresets from '@/components/provider/CnBaseUrlPresets.vue'
import HeaderOverrideEditor from '@/components/provider/HeaderOverrideEditor.vue'
import OllamaCloudUsageSettings from '@/components/provider/OllamaCloudUsageSettings.vue'
import UpstreamUsageConfigEditor from '@/components/provider/UpstreamUsageConfigEditor.vue'
import {
  ANTIGRAVITY_PROJECT_ID_CREDENTIAL_KEY,
  applyAntigravityProjectID,
  applyHeaderOverride,
  applyInterceptWarmup,
  applyPlanType,
  buildPlanTypeOptions,
  readPlanType,
  isCustomGrokBaseUrl,
  isHeaderOverrideCapable,
  splitHeaderOverridesObject,
  validateHeaderOverrideRows,
  cnSupportsNativeResponses,
  defaultCNAdaptiveBaseUrls,
  defaultCNBaseUrl,
  HEADER_OVERRIDE_ENABLED_CREDENTIAL_KEY,
  HEADER_OVERRIDES_CREDENTIAL_KEY,
  type CnProviderMode,
  type CnApiProtocol,
  type CnNativeApiProtocol,
  type HeaderOverrideRow
} from '@/components/provider/credentialsBuilder'
import { formatDateTimeLocalInput, parseDateTimeLocalInput } from '@/utils/format'
import {
  applyCodexImageToolMode,
  readCodexImageToolMode,
  type CodexImageToolMode
} from '@/utils/codexImageToolMode'
import {
  VERTEX_LOCATION_OPTIONS,
  groupedProviderSelectOptions
} from '@/constants/provider'
import {
  OPENAI_WS_MODE_CTX_POOL,
  OPENAI_WS_MODE_HTTP_BRIDGE,
  OPENAI_WS_MODE_OFF,
  OPENAI_WS_MODE_PASSTHROUGH,
  isOpenAIWSModeEnabled,
  resolveOpenAIWSModeConcurrencyHintKey,
  type OpenAIWSMode,
  resolveOpenAIWSModeFromExtra
} from '@/utils/openaiWsMode'
import {
  getPresetMappingsByPlatform,
  getModelsByPlatform,
  commonErrorCodes,
  buildModelMappingObject,
  buildPersistedModelRestriction,
  splitQoderPersistedModelRestriction,
  splitPersistedModelRestriction,
  type QoderSite
} from '@/composables/useModelWhitelist'

interface Props {
  show: boolean
  provider: Provider | null
  proxies: Proxy[]
  groups: AdminGroup[]
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
  updated: [provider: Provider]
}>()

const { t } = useI18n()
const appStore = useAppStore()

// Spark 影子提供商(parent_provider_id 非空):代理恒继承母提供商,不可独立编辑(外审 B/P1),
// 故隐藏代理选择器。
const isSparkShadow = computed(() => props.provider?.parent_provider_id != null)

const handleOllamaCloudUsageUpdated = (state: OllamaCloudUsageState) => {
  if (props.provider) emit('updated', { ...props.provider, ollama_cloud_usage: state })
}

// Platform-specific hint for Base URL
const baseUrlHint = computed(() => {
  if (!props.provider) return t('admin.providers.baseUrlHint')
  if (props.provider.platform === 'openai') return t('admin.providers.openai.baseUrlHint')
  if (props.provider.platform === 'gemini' && geminiProviderType.value === 'third_party') {
    return t('admin.providers.gemini.connectionSource.thirdPartyBaseUrlHint')
  }
  if (props.provider.platform === 'gemini') return t('admin.providers.gemini.baseUrlHint')
  if (props.provider.platform === 'grok') return ''
  return t('admin.providers.baseUrlHint')
})

const antigravityPresetMappings = computed(() => getPresetMappingsByPlatform('antigravity'))
const bedrockPresets = computed(() => getPresetMappingsByPlatform('bedrock'))

// Model mapping type
// State
const submitting = ref(false)
const ticketSettings = ref<InstanceType<typeof CodexTicketAccountSettings>>()

// 已绑定停用分组仍可查看或移除；当前活跃列表的数据优先于账号旧快照。
const selectableGroups = computed(() => {
  const current: Group[] = [...props.groups]
  const known = new Set(current.map(group => group.id))
  for (const group of props.provider?.groups ?? []) {
    if (props.provider?.group_ids?.includes(group.id) && !known.has(group.id)) { current.push(group); known.add(group.id) }
  }
  return current
})
const editBaseUrl = ref('https://api.anthropic.com')
const editApiKey = ref('')
const upstreamUsageEnabled = ref(true)
const upstreamUsageAdapter = ref<UpstreamUsageAdapter>('sub2api')
const upstreamUsageBaseUrl = ref('')
const upstreamUsageWalletAccessToken = ref('')
const upstreamUsageWalletUserId = ref('')
type GeminiProviderType = 'official' | 'third_party'
const geminiProviderType = ref<GeminiProviderType>('official')
const geminiAIStudioTier = ref<'aistudio_free' | 'aistudio_paid'>('aistudio_free')
const geminiProviderTypeOptions = computed(() => [
  { value: 'official', label: t('admin.providers.gemini.connectionSource.official') },
  { value: 'third_party', label: t('admin.providers.gemini.connectionSource.thirdParty') }
])
const geminiAIStudioTierOptions = computed(() => [
  { value: 'aistudio_free', label: t('admin.providers.gemini.tier.aiStudio.free') },
  { value: 'aistudio_paid', label: t('admin.providers.gemini.tier.aiStudio.paid') }
])
const geminiProviderTypeHint = computed(() =>
  geminiProviderType.value === 'third_party'
    ? t('admin.providers.gemini.connectionSource.thirdPartyHint')
    : t('admin.providers.gemini.connectionSource.officialHint')
)

// 国产供应商提供商允许修正历史数据中的模式、协议和自定义端点。
const isCNApiKeyProvider = computed(
  () =>
    props.provider?.type === 'apikey' &&
    (props.provider.platform === 'kimi' ||
      props.provider.platform === 'zhipu' ||
      props.provider.platform === 'deepseek' || props.provider.platform === 'minimax' || props.provider.platform === 'opencode_go')
)
// 模板不支持联合类型断言，因此在脚本中收窄预设组件的平台类型。
const cnPresetPlatform = computed<'kimi' | 'zhipu' | 'deepseek' | 'minimax' | 'opencode_go'>(() => {
  const platform = props.provider?.platform
  if (platform === 'kimi' || platform === 'zhipu' || platform === 'deepseek' || platform === 'minimax' || platform === 'opencode_go') {
    return platform
  }
  return 'kimi'
})
const editApiProtocol = ref<CnApiProtocol>('adaptive')
const editProviderMode = ref<CnProviderMode>('payg')
const editOpenCodeRules = ref(cloneOpenCodeGoProtocolRules())
// 智谱团队版 Coding Plan 的组织/项目 ID，清空后随完整凭据更新一并移除。
const editZhipuOrganization = ref('')
const editZhipuProject = ref('')
const editAdaptiveBaseUrls = ref<Record<CnNativeApiProtocol, string>>({
  chat_completions: '',
  anthropic: '',
  responses: ''
})
// 回填窗口标志：syncFormFromProvider 会同步改写 editProviderMode / editApiProtocol，
// 而 watcher（pre-flush）在同步代码执行完之后才触发——若不抑制，会把刚恢复的
// 存储版 base_url（可能是用户自定义/中转地址）覆盖为官方预设并在下次保存时持久化。
// nextTick 后解除，此后用户主动切换模式/协议仍正常联动重置。
const syncingForm = ref(false)
const cnProviderModeOptions = computed<Array<{ value: CnProviderMode; labelKey: CnProviderMode }>>(
  () => {
    if (props.provider?.platform === 'opencode_go') return [{ value: 'zen', labelKey: 'zen' }, { value: 'go', labelKey: 'go' }]
    if (props.provider?.platform === 'deepseek') {
      return [{ value: 'payg', labelKey: 'payg' }]
    }
    return [
      { value: 'payg', labelKey: 'payg' },
      { value: 'coding', labelKey: 'coding' }
    ]
  }
)
const editAdaptiveProtocolOptions = computed<Array<{ value: CnNativeApiProtocol; labelKey: string }>>(() => {
  const opts: Array<{ value: CnNativeApiProtocol; labelKey: string }> = [
    { value: 'chat_completions', labelKey: 'chatCompletions' },
    { value: 'anthropic', labelKey: 'anthropic' }
  ]
  if (cnSupportsNativeResponses(props.provider?.platform ?? '')) opts.push({ value: 'responses', labelKey: 'responses' })
  return opts
})
watch(editApiProtocol, (protocol, previousProtocol) => {
  if (!isCNApiKeyProvider.value || syncingForm.value) return
  if (protocol === 'adaptive') {
    const defaults = defaultCNAdaptiveBaseUrls(cnPresetPlatform.value, editProviderMode.value)
    for (const item of editAdaptiveProtocolOptions.value) {
      if (!editAdaptiveBaseUrls.value[item.value]) editAdaptiveBaseUrls.value[item.value] = defaults[item.value]
    }
    if (previousProtocol !== 'adaptive' && editBaseUrl.value.trim()) {
      editAdaptiveBaseUrls.value[previousProtocol] = editBaseUrl.value.trim()
    }
    editBaseUrl.value = editAdaptiveBaseUrls.value.chat_completions
    return
  }
  if (previousProtocol === 'adaptive') {
    editBaseUrl.value = editAdaptiveBaseUrls.value[protocol] ||
      defaultCNBaseUrl(props.provider!.platform, editProviderMode.value, protocol)
    return
  }
  editBaseUrl.value = defaultCNBaseUrl(props.provider!.platform, editProviderMode.value, protocol)
})
watch(editProviderMode, (mode, previousMode) => {
  if (!isCNApiKeyProvider.value || syncingForm.value) return
  const effectiveMode = props.provider!.platform === 'deepseek' && mode === 'coding' ? 'payg' : mode
  if (effectiveMode !== mode) {
    editProviderMode.value = effectiveMode
    return
  }
  if (editApiProtocol.value === 'adaptive') {
    const previousDefaults = defaultCNAdaptiveBaseUrls(cnPresetPlatform.value, previousMode)
    const nextDefaults = defaultCNAdaptiveBaseUrls(cnPresetPlatform.value, mode)
    for (const item of editAdaptiveProtocolOptions.value) {
      if (!editAdaptiveBaseUrls.value[item.value] || editAdaptiveBaseUrls.value[item.value] === previousDefaults[item.value]) {
        editAdaptiveBaseUrls.value[item.value] = nextDefaults[item.value]
      }
    }
    editBaseUrl.value = editAdaptiveBaseUrls.value.chat_completions
    return
  }
  editBaseUrl.value = defaultCNBaseUrl(props.provider!.platform, mode, editApiProtocol.value)
})

// 端点预设同时更新模式和协议，保持表单字段一致。
function onCnPresetSelect(preset: { mode: CnProviderMode; protocol: CnApiProtocol; url: string }) {
  editProviderMode.value = preset.mode
  editApiProtocol.value = 'adaptive'
  if (preset.protocol !== 'adaptive') editAdaptiveBaseUrls.value[preset.protocol] = preset.url
  editBaseUrl.value = editAdaptiveBaseUrls.value.chat_completions
}

const isGeminiThirdPartyBaseUrl = (value: string) => {
  const normalized = value.trim()
  if (!normalized) return false
  try {
    return new URL(normalized).hostname.toLowerCase() !== 'generativelanguage.googleapis.com'
  } catch {
    return false
  }
}
// Bedrock credentials
const editBedrockAccessKeyId = ref('')
const editBedrockSecretAccessKey = ref('')
const editBedrockSessionToken = ref('')
const editBedrockRegion = ref('')
const editBedrockForceGlobal = ref(false)
const editBedrockApiKeyValue = ref('')
const editVertexProjectId = ref('')
const editVertexClientEmail = ref('')
const editVertexLocation = ref('us-central1')
const isBedrockAPIKeyMode = computed(() =>
  props.provider?.type === 'bedrock' &&
  (props.provider?.credentials as Record<string, unknown>)?.auth_mode === 'apikey'
)
const modelMappings = ref<ModelMappingRow[]>([])
const openAICompactModelMappings = ref<ModelMappingRow[]>([])
const modelRestrictionMode = ref<'whitelist' | 'mapping'>('whitelist')
const allowedModels = ref<string[]>([])
const qoderModelRestrictionConfigured = ref(false)
const qoderModelRestrictionTouched = ref(false)
const qoderModelWhitelistConfigured = ref(false)
const qoderModelWhitelistTouched = ref(false)
const qoderSite = ref<QoderSite>('global')
const DEFAULT_POOL_MODE_RETRY_COUNT = 3
const MAX_POOL_MODE_RETRY_COUNT = 10
const DEFAULT_POOL_MODE_RETRY_STATUS_CODES = [401, 403, 429]
const GROK_CLIENT_TOOL_CACHE_EXTRA_KEY = 'grok_client_tool_cache_enabled'
const poolModeEnabled = ref(false)
const poolModeRetryCount = ref(DEFAULT_POOL_MODE_RETRY_COUNT)
const poolModeRetryStatusCodesInput = ref('')

function parsePoolModeRetryStatusCodes(input: string): number[] {
  if (!input || !input.trim()) return []
  const seen = new Set<number>()
  const out: number[] = []
  for (const token of input.split(/[,\s]+/)) {
    const trimmed = token.trim()
    if (!trimmed) continue
    const n = Number(trimmed)
    if (!Number.isFinite(n) || !Number.isInteger(n)) continue
    if (n < 100 || n > 599) continue
    if (seen.has(n)) continue
    seen.add(n)
    out.push(n)
  }
  return out.sort((a, b) => a - b)
}

function formatPoolModeRetryStatusCodes(value: unknown): string {
  if (!Array.isArray(value)) return ''
  const out: number[] = []
  const seen = new Set<number>()
  for (const v of value) {
    const n = typeof v === 'string' ? Number(v.trim()) : Number(v)
    if (!Number.isFinite(n) || !Number.isInteger(n)) continue
    if (n < 100 || n > 599) continue
    if (seen.has(n)) continue
    seen.add(n)
    out.push(n)
  }
  return out.sort((a, b) => a - b).join(', ')
}
const customErrorCodesEnabled = ref(false)
const selectedErrorCodes = ref<number[]>([])
const customErrorCodeInput = ref<number | null>(null)
const headerOverrideEnabled = ref(false)
const headerOverrideRows = ref<HeaderOverrideRow[]>([])

const headerOverrideCapable = computed(
  () => !!props.provider && isHeaderOverrideCapable(props.provider.platform, props.provider.type)
)

// Grok OAuth 自定义上游地址（仅转发端点；OAuth 授权/令牌刷新不受影响）
const grokOAuthCustomBaseUrlEnabled = ref(false)
const grokOAuthBaseUrl = ref('')
// Grok Free OAuth 提供商默认使用客户端工具提示缓存；extra 中的显式 false 作为退出信号。
const grokClientToolCacheEnabled = ref(true)

const interceptWarmupRequests = ref(false)
const autoPauseOnExpired = ref(false)
const autoPause5hThreshold = ref<number | null>(null)
const autoPause7dThreshold = ref<number | null>(null)
const autoPause5hDisabled = ref(false)
const autoPause7dDisabled = ref(false)
// 上游ID：直接上游声明请求标识的响应头名，留空不记录。
const upstreamRequestIdHeader = ref('')
const readUpstreamRequestIdHeader = (extra: unknown): string => {
  const value = (extra as Record<string, unknown> | undefined)?.upstream_request_id_header
  return typeof value === 'string' ? value : ''
}
const allowOverages = ref(false) // For antigravity providers: enable AI Credits overages
const antigravityProjectId = ref('')
const antigravityModelRestrictionMode = ref<'whitelist' | 'mapping'>('whitelist')
const antigravityWhitelistModels = ref<string[]>([])
const antigravityModelMappings = ref<ModelMappingRow[]>([])
const isSyncingAntigravityUpstream = ref(false)
const tempUnschedEnabled = ref(false)
const providerSchedulingThresholdOverrideEnabled = ref(false)
const providerSchedulingThresholdOverrideValue = ref(100)
const PROVIDER_SCHEDULING_THRESHOLD_CREDENTIAL_KEY = 'provider_scheduling_threshold'
const supportsProviderSchedulingThresholdOverride = computed(() =>
  supportsProviderSchedulingThresholdOverridePlatform(props.provider?.platform)
)
const tempUnschedRules = ref<TempUnschedRuleForm[]>([])

// Quota control state (Anthropic OAuth/SetupToken only)
const windowCostEnabled = ref(false)
const windowCostLimit = ref<number | null>(null)
const windowCostStickyReserve = ref<number | null>(null)
const sessionLimitEnabled = ref(false)
const maxSessions = ref<number | null>(null)
const sessionIdleTimeout = ref<number | null>(null)
const rpmLimitEnabled = ref(false)
const baseRpm = ref<number | null>(null)
const rpmStrategy = ref<'tiered' | 'sticky_exempt'>('tiered')
const rpmStickyBuffer = ref<number | null>(null)
const userMsgQueueMode = ref('')
const umqModeOptions = computed(() => [
  { value: '', label: t('admin.providers.quotaControl.rpmLimit.umqModeOff') },
  { value: 'throttle', label: t('admin.providers.quotaControl.rpmLimit.umqModeThrottle') },
  { value: 'serialize', label: t('admin.providers.quotaControl.rpmLimit.umqModeSerialize') },
])
const tlsFingerprintEnabled = ref(false)
const tlsFingerprintProfileId = ref<number | null>(null)
const tlsFingerprintProfiles = ref<{ id: number; name: string }[]>([])
const tlsFingerprintRouterId = ref<number | null>(null)
const tlsFingerprintRouters = ref<{ id: number; name: string }[]>([])
const tlsFingerprintProfileOptions = computed(() => [
  { value: null, label: t('admin.providers.quotaControl.tlsFingerprint.defaultProfile') },
  ...(tlsFingerprintProfiles.value.length > 0
    ? [{ value: -1, label: t('admin.providers.quotaControl.tlsFingerprint.randomProfile') }]
    : []),
  ...tlsFingerprintProfiles.value.map((profile) => ({ value: profile.id, label: profile.name }))
])
const tlsFingerprintRouterOptions = computed(() => [
  { value: null, label: t('admin.providers.quotaControl.tlsFingerprint.noRouter') },
  ...tlsFingerprintRouters.value.map((router) => ({ value: router.id, label: router.name }))
])
const sessionIdMaskingEnabled = ref(false)
const cacheTTLOverrideEnabled = ref(false)
const cacheTTLOverrideTarget = ref<string>('5m')
const cacheTTLOverrideTargetOptions = [
  { value: '5m', label: '5m' },
  { value: '1h', label: '1h' }
]
const customBaseUrlEnabled = ref(false)
const customBaseUrl = ref('')

// OpenAI 自动透传开关（OAuth/API Key）
const openaiPassthroughEnabled = ref(false)
// OpenAI OAuth namespace 工具摊平兼容开关，缺省关闭即原样保留。
const openaiFlattenNamespacesEnabled = ref(false)
// OpenAI 订阅档位（Plus/Pro/Free）手动覆盖值,存于 credentials.plan_type;'' 表示清空/自动识别
const editPlanType = ref<string>('')
const openAICompactMode = ref<OpenAICompactMode>('force_on')
const openAINativeCompactionV2Mode = ref<OpenAICompactMode>('force_on')
// HTTP continuation 缺省关闭，只有管理员确认上游支持时才发送 previous_response_id。
const openAIResponsesContinuationSupported = ref(false)
// 图片回填默认关闭，只对 OpenAI API Key 提供商生效。
const openAIImagesURLToB64JSON = ref(false)
const openaiOAuthResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const openaiAPIKeyResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const codexCLIOnlyAllowClaudeCodeEnabled = ref(false)
const openAIOAuthClientPolicy = ref<OpenAIOAuthClientPolicy>('any')
type CodexFingerprintMode = 'off' | 'device' | 'session' | 'full'
const codexFingerprintMode = ref<CodexFingerprintMode>('off')
const codexImageToolMode = ref<CodexImageToolMode>('inherit')
type AnthropicAPIKeyAuthScheme = 'x_api_key' | 'authorization_bearer'
const anthropicPassthroughEnabled = ref(false)
const anthropicAPIKeyAuthScheme = ref<AnthropicAPIKeyAuthScheme>('x_api_key')
const webSearchEmulationMode = ref('default')
const webSearchEmulationOptions = computed(() => [
  { value: 'default', label: t('admin.providers.anthropic.webSearchDefault') },
  { value: 'enabled', label: t('admin.providers.anthropic.webSearchEnabled') },
  { value: 'disabled', label: t('admin.providers.anthropic.webSearchDisabled') }
])
const anthropicAPIKeyAuthSchemeOptions = computed(() => [
  { value: 'x_api_key', label: t('admin.providers.anthropic.apiKeyAuthSchemeXApiKey') },
  { value: 'authorization_bearer', label: t('admin.providers.anthropic.apiKeyAuthSchemeBearer') }
])
const webSearchGlobalEnabled = ref(false)
const {
  globalEnabled: quotaNotifyGlobalEnabled,
  state: quotaNotifyState,
  loadGlobalState: loadQuotaNotifyGlobal,
  loadFromExtra: loadQuotaNotifyFromExtra,
  writeToExtra: writeQuotaNotifyToExtra,
  reset: resetQuotaNotify,
} = useQuotaNotifyState()

const supportsTLSFingerprint = (provider: Provider | null | undefined) => {
  // TLS 指纹伪装开放给实际走 OAuth/COSY 客户端模拟链路的提供商。
  return !!provider && (
    (provider.platform === 'anthropic' && (provider.type === 'oauth' || provider.type === 'setup-token')) ||
    (provider.platform === 'openai' && provider.type === 'oauth') ||
    (provider.platform === 'qoder' && provider.type === 'cosy')
  )
}

const isQoderCosyProvider = computed(() =>
  props.provider?.platform === 'qoder' && props.provider?.type === 'cosy'
)
const originalQoderSite = computed<QoderSite>(() => {
  const credentials = props.provider?.credentials as Record<string, unknown> | undefined
  return credentials?.site === 'cn' ? 'cn' : 'global'
})
const qoderSiteChanged = computed(() => isQoderCosyProvider.value && qoderSite.value !== originalQoderSite.value)
const qoderAvailableModels = computed(() => getModelsByPlatform('qoder', qoderSite.value))
const supportsOAuthLikeModelRestriction = computed(() =>
  (props.provider?.platform === 'openai' && props.provider?.type === 'oauth') ||
  (props.provider?.platform === 'grok' && props.provider?.type === 'oauth') ||
  (props.provider?.platform === 'gemini' && props.provider?.type === 'oauth') ||
  isQoderCosyProvider.value
)
const isAnthropicOAuthLikeProvider = computed(() =>
  props.provider?.platform === 'anthropic' &&
  (props.provider?.type === 'oauth' || props.provider?.type === 'setup-token')
)
const supportsTLSFingerprintRouter = computed(() =>
  props.provider?.platform === 'openai' && props.provider?.type === 'oauth'
)
const showStandaloneTLSFingerprint = computed(() =>
  supportsTLSFingerprint(props.provider) && !isAnthropicOAuthLikeProvider.value
)

// Load global feature states once
adminAPI.settings.getWebSearchEmulationConfig().then(cfg => {
  webSearchGlobalEnabled.value = cfg?.enabled === true && (cfg?.providers?.length ?? 0) > 0
}).catch(() => { webSearchGlobalEnabled.value = false })

loadQuotaNotifyGlobal()
const editQuotaLimit = ref<number | null>(null)
const editQuotaDailyLimit = ref<number | null>(null)
const editQuotaWeeklyLimit = ref<number | null>(null)
const editDailyResetMode = ref<'rolling' | 'fixed' | null>(null)
const editDailyResetHour = ref<number | null>(null)
const editWeeklyResetMode = ref<'rolling' | 'fixed' | null>(null)
const editWeeklyResetDay = ref<number | null>(null)
const editWeeklyResetHour = ref<number | null>(null)
const editResetTimezone = ref<string | null>(null)
const codexFingerprintModeOptions = computed(() => [
  { value: 'off' as CodexFingerprintMode, label: t('admin.providers.openai.codexFingerprintOff') },
  { value: 'device' as CodexFingerprintMode, label: t('admin.providers.openai.codexFingerprintDevice') },
  { value: 'session' as CodexFingerprintMode, label: t('admin.providers.openai.codexFingerprintSession') },
  { value: 'full' as CodexFingerprintMode, label: t('admin.providers.openai.codexFingerprintFull') },
])

const openAIWSModeOptions = computed(() => [
  { value: OPENAI_WS_MODE_OFF, label: t('admin.providers.openai.wsModeOff') },
  { value: OPENAI_WS_MODE_CTX_POOL, label: t('admin.providers.openai.wsModeCtxPool') },
  { value: OPENAI_WS_MODE_PASSTHROUGH, label: t('admin.providers.openai.wsModePassthrough') },
  { value: OPENAI_WS_MODE_HTTP_BRIDGE, label: t('admin.providers.openai.wsModeHttpBridge') }
])
const openaiResponsesWebSocketV2Mode = computed({
  get: () => {
    if (props.provider?.type === 'apikey') {
      return openaiAPIKeyResponsesWebSocketV2Mode.value
    }
    return openaiOAuthResponsesWebSocketV2Mode.value
  },
  set: (mode: OpenAIWSMode) => {
    if (props.provider?.type === 'apikey') {
      openaiAPIKeyResponsesWebSocketV2Mode.value = mode
      return
    }
    openaiOAuthResponsesWebSocketV2Mode.value = mode
  }
})
const openAIWSModeConcurrencyHintKey = computed(() =>
  resolveOpenAIWSModeConcurrencyHintKey(openaiResponsesWebSocketV2Mode.value)
)

// OpenAI 订阅档位手动覆盖选项(清空 + Plus/Pro/Free;别名/自定义值友好显示且保留 canonical)
const planTypeOptions = computed(() =>
  buildPlanTypeOptions(editPlanType.value, t('admin.providers.openai.planTypeClear'))
)
const openAIOAuthClientPolicyOptions = computed(() => [
  { value: 'any', label: t('admin.providers.openai.clientPolicyAny') },
  { value: 'codex_only', label: t('admin.providers.openai.clientPolicyCodexOnly') },
  { value: 'tls_router_matched_only', label: t('admin.providers.openai.clientPolicyTLSRouterMatchedOnly') }
])

// Computed: current preset mappings based on platform
const presetMappings = computed(() =>
  getPresetMappingsByPlatform(
    props.provider?.platform || 'anthropic',
    isQoderCosyProvider.value ? qoderSite.value : undefined
  )
)

// Computed: default base URL based on platform
const defaultBaseUrl = computed(() => {
  if (props.provider?.platform === 'openai') return 'https://api.openai.com'
  if (props.provider?.platform === 'gemini') return 'https://generativelanguage.googleapis.com'
  if (props.provider?.platform === 'grok') return 'https://api.x.ai/v1'
  // CN 供应商：按当前模式/协议回落到官方预设（清空输入框提交时使用），
  // 不能落到 anthropic 默认值（会被当 CC base 拼出错误端点）。
  if (
    props.provider?.platform === 'kimi' ||
    props.provider?.platform === 'zhipu' ||
    props.provider?.platform === 'deepseek'
  ) {
    return defaultCNBaseUrl(props.provider.platform, editProviderMode.value, editApiProtocol.value)
  }
  return 'https://api.anthropic.com'
})

watch(geminiProviderType, (providerType) => {
  if (props.provider?.platform !== 'gemini' || providerType !== 'third_party') return
  // 切换到第三方来源时，不能把官方默认端点误当成第三方地址提交。
  if (!isGeminiThirdPartyBaseUrl(editBaseUrl.value)) {
    editBaseUrl.value = ''
  }
})

const form = reactive({
  name: '',
  notes: '',
  proxy_id: null as number | null,
  concurrency: 1,
  load_factor: null as number | null,
  priority: 1,
  rate_multiplier: 1,
  status: 'active' as 'active' | 'inactive' | 'error',
  group_ids: [] as number[],
  expires_at: null as number | null
})

const statusOptions = computed(() => {
  const options = [
    { value: 'active', label: t('common.active') },
    { value: 'inactive', label: t('common.inactive') }
  ]
  if (form.status === 'error') {
    options.push({ value: 'error', label: t('admin.providers.status.error') })
  }
  return options
})
const vertexLocationOptions = groupedProviderSelectOptions(VERTEX_LOCATION_OPTIONS)

const expiresAtInput = computed({
  get: () => formatDateTimeLocal(form.expires_at),
  set: (value: string) => {
    form.expires_at = parseDateTimeLocal(value)
  }
})

// Watchers
const normalizePoolModeRetryCount = (value: number) => {
  if (!Number.isFinite(value)) {
    return DEFAULT_POOL_MODE_RETRY_COUNT
  }
  const normalized = Math.trunc(value)
  if (normalized < 0) {
    return 0
  }
  if (normalized > MAX_POOL_MODE_RETRY_COUNT) {
    return MAX_POOL_MODE_RETRY_COUNT
  }
  return normalized
}

const hydrateModelRestrictionFromMapping = (
  existingMappings?: Record<string, string>,
  rawWhitelist?: unknown
) => {
  const parsed = splitPersistedModelRestriction(existingMappings, rawWhitelist)
  allowedModels.value = parsed.allowedModels
  modelMappings.value = parsed.modelMappings
  modelRestrictionMode.value = parsed.modelMappings.length > 0 ? 'mapping' : 'whitelist'
}

const hasConfiguredModelRestriction = (
  existingMappings?: Record<string, string>,
  rawWhitelist?: unknown
) =>
  (!!existingMappings && typeof existingMappings === 'object' && Object.keys(existingMappings).length > 0) ||
  Array.isArray(rawWhitelist)

const hydrateQoderModelRestrictionFromMapping = (
  existingMappings?: Record<string, string>,
  rawWhitelist?: unknown
) => {
  qoderModelRestrictionConfigured.value = hasConfiguredModelRestriction(existingMappings, rawWhitelist)
  qoderModelWhitelistConfigured.value = Array.isArray(rawWhitelist)
  qoderModelRestrictionTouched.value = false
  qoderModelWhitelistTouched.value = false
  if (!qoderModelRestrictionConfigured.value) {
    allowedModels.value = []
    modelMappings.value = []
    modelRestrictionMode.value = 'mapping'
    return
  }

  const parsed = splitQoderPersistedModelRestriction(existingMappings, rawWhitelist)
  allowedModels.value = parsed.allowedModels
  modelMappings.value = parsed.modelMappings
  modelRestrictionMode.value = modelMappings.value.length > 0 ? 'mapping' : 'whitelist'
}

const applyPersistedModelRestriction = (credentials: Record<string, unknown>) => {
  // 普通提供商将请求侧映射与最终白名单分开持久化。
  // 这里即使白名单为空，也要显式写入 []，避免后端回退到 legacy 的自映射白名单解析。
  const persisted = buildPersistedModelRestriction(allowedModels.value, modelMappings.value)
  if (persisted.modelMapping) {
    credentials.model_mapping = persisted.modelMapping
  } else {
    delete credentials.model_mapping
  }
  credentials.model_whitelist = persisted.modelWhitelist
}

const applyQoderModelRestriction = (credentials: Record<string, unknown>) => {
  if (!qoderModelRestrictionConfigured.value && !qoderModelRestrictionTouched.value) {
    delete credentials.model_mapping
    delete credentials.model_whitelist
    return
  }

  const persisted = buildPersistedModelRestriction(
    qoderModelWhitelistConfigured.value || qoderModelWhitelistTouched.value ? allowedModels.value : [],
    modelMappings.value
  )
  if (persisted.modelMapping) {
    credentials.model_mapping = persisted.modelMapping
  } else {
    delete credentials.model_mapping
  }
  credentials.model_whitelist = persisted.modelWhitelist
}

const applyOpenAIModelMappingCredentials = (credentials: Record<string, unknown>) => {
  const shouldApplyModelMapping = true

  if (shouldApplyModelMapping) {
    if (isSparkShadow.value) {
      // Spark 影子提供商只允许持久化请求侧映射，不能写入独立白名单字段。
      const modelMapping = buildModelMappingObject(modelRestrictionMode.value, allowedModels.value, modelMappings.value)
      if (modelMapping) {
        credentials.model_mapping = modelMapping
      } else {
        delete credentials.model_mapping
      }
    } else {
      // OpenAI OAuth 与普通提供商一样，将请求映射和最终白名单分开保存。
      applyPersistedModelRestriction(credentials)
    }
  } else if (!credentials.model_mapping) {
    delete credentials.model_mapping
  }

  const compactModelMapping = buildModelMappingObject('mapping', [], openAICompactModelMappings.value)
  if (compactModelMapping) {
    credentials.compact_model_mapping = compactModelMapping
  } else {
    delete credentials.compact_model_mapping
  }
}

const syncFormFromProvider = (newProvider: Provider | null) => {
  upstreamProtocols.value = Array.isArray(newProvider?.credentials?.upstream_protocols) ? [...newProvider.credentials.upstream_protocols] as ProtocolID[] : undefined

  if (!newProvider) {
    return
  }
  // 进入回填窗口：抑制 CN 模式/协议 watcher 联动重置 base_url（见 syncingForm 注释）。
  syncingForm.value = true
  void nextTick(() => {
    syncingForm.value = false
  })
  form.name = newProvider.name
  form.notes = newProvider.notes || ''
  form.proxy_id = newProvider.proxy_id
  form.concurrency = newProvider.concurrency
  form.load_factor = newProvider.load_factor ?? null
  form.priority = newProvider.priority
  form.rate_multiplier = newProvider.rate_multiplier ?? 1
  form.status = (newProvider.status === 'active' || newProvider.status === 'inactive' || newProvider.status === 'error')
    ? newProvider.status
    : 'active'
  form.group_ids = newProvider.group_ids || []
  form.expires_at = newProvider.expires_at ?? null

  // Load intercept warmup requests setting (applies to all provider types)
  const credentials = newProvider.credentials as Record<string, unknown> | undefined
	editOpenCodeRules.value = parseOpenCodeGoProtocolRules(credentials?.protocol_rules) || cloneOpenCodeGoProtocolRules(defaultOpenCodeProtocolRules(credentials?.provider_mode === 'zen' ? 'zen' : 'go'))
  editZhipuOrganization.value = ''
  editZhipuProject.value = ''
  geminiProviderType.value = 'official'
  geminiAIStudioTier.value = 'aistudio_free'
  if (newProvider.platform === 'gemini' && newProvider.type === 'apikey') {
    geminiProviderType.value = credentials?.provider_type === 'third_party' ? 'third_party' : 'official'
    if (credentials?.tier_id === 'aistudio_paid') {
      geminiAIStudioTier.value = 'aistudio_paid'
    }
  }
  qoderSite.value = newProvider.platform === 'qoder' && credentials?.site === 'cn' ? 'cn' : 'global'
  interceptWarmupRequests.value = credentials?.intercept_warmup_requests === true
  autoPauseOnExpired.value = newProvider.auto_pause_on_expired === true
  editVertexProjectId.value = ''
  editVertexClientEmail.value = ''
  editVertexLocation.value = 'us-central1'

  // Load mixed scheduling setting (only for antigravity providers)
  allowOverages.value = false
  const extra = newProvider.extra as Record<string, unknown> | undefined
  upstreamUsageEnabled.value = true
  upstreamUsageAdapter.value = 'sub2api'
  upstreamUsageBaseUrl.value = ''
  upstreamUsageWalletAccessToken.value = ''
  upstreamUsageWalletUserId.value = ''
  if (newProvider.type === 'apikey') {
    const rawUsageConfig = extra?.upstream_usage_query as Record<string, unknown> | undefined
    if (rawUsageConfig && typeof rawUsageConfig === 'object') {
      upstreamUsageEnabled.value = rawUsageConfig.enabled !== false
      if (rawUsageConfig.adapter === 'new_api' || rawUsageConfig.adapter === 'zivv') {
        upstreamUsageAdapter.value = rawUsageConfig.adapter
      }
      if (typeof rawUsageConfig.base_url === 'string') upstreamUsageBaseUrl.value = rawUsageConfig.base_url
    }
    if (typeof credentials?.new_api_user_id === 'string' || typeof credentials?.new_api_user_id === 'number') {
      upstreamUsageWalletUserId.value = String(credentials.new_api_user_id)
    }
  }
  allowOverages.value = extra?.allow_overages === true
  autoPause5hThreshold.value = typeof extra?.auto_pause_5h_threshold === 'number' ? extra.auto_pause_5h_threshold * 100 : null
  autoPause7dThreshold.value = typeof extra?.auto_pause_7d_threshold === 'number' ? extra.auto_pause_7d_threshold * 100 : null
  autoPause5hDisabled.value = extra?.auto_pause_5h_disabled === true
  autoPause7dDisabled.value = extra?.auto_pause_7d_disabled === true
	upstreamRequestIdHeader.value = readUpstreamRequestIdHeader(extra)

  // 加载 OpenAI OAuth、SetupToken 和 API Key 提供商的透传设置。
  openaiPassthroughEnabled.value = false
  openaiFlattenNamespacesEnabled.value = false
  editPlanType.value = ''
  openAICompactMode.value = 'force_on'
  openAINativeCompactionV2Mode.value = 'force_on'
  openAIResponsesContinuationSupported.value = false
  openAIImagesURLToB64JSON.value = false
  openAICompactModelMappings.value = []
  openaiOAuthResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
  openaiAPIKeyResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
  codexCLIOnlyAllowClaudeCodeEnabled.value = false
  openAIOAuthClientPolicy.value = 'any'
  codexFingerprintMode.value = 'off'
  codexImageToolMode.value = 'inherit'
  anthropicPassthroughEnabled.value = false
  anthropicAPIKeyAuthScheme.value = 'x_api_key'
  webSearchEmulationMode.value = 'default'
  if (newProvider.platform === 'openai' && (newProvider.type === 'oauth' || newProvider.type === 'setup-token' || newProvider.type === 'apikey')) {
    openaiPassthroughEnabled.value = extra?.openai_passthrough === true || extra?.openai_oauth_passthrough === true
    openaiFlattenNamespacesEnabled.value =
      newProvider.type === 'oauth' && extra?.openai_responses_flatten_namespaces === true
    // plan_type 手动覆盖仅 OAuth 有实际调度语义(IsOpenAIChatGPTSubscription 要求 oauth),故只对 oauth 回填
    editPlanType.value = newProvider.type === 'oauth'
      ? readPlanType(newProvider.credentials as Record<string, unknown> | undefined)
      : ''
    openAICompactMode.value = normalizeOpenAICompactMode(extra?.openai_compact_mode)
    openAINativeCompactionV2Mode.value = normalizeOpenAICompactMode(extra?.openai_native_compaction_v2_mode)
    if (newProvider.type === 'apikey') {
      openAIResponsesContinuationSupported.value = extra?.openai_responses_continuation_supported === true
      openAIImagesURLToB64JSON.value = extra?.images_url_to_b64_json === true
    }
    codexImageToolMode.value = readCodexImageToolMode(extra)
    openaiOAuthResponsesWebSocketV2Mode.value = resolveOpenAIWSModeFromExtra(extra, {
      modeKey: 'openai_oauth_responses_websockets_v2_mode',
      enabledKey: 'openai_oauth_responses_websockets_v2_enabled',
      fallbackEnabledKeys: ['responses_websockets_v2_enabled', 'openai_ws_enabled'],
      defaultMode: OPENAI_WS_MODE_OFF
    })
    openaiAPIKeyResponsesWebSocketV2Mode.value = resolveOpenAIWSModeFromExtra(extra, {
      modeKey: 'openai_apikey_responses_websockets_v2_mode',
      enabledKey: 'openai_apikey_responses_websockets_v2_enabled',
      fallbackEnabledKeys: ['responses_websockets_v2_enabled', 'openai_ws_enabled'],
      defaultMode: OPENAI_WS_MODE_OFF
    })
    if (newProvider.type === 'oauth') {
      openAIOAuthClientPolicy.value = normalizeOpenAIOAuthClientPolicy(extra?.openai_oauth_client_policy, extra?.codex_cli_only)
      codexCLIOnlyAllowClaudeCodeEnabled.value =
        Array.isArray(extra?.codex_cli_only_allowed_clients) &&
        (extra.codex_cli_only_allowed_clients as unknown[]).includes('claude_code')
    }
    if (newProvider.type === 'oauth') {
      const fpMode = extra?.codex_fingerprint_mode as string | undefined
      codexFingerprintMode.value = (['off', 'device', 'session', 'full'].includes(fpMode || '')
        ? fpMode as CodexFingerprintMode
        : 'off')
    }
    const credentials = newProvider.credentials as Record<string, unknown> | undefined
    const compactMappings = credentials?.compact_model_mapping as Record<string, string> | undefined
    if (compactMappings && typeof compactMappings === 'object') {
      openAICompactModelMappings.value = Object.entries(compactMappings).map(([from, to]) => ({ from, to }))
    }
  }
  if (newProvider.platform === 'anthropic' && newProvider.type === 'apikey') {
    anthropicPassthroughEnabled.value = extra?.anthropic_passthrough === true
    anthropicAPIKeyAuthScheme.value = extra?.anthropic_apikey_auth_scheme === 'authorization_bearer'
      ? 'authorization_bearer'
      : 'x_api_key'
    // 三态：string "default"/"enabled"/"disabled"，向后兼容旧 bool
    const wsVal = extra?.web_search_emulation
    if (wsVal === 'enabled' || wsVal === 'disabled') {
      webSearchEmulationMode.value = wsVal
    } else if (wsVal === true) {
      webSearchEmulationMode.value = 'enabled'
    } else {
      webSearchEmulationMode.value = 'default'
    }
  }

  // Load quota limit for apikey/bedrock providers (bedrock quota is also loaded in its own branch above)
  if (newProvider.type === 'apikey' || newProvider.type === 'bedrock') {
    const quotaVal = extra?.quota_limit as number | undefined
    editQuotaLimit.value = (quotaVal && quotaVal > 0) ? quotaVal : null
    const dailyVal = extra?.quota_daily_limit as number | undefined
    editQuotaDailyLimit.value = (dailyVal && dailyVal > 0) ? dailyVal : null
    const weeklyVal = extra?.quota_weekly_limit as number | undefined
    editQuotaWeeklyLimit.value = (weeklyVal && weeklyVal > 0) ? weeklyVal : null
    // Load quota reset mode config
    editDailyResetMode.value = (extra?.quota_daily_reset_mode as 'rolling' | 'fixed') || null
    editDailyResetHour.value = (extra?.quota_daily_reset_hour as number) ?? null
    editWeeklyResetMode.value = (extra?.quota_weekly_reset_mode as 'rolling' | 'fixed') || null
    editWeeklyResetDay.value = (extra?.quota_weekly_reset_day as number) ?? null
    editWeeklyResetHour.value = (extra?.quota_weekly_reset_hour as number) ?? null
    editResetTimezone.value = (extra?.quota_reset_timezone as string) || null
    // Load quota notify config
    loadQuotaNotifyFromExtra(extra)
  } else {
    editQuotaLimit.value = null
    editQuotaDailyLimit.value = null
    editQuotaWeeklyLimit.value = null
    editDailyResetMode.value = null
    editDailyResetHour.value = null
    editWeeklyResetMode.value = null
    editWeeklyResetDay.value = null
    editWeeklyResetHour.value = null
    editResetTimezone.value = null
    resetQuotaNotify()
  }

  // Load antigravity model mapping (Antigravity 只支持映射模式)
  if (newProvider.platform === 'antigravity') {
    const credentials = newProvider.credentials as Record<string, unknown> | undefined
    const configuredProjectID = credentials?.[ANTIGRAVITY_PROJECT_ID_CREDENTIAL_KEY]
    antigravityProjectId.value = typeof configuredProjectID === 'string' ? configuredProjectID : ''

    // Antigravity 始终使用映射模式
    antigravityModelRestrictionMode.value = 'mapping'
    antigravityWhitelistModels.value = Array.isArray(credentials?.model_whitelist) ? [...credentials.model_whitelist as string[]] : []

    // 从 model_mapping 读取映射配置
    const rawAgMapping = credentials?.model_mapping as Record<string, string> | undefined
    if (rawAgMapping && typeof rawAgMapping === 'object') {
      const entries = Object.entries(rawAgMapping)
      // 无论是白名单样式(key===value)还是真正的映射，都统一转换为映射列表
      antigravityModelMappings.value = entries.map(([from, to]) => ({ from, to }))
    } else {
      // 兼容旧数据：从 model_whitelist 读取，转换为映射格式
      const rawWhitelist = credentials?.model_whitelist
      if (Array.isArray(rawWhitelist) && rawWhitelist.length > 0) {
        antigravityModelMappings.value = rawWhitelist
          .map((v) => String(v).trim())
          .filter((v) => v.length > 0)
          .map((m) => ({ from: m, to: m }))
      } else {
        antigravityModelMappings.value = []
      }
    }
  } else {
    antigravityProjectId.value = ''
    antigravityModelRestrictionMode.value = 'mapping'
    antigravityWhitelistModels.value = []
    antigravityModelMappings.value = []
  }

  // Load quota control settings (Anthropic OAuth/SetupToken only)
  loadQuotaControlSettings(newProvider)

  loadTempUnschedRules(credentials)
  loadProviderSchedulingThresholdOverride(newProvider.platform, credentials)

  // 加载支持的平台 API Key 与 Grok API Key/OAuth 提供商的请求头覆写状态。
  headerOverrideEnabled.value = false
  headerOverrideRows.value = []
  if (newProvider.credentials && isHeaderOverrideCapable(newProvider.platform, newProvider.type)) {
    const overrideCreds = newProvider.credentials as Record<string, unknown>
    headerOverrideEnabled.value = overrideCreds[HEADER_OVERRIDE_ENABLED_CREDENTIAL_KEY] === true
    headerOverrideRows.value = splitHeaderOverridesObject(
      overrideCreds[HEADER_OVERRIDES_CREDENTIAL_KEY]
    )
  }

  // 加载 Grok OAuth 自定义上游地址状态（存储的官方地址视同未定制）。
  grokOAuthCustomBaseUrlEnabled.value = false
  grokOAuthBaseUrl.value = ''
  const grokClientToolCacheSetting =
    newProvider.platform === 'grok' && newProvider.type === 'oauth'
      ? newProvider.extra?.[GROK_CLIENT_TOOL_CACHE_EXTRA_KEY]
      : undefined
  grokClientToolCacheEnabled.value =
    newProvider.platform === 'grok' &&
    newProvider.type === 'oauth' &&
    (grokClientToolCacheSetting === undefined || grokClientToolCacheSetting === true)
  if (newProvider.platform === 'grok' && newProvider.type === 'oauth' && newProvider.credentials) {
    const grokCreds = newProvider.credentials as Record<string, unknown>
    if (isCustomGrokBaseUrl(grokCreds.base_url)) {
      grokOAuthCustomBaseUrlEnabled.value = true
      grokOAuthBaseUrl.value = (grokCreds.base_url as string).trim()
    }
  }

  // Initialize API Key fields for apikey type
  if (newProvider.type === 'apikey' && newProvider.credentials) {
    const credentials = newProvider.credentials as Record<string, unknown>
    // 国产供应商：读取 provider_mode 与 api_protocol 作为可编辑初始值
    // （编辑弹窗允许修正两者，用于修复早期存错默认值的提供商）。
    if (newProvider.platform === 'kimi' || newProvider.platform === 'zhipu' || newProvider.platform === 'deepseek' || newProvider.platform === 'minimax' || newProvider.platform === 'opencode_go') {
      editProviderMode.value = newProvider.platform === 'opencode_go' ? (credentials.provider_mode === 'zen' ? 'zen' : 'go') : credentials.provider_mode === 'coding' ? 'coding' : 'payg'
      const storedProtocol = credentials.upstream_protocols !== undefined ? 'adaptive' : credentials.api_protocol
      editApiProtocol.value =
        storedProtocol === 'adaptive' ||
        storedProtocol === 'chat_completions' ||
        storedProtocol === 'anthropic' ||
        storedProtocol === 'responses'
          ? storedProtocol
          : 'chat_completions'
      if (!cnSupportsNativeResponses(newProvider.platform) && editApiProtocol.value === 'responses') {
        editApiProtocol.value = 'chat_completions'
      }
      const adaptiveDefaults = defaultCNAdaptiveBaseUrls(newProvider.platform, editProviderMode.value)
      const storedBaseUrls = (credentials.api_base_urls as Record<string, unknown> | undefined) || {}
      const legacyBaseUrl = typeof credentials.base_url === 'string' ? credentials.base_url.trim() : ''
      const storedChatBaseUrl = typeof storedBaseUrls.chat_completions === 'string'
        ? storedBaseUrls.chat_completions.trim()
        : ''
      const storedAnthropicBaseUrl = typeof storedBaseUrls.anthropic === 'string'
        ? storedBaseUrls.anthropic.trim()
        : ''
      const storedResponsesBaseUrl = typeof storedBaseUrls.responses === 'string'
        ? storedBaseUrls.responses.trim()
        : ''
      const nextAdaptiveBaseUrls: Record<CnNativeApiProtocol, string> = {
        chat_completions: storedChatBaseUrl || adaptiveDefaults.chat_completions,
        anthropic: storedAnthropicBaseUrl || adaptiveDefaults.anthropic,
        responses: storedResponsesBaseUrl || adaptiveDefaults.responses
      }
      const legacyProtocol: CnNativeApiProtocol = editApiProtocol.value === 'anthropic'
        ? 'anthropic'
        : editApiProtocol.value === 'responses'
          ? 'responses'
          : 'chat_completions'
      const storedLegacyBaseUrl = legacyProtocol === 'anthropic'
        ? storedAnthropicBaseUrl
        : legacyProtocol === 'responses'
          ? storedResponsesBaseUrl
          : storedChatBaseUrl
      if (legacyBaseUrl && !storedLegacyBaseUrl) {
        nextAdaptiveBaseUrls[legacyProtocol] = legacyBaseUrl
      }
      editAdaptiveBaseUrls.value = nextAdaptiveBaseUrls
      if (newProvider.platform === 'zhipu') {
        editZhipuOrganization.value = typeof credentials.zhipu_organization === 'string'
          ? credentials.zhipu_organization
          : ''
        editZhipuProject.value = typeof credentials.zhipu_project === 'string'
          ? credentials.zhipu_project
          : ''
      }
    }
    const platformDefaultUrl =
      newProvider.platform === 'openai'
        ? 'https://api.openai.com'
        : newProvider.platform === 'gemini'
          ? 'https://generativelanguage.googleapis.com'
          : newProvider.platform === 'grok'
            ? 'https://api.x.ai/v1'
            : newProvider.platform === 'kimi' ||
                newProvider.platform === 'zhipu' ||
                newProvider.platform === 'deepseek'
              ? defaultCNBaseUrl(newProvider.platform, editProviderMode.value, editApiProtocol.value)
              : 'https://api.anthropic.com'
    editBaseUrl.value = isCNApiKeyProvider.value && editApiProtocol.value === 'adaptive'
      ? editAdaptiveBaseUrls.value.chat_completions
      : (credentials.base_url as string) || platformDefaultUrl

    // 统一从 model_mapping 恢复白名单与映射两个视图，避免配置映射后把白名单误判为空。
    const existingMappings = credentials.model_mapping as Record<string, string> | undefined
    hydrateModelRestrictionFromMapping(existingMappings, credentials.model_whitelist)

    // Load pool mode
    poolModeEnabled.value = credentials.pool_mode === true
    poolModeRetryCount.value = normalizePoolModeRetryCount(
      Number(credentials.pool_mode_retry_count ?? DEFAULT_POOL_MODE_RETRY_COUNT)
    )
    poolModeRetryStatusCodesInput.value = formatPoolModeRetryStatusCodes(credentials.pool_mode_retry_status_codes)

    // Load custom error codes
    customErrorCodesEnabled.value = credentials.custom_error_codes_enabled === true
    const existingErrorCodes = credentials.custom_error_codes as number[] | undefined
    if (existingErrorCodes && Array.isArray(existingErrorCodes)) {
      selectedErrorCodes.value = [...existingErrorCodes]
    } else {
      selectedErrorCodes.value = []
    }

  } else if (newProvider.type === 'bedrock' && newProvider.credentials) {
    const bedrockCreds = newProvider.credentials as Record<string, unknown>
    const authMode = (bedrockCreds.auth_mode as string) || 'sigv4'
    editBedrockRegion.value = (bedrockCreds.aws_region as string) || ''
    editBedrockForceGlobal.value = (bedrockCreds.aws_force_global as string) === 'true'

    if (authMode === 'apikey') {
      editBedrockApiKeyValue.value = ''
    } else {
      editBedrockAccessKeyId.value = (bedrockCreds.aws_access_key_id as string) || ''
      editBedrockSecretAccessKey.value = ''
      editBedrockSessionToken.value = ''
    }

    // Load pool mode for bedrock
    poolModeEnabled.value = bedrockCreds.pool_mode === true
    const retryCount = bedrockCreds.pool_mode_retry_count
    poolModeRetryCount.value = (typeof retryCount === 'number' && retryCount >= 0) ? retryCount : DEFAULT_POOL_MODE_RETRY_COUNT
    poolModeRetryStatusCodesInput.value = formatPoolModeRetryStatusCodes(bedrockCreds.pool_mode_retry_status_codes)

    // Load quota limits for bedrock
    const bedrockExtra = (newProvider.extra as Record<string, unknown>) || {}
    editQuotaLimit.value = typeof bedrockExtra.quota_limit === 'number' ? bedrockExtra.quota_limit : null
    editQuotaDailyLimit.value = typeof bedrockExtra.quota_daily_limit === 'number' ? bedrockExtra.quota_daily_limit : null
    editQuotaWeeklyLimit.value = typeof bedrockExtra.quota_weekly_limit === 'number' ? bedrockExtra.quota_weekly_limit : null
    // Load quota notify for bedrock
    loadQuotaNotifyFromExtra(bedrockExtra)

    // Load model mappings for bedrock
    const existingMappings = bedrockCreds.model_mapping as Record<string, string> | undefined
    hydrateModelRestrictionFromMapping(existingMappings, bedrockCreds.model_whitelist)
  } else if (newProvider.type === 'upstream' && newProvider.credentials) {
    const credentials = newProvider.credentials as Record<string, unknown>
    editBaseUrl.value = (credentials.base_url as string) || ''
  } else if ((newProvider.platform === 'gemini' || newProvider.platform === 'anthropic') && newProvider.type === 'service_account' && newProvider.credentials) {
    const credentials = newProvider.credentials as Record<string, unknown>
    editVertexProjectId.value = (credentials.project_id as string) || ''
    editVertexClientEmail.value = (credentials.client_email as string) || ''
    editVertexLocation.value = (credentials.location as string) || (credentials.vertex_location as string) || 'us-central1'

    // 服务账号也分别回显请求映射与最终白名单。
    hydrateModelRestrictionFromMapping(credentials.model_mapping as Record<string, string> | undefined, credentials.model_whitelist)

  } else {
    const platformDefaultUrl =
      newProvider.platform === 'openai'
        ? 'https://api.openai.com'
        : newProvider.platform === 'gemini'
          ? 'https://generativelanguage.googleapis.com'
          : newProvider.platform === 'grok'
            ? 'https://api.x.ai/v1'
            : 'https://api.anthropic.com'
    editBaseUrl.value = platformDefaultUrl

    // 加载 OpenAI/Grok OAuth 和 Qoder COSY 提供商的模型映射。
    if (
      ((newProvider.platform === 'openai' && newProvider.type === 'oauth') ||
        (newProvider.platform === 'grok' && newProvider.type === 'oauth') ||
        (newProvider.platform === 'gemini' && newProvider.type === 'oauth') ||
        (newProvider.platform === 'qoder' && newProvider.type === 'cosy')) &&
      newProvider.credentials
    ) {
      const oauthCredentials = newProvider.credentials as Record<string, unknown>
      const existingMappings = oauthCredentials.model_mapping as Record<string, string> | undefined
      if (newProvider.platform === 'qoder') {
        hydrateQoderModelRestrictionFromMapping(existingMappings, oauthCredentials.model_whitelist)
      } else {
        hydrateModelRestrictionFromMapping(existingMappings, oauthCredentials.model_whitelist)
      }
    } else {
      hydrateModelRestrictionFromMapping()
    }
    poolModeEnabled.value = false
    poolModeRetryCount.value = DEFAULT_POOL_MODE_RETRY_COUNT
    poolModeRetryStatusCodesInput.value = ''
    customErrorCodesEnabled.value = false
    selectedErrorCodes.value = []
  }
  editApiKey.value = ''
}

async function loadTLSProfiles() {
  try {
    const profiles = await adminAPI.tlsFingerprintProfiles.list()
    tlsFingerprintProfiles.value = profiles.map(p => ({ id: p.id, name: p.name }))
  } catch {
    tlsFingerprintProfiles.value = []
  }
}

async function loadTLSRouters() {
  try {
    const routers = await adminAPI.tlsFingerprintRouters.list()
    tlsFingerprintRouters.value = routers.map(router => ({ id: router.id, name: router.name }))
  } catch {
    tlsFingerprintRouters.value = []
  }
}

const normalizeOpenAIOAuthClientPolicy = (policy: unknown, legacyCodexOnly?: unknown): OpenAIOAuthClientPolicy => {
  if (policy === 'codex_only' || policy === 'tls_router_matched_only' || policy === 'any') {
    return policy
  }
  return legacyCodexOnly === true ? 'codex_only' : 'any'
}

watch(
  [() => props.show, () => props.provider],
  ([show, newProvider], [wasShow, previousProvider]) => {
    if (!show || !newProvider) {
      return
    }
    if (!wasShow || newProvider !== previousProvider) {
      syncFormFromProvider(newProvider)
      loadTLSProfiles()
      loadTLSRouters()
    }
  },
  { immediate: true }
)

// Model mapping helpers
const touchQoderModelRestriction = () => {
  if (isQoderCosyProvider.value) {
    qoderModelRestrictionTouched.value = true
  }
}

const setAllowedModels = (models: string[]) => {
  touchQoderModelRestriction()
  if (isQoderCosyProvider.value) {
    qoderModelWhitelistTouched.value = true
  }
  allowedModels.value = models
}

const addPresetMapping = (from: string, to: string) => {
  touchQoderModelRestriction()
  const exists = modelMappings.value.some((m) => m.from === from)
  if (exists) {
    appStore.showInfo(t('admin.providers.mappingExists', { model: from }))
    return
  }
  modelMappings.value.push({ from, to })
}

const addAntigravityPresetMapping = (from: string, to: string) => {
  const exists = antigravityModelMappings.value.some((m) => m.from === from)
  if (exists) {
    appStore.showInfo(t('admin.providers.mappingExists', { model: from }))
    return
  }
  antigravityModelMappings.value.push({ from, to })
}

const syncAntigravityUpstreamModels = async () => {
  if (!props.provider?.id || isSyncingAntigravityUpstream.value) return

  isSyncingAntigravityUpstream.value = true
  try {
    const result = await adminAPI.providers.syncUpstreamModels(props.provider.id)
    const upstreamModels = result.models.map((model) => model.trim()).filter(Boolean)
    if (upstreamModels.length === 0) {
      appStore.showInfo(t('admin.providers.syncUpstreamModelsEmpty'))
      return
    }

    let addedCount = 0
    for (const model of upstreamModels) {
      const exists = antigravityModelMappings.value.some((mapping) => mapping.from === model)
      if (!exists) {
        antigravityModelMappings.value.push({ from: model, to: model })
        addedCount += 1
      }
    }

    if (addedCount > 0) {
      appStore.showSuccess(t('admin.providers.syncUpstreamModelsSuccess', { count: addedCount, total: upstreamModels.length }))
    } else {
      appStore.showInfo(t('admin.providers.syncUpstreamModelsNoChanges', { count: upstreamModels.length }))
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : t('admin.providers.syncUpstreamModelsFailed')
    appStore.showError(t('admin.providers.syncUpstreamModelsError', { message }))
  } finally {
    isSyncingAntigravityUpstream.value = false
  }
}

// Error code toggle helper
const toggleErrorCode = (code: number) => {
  const index = selectedErrorCodes.value.indexOf(code)
  if (index === -1) {
    // Adding code - check for 429/529 warning
    if (code === 429) {
      if (!confirm(t('admin.providers.customErrorCodes429Warning'))) {
        return
      }
    } else if (code === 529) {
      if (!confirm(t('admin.providers.customErrorCodes529Warning'))) {
        return
      }
    }
    selectedErrorCodes.value.push(code)
  } else {
    selectedErrorCodes.value.splice(index, 1)
  }
}

// Add custom error code from input
const addCustomErrorCode = () => {
  const code = customErrorCodeInput.value
  if (code === null || code < 100 || code > 599) {
    appStore.showError(t('admin.providers.invalidErrorCode'))
    return
  }
  if (selectedErrorCodes.value.includes(code)) {
    appStore.showInfo(t('admin.providers.errorCodeExists'))
    return
  }
  // Check for 429/529 warning
  if (code === 429) {
    if (!confirm(t('admin.providers.customErrorCodes429Warning'))) {
      return
    }
  } else if (code === 529) {
    if (!confirm(t('admin.providers.customErrorCodes529Warning'))) {
      return
    }
  }
  selectedErrorCodes.value.push(code)
  customErrorCodeInput.value = null
}

// Remove error code
const removeErrorCode = (code: number) => {
  const index = selectedErrorCodes.value.indexOf(code)
  if (index !== -1) {
    selectedErrorCodes.value.splice(index, 1)
  }
}

const buildTempUnschedRules = (rules: TempUnschedRuleForm[]) => {
  const out: Array<{
    error_code: number
    keywords: string[]
    duration_minutes: number
    description: string
  }> = []

  for (const rule of rules) {
    const errorCode = Number(rule.error_code)
    const duration = Number(rule.duration_minutes)
    const keywords = splitTempUnschedKeywords(rule.keywords)
    if (!Number.isFinite(errorCode) || errorCode < 100 || errorCode > 599) {
      continue
    }
    if (!Number.isFinite(duration) || duration <= 0) {
      continue
    }
    if (keywords.length === 0) {
      continue
    }
    out.push({
      error_code: Math.trunc(errorCode),
      keywords,
      duration_minutes: Math.trunc(duration),
      description: rule.description.trim()
    })
  }

  return out
}

const applyTempUnschedConfig = (credentials: Record<string, unknown>) => {
  if (!tempUnschedEnabled.value) {
    delete credentials.temp_unschedulable_enabled
    delete credentials.temp_unschedulable_rules
    return true
  }

  const rules = buildTempUnschedRules(tempUnschedRules.value)
  if (rules.length === 0) {
    appStore.showError(t('admin.providers.tempUnschedulable.rulesInvalid'))
    return false
  }

  credentials.temp_unschedulable_enabled = true
  credentials.temp_unschedulable_rules = rules
  return true
}

function supportsProviderSchedulingThresholdOverridePlatform(
  platform: Provider['platform'] | undefined
) {
  return platform === 'openai' || platform === 'anthropic' || platform === 'grok'
}

function normalizeProviderSchedulingThresholdOverride(value: unknown): number | null {
  if (value === null || value === undefined || value === '') {
    return null
  }
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) {
    return null
  }
  const integer = Math.trunc(numeric)
  return integer >= 1 && integer <= 100 ? integer : null
}

function clampProviderSchedulingThresholdOverride(value: unknown): number {
  return Math.min(100, Math.max(1, Math.trunc(Number(value) || 100)))
}

// loadProviderSchedulingThresholdOverride 从提供商凭据恢复单提供商阈值覆盖。
function loadProviderSchedulingThresholdOverride(
  platform: Provider['platform'] | undefined,
  credentials: Record<string, unknown> | undefined
) {
  if (!supportsProviderSchedulingThresholdOverridePlatform(platform)) {
    providerSchedulingThresholdOverrideEnabled.value = false
    providerSchedulingThresholdOverrideValue.value = 100
    return
  }
  const value = normalizeProviderSchedulingThresholdOverride(
    credentials?.[PROVIDER_SCHEDULING_THRESHOLD_CREDENTIAL_KEY]
  )
  providerSchedulingThresholdOverrideEnabled.value = value !== null
  providerSchedulingThresholdOverrideValue.value = value ?? 100
}

// applyProviderSchedulingThresholdOverridePatch 仅在值变化时写入凭据补丁。
const applyProviderSchedulingThresholdOverridePatch = (
  credentials: Record<string, unknown>,
  currentCredentials: Record<string, unknown>,
  platform: Provider['platform'] | undefined = props.provider?.platform
) => {
  if (!supportsProviderSchedulingThresholdOverridePlatform(platform)) {
    return
  }
  const current = normalizeProviderSchedulingThresholdOverride(
    currentCredentials[PROVIDER_SCHEDULING_THRESHOLD_CREDENTIAL_KEY]
  )
  if (!providerSchedulingThresholdOverrideEnabled.value) {
    if (current !== null) {
      credentials[PROVIDER_SCHEDULING_THRESHOLD_CREDENTIAL_KEY] = null
    }
    return
  }
  const next = clampProviderSchedulingThresholdOverride(
    providerSchedulingThresholdOverrideValue.value
  )
  if (current !== next) {
    credentials[PROVIDER_SCHEDULING_THRESHOLD_CREDENTIAL_KEY] = next
  }
}

const applyTLSFingerprintExtra = (extra: Record<string, unknown>) => {
  if (tlsFingerprintEnabled.value) {
    extra.enable_tls_fingerprint = true
    if (tlsFingerprintProfileId.value) {
      extra.tls_fingerprint_profile_id = tlsFingerprintProfileId.value
    } else {
      delete extra.tls_fingerprint_profile_id
    }
    if (supportsTLSFingerprintRouter.value && tlsFingerprintRouterId.value) {
      extra.tls_fingerprint_router_id = tlsFingerprintRouterId.value
    } else {
      delete extra.tls_fingerprint_router_id
    }
  } else {
    delete extra.enable_tls_fingerprint
    delete extra.tls_fingerprint_profile_id
    delete extra.tls_fingerprint_router_id
  }
}

function loadTempUnschedRules(credentials?: Record<string, unknown>) {
  tempUnschedEnabled.value = credentials?.temp_unschedulable_enabled === true
  const rawRules = credentials?.temp_unschedulable_rules
  if (!Array.isArray(rawRules)) {
    tempUnschedRules.value = []
    return
  }

  tempUnschedRules.value = rawRules.map((rule) => {
    const entry = rule as Record<string, unknown>
    return {
      error_code: toPositiveNumber(entry.error_code),
      keywords: formatTempUnschedKeywords(entry.keywords),
      duration_minutes: toPositiveNumber(entry.duration_minutes),
      description: typeof entry.description === 'string' ? entry.description : ''
    }
  })
}

// 从提供商加载配额控制配置（TLS 支持 OpenAI OAuth，其余配额控制仍仅限 Anthropic）
function loadQuotaControlSettings(provider: Provider) {
  // Reset all quota control state first
  windowCostEnabled.value = false
  windowCostLimit.value = null
  windowCostStickyReserve.value = null
  sessionLimitEnabled.value = false
  maxSessions.value = null
  sessionIdleTimeout.value = null
  rpmLimitEnabled.value = false
  baseRpm.value = null
  rpmStrategy.value = 'tiered'
  rpmStickyBuffer.value = null
  userMsgQueueMode.value = ''
  tlsFingerprintEnabled.value = false
  tlsFingerprintProfileId.value = null
  tlsFingerprintRouterId.value = null
  sessionIdMaskingEnabled.value = false
  cacheTTLOverrideEnabled.value = false
  cacheTTLOverrideTarget.value = '5m'
  customBaseUrlEnabled.value = false
  customBaseUrl.value = ''

  // TLS 指纹伪装跨 Anthropic OAuth/SetupToken 与 OpenAI OAuth 复用同一组字段。
  if (supportsTLSFingerprint(provider)) {
    const extra = provider.extra as Record<string, unknown> | undefined
    tlsFingerprintEnabled.value =
      provider.enable_tls_fingerprint === true || extra?.enable_tls_fingerprint === true
    tlsFingerprintProfileId.value =
      provider.tls_fingerprint_profile_id ?? toPositiveOrSpecialID(extra?.tls_fingerprint_profile_id)
    tlsFingerprintRouterId.value =
      provider.tls_fingerprint_router_id ?? toPositiveOrSpecialID(extra?.tls_fingerprint_router_id)
  }

  // Remaining quota control settings only apply to Anthropic providers
  if (provider.platform !== 'anthropic') {
    return
  }

  // Window cost / session limit only apply to Anthropic OAuth/SetupToken providers
  if (provider.type !== 'oauth' && provider.type !== 'setup-token') {
    return
  }

  // Load from extra field (via backend DTO fields)
  if (provider.window_cost_limit != null && provider.window_cost_limit > 0) {
    windowCostEnabled.value = true
    windowCostLimit.value = provider.window_cost_limit
    windowCostStickyReserve.value = provider.window_cost_sticky_reserve ?? 10
  }

  if (provider.max_sessions != null && provider.max_sessions > 0) {
    sessionLimitEnabled.value = true
    maxSessions.value = provider.max_sessions
    sessionIdleTimeout.value = provider.session_idle_timeout_minutes ?? 5
  }

  // RPM limit
  if (provider.base_rpm != null && provider.base_rpm > 0) {
    rpmLimitEnabled.value = true
    baseRpm.value = provider.base_rpm
    rpmStrategy.value = (provider.rpm_strategy as 'tiered' | 'sticky_exempt') || 'tiered'
    rpmStickyBuffer.value = provider.rpm_sticky_buffer ?? null
  }

  // UMQ mode（独立于 RPM 加载，防止编辑无 RPM 提供商时丢失已有配置）
  userMsgQueueMode.value = provider.user_msg_queue_mode ?? ''

  // Load session ID masking setting
  if (provider.session_id_masking_enabled === true) {
    sessionIdMaskingEnabled.value = true
  }

  // Load cache TTL override setting
  if (provider.cache_ttl_override_enabled === true) {
    cacheTTLOverrideEnabled.value = true
    cacheTTLOverrideTarget.value = provider.cache_ttl_override_target || '5m'
  }

  // Load custom base URL setting
  if (provider.custom_base_url_enabled === true) {
    customBaseUrlEnabled.value = true
    customBaseUrl.value = provider.custom_base_url || ''
  }
}

function formatTempUnschedKeywords(value: unknown) {
  if (Array.isArray(value)) {
    return value
      .filter((item): item is string => typeof item === 'string')
      .map((item) => item.trim())
      .filter((item) => item.length > 0)
      .join(', ')
  }
  if (typeof value === 'string') {
    return value
  }
  return ''
}

const splitTempUnschedKeywords = (value: string) => {
  return value
    .split(/[,;]/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

function toPositiveNumber(value: unknown) {
  const num = Number(value)
  if (!Number.isFinite(num) || num <= 0) {
    return null
  }
  return Math.trunc(num)
}

function toPositiveOrSpecialID(value: unknown) {
  const num = Number(value)
  if (!Number.isFinite(num) || num === 0) {
    return null
  }
  return Math.trunc(num)
}

const formatDateTimeLocal = formatDateTimeLocalInput
const parseDateTimeLocal = parseDateTimeLocalInput

// Methods
const handleClose = () => {
  emit('close')
}

const submitUpdateProvider = async (providerID: number, updatePayload: Record<string, unknown>) => {
  submitting.value = true
	let providerSaved = false
  try {
    if (props.provider && !props.provider.parent_provider_id) {
      await loadProtocolCatalog()
      const credentials = { ...(updatePayload.credentials as Record<string, unknown> ?? props.provider.credentials ?? {}), upstream_protocols: upstreamProtocols.value ?? nativeProtocolOptions(props.provider.platform, props.provider.type, String(props.provider.credentials?.auth_mode ?? props.provider.credentials?.openai_auth_mode ?? '')) } as Record<string, unknown>
      delete credentials.api_protocol
      delete credentials.openai_workload_capabilities
      updatePayload.credentials = credentials
      const extra = updatePayload.extra as Record<string, unknown> | undefined
      if (extra) delete extra.openai_text_route_mode
    }
    const saveTicket = await ticketSettings.value?.prepareSave()
    if (props.provider?.id !== providerID || !props.show) return
    const updatedProvider = await adminAPI.providers.update(providerID, updatePayload)
    providerSaved = true
    if (saveTicket) await saveTicket()
    appStore.showSuccess(t('admin.providers.providerUpdated'))
    emit('updated', updatedProvider)
    handleClose()
  } catch (error: any) {
    appStore.showError(providerSaved ? t('admin.accounts.ticketPolicy.partialSave', { error: error.message }) : error.message || t('admin.providers.failedToUpdate'))
  } finally {
    submitting.value = false
  }
}

const handleSubmit = async () => {
  if (!props.provider) return
  const providerID = props.provider.id

  if (form.status !== 'active' && form.status !== 'inactive' && form.status !== 'error') {
    appStore.showError(t('admin.providers.pleaseSelectStatus'))
    return
  }

  const updatePayload: Record<string, unknown> = { ...form }
  try {
    // 后端期望 proxy_id: 0 表示清除代理，而不是 null
    if (updatePayload.proxy_id === null) {
      updatePayload.proxy_id = 0
    }
    if (form.expires_at === null) {
      updatePayload.expires_at = 0
    }
    // load_factor: 空值/NaN/0/负数 时发送 0（后端约定 <= 0 = 清除）
    const lf = form.load_factor
    if (lf == null || Number.isNaN(lf) || lf <= 0) {
      updatePayload.load_factor = 0
    }
    updatePayload.auto_pause_on_expired = autoPauseOnExpired.value

    // For apikey type, handle credentials update
    if (props.provider.type === 'apikey') {
      const currentCredentials = (props.provider.credentials as Record<string, unknown>) || {}
      const enteredBaseUrl = editBaseUrl.value.trim()
      if (
        props.provider.platform === 'gemini' &&
        geminiProviderType.value === 'third_party' &&
        !isGeminiThirdPartyBaseUrl(enteredBaseUrl)
      ) {
        appStore.showError(t('admin.providers.gemini.connectionSource.thirdPartyBaseUrlRequired'))
        return
      }
      const newBaseUrl = enteredBaseUrl || defaultBaseUrl.value
      const shouldApplyModelMapping = true

      // API Key 类型始终提交 credentials，以便同步模型映射变更。
      const newCredentials: Record<string, unknown> = {
        ...currentCredentials,
        base_url: newBaseUrl
      }

      // 国产供应商：模式与协议写入凭据（决定额度/余额探测与转发端点/格式）。
      if (isCNApiKeyProvider.value) {
        newCredentials.provider_mode = editProviderMode.value
        newCredentials.api_protocol = editApiProtocol.value
        if (props.provider.platform === 'opencode_go') applyOpenCodeGoProtocolRules(newCredentials, editOpenCodeRules.value, 'edit')
        if (editApiProtocol.value === 'adaptive') {
          const defaults = defaultCNAdaptiveBaseUrls(cnPresetPlatform.value, editProviderMode.value)
          const protocolBaseUrls: Record<string, string> = {}
          for (const item of editAdaptiveProtocolOptions.value) {
            protocolBaseUrls[item.value] = (editAdaptiveBaseUrls.value[item.value] || defaults[item.value]).trim()
          }
          newCredentials.api_base_urls = protocolBaseUrls
          newCredentials.base_url = protocolBaseUrls.chat_completions
        } else {
          delete newCredentials.api_base_urls
        }
        if (props.provider.platform === 'zhipu') {
          const organization = editZhipuOrganization.value.trim()
          const project = editZhipuProject.value.trim()
          if (editProviderMode.value === 'coding' && organization) {
            newCredentials.zhipu_organization = organization
            if (project) newCredentials.zhipu_project = project
            else delete newCredentials.zhipu_project
          } else {
            delete newCredentials.zhipu_organization
            delete newCredentials.zhipu_project
          }
        }
      }

      // 处理 API Key。
      // 后端响应已脱敏：currentCredentials 不会再包含 api_key 原文。
      // 用户填入新值则覆盖；留空时优先看 credentials_status.has_api_key；
      // 若后端尚未升级（无 credentials_status），回退读旧结构 currentCredentials.api_key。
      // 两者都无才报错。
      const hasExistingApiKey =
        props.provider.credentials_status?.has_api_key ?? Boolean(currentCredentials.api_key)
      if (editApiKey.value.trim()) {
        newCredentials.api_key = editApiKey.value.trim()
      } else if (!hasExistingApiKey) {
        appStore.showError(t('admin.providers.apiKeyIsRequired'))
        return
      }

      // New API 用户访问令牌属于敏感凭据；留空表示沿用后端已保存的值。
      if (upstreamUsageWalletAccessToken.value.trim()) {
        newCredentials.new_api_user_access_token = upstreamUsageWalletAccessToken.value.trim()
      }
      if (upstreamUsageWalletUserId.value.trim()) {
        newCredentials.new_api_user_id = upstreamUsageWalletUserId.value.trim()
      } else {
        delete newCredentials.new_api_user_id
      }

      if (props.provider.platform === 'gemini') {
        newCredentials.provider_type = geminiProviderType.value
        if (geminiProviderType.value === 'third_party') {
          // tier_id 不是敏感字段，删除后会随本次完整凭据更新一起清除。
          delete newCredentials.tier_id
        } else {
          newCredentials.tier_id = geminiAIStudioTier.value
        }
      }

      // Add model mapping if configured（OpenAI 开启自动透传时保留现有映射，不再编辑）
      if (shouldApplyModelMapping) {
        if (props.provider.platform === 'qoder') {
          applyQoderModelRestriction(newCredentials)
        } else {
          applyPersistedModelRestriction(newCredentials)
        }
      } else if (currentCredentials.model_mapping) {
        newCredentials.model_mapping = currentCredentials.model_mapping
        if ('model_whitelist' in currentCredentials) {
          newCredentials.model_whitelist = currentCredentials.model_whitelist
        }
      } else if ('model_whitelist' in currentCredentials) {
        newCredentials.model_whitelist = currentCredentials.model_whitelist
      }
      if (props.provider.platform === 'openai') {
        const compactModelMapping = buildModelMappingObject('mapping', [], openAICompactModelMappings.value)
        if (compactModelMapping) {
          newCredentials.compact_model_mapping = compactModelMapping
        } else {
          delete newCredentials.compact_model_mapping
        }
      }

      // Add pool mode if enabled
      if (poolModeEnabled.value) {
        newCredentials.pool_mode = true
        newCredentials.pool_mode_retry_count = normalizePoolModeRetryCount(poolModeRetryCount.value)
        const parsedRetryStatusCodes = parsePoolModeRetryStatusCodes(poolModeRetryStatusCodesInput.value)
        if (parsedRetryStatusCodes.length > 0) {
          newCredentials.pool_mode_retry_status_codes = parsedRetryStatusCodes
        } else {
          delete newCredentials.pool_mode_retry_status_codes
        }
      } else {
        delete newCredentials.pool_mode
        delete newCredentials.pool_mode_retry_count
        delete newCredentials.pool_mode_retry_status_codes
      }

      // Add custom error codes if enabled
      if (customErrorCodesEnabled.value) {
        newCredentials.custom_error_codes_enabled = true
        newCredentials.custom_error_codes = [...selectedErrorCodes.value]
      } else {
        delete newCredentials.custom_error_codes_enabled
        delete newCredentials.custom_error_codes
      }

      // 为支持该功能的平台 API Key 提供商写入请求头覆写。
      if (isHeaderOverrideCapable(props.provider.platform, 'apikey')) {
        if (headerOverrideEnabled.value) {
          const headerError = validateHeaderOverrideRows(headerOverrideRows.value)
          if (headerError) {
            appStore.showError(t(`admin.providers.headerOverride.${headerError}`))
            return
          }
        }
        applyHeaderOverride(newCredentials, headerOverrideEnabled.value, headerOverrideRows.value, 'edit')
      }

      // Add intercept warmup requests setting
      applyInterceptWarmup(newCredentials, interceptWarmupRequests.value, 'edit')
      applyProviderSchedulingThresholdOverridePatch(newCredentials, currentCredentials)
      if (!applyTempUnschedConfig(newCredentials)) {
        return
      }

      updatePayload.credentials = newCredentials
    } else if (props.provider.type === 'upstream') {
      const currentCredentials = (props.provider.credentials as Record<string, unknown>) || {}
      const newCredentials: Record<string, unknown> = { ...currentCredentials }

      newCredentials.base_url = editBaseUrl.value.trim()

      if (editApiKey.value.trim()) {
        newCredentials.api_key = editApiKey.value.trim()
      }

      // Add intercept warmup requests setting
      applyInterceptWarmup(newCredentials, interceptWarmupRequests.value, 'edit')

      applyProviderSchedulingThresholdOverridePatch(newCredentials, currentCredentials)
      if (!applyTempUnschedConfig(newCredentials)) {
        return
      }

      updatePayload.credentials = newCredentials
    } else if ((props.provider.platform === 'gemini' || props.provider.platform === 'anthropic') && props.provider.type === 'service_account') {
      const currentCredentials = (props.provider.credentials as Record<string, unknown>) || {}
      const newCredentials: Record<string, unknown> = { ...currentCredentials }

      if (!editVertexProjectId.value.trim()) {
        appStore.showError(t('admin.providers.vertexSaJsonMissingProjectId'))
        return
      }
      if (!editVertexClientEmail.value.trim()) {
        appStore.showError(t('admin.providers.vertexSaJsonMissingClientEmail'))
        return
      }
      if (!editVertexLocation.value.trim()) {
        appStore.showError(t('admin.providers.vertexLocationRequired'))
        return
      }

      // SA JSON 已脱敏不再随 credentials 返回，存在性优先读 credentials_status。
      // 若后端尚未升级（无 credentials_status），回退读旧结构 service_account_json / service_account。
      const credentialsStatus = props.provider.credentials_status
      const hasExistingServiceAccountJson = credentialsStatus
        ? Boolean(
            credentialsStatus.has_service_account_json || credentialsStatus.has_service_account
          )
        : Boolean(currentCredentials.service_account_json || currentCredentials.service_account)
      if (!hasExistingServiceAccountJson) {
        appStore.showError(t('admin.providers.vertexSaJsonRequired'))
        return
      }
      newCredentials.project_id = editVertexProjectId.value.trim()
      newCredentials.client_email = editVertexClientEmail.value.trim()
      newCredentials.location = editVertexLocation.value.trim()
      newCredentials.tier_id = 'vertex'

      applyPersistedModelRestriction(newCredentials)

      applyInterceptWarmup(newCredentials, interceptWarmupRequests.value, 'edit')
      applyProviderSchedulingThresholdOverridePatch(newCredentials, currentCredentials)
      if (!applyTempUnschedConfig(newCredentials)) {
        return
      }

      updatePayload.credentials = newCredentials
    } else if (props.provider.type === 'bedrock') {
      const currentCredentials = (props.provider.credentials as Record<string, unknown>) || {}
      const newCredentials: Record<string, unknown> = { ...currentCredentials }

      newCredentials.aws_region = editBedrockRegion.value.trim()
      if (editBedrockForceGlobal.value) {
        newCredentials.aws_force_global = 'true'
      } else {
        delete newCredentials.aws_force_global
      }

      if (isBedrockAPIKeyMode.value) {
        // API Key mode: only update api_key if user provided new value
        if (editBedrockApiKeyValue.value.trim()) {
          newCredentials.api_key = editBedrockApiKeyValue.value.trim()
        }
      } else {
        // SigV4 mode
        newCredentials.aws_access_key_id = editBedrockAccessKeyId.value.trim()
        if (editBedrockSecretAccessKey.value.trim()) {
          newCredentials.aws_secret_access_key = editBedrockSecretAccessKey.value.trim()
        }
        if (editBedrockSessionToken.value.trim()) {
          newCredentials.aws_session_token = editBedrockSessionToken.value.trim()
        }
      }

      // Pool mode
      if (poolModeEnabled.value) {
        newCredentials.pool_mode = true
        newCredentials.pool_mode_retry_count = normalizePoolModeRetryCount(poolModeRetryCount.value)
        const parsedRetryStatusCodes = parsePoolModeRetryStatusCodes(poolModeRetryStatusCodesInput.value)
        if (parsedRetryStatusCodes.length > 0) {
          newCredentials.pool_mode_retry_status_codes = parsedRetryStatusCodes
        } else {
          delete newCredentials.pool_mode_retry_status_codes
        }
      } else {
        delete newCredentials.pool_mode
        delete newCredentials.pool_mode_retry_count
        delete newCredentials.pool_mode_retry_status_codes
      }

      // Model mapping
      applyPersistedModelRestriction(newCredentials)

      applyInterceptWarmup(newCredentials, interceptWarmupRequests.value, 'edit')
      applyProviderSchedulingThresholdOverridePatch(newCredentials, currentCredentials)
      if (!applyTempUnschedConfig(newCredentials)) {
        return
      }

      updatePayload.credentials = newCredentials
    } else {
      // For oauth/setup-token types, only update intercept_warmup_requests if changed
      const currentCredentials = (props.provider.credentials as Record<string, unknown>) || {}
      const newCredentials: Record<string, unknown> = { ...currentCredentials }

      applyInterceptWarmup(newCredentials, interceptWarmupRequests.value, 'edit')
      applyProviderSchedulingThresholdOverridePatch(newCredentials, currentCredentials)
      if (!applyTempUnschedConfig(newCredentials)) {
        return
      }

      updatePayload.credentials = newCredentials
    }

    // OpenAI/Grok OAuth 与 Qoder COSY：将模型映射保存到 credentials。
    if (supportsOAuthLikeModelRestriction.value) {
      const currentCredentials = props.provider.platform === 'openai' && isSparkShadow.value
        ? {}
        : (updatePayload.credentials as Record<string, unknown>) ||
          ((props.provider.credentials as Record<string, unknown>) || {})
      const newCredentials: Record<string, unknown> = { ...currentCredentials }

      if (props.provider.platform === 'qoder') {
        applyQoderModelRestriction(newCredentials)
        newCredentials.site = qoderSite.value
      } else if (props.provider.platform === 'openai') {
        applyOpenAIModelMappingCredentials(newCredentials)
      } else {
        applyPersistedModelRestriction(newCredentials)
      }

      updatePayload.credentials = newCredentials
    }

    // Grok OAuth：保存自定义上游地址与请求头覆写。base_url 仅改写转发端点，
    // OAuth 授权与令牌刷新链路不读取该值；关闭开关即恢复默认官方网关。
    if (props.provider.platform === 'grok' && props.provider.type === 'oauth') {
      const currentCredentials =
        (updatePayload.credentials as Record<string, unknown>) ||
        ((props.provider.credentials as Record<string, unknown>) || {})
      const newCredentials: Record<string, unknown> = { ...currentCredentials }

      if (grokOAuthCustomBaseUrlEnabled.value) {
        const trimmedBaseUrl = grokOAuthBaseUrl.value.trim()
        if (!trimmedBaseUrl) {
          appStore.showError(t('admin.providers.grokCustomBaseUrl.required'))
          return
        }
        if (!/^https?:\/\//i.test(trimmedBaseUrl)) {
          appStore.showError(t('admin.providers.grokCustomBaseUrl.invalid'))
          return
        }
        newCredentials.base_url = trimmedBaseUrl
      } else {
        delete newCredentials.base_url
      }

      if (headerOverrideEnabled.value) {
        const headerError = validateHeaderOverrideRows(headerOverrideRows.value)
        if (headerError) {
          appStore.showError(t(`admin.providers.headerOverride.${headerError}`))
          return
        }
      }
      applyHeaderOverride(newCredentials, headerOverrideEnabled.value, headerOverrideRows.value, 'edit')

      updatePayload.credentials = newCredentials

      const newExtra: Record<string, unknown> = {
        ...((props.provider.extra as Record<string, unknown>) || {})
      }
      // 两种状态都持久化，避免后端对缺失值应用默认启用策略后重新开启已关闭提供商。
      newExtra[GROK_CLIENT_TOOL_CACHE_EXTRA_KEY] = grokClientToolCacheEnabled.value
      updatePayload.extra = newExtra
    }

    // OpenAI: 手动覆盖订阅档位 plan_type（Plus/Pro/Free）。仅 OAuth 非影子提供商：
    // 影子提供商凭据由母提供商管理(且后端会 sanitize),setup-token 无订阅调度语义。
    if (props.provider.platform === 'openai' && props.provider.type === 'oauth' && !isSparkShadow.value) {
      const currentCredentials = (updatePayload.credentials as Record<string, unknown>) ||
        ((props.provider.credentials as Record<string, unknown>) || {})
      updatePayload.credentials = applyPlanType({ ...currentCredentials }, editPlanType.value)
    }

    // Antigravity: persist model mapping to credentials (applies to all antigravity types)
    // Antigravity 只支持映射模式
    if (props.provider.platform === 'antigravity') {
      const currentCredentials = (updatePayload.credentials as Record<string, unknown>) ||
        ((props.provider.credentials as Record<string, unknown>) || {})
      const newCredentials: Record<string, unknown> = { ...currentCredentials }

      // 移除旧字段
      newCredentials.model_whitelist = [...antigravityWhitelistModels.value]
      delete newCredentials.model_mapping

      // 只使用映射模式
      const antigravityModelMapping = buildModelMappingObject(
        'mapping',
        [],
        antigravityModelMappings.value
      )
      if (antigravityModelMapping) {
        newCredentials.model_mapping = antigravityModelMapping
      }
      if (props.provider.type === 'oauth') {
        applyAntigravityProjectID(newCredentials, antigravityProjectId.value, 'edit')
      }

      updatePayload.credentials = newCredentials
    }

    // Antigravity 提供商单独保存上游超额额度策略。
    if (props.provider.platform === 'antigravity') {
      const currentExtra = (props.provider.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }
      delete newExtra.mixed_scheduling
      if (allowOverages.value) {
        newExtra.allow_overages = true
      } else {
        delete newExtra.allow_overages
      }
      updatePayload.extra = newExtra
    }

    // For Anthropic OAuth/SetupToken providers, handle quota control settings in extra
    if (props.provider.platform === 'anthropic' && (props.provider.type === 'oauth' || props.provider.type === 'setup-token')) {
      const currentExtra = (updatePayload.extra as Record<string, unknown>) || (props.provider.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }

      // Window cost limit settings
      if (windowCostEnabled.value && windowCostLimit.value != null && windowCostLimit.value > 0) {
        newExtra.window_cost_limit = windowCostLimit.value
        newExtra.window_cost_sticky_reserve = windowCostStickyReserve.value ?? 10
      } else {
        delete newExtra.window_cost_limit
        delete newExtra.window_cost_sticky_reserve
      }

      // Session limit settings
      if (sessionLimitEnabled.value && maxSessions.value != null && maxSessions.value > 0) {
        newExtra.max_sessions = maxSessions.value
        newExtra.session_idle_timeout_minutes = sessionIdleTimeout.value ?? 5
      } else {
        delete newExtra.max_sessions
        delete newExtra.session_idle_timeout_minutes
      }

      // RPM limit settings
      if (rpmLimitEnabled.value) {
        const DEFAULT_BASE_RPM = 15
        newExtra.base_rpm = (baseRpm.value != null && baseRpm.value > 0)
          ? baseRpm.value
          : DEFAULT_BASE_RPM
        newExtra.rpm_strategy = rpmStrategy.value
        if (rpmStickyBuffer.value != null && rpmStickyBuffer.value > 0) {
          newExtra.rpm_sticky_buffer = rpmStickyBuffer.value
        } else {
          delete newExtra.rpm_sticky_buffer
        }
      } else {
        delete newExtra.base_rpm
        delete newExtra.rpm_strategy
        delete newExtra.rpm_sticky_buffer
      }

      // UMQ mode（独立于 RPM 保存）
      if (userMsgQueueMode.value) {
        newExtra.user_msg_queue_mode = userMsgQueueMode.value
      } else {
        delete newExtra.user_msg_queue_mode
      }
      delete newExtra.user_msg_queue_enabled  // 清理旧字段

      // TLS fingerprint setting
      if (tlsFingerprintEnabled.value) {
        newExtra.enable_tls_fingerprint = true
        if (tlsFingerprintProfileId.value) {
          newExtra.tls_fingerprint_profile_id = tlsFingerprintProfileId.value
        } else {
          delete newExtra.tls_fingerprint_profile_id
        }
      } else {
        delete newExtra.enable_tls_fingerprint
        delete newExtra.tls_fingerprint_profile_id
        delete newExtra.tls_fingerprint_router_id
      }

      // Session ID masking setting
      if (sessionIdMaskingEnabled.value) {
        newExtra.session_id_masking_enabled = true
      } else {
        delete newExtra.session_id_masking_enabled
      }

      // Cache TTL override setting
      if (cacheTTLOverrideEnabled.value) {
        newExtra.cache_ttl_override_enabled = true
        newExtra.cache_ttl_override_target = cacheTTLOverrideTarget.value
      } else {
        delete newExtra.cache_ttl_override_enabled
        delete newExtra.cache_ttl_override_target
      }

      // Custom base URL relay setting
      if (customBaseUrlEnabled.value && customBaseUrl.value.trim()) {
        newExtra.custom_base_url_enabled = true
        newExtra.custom_base_url = customBaseUrl.value.trim()
      } else {
        delete newExtra.custom_base_url_enabled
        delete newExtra.custom_base_url
      }

      updatePayload.extra = newExtra
    }

    // For Anthropic API Key providers, handle passthrough mode + web search emulation in extra
    if (props.provider.platform === 'anthropic' && props.provider.type === 'apikey') {
      const currentExtra = (updatePayload.extra as Record<string, unknown>) || (props.provider.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }
      if (anthropicPassthroughEnabled.value) {
        newExtra.anthropic_passthrough = true
      } else {
        delete newExtra.anthropic_passthrough
      }
      if (anthropicAPIKeyAuthScheme.value === 'authorization_bearer') {
        newExtra.anthropic_apikey_auth_scheme = 'authorization_bearer'
      } else {
        delete newExtra.anthropic_apikey_auth_scheme
      }
      if (webSearchEmulationMode.value === 'default') {
        delete newExtra.web_search_emulation
      } else {
        newExtra.web_search_emulation = webSearchEmulationMode.value
      }
      updatePayload.extra = newExtra
    }

    if (isQoderCosyProvider.value) {
      const currentExtra = (updatePayload.extra as Record<string, unknown>) || (props.provider.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }
      applyTLSFingerprintExtra(newExtra)
      updatePayload.extra = newExtra
    }

    // OpenAI OAuth、SetupToken 和 API Key 提供商：更新透传与计费设置。
    if (props.provider.platform === 'openai' && (props.provider.type === 'oauth' || props.provider.type === 'setup-token' || props.provider.type === 'apikey')) {
      const currentExtra = (props.provider.extra as Record<string, unknown>) || {}
      const newExtra = normalizeLegacyOpenAIExtra(currentExtra)
      const hadCodexCLIOnlyEnabled = currentExtra.codex_cli_only === true
      if (props.provider.type === 'oauth' || props.provider.type === 'setup-token') {
        newExtra.openai_oauth_responses_websockets_v2_mode = openaiOAuthResponsesWebSocketV2Mode.value
        newExtra.openai_oauth_responses_websockets_v2_enabled = isOpenAIWSModeEnabled(openaiOAuthResponsesWebSocketV2Mode.value)
      } else if (props.provider.type === 'apikey') {
        newExtra.openai_apikey_responses_websockets_v2_mode = openaiAPIKeyResponsesWebSocketV2Mode.value
        newExtra.openai_apikey_responses_websockets_v2_enabled = isOpenAIWSModeEnabled(openaiAPIKeyResponsesWebSocketV2Mode.value)
      }
      delete newExtra.responses_websockets_v2_enabled
      delete newExtra.openai_ws_enabled
      delete newExtra.openai_long_context_billing_enabled
      if (openaiPassthroughEnabled.value) {
        newExtra.openai_passthrough = true
      } else {
        delete newExtra.openai_passthrough
        delete newExtra.openai_oauth_passthrough
      }
      // 关闭时删除默认项，避免提供商 extra 堆积无意义的 false。
      if (props.provider.type === 'oauth' && openaiFlattenNamespacesEnabled.value) {
        newExtra.openai_responses_flatten_namespaces = true
      } else {
        delete newExtra.openai_responses_flatten_namespaces
      }
      newExtra.openai_compact_mode = openAICompactMode.value
      newExtra.openai_native_compaction_v2_mode = openAINativeCompactionV2Mode.value
      if (props.provider.type === 'apikey' && openAIImagesURLToB64JSON.value) {
        newExtra.images_url_to_b64_json = true
      } else {
        delete newExtra.images_url_to_b64_json
      }
      if (props.provider.type === 'apikey') {
		delete newExtra.openai_responses_mode
		newExtra.openai_responses_continuation_supported = openAIResponsesContinuationSupported.value
	  }
		if (autoPause5hThreshold.value != null && autoPause5hThreshold.value > 0) {
			newExtra.auto_pause_5h_threshold = autoPause5hThreshold.value / 100
		} else {
			delete newExtra.auto_pause_5h_threshold
		}
		if (autoPause7dThreshold.value != null && autoPause7dThreshold.value > 0) {
			newExtra.auto_pause_7d_threshold = autoPause7dThreshold.value / 100
		} else {
			delete newExtra.auto_pause_7d_threshold
		}
		if (autoPause5hDisabled.value) {
			newExtra.auto_pause_5h_disabled = true
		} else {
			delete newExtra.auto_pause_5h_disabled
		}
		if (autoPause7dDisabled.value) {
			newExtra.auto_pause_7d_disabled = true
		} else {
			delete newExtra.auto_pause_7d_disabled
		}

      applyCodexImageToolMode(newExtra, codexImageToolMode.value)

      if (props.provider.type === 'oauth') {
        newExtra.openai_oauth_client_policy = openAIOAuthClientPolicy.value
        if (openAIOAuthClientPolicy.value === 'codex_only') {
          newExtra.codex_cli_only = true
        } else if (hadCodexCLIOnlyEnabled || currentExtra.openai_oauth_client_policy != null) {
          // 关闭时显式写 false，避免 extra 为空被后端忽略导致旧值无法清除
          newExtra.codex_cli_only = false
        } else {
          delete newExtra.codex_cli_only
        }
        // 仅当 codex_cli_only 开启且子开关开启时写入 Claude Code 插件白名单，否则清除避免孤立字段
        if (openAIOAuthClientPolicy.value === 'codex_only' && codexCLIOnlyAllowClaudeCodeEnabled.value) {
          newExtra.codex_cli_only_allowed_clients = ['claude_code']
        } else {
          delete newExtra.codex_cli_only_allowed_clients
        }

        // OpenAI OAuth 复用 Anthropic 的 TLS 指纹伪装字段。
        if (tlsFingerprintEnabled.value) {
          newExtra.enable_tls_fingerprint = true
          if (tlsFingerprintProfileId.value) {
            newExtra.tls_fingerprint_profile_id = tlsFingerprintProfileId.value
          } else {
            delete newExtra.tls_fingerprint_profile_id
          }
          if (tlsFingerprintRouterId.value) {
            newExtra.tls_fingerprint_router_id = tlsFingerprintRouterId.value
          } else {
            delete newExtra.tls_fingerprint_router_id
          }
        } else {
          delete newExtra.enable_tls_fingerprint
          delete newExtra.tls_fingerprint_profile_id
          delete newExtra.tls_fingerprint_router_id
        }
      }

      // 指纹收敛模式：默认 off，不写入；其它模式必须显式写入。
      if (props.provider.type === 'oauth') {
        if (codexFingerprintMode.value !== 'off') {
          newExtra.codex_fingerprint_mode = codexFingerprintMode.value
        } else {
          delete newExtra.codex_fingerprint_mode
        }
      }

      updatePayload.extra = newExtra
    }

    // For apikey/bedrock providers, handle quota_limit in extra
    if (props.provider.type === 'apikey' || props.provider.type === 'bedrock') {
      const currentExtra = (updatePayload.extra as Record<string, unknown>) ||
        (props.provider.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }
      // Total quota
      if (editQuotaLimit.value != null && editQuotaLimit.value > 0) {
        newExtra.quota_limit = editQuotaLimit.value
      } else {
        delete newExtra.quota_limit
      }
      // Daily quota
      if (editQuotaDailyLimit.value != null && editQuotaDailyLimit.value > 0) {
        newExtra.quota_daily_limit = editQuotaDailyLimit.value
      } else {
        delete newExtra.quota_daily_limit
        delete newExtra.quota_daily_used
        delete newExtra.quota_daily_start
      }
      // Weekly quota
      if (editQuotaWeeklyLimit.value != null && editQuotaWeeklyLimit.value > 0) {
        newExtra.quota_weekly_limit = editQuotaWeeklyLimit.value
      } else {
        delete newExtra.quota_weekly_limit
        delete newExtra.quota_weekly_used
        delete newExtra.quota_weekly_start
      }
      // Quota reset mode config
      if (editDailyResetMode.value === 'fixed') {
        newExtra.quota_daily_reset_mode = 'fixed'
        newExtra.quota_daily_reset_hour = editDailyResetHour.value ?? 0
      } else {
        delete newExtra.quota_daily_reset_mode
        delete newExtra.quota_daily_reset_hour
      }
      if (editWeeklyResetMode.value === 'fixed') {
        newExtra.quota_weekly_reset_mode = 'fixed'
        newExtra.quota_weekly_reset_day = editWeeklyResetDay.value ?? 1
        newExtra.quota_weekly_reset_hour = editWeeklyResetHour.value ?? 0
      } else {
        delete newExtra.quota_weekly_reset_mode
        delete newExtra.quota_weekly_reset_day
        delete newExtra.quota_weekly_reset_hour
      }
      if (editDailyResetMode.value === 'fixed' || editWeeklyResetMode.value === 'fixed') {
        newExtra.quota_reset_timezone = editResetTimezone.value || 'UTC'
      } else {
        delete newExtra.quota_reset_timezone
      }
      // Quota notify config
      writeQuotaNotifyToExtra(newExtra, 'update')
      if (props.provider.type === 'apikey') {
        const upstreamConfig: Record<string, unknown> = {
          enabled: upstreamUsageEnabled.value,
          adapter: upstreamUsageAdapter.value
        }
        if (upstreamUsageBaseUrl.value.trim()) {
          upstreamConfig.base_url = upstreamUsageBaseUrl.value.trim()
        }
        newExtra.upstream_usage_query = upstreamConfig
      }
      updatePayload.extra = newExtra
    }

    // 上游ID头名只在改动时写回 extra，避免用弹窗打开时的快照覆盖运行态键。
    const nextUpstreamRequestIdHeader = upstreamRequestIdHeader.value.trim()
    if (nextUpstreamRequestIdHeader !== readUpstreamRequestIdHeader(props.provider.extra)) {
      const currentExtra = (updatePayload.extra as Record<string, unknown>) || (props.provider.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }
      if (nextUpstreamRequestIdHeader) {
        newExtra.upstream_request_id_header = nextUpstreamRequestIdHeader
      } else {
        delete newExtra.upstream_request_id_header
      }
      updatePayload.extra = newExtra
    }

    await submitUpdateProvider(providerID, updatePayload)
  } catch (error: any) {
    appStore.showError(error.message || t('admin.providers.failedToUpdate'))
  }
}
</script>
