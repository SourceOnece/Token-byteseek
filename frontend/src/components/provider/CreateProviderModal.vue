<template>
  <BaseDialog
    :show="show"
    :title="t('admin.providers.createProvider')"
    width="wide"
    @close="handleClose"
  >
    <!-- Step Indicator for OAuth providers -->
    <div v-if="isOAuthFlow" class="mb-6 flex items-center justify-center">
      <div class="flex items-center space-x-4">
        <div class="flex items-center">
          <div
            :class="[
              'flex h-8 w-8 items-center justify-center rounded-full text-sm font-semibold',
              step >= 1 ? 'bg-primary-500 text-white' : 'bg-gray-200 text-gray-500 dark:bg-dark-600'
            ]"
          >
            1
          </div>
          <span class="ml-2 text-sm font-medium text-gray-700 dark:text-gray-300">{{
            t('admin.providers.oauth.authMethod')
          }}</span>
        </div>
        <div class="h-0.5 w-8 bg-gray-300 dark:bg-dark-600" />
        <div class="flex items-center">
          <div
            :class="[
              'flex h-8 w-8 items-center justify-center rounded-full text-sm font-semibold',
              step >= 2 ? 'bg-primary-500 text-white' : 'bg-gray-200 text-gray-500 dark:bg-dark-600'
            ]"
          >
            2
          </div>
          <span class="ml-2 text-sm font-medium text-gray-700 dark:text-gray-300">{{
            oauthStepTitle
          }}</span>
        </div>
      </div>
    </div>

    <!-- Step 1: Basic Info -->
    <form
      v-if="step === 1"
      id="create-provider-form"
      @submit.prevent="handleSubmit"
      class="space-y-5"
    >
      <div>
        <label class="input-label">{{ t('admin.providers.providerName') }}</label>
        <input
          v-model="form.name"
          type="text"
          :required="!isGrokSSOInputMethod"
          class="input"
          :placeholder="t('admin.providers.enterProviderName')"
          data-tour="provider-form-name"
        />
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

      <!-- Platform Selection - Segmented Control Style -->
      <div>
        <label class="input-label">{{ t('admin.providers.platform') }}</label>
        <div class="mt-2 flex flex-wrap rounded-control bg-gray-100 p-1 dark:bg-dark-700" data-tour="provider-form-platform">
          <button
            type="button"
            @click="form.platform = 'anthropic'"
            :class="[
              'flex h-9 flex-1 items-center justify-center gap-2 rounded-control px-4 py-1.5 text-sm font-medium transition-all',
              form.platform === 'anthropic'
                ? 'bg-white text-orange-600 shadow-sm dark:bg-dark-600 dark:text-orange-400'
                : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200'
            ]"
          >
            <Icon name="sparkles" size="sm" />
            Anthropic
          </button>
          <button
            type="button"
            @click="form.platform = 'openai'"
            :class="[
              'flex h-9 flex-1 items-center justify-center gap-2 rounded-control px-4 py-1.5 text-sm font-medium transition-all',
              form.platform === 'openai'
                ? 'bg-white text-green-600 shadow-sm dark:bg-dark-600 dark:text-green-400'
                : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200'
            ]"
          >
            <svg
              class="h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="1.5"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z"
              />
            </svg>
            OpenAI
          </button>
          <button
            type="button"
            @click="form.platform = 'gemini'"
            data-testid="create-provider-platform-gemini"
            :class="[
              'flex h-9 flex-1 items-center justify-center gap-2 rounded-control px-4 py-1.5 text-sm font-medium transition-all',
              form.platform === 'gemini'
                ? 'bg-white text-blue-600 shadow-sm dark:bg-dark-600 dark:text-blue-400'
                : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200'
            ]"
          >
            <svg
              class="h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="1.5"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M12 2l1.5 6.5L20 10l-6.5 1.5L12 18l-1.5-6.5L4 10l6.5-1.5L12 2z"
              />
            </svg>
            Gemini
          </button>
          <button
            type="button"
            @click="form.platform = 'antigravity'"
            :class="[
              'flex h-9 flex-1 items-center justify-center gap-2 rounded-control px-4 py-1.5 text-sm font-medium transition-all',
              form.platform === 'antigravity'
                ? 'bg-white text-purple-600 shadow-sm dark:bg-dark-600 dark:text-purple-400'
                : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200'
            ]"
          >
            <Icon name="cloud" size="sm" />
            Antigravity
          </button>
          <button
            type="button"
            @click="form.platform = 'qoder'"
            data-testid="create-provider-platform-qoder"
            :class="[
              'flex h-9 flex-1 items-center justify-center gap-2 rounded-control px-4 py-1.5 text-sm font-medium transition-all',
              form.platform === 'qoder'
                ? 'bg-white text-cyan-600 shadow-sm dark:bg-dark-600 dark:text-cyan-300'
                : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200'
            ]"
          >
            <Icon name="terminal" size="sm" />
            Qoder
          </button>
          <button
            type="button"
            @click="form.platform = 'grok'"
            :class="[
              'flex h-9 flex-1 items-center justify-center gap-2 rounded-control px-4 py-1.5 text-sm font-medium transition-all',
              form.platform === 'grok'
                ? 'bg-white text-zinc-900 shadow-sm dark:bg-dark-600 dark:text-zinc-100'
                : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200'
            ]"
          >
            <PlatformIcon platform="grok" size="sm" />
            Grok
          </button>
        </div>
        <!-- 国产供应商行：Kimi / 智谱 GLM / DeepSeek -->
        <div class="mt-2 flex flex-wrap rounded-control bg-gray-100 p-1 dark:bg-dark-700">
          <button
            type="button"
            @click="selectCNPlatform('kimi')"
            :class="[
              'flex h-9 flex-1 items-center justify-center gap-2 rounded-control px-4 py-1.5 text-sm font-medium transition-all',
              form.platform === 'kimi'
                ? 'bg-white text-pink-600 shadow-sm dark:bg-dark-600 dark:text-pink-400'
                : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200'
            ]"
          >
            <PlatformIcon platform="kimi" size="sm" />
            Kimi
          </button>
          <button
            type="button"
            @click="selectCNPlatform('zhipu')"
            :class="[
              'flex h-9 flex-1 items-center justify-center gap-2 rounded-control px-4 py-1.5 text-sm font-medium transition-all',
              form.platform === 'zhipu'
                ? 'bg-white text-indigo-600 shadow-sm dark:bg-dark-600 dark:text-indigo-400'
                : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200'
            ]"
          >
            <PlatformIcon platform="zhipu" size="sm" />
            Zhipu GLM
          </button>
          <button
            type="button"
            @click="selectCNPlatform('deepseek')"
            :class="[
              'flex h-9 flex-1 items-center justify-center gap-2 rounded-control px-4 py-1.5 text-sm font-medium transition-all',
              form.platform === 'deepseek'
                ? 'bg-white text-teal-600 shadow-sm dark:bg-dark-600 dark:text-teal-400'
                : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200'
            ]"
          >
            <PlatformIcon platform="deepseek" size="sm" />
            DeepSeek
          </button>
          <!-- 新平台沿用同组分段选项样式，不叠加操作按钮的常驻硬阴影。 -->
          <button
            type="button"
            @click="selectCNPlatform('minimax')"
            :class="[
              'flex h-9 flex-1 items-center justify-center gap-2 rounded-control px-4 py-1.5 text-sm font-medium transition-all',
              form.platform === 'minimax'
                ? 'bg-white text-red-600 shadow-sm dark:bg-dark-600 dark:text-red-400'
                : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200'
            ]"
            :aria-pressed="form.platform === 'minimax'"
          >
            <PlatformIcon platform="minimax" size="sm" />
            MiniMax
          </button>
          <button
            type="button"
            @click="selectCNPlatform('opencode_go')"
            :class="[
              'flex h-9 flex-1 items-center justify-center gap-2 rounded-control px-4 py-1.5 text-sm font-medium transition-all',
              form.platform === 'opencode_go'
                ? 'bg-white text-blue-600 shadow-sm dark:bg-dark-600 dark:text-blue-400'
                : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200'
            ]"
            :aria-pressed="form.platform === 'opencode_go'"
          >
            <Icon name="terminal" size="sm" />OpenCode
          </button>
        </div>
      </div>

      <!-- Provider Type Selection (Anthropic) -->
      <div v-if="form.platform === 'anthropic'">
        <label class="input-label">{{ t('admin.providers.providerType') }}</label>
        <div class="mt-2 grid grid-cols-2 gap-3 sm:grid-cols-4" data-tour="provider-form-type">
          <button
            type="button"
            @click="providerCategory = 'oauth-based'"
            :class="[
              'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
              providerCategory === 'oauth-based'
                ? 'border-orange-500 bg-orange-50 dark:bg-orange-900/20'
                : 'border-gray-200 hover:border-orange-300 dark:border-dark-600 dark:hover:border-orange-700'
            ]"
          >
            <div
              :class="[
                'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                providerCategory === 'oauth-based'
                  ? 'bg-orange-500 text-white'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
              ]"
            >
              <Icon name="sparkles" size="sm" />
            </div>
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">{{
                t('admin.providers.claudeCode')
              }}</span>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{
                t('admin.providers.oauthSetupToken')
              }}</span>
            </div>
          </button>

          <button
            type="button"
            @click="providerCategory = 'apikey'"
            :class="[
              'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
              providerCategory === 'apikey'
                ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
                : 'border-gray-200 hover:border-purple-300 dark:border-dark-600 dark:hover:border-purple-700'
            ]"
          >
            <div
              :class="[
                'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                providerCategory === 'apikey'
                  ? 'bg-purple-500 text-white'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
              ]"
            >
              <Icon name="key" size="sm" />
            </div>
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">{{
                t('admin.providers.claudeConsole')
              }}</span>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{
                t('admin.providers.apiKey')
              }}</span>
            </div>
          </button>

          <button
            type="button"
            @click="providerCategory = 'bedrock'"
            :class="[
              'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
              providerCategory === 'bedrock'
                ? 'border-amber-500 bg-amber-50 dark:bg-amber-900/20'
                : 'border-gray-200 hover:border-amber-300 dark:border-dark-600 dark:hover:border-amber-700'
            ]"
          >
            <div
              :class="[
                'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                providerCategory === 'bedrock'
                  ? 'bg-amber-500 text-white'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
              ]"
            >
              <Icon name="cloud" size="sm" />
            </div>
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">{{
                t('admin.providers.bedrockLabel')
              }}</span>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{
                t('admin.providers.bedrockDesc')
              }}</span>
            </div>
          </button>

          <button
            type="button"
            @click="providerCategory = 'service_account'"
            :class="[
              'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
              providerCategory === 'service_account'
                ? 'border-sky-500 bg-sky-50 dark:bg-sky-900/20'
                : 'border-gray-200 hover:border-sky-300 dark:border-dark-600 dark:hover:border-sky-700'
            ]"
          >
            <div
              :class="[
                'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                providerCategory === 'service_account'
                  ? 'bg-sky-500 text-white'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
              ]"
            >
              <Icon name="cloud" size="sm" />
            </div>
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">Vertex</span>
              <span class="text-xs text-gray-500 dark:text-gray-400">Service Account</span>
            </div>
          </button>

        </div>

        <div
          v-if="providerCategory === 'service_account'"
          class="mt-3 rounded-control border border-sky-200 bg-sky-50 px-3 py-2 text-xs text-sky-800 dark:border-sky-800/40 dark:bg-sky-900/20 dark:text-sky-200"
        >
          <p>{{ t('admin.providers.vertexAnthropicHint') }}</p>
        </div>
      </div>

      <!-- Provider Type Selection (OpenAI) -->
      <div v-if="form.platform === 'openai'">
        <label class="input-label">{{ t('admin.providers.providerType') }}</label>
        <div class="mt-2 grid grid-cols-2 gap-3" data-tour="provider-form-type">
          <button
            type="button"
            @click="providerCategory = 'oauth-based'"
            :class="[
              'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
              providerCategory === 'oauth-based'
                ? 'border-green-500 bg-green-50 dark:bg-green-900/20'
                : 'border-gray-200 hover:border-green-300 dark:border-dark-600 dark:hover:border-green-700'
            ]"
          >
            <div
              :class="[
                'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                providerCategory === 'oauth-based'
                  ? 'bg-green-500 text-white'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
              ]"
            >
              <Icon name="key" size="sm" />
            </div>
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">OAuth</span>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.providers.types.chatgptOauth') }}</span>
            </div>
          </button>

          <button
            type="button"
            @click="providerCategory = 'apikey'"
            :class="[
              'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
              providerCategory === 'apikey'
                ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
                : 'border-gray-200 hover:border-purple-300 dark:border-dark-600 dark:hover:border-purple-700'
            ]"
          >
            <div
              :class="[
                'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                providerCategory === 'apikey'
                  ? 'bg-purple-500 text-white'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
              ]"
            >
              <Icon name="key" size="sm" />
            </div>
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">API Key</span>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.providers.types.responsesApi') }}</span>
            </div>
          </button>

        </div>
      </div>

      <!-- 提供商类型选择（Grok） -->
      <div v-if="form.platform === 'grok'">
        <label class="input-label">{{ t('admin.providers.providerType') }}</label>
        <div class="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2" data-tour="provider-form-type">
          <button
            type="button"
            @click="providerCategory = 'oauth-based'"
            :class="[
              'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
              providerCategory === 'oauth-based'
                ? 'border-zinc-800 bg-zinc-50 dark:bg-zinc-900/30'
                : 'border-gray-200 hover:border-zinc-400 dark:border-dark-600 dark:hover:border-zinc-600'
            ]"
          >
            <div
              :class="[
                'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                providerCategory === 'oauth-based'
                  ? 'bg-zinc-900 text-white'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
              ]"
            >
              <PlatformIcon platform="grok" size="sm" />
            </div>
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">OAuth</span>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.providers.types.grokOauth') }}</span>
            </div>
          </button>

          <button
            type="button"
            data-testid="grok-provider-type-api-key"
            @click="providerCategory = 'apikey'"
            :class="[
              'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
              providerCategory === 'apikey'
                ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
                : 'border-gray-200 hover:border-purple-300 dark:border-dark-600 dark:hover:border-purple-700'
            ]"
          >
            <div
              :class="[
                'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                providerCategory === 'apikey'
                  ? 'bg-purple-500 text-white'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
              ]"
            >
              <Icon name="key" size="sm" />
            </div>
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">API Key</span>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.providers.types.responsesApi') }}</span>
            </div>
          </button>
        </div>
      </div>

      <!-- 模式与协议分流保留本地供应商能力，统一使用同一组控件。 -->
      <div v-if="isCNPlatform">
        <label class="input-label">{{ t('admin.providers.cnProviders.providerMode.title') }}</label>
        <div class="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2" data-tour="provider-form-mode">
          <button v-for="mode in cnModeOptions" :key="mode" type="button" :aria-pressed="providerMode === mode" @click="providerMode = mode"
            :class="['flex items-center gap-3 border-2 p-3 text-left transition-all', providerMode === mode ? cnAccentActiveClass : 'border-gray-200 hover:border-gray-400 dark:border-dark-600 dark:hover:border-gray-600']">
            <span :class="['flex h-8 w-8 shrink-0 items-center justify-center', providerMode === mode ? cnAccentIconClass : 'bg-gray-100 dark:bg-dark-600']"><Icon :name="mode === 'payg' || mode === 'zen' ? 'creditCard' : 'bolt'" size="sm" /></span>
            <div>
              <span class="block text-sm font-bold">{{ t('admin.providers.cnProviders.providerMode.' + mode) }}</span>
              <span class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.providers.cnProviders.providerMode.' + mode + 'Desc') }}</span>
            </div>
          </button>
        </div>
        <OpenCodeGoProtocolRulesEditor v-if="form.platform === 'opencode_go' && apiProtocol === 'adaptive'" v-model:rows="openCodeRules" :plan="providerMode === 'zen' ? 'zen' : 'go'" class="mt-4" />
      </div>

      <!-- 智谱团队版 Coding Plan：组织/项目 ID 可选，填写组织 ID 后切换团队额度端点。 -->
      <div v-if="form.platform === 'zhipu' && providerMode === 'coding'" class="mt-4">
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
              v-model="zhipuOrganization"
              type="text"
              class="input"
              :placeholder="t('admin.providers.cnProviders.zhipuTeam.organizationPlaceholder')"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.providers.cnProviders.zhipuTeam.project') }}</label>
            <input
              v-model="zhipuProject"
              type="text"
              class="input"
              :placeholder="t('admin.providers.cnProviders.zhipuTeam.projectPlaceholder')"
            />
          </div>
        </div>
        <p class="input-hint mt-2">{{ t('admin.providers.cnProviders.zhipuTeam.hint') }}</p>
      </div>

      <!-- Provider Type Selection (Gemini) -->
      <div v-if="form.platform === 'gemini'">
        <div class="flex items-center justify-between">
          <label class="input-label">{{ t('admin.providers.providerType') }}</label>
          <button
            type="button"
            @click="showGeminiHelpDialog = true"
            class="flex items-center gap-1 rounded-compact px-2 py-1 text-xs text-blue-600 hover:bg-blue-50 dark:text-blue-400 dark:hover:bg-blue-900/20"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9 5.25h.008v.008H12v-.008z" />
            </svg>
            {{ t('admin.providers.gemini.helpButton') }}
          </button>
        </div>
        <div class="mt-2 grid grid-cols-3 gap-3" data-tour="provider-form-type">
          <button
            type="button"
            @click="providerCategory = 'oauth-based'"
            :class="[
              'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
              providerCategory === 'oauth-based'
                ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                : 'border-gray-200 hover:border-blue-300 dark:border-dark-600 dark:hover:border-blue-700'
            ]"
          >
            <div
              :class="[
                'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                providerCategory === 'oauth-based'
                  ? 'bg-blue-500 text-white'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
              ]"
            >
              <Icon name="key" size="sm" />
            </div>
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">
                {{ t('admin.providers.gemini.providerType.oauthTitle') }}
              </span>
              <span class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.gemini.providerType.oauthDesc') }}
              </span>
            </div>
          </button>

          <button
            type="button"
            @click="providerCategory = 'apikey'"
            data-testid="create-gemini-apikey-type"
            :class="[
              'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
              providerCategory === 'apikey'
                ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
                : 'border-gray-200 hover:border-purple-300 dark:border-dark-600 dark:hover:border-purple-700'
            ]"
          >
            <div
              :class="[
                'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                providerCategory === 'apikey'
                  ? 'bg-purple-500 text-white'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
              ]"
            >
              <svg
                class="h-4 w-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1721.75 8.25z"
                />
              </svg>
            </div>
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">
                {{ t('admin.providers.gemini.providerType.apiKeyTitle') }}
              </span>
              <span class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.gemini.providerType.apiKeyDesc') }}
              </span>
            </div>
          </button>

          <button
            type="button"
            @click="providerCategory = 'service_account'"
            :class="[
              'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
              providerCategory === 'service_account'
                ? 'border-sky-500 bg-sky-50 dark:bg-sky-900/20'
                : 'border-gray-200 hover:border-sky-300 dark:border-dark-600 dark:hover:border-sky-700'
            ]"
          >
            <div
              :class="[
                'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                providerCategory === 'service_account'
                  ? 'bg-sky-500 text-white'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
              ]"
            >
              <Icon name="cloud" size="sm" />
            </div>
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">
                Vertex
              </span>
              <span class="text-xs text-gray-500 dark:text-gray-400">
                Service Account
              </span>
            </div>
          </button>
        </div>

        <div
          v-if="providerCategory === 'apikey' && geminiProviderType === 'official'"
          class="mt-3 rounded-control border border-purple-200 bg-purple-50 px-3 py-2 text-xs text-purple-800 dark:border-purple-800/40 dark:bg-purple-900/20 dark:text-purple-200"
        >
          <p>{{ t('admin.providers.gemini.providerType.apiKeyNote') }}</p>
          <div class="mt-2 flex flex-wrap gap-2">
            <a
              :href="geminiHelpLinks.apiKey"
              class="font-medium text-blue-600 hover:underline dark:text-blue-400"
              target="_blank"
              rel="noreferrer"
            >
              {{ t('admin.providers.gemini.providerType.apiKeyLink') }}
            </a>
          </div>
        </div>

        <div
          v-if="providerCategory === 'service_account'"
          class="mt-3 rounded-control border border-sky-200 bg-sky-50 px-3 py-2 text-xs text-sky-800 dark:border-sky-800/40 dark:bg-sky-900/20 dark:text-sky-200"
        >
          <p>{{ t('admin.providers.vertexGeminiHint') }}</p>
        </div>

        <!-- OAuth Type Selection (only show when oauth-based is selected) -->
        <div v-if="providerCategory === 'oauth-based'" class="mt-4">
          <label class="input-label">{{ t('admin.providers.oauth.gemini.oauthTypeLabel') }}</label>
          <div class="mt-2 grid grid-cols-2 gap-3">
            <!-- Google One OAuth -->
            <button
              type="button"
              @click="handleSelectGeminiOAuthType('google_one')"
              :class="[
                'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
                geminiOAuthType === 'google_one'
                  ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
                  : 'border-gray-200 hover:border-purple-300 dark:border-dark-600 dark:hover:border-purple-700'
              ]"
            >
              <div
                :class="[
                  'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                  geminiOAuthType === 'google_one'
                    ? 'bg-purple-500 text-white'
                    : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
                ]"
              >
                <Icon name="user" size="sm" />
              </div>
              <div class="min-w-0">
                <span class="block text-sm font-medium text-gray-900 dark:text-white">
                  Google One
                </span>
                <span class="text-xs text-gray-500 dark:text-gray-400">
                  个人提供商，享受 Google One 订阅配额
                </span>
                <div class="mt-2 flex flex-wrap gap-1">
                  <span
                    class="rounded-compact bg-purple-100 px-2 py-0.5 text-xs font-semibold text-purple-700 dark:bg-purple-900/40 dark:text-purple-300"
                  >
                    推荐个人用户
                  </span>
                  <span
                    class="rounded-compact bg-emerald-100 px-2 py-0.5 text-xs font-semibold text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300"
                  >
                    无需 GCP
                  </span>
                </div>
              </div>
            </button>

            <!-- GCP Code Assist OAuth -->
            <button
              type="button"
              @click="handleSelectGeminiOAuthType('code_assist')"
              :class="[
                'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
                geminiOAuthType === 'code_assist'
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                  : 'border-gray-200 hover:border-blue-300 dark:border-dark-600 dark:hover:border-blue-700'
              ]"
            >
              <div
                :class="[
                  'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                  geminiOAuthType === 'code_assist'
                    ? 'bg-blue-500 text-white'
                    : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
                ]"
              >
                <Icon name="cloud" size="sm" />
              </div>
              <div class="min-w-0">
                <span class="block text-sm font-medium text-gray-900 dark:text-white">
                  GCP Code Assist
                </span>
                <span class="text-xs text-gray-500 dark:text-gray-400">
                  企业级，需要 GCP 项目
                </span>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  需要激活 GCP 项目并绑定信用卡
                  <a
                    :href="geminiHelpLinks.gcpProject"
                    class="ml-1 text-blue-600 hover:underline dark:text-blue-400"
                    target="_blank"
                    rel="noreferrer"
                  >
                    {{ t('admin.providers.gemini.oauthType.gcpProjectLink') }}
                  </a>
                </div>
                <div class="mt-2 flex flex-wrap gap-1">
                  <span
                    class="rounded-compact bg-blue-100 px-2 py-0.5 text-xs font-semibold text-blue-700 dark:bg-blue-900/40 dark:text-blue-300"
                  >
                    企业用户
                  </span>
                  <span
                    class="rounded-compact bg-emerald-100 px-2 py-0.5 text-xs font-semibold text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300"
                  >
                    高并发
                  </span>
                </div>
              </div>
            </button>
          </div>

          <!-- Advanced Options Toggle -->
          <div class="mt-3">
            <button
              type="button"
              @click="showAdvancedOAuth = !showAdvancedOAuth"
              class="flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200"
            >
              <svg
                :class="['h-4 w-4 transition-transform', showAdvancedOAuth ? 'rotate-90' : '']"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="2"
              >
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
              </svg>
              <span>{{ showAdvancedOAuth ? '隐藏' : '显示' }}高级选项（自建 OAuth Client）</span>
            </button>
          </div>

          <!-- Custom OAuth Client (Advanced) -->
          <div v-if="showAdvancedOAuth" class="mt-3 group relative">
            <button
              type="button"
              :disabled="!geminiAIStudioOAuthEnabled"
              @click="handleSelectGeminiOAuthType('ai_studio')"
              :class="[
                'flex w-full items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
                !geminiAIStudioOAuthEnabled ? 'cursor-not-allowed opacity-60' : '',
                geminiOAuthType === 'ai_studio'
                  ? 'border-amber-500 bg-amber-50 dark:bg-amber-900/20'
                  : 'border-gray-200 hover:border-amber-300 dark:border-dark-600 dark:hover:border-amber-700'
              ]"
            >
              <div
                :class="[
                  'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                  geminiOAuthType === 'ai_studio'
                    ? 'bg-amber-500 text-white'
                    : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
                ]"
              >
                <svg
                  class="h-4 w-4"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  stroke-width="1.5"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z"
                  />
                </svg>
              </div>
              <div class="min-w-0">
                <span class="block text-sm font-medium text-gray-900 dark:text-white">
                  {{ t('admin.providers.gemini.oauthType.customTitle') }}
                </span>
                <span class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.providers.gemini.oauthType.customDesc') }}
                </span>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.providers.gemini.oauthType.customRequirement') }}
                </div>
                <div class="mt-2 flex flex-wrap gap-1">
                  <span
                    class="rounded-compact bg-amber-100 px-2 py-0.5 text-xs font-semibold text-amber-700 dark:bg-amber-900/40 dark:text-amber-300"
                  >
                    {{ t('admin.providers.gemini.oauthType.badges.orgManaged') }}
                  </span>
                  <span
                    class="rounded-compact bg-amber-100 px-2 py-0.5 text-xs font-semibold text-amber-700 dark:bg-amber-900/40 dark:text-amber-300"
                  >
                    {{ t('admin.providers.gemini.oauthType.badges.adminRequired') }}
                  </span>
                </div>
              </div>
              <span
                v-if="!geminiAIStudioOAuthEnabled"
                class="ml-auto shrink-0 rounded-compact bg-amber-100 px-2 py-0.5 text-xs text-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
              >
                {{ t('admin.providers.oauth.gemini.aiStudioNotConfiguredShort') }}
              </span>
            </button>

            <div
              v-if="!geminiAIStudioOAuthEnabled"
              class="pointer-events-none absolute right-0 top-full z-50 mt-2 w-80 rounded-control border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800 opacity-0 shadow-lg transition-opacity group-hover:opacity-100 dark:border-amber-700 dark:bg-amber-900/40 dark:text-amber-200"
            >
              {{ t('admin.providers.oauth.gemini.aiStudioNotConfiguredTip') }}
            </div>
          </div>
        </div>

        <!-- Tier selection (used as fallback when auto-detection is unavailable/fails) -->
        <div v-if="providerCategory === 'oauth-based'" class="mt-4">
          <label class="input-label">{{ t('admin.providers.gemini.tier.label') }}</label>
          <div class="mt-2">
            <Select
              v-if="geminiOAuthType === 'google_one'"
              v-model="geminiTierGoogleOne"
              :options="geminiGoogleOneTierOptions"
            />

            <Select
              v-else-if="geminiOAuthType === 'code_assist'"
              v-model="geminiTierGcp"
              :options="geminiGCPTierOptions"
            />

            <Select
              v-else
              v-model="geminiTierAIStudio"
              :options="geminiAIStudioTierOptions"
            />
          </div>
          <p class="input-hint">{{ t('admin.providers.gemini.tier.hint') }}</p>
        </div>
      </div>

      <!-- Provider Type Selection (Antigravity - OAuth or Upstream) -->
      <div v-if="form.platform === 'antigravity'">
        <label class="input-label">{{ t('admin.providers.providerType') }}</label>
        <div class="mt-2 grid grid-cols-2 gap-3">
          <button
            type="button"
            @click="antigravityProviderType = 'oauth'"
            :class="[
              'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
              antigravityProviderType === 'oauth'
                ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
                : 'border-gray-200 hover:border-purple-300 dark:border-dark-600 dark:hover:border-purple-700'
            ]"
          >
            <div
              :class="[
                'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                antigravityProviderType === 'oauth'
                  ? 'bg-purple-500 text-white'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
              ]"
            >
              <Icon name="key" size="sm" />
            </div>
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">OAuth</span>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.providers.types.antigravityOauth') }}</span>
            </div>
          </button>

          <button
            type="button"
            @click="antigravityProviderType = 'upstream'"
            :class="[
              'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
              antigravityProviderType === 'upstream'
                ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
                : 'border-gray-200 hover:border-purple-300 dark:border-dark-600 dark:hover:border-purple-700'
            ]"
          >
            <div
              :class="[
                'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                antigravityProviderType === 'upstream'
                  ? 'bg-purple-500 text-white'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
              ]"
            >
              <Icon name="cloud" size="sm" />
            </div>
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">API Key</span>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.providers.types.antigravityApikey') }}</span>
            </div>
          </button>
        </div>
      </div>

      <div v-if="form.platform === 'antigravity' && antigravityProviderType === 'oauth'">
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

      <!-- Upstream config (only for Antigravity upstream type) -->
      <div v-if="form.platform === 'antigravity' && antigravityProviderType === 'upstream'" class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.providers.upstream.baseUrl') }}</label>
          <input
            v-model="upstreamBaseUrl"
            type="text"
            required
            class="input"
            placeholder="https://cloudcode-pa.googleapis.com"
          />
          <p class="input-hint">{{ t('admin.providers.upstream.baseUrlHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.providers.upstream.apiKey') }}</label>
          <input
            v-model="upstreamApiKey"
            type="password"
            required
            class="input font-mono"
            placeholder="sk-..."
          />
          <p class="input-hint">{{ t('admin.providers.upstream.apiKeyHint') }}</p>
        </div>
      </div>

      <!-- Qoder 站点选择，必须在登录方式之前冻结。 -->
      <div v-if="form.platform === 'qoder'" class="space-y-2">
        <label class="input-label">{{ t('admin.providers.qoder.site.label') }}</label>
        <div class="grid grid-cols-2 gap-2" role="group" :aria-label="t('admin.providers.qoder.site.label')">
          <button
            type="button"
            data-testid="create-qoder-site-global"
            :disabled="submitting || isQoderOAuthProviderCreating"
            :aria-pressed="qoderSite === 'global'"
            @click="qoderSite = 'global'"
            :class="[
              'rounded-control border px-4 py-2 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50',
              qoderSite === 'global'
                ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300'
                : 'border-gray-200 bg-white text-gray-600 hover:bg-gray-50 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-300'
            ]"
          >
            {{ t('admin.providers.qoder.site.global') }}
          </button>
          <button
            type="button"
            data-testid="create-qoder-site-cn"
            :disabled="submitting || isQoderOAuthProviderCreating"
            :aria-pressed="qoderSite === 'cn'"
            @click="qoderSite = 'cn'"
            :class="[
              'rounded-control border px-4 py-2 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50',
              qoderSite === 'cn'
                ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300'
                : 'border-gray-200 bg-white text-gray-600 hover:bg-gray-50 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-300'
            ]"
          >
            {{ t('admin.providers.qoder.site.cn') }}
          </button>
        </div>
      </div>

      <!-- Qoder 提供商类型选择 -->
      <div v-if="form.platform === 'qoder'">
        <label class="input-label">{{ t('admin.providers.providerType') }}</label>
        <div class="mt-2 grid grid-cols-2 gap-3">
          <button
            type="button"
            @click="qoderProviderType = 'oauth'"
            :class="[
              'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
              qoderProviderType === 'oauth'
                ? 'border-cyan-500 bg-cyan-50 dark:bg-cyan-900/20'
                : 'border-gray-200 hover:border-cyan-300 dark:border-dark-600 dark:hover:border-cyan-700'
            ]"
          >
            <div
              :class="[
                'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                qoderProviderType === 'oauth'
                  ? 'bg-cyan-500 text-white'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
              ]"
            >
              <Icon name="link" size="sm" />
            </div>
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">
                {{ t('admin.providers.qoder.providerType.oauthTitle') }}
              </span>
              <span class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.qoder.providerType.oauthDesc') }}
              </span>
            </div>
          </button>

          <button
            type="button"
            @click="qoderProviderType = 'manual'"
            :class="[
              'flex items-center gap-3 rounded-control border-2 p-3 text-left transition-all',
              qoderProviderType === 'manual'
                ? 'border-cyan-500 bg-cyan-50 dark:bg-cyan-900/20'
                : 'border-gray-200 hover:border-cyan-300 dark:border-dark-600 dark:hover:border-cyan-700'
            ]"
          >
            <div
              :class="[
                'flex h-8 w-8 shrink-0 items-center justify-center rounded-control',
                qoderProviderType === 'manual'
                  ? 'bg-cyan-500 text-white'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
              ]"
            >
              <Icon name="key" size="sm" />
            </div>
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">
                {{ t('admin.providers.qoder.providerType.manualTitle') }}
              </span>
              <span class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.qoder.providerType.manualDesc') }}
              </span>
            </div>
          </button>
        </div>
      </div>

      <!-- Qoder 手动凭据 -->
      <div v-if="form.platform === 'qoder' && qoderProviderType === 'manual'" class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.providers.qoder.pat') }}</label>
          <input
            v-model="qoderPAT"
            type="password"
            class="input font-mono"
            autocomplete="off"
            placeholder="pat-..."
          />
          <p class="input-hint">{{ t('admin.providers.qoder.patHint') }}</p>
        </div>
        <div class="flex items-center gap-3">
          <div class="h-px flex-1 bg-gray-200 dark:bg-dark-600"></div>
          <span class="text-xs uppercase tracking-wide text-gray-400">{{ t('common.or') }}</span>
          <div class="h-px flex-1 bg-gray-200 dark:bg-dark-600"></div>
        </div>
        <div>
          <label class="input-label">{{ t('admin.providers.qoder.securityOauthToken') }}</label>
          <input
            v-model="qoderSecurityOauthToken"
            type="password"
            class="input font-mono"
            autocomplete="off"
            placeholder="dt-..."
          />
          <p class="input-hint">{{ t('admin.providers.qoder.securityOauthTokenHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.providers.qoder.machineId') }}</label>
          <input
            v-model="qoderMachineId"
            type="text"
            class="input font-mono"
            autocomplete="off"
            placeholder="machine_id"
          />
          <p class="input-hint">{{ t('admin.providers.qoder.machineIdHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.providers.qoder.uidAid') }}</label>
          <input
            v-model="qoderUidAid"
            type="text"
            class="input font-mono"
            autocomplete="off"
            placeholder="uid or aid"
          />
          <p class="input-hint">{{ t('admin.providers.qoder.uidAidHint') }}</p>
        </div>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.providers.qoder.refreshToken') }}</label>
            <input
              v-model="qoderRefreshToken"
              type="password"
              class="input font-mono"
              autocomplete="off"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.providers.qoder.userType') }}</label>
            <input
              v-model="qoderUserType"
              type="text"
              class="input font-mono"
              autocomplete="off"
              placeholder="personal_standard"
            />
          </div>
        </div>
      </div>

      <!-- Qoder 模型限制，适用于 OAuth 和手动凭据 -->
      <div v-if="form.platform === 'qoder'" class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <label class="input-label">{{ t('admin.providers.modelRestriction') }}</label>

        <div class="mb-4 flex gap-2">
          <button
            type="button"
            @click="modelRestrictionMode = 'whitelist'"
            :class="[
              'flex-1 rounded-control px-4 py-2 text-sm font-medium transition-all',
              modelRestrictionMode === 'whitelist'
                ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
            ]"
          >
            {{ t('admin.providers.modelWhitelist') }}
          </button>
          <button
            type="button"
            @click="modelRestrictionMode = 'mapping'"
            :class="[
              'flex-1 rounded-control px-4 py-2 text-sm font-medium transition-all',
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

        <div v-if="modelRestrictionMode === 'whitelist'">
          <ModelWhitelistSelector :model-value="allowedModels" platform="qoder" :models="qoderAvailableModels" @update:model-value="setAllowedModels" />
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.providers.selectedModels', { count: allowedModels.length }) }}
            <span v-if="allowedModels.length === 0">{{ t('admin.providers.supportsAllModels') }}</span>
          </p>
        </div>

        <div v-else>
          <div class="mb-3 rounded-control bg-purple-50 p-3 dark:bg-purple-900/20">
            <p class="text-xs text-purple-700 dark:text-purple-400">
              {{ t('admin.providers.mapRequestModels') }}
            </p>
          </div>

          <div v-if="modelMappings.length > 0" class="mb-3 space-y-2">
            <div
              v-for="(mapping, index) in modelMappings"
              :key="'qoder-' + getModelMappingKey(mapping)"
              class="flex items-center gap-2"
            >
              <input
                v-model="mapping.from"
                type="text"
                class="input flex-1"
                :placeholder="t('admin.providers.requestModel')"
              />
              <svg class="h-4 w-4 flex-shrink-0 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
              </svg>
              <input
                v-model="mapping.to"
                type="text"
                class="input flex-1"
                :placeholder="t('admin.providers.actualModel')"
              />
              <button
                type="button"
                @click="removeModelMapping(index)"
                class="rounded-control p-2 text-red-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
              >
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                  />
                </svg>
              </button>
            </div>
          </div>

          <button
            type="button"
            @click="addModelMapping"
            class="mb-3 w-full rounded-control border-2 border-dashed border-gray-300 px-4 py-2 text-gray-600 transition-colors hover:border-gray-400 hover:text-gray-700 dark:border-dark-500 dark:text-gray-400 dark:hover:border-dark-400 dark:hover:text-gray-300"
          >
            + {{ t('admin.providers.addMapping') }}
          </button>

          <div class="flex flex-wrap gap-2">
            <button
              v-for="preset in presetMappings"
              :key="'qoder-' + preset.label"
              type="button"
              @click="addPresetMapping(preset.from, preset.to)"
              :class="['rounded-control px-3 py-1 text-xs transition-colors', preset.color]"
            >
              + {{ preset.label }}
            </button>
          </div>
        </div>
      </div>

      <!-- Qoder COSY TLS 指纹伪装 -->
      <div
        v-if="form.platform === 'qoder'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{ t('admin.providers.quotaControl.tlsFingerprint.label') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.quotaControl.tlsFingerprint.hint') }}
            </p>
          </div>
          <Toggle v-model="tlsFingerprintEnabled" variant="flush" off-tone="soft" data-testid="create-qoder-tls-fingerprint-toggle" />
        </div>
        <div v-if="tlsFingerprintEnabled" class="mt-3 space-y-3">
          <Select
            v-model="tlsFingerprintProfileId"
            data-testid="create-qoder-tls-fingerprint-profile"
            :options="tlsFingerprintProfileOptions"
          />
        </div>
      </div>

      <!-- Vertex Service Account -->
      <div v-if="(form.platform === 'gemini' || form.platform === 'anthropic') && providerCategory === 'service_account'" class="space-y-4">
        <div>
          <label class="input-label">Service Account JSON</label>
          <input
            ref="vertexServiceAccountFileInput"
            type="file"
            accept="application/json,.json"
            class="hidden"
            @change="handleVertexServiceAccountFile"
          />
          <div
            :class="[
              'rounded-control border-2 border-dashed px-4 py-5 transition-colors',
              vertexServiceAccountDragActive
                ? 'border-sky-500 bg-sky-50 dark:border-sky-500 dark:bg-sky-900/20'
                : 'border-gray-300 bg-gray-50 hover:border-sky-400 hover:bg-sky-50/60 dark:border-dark-500 dark:bg-dark-700/40 dark:hover:border-sky-600 dark:hover:bg-sky-900/10'
            ]"
            @dragenter.prevent="vertexServiceAccountDragActive = true"
            @dragover.prevent="vertexServiceAccountDragActive = true"
            @dragleave.prevent="vertexServiceAccountDragActive = false"
            @drop.prevent="handleVertexServiceAccountDrop"
          >
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div class="min-w-0">
                <div class="flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white">
                  <Icon name="upload" size="sm" />
                  <span>{{ vertexClientEmail ? t('admin.providers.vertexSaJsonLoaded') : t('admin.providers.vertexSaJsonDrop') }}</span>
                </div>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ vertexClientEmail ? t('admin.providers.vertexSaJsonKeyHidden') : t('admin.providers.vertexSaJsonDropHint') }}
                </p>
              </div>
              <button
                type="button"
                class="btn btn-secondary shrink-0"
                @click="vertexServiceAccountFileInput?.click()"
              >
                <Icon name="upload" size="sm" />
                {{ t('admin.providers.vertexSaJsonSelectBtn') }}
              </button>
            </div>
            <div
              v-if="vertexClientEmail"
              class="mt-3 rounded-control border border-sky-200 bg-white px-3 py-2 text-xs text-sky-900 dark:border-sky-800/50 dark:bg-dark-800 dark:text-sky-200"
            >
              <div class="truncate">Project ID: <span class="font-mono">{{ vertexProjectId }}</span></div>
              <div class="truncate">Client Email: <span class="font-mono">{{ vertexClientEmail }}</span></div>
            </div>
          </div>
          <p class="input-hint">{{ t('admin.providers.vertexSaJsonUploadHint') }}</p>
        </div>

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">Project ID</label>
            <input
              v-model="vertexProjectId"
              type="text"
              class="input font-mono"
              readonly
              :placeholder="t('admin.providers.vertexProjectIdPlaceholder')"
            />
          </div>
          <div>
            <label class="input-label">Location</label>
            <Select
              v-model="vertexLocation"
              :options="vertexLocationOptions"
              class="font-mono"
              searchable
            />
            <p class="input-hint">{{ t('admin.providers.vertexLocationHint') }}</p>
          </div>
        </div>
      </div>

      <!-- Antigravity model restriction (applies to OAuth + Upstream) -->
      <!-- 白名单与映射分别控制最终范围和请求改写。 -->
      <div v-if="form.platform === 'antigravity'" class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <label class="input-label">{{ t('admin.providers.modelRestriction') }}</label>
        <p class="input-hint">{{ t('admin.providers.selectAllowedModels') }}</p>
        <ModelWhitelistSelector v-model="antigravityWhitelistModels" platform="antigravity" />

        <!-- Mapping Mode Only (no toggle for Antigravity) -->
        <div>
          <div class="mb-3 rounded-control bg-purple-50 p-3 dark:bg-purple-900/20">
            <p class="text-xs text-purple-700 dark:text-purple-400">
              {{ t('admin.providers.mapRequestModels') }}
            </p>
          </div>

          <div v-if="antigravityModelMappings.length > 0" class="mb-3 space-y-2">
            <div
              v-for="(mapping, index) in antigravityModelMappings"
              :key="getAntigravityModelMappingKey(mapping)"
              class="space-y-1"
            >
              <div class="flex items-center gap-2">
                <input
                  v-model="mapping.from"
                  type="text"
                  :class="[
                    'input flex-1',
                    !isValidWildcardPattern(mapping.from) ? 'border-red-500 dark:border-red-500' : ''
                  ]"
                  :placeholder="t('admin.providers.requestModel')"
                />
                <svg class="h-4 w-4 flex-shrink-0 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
                </svg>
                <input
                  v-model="mapping.to"
                  type="text"
                  :class="[
                    'input flex-1',
                    mapping.to.includes('*') ? 'border-red-500 dark:border-red-500' : ''
                  ]"
                  :placeholder="t('admin.providers.actualModel')"
                />
                <button
                  type="button"
                  @click="removeAntigravityModelMapping(index)"
                  class="rounded-control p-2 text-red-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
                >
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                    />
                  </svg>
                </button>
              </div>
              <!-- 校验错误提示 -->
              <p v-if="!isValidWildcardPattern(mapping.from)" class="text-xs text-red-500">
                {{ t('admin.providers.wildcardOnlyAtEnd') }}
              </p>
              <p v-if="mapping.to.includes('*')" class="text-xs text-red-500">
                {{ t('admin.providers.targetNoWildcard') }}
              </p>
            </div>
          </div>

          <button
            type="button"
            @click="addAntigravityModelMapping"
            class="mb-3 w-full rounded-control border-2 border-dashed border-gray-300 px-4 py-2 text-gray-600 transition-colors hover:border-gray-400 hover:text-gray-700 dark:border-dark-500 dark:text-gray-400 dark:hover:border-dark-400 dark:hover:text-gray-300"
          >
            <svg class="mr-1 inline h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            {{ t('admin.providers.addMapping') }}
          </button>

          <div class="flex flex-wrap gap-2">
            <button
              v-for="preset in antigravityPresetMappings"
              :key="preset.label"
              type="button"
              @click="addAntigravityPresetMapping(preset.from, preset.to)"
              :class="['rounded-control px-3 py-1 text-xs transition-colors', preset.color]"
            >
              + {{ preset.label }}
            </button>
          </div>
        </div>
      </div>

      <!-- Add Method (only for Anthropic OAuth-based type) -->
      <div v-if="form.platform === 'anthropic' && isOAuthFlow">
        <label class="input-label">{{ t('admin.providers.addMethod') }}</label>
        <div class="mt-2 flex gap-4">
          <label class="flex cursor-pointer items-center">
            <input
              v-model="addMethod"
              type="radio"
              value="oauth"
              class="mr-2 text-primary-600 focus:ring-primary-500"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.providers.types.oauth') }}</span>
          </label>
          <label class="flex cursor-pointer items-center">
            <input
              v-model="addMethod"
              type="radio"
              value="setup-token"
              class="mr-2 text-primary-600 focus:ring-primary-500"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">{{
              t('admin.providers.setupTokenLongLived')
            }}</span>
          </label>
        </div>
      </div>

      <!-- API Key input (only for apikey type, excluding Antigravity which has its own fields) -->
      <div v-if="form.type === 'apikey' && form.platform !== 'antigravity'" class="space-y-4">
        <div v-if="form.platform === 'gemini'">
          <label class="input-label">{{ t('admin.providers.gemini.connectionSource.label') }}</label>
          <Select
            v-model="geminiProviderType"
            :options="geminiProviderTypeOptions"
            data-testid="create-gemini-provider-type"
          />
          <p class="input-hint">{{ geminiProviderTypeHint }}</p>
        </div>
        <div v-if="!isCNPlatform || apiProtocol !== 'adaptive'">
          <label class="input-label">{{ t('admin.providers.baseUrl') }}</label>
          <input
            v-model="apiKeyBaseUrl"
            type="text"
            class="input"
            data-testid="create-provider-base-url"
            :placeholder="
              form.platform === 'openai'
                ? 'https://api.openai.com'
                : form.platform === 'gemini'
                  ? geminiProviderType === 'third_party'
                    ? 'https://'
                    : 'https://generativelanguage.googleapis.com'
                  : form.platform === 'grok'
                    ? 'https://api.x.ai/v1'
                    : 'https://api.anthropic.com'
            "
          />
          <p v-if="baseUrlHint" class="input-hint">{{ baseUrlHint }}</p>
          <GrokBaseUrlPresets
            v-if="form.platform === 'grok'"
            class="mt-2"
            @select="apiKeyBaseUrl = $event"
          />
          <CnBaseUrlPresets
            v-if="isCNPlatform"
            class="mt-2"
            :platform="cnPresetPlatform"
            :mode="providerMode"
            :protocol="apiProtocol"
            :current-url="apiKeyBaseUrl"
            @select="onCnPresetSelect"
          />
        </div>
        <div v-else>
          <label class="input-label">{{ t('admin.providers.cnProviders.apiProtocol.endpoints') }}</label>
          <div class="mt-2 space-y-3">
            <div v-for="item in cnAdaptiveProtocolOptions" :key="item.value">
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                {{ t(`admin.providers.cnProviders.apiProtocol.${item.labelKey}`) }}
              </label>
              <input
                v-model="adaptiveBaseUrls[item.value]"
                type="text"
                class="input"
                :data-testid="`cn-adaptive-base-url-${item.value}`"
              />
            </div>
          </div>
          <p v-if="!cnSupportsNativeResponses(form.platform)" class="input-hint">
            {{ t('admin.providers.cnProviders.apiProtocol.responsesFallbackDesc') }}
          </p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.providers.apiKeyRequired') }}</label>
          <input
            v-model="apiKeyValue"
            type="password"
            required
            class="input font-mono"
            :placeholder="
              form.platform === 'openai'
                ? 'sk-proj-...'
                : form.platform === 'gemini'
                  ? geminiProviderType === 'third_party'
                    ? 'api-key-...'
                    : 'AIza...'
                  : form.platform === 'grok'
                    ? 'xai-...'
                    : 'sk-ant-...'
            "
          />
          <p v-if="apiKeyHint" class="input-hint">{{ apiKeyHint }}</p>
        </div>

        <!-- Gemini API Key tier selection -->
        <div v-if="form.platform === 'gemini' && geminiProviderType === 'official'" data-testid="create-gemini-tier">
          <label class="input-label">{{ t('admin.providers.gemini.tier.label') }}</label>
          <Select v-model="geminiTierAIStudio" :options="geminiAIStudioTierOptions" data-testid="create-gemini-tier-select" />
          <p class="input-hint">{{ t('admin.providers.gemini.tier.aiStudioHint') }}</p>
        </div>

        <!-- Model Restriction Section (Antigravity 已在上层条件排除) -->
        <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
          <label class="input-label">{{ t('admin.providers.modelRestriction') }}</label>

            <!-- Mode Toggle -->
            <div class="mb-4 flex gap-2">
              <button
                type="button"
                @click="modelRestrictionMode = 'whitelist'"
                :class="[
                  'flex-1 rounded-control px-4 py-2 text-sm font-medium transition-all',
                  modelRestrictionMode === 'whitelist'
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
                ]"
              >
                <svg
                  class="mr-1.5 inline h-4 w-4"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
                {{ t('admin.providers.modelWhitelist') }}
              </button>
              <button
                type="button"
                @click="modelRestrictionMode = 'mapping'"
                :class="[
                  'flex-1 rounded-control px-4 py-2 text-sm font-medium transition-all',
                  modelRestrictionMode === 'mapping'
                    ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
                ]"
              >
                <svg
                  class="mr-1.5 inline h-4 w-4"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"
                  />
                </svg>
                {{ t('admin.providers.modelMapping') }}
              </button>
            </div>
            <p class="mb-3 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.modelRestrictionCombinedHint') }}
            </p>

            <!-- Whitelist Mode -->
            <div v-if="modelRestrictionMode === 'whitelist'">
              <ModelWhitelistSelector :model-value="allowedModels" :platform="form.platform" :sync-credentials="syncPreviewCredentials" @update:model-value="setAllowedModels" />
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.selectedModels', { count: allowedModels.length }) }}
                <span v-if="allowedModels.length === 0">{{
                  t('admin.providers.supportsAllModels')
                }}</span>
              </p>
            </div>

            <!-- Mapping Mode -->
            <div v-else>
              <div class="mb-3 rounded-control bg-purple-50 p-3 dark:bg-purple-900/20">
                <p class="text-xs text-purple-700 dark:text-purple-400">
                  <svg
                    class="mr-1 inline h-4 w-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                  </svg>
                  {{ t('admin.providers.mapRequestModels') }}
                </p>
              </div>

            <!-- Model Mapping List -->
            <div v-if="modelMappings.length > 0" class="mb-3 space-y-2">
              <div
                v-for="(mapping, index) in modelMappings"
                :key="getModelMappingKey(mapping)"
                class="flex items-center gap-2"
              >
                <input
                  v-model="mapping.from"
                  type="text"
                  class="input flex-1"
                  :placeholder="t('admin.providers.requestModel')"
                />
                <svg
                  class="h-4 w-4 flex-shrink-0 text-gray-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M14 5l7 7m0 0l-7 7m7-7H3"
                  />
                </svg>
                <input
                  v-model="mapping.to"
                  type="text"
                  class="input flex-1"
                  :placeholder="t('admin.providers.actualModel')"
                />
                <button
                  type="button"
                  @click="removeModelMapping(index)"
                  class="rounded-control p-2 text-red-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
                >
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                    />
                  </svg>
                </button>
              </div>
            </div>

            <button
              type="button"
              @click="addModelMapping"
              class="mb-3 w-full rounded-control border-2 border-dashed border-gray-300 px-4 py-2 text-gray-600 transition-colors hover:border-gray-400 hover:text-gray-700 dark:border-dark-500 dark:text-gray-400 dark:hover:border-dark-400 dark:hover:text-gray-300"
            >
              <svg
                class="mr-1 inline h-4 w-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 4v16m8-8H4"
                />
              </svg>
              {{ t('admin.providers.addMapping') }}
            </button>

              <!-- Quick Add Buttons -->
              <div class="flex flex-wrap gap-2">
                <button
                  v-for="preset in presetMappings"
                  :key="preset.label"
                  type="button"
                  @click="addPresetMapping(preset.from, preset.to)"
                  :class="['rounded-control px-3 py-1 text-xs transition-colors', preset.color]"
                >
                  + {{ preset.label }}
                </button>
              </div>
            </div>

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
          <div v-if="poolModeEnabled" class="rounded-control bg-blue-50 p-3 dark:bg-blue-900/20">
            <p class="text-xs text-blue-700 dark:text-blue-400">
              <Icon name="exclamationCircle" size="sm" class="mr-1 inline" :stroke-width="2" />
              {{ t('admin.providers.poolModeInfo') }}
            </p>
          </div>
          <div v-if="poolModeEnabled" class="mt-3">
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
          <div v-if="poolModeEnabled" class="mt-3">
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
        </div>

        <!-- Custom Error Codes Section -->
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

          <div v-if="customErrorCodesEnabled" class="space-y-3">
            <div class="rounded-control bg-amber-50 p-3 dark:bg-amber-900/20">
              <p class="text-xs text-amber-700 dark:text-amber-400">
                <Icon name="exclamationTriangle" size="sm" class="mr-1 inline" :stroke-width="2" />
                {{ t('admin.providers.customErrorCodesWarning') }}
              </p>
            </div>

            <!-- Error Code Buttons -->
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

            <!-- Manual input -->
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
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M12 4v16m8-8H4"
                  />
                </svg>
              </button>
            </div>

            <!-- Selected codes summary -->
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
        </div>

        <!-- 请求头覆写区域（支持的平台 API Key 提供商） -->
        <div
          v-if="isHeaderOverrideCapable(form.platform, 'apikey')"
          class="border-t border-gray-200 pt-4 dark:border-dark-600"
        >
          <div class="mb-3 flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.providers.headerOverride.title') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.providers.headerOverride.hint') }}
              </p>
            </div>
            <Toggle v-model="headerOverrideEnabled" variant="flush" off-tone="soft" />
          </div>

          <div v-if="headerOverrideEnabled" class="space-y-3">
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
        </div>

      </div>

      <!-- Bedrock credentials (only for Anthropic Bedrock type) -->
      <div v-if="form.platform === 'anthropic' && providerCategory === 'bedrock'" class="space-y-4">
        <!-- Auth Mode Radio -->
        <div>
          <label class="input-label">{{ t('admin.providers.bedrockAuthMode') }}</label>
          <div class="mt-2 flex gap-4">
            <label class="flex cursor-pointer items-center">
              <input
                v-model="bedrockAuthMode"
                type="radio"
                value="sigv4"
                class="mr-2 text-primary-600 focus:ring-primary-500"
              />
              <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.providers.bedrockAuthModeSigv4') }}</span>
            </label>
            <label class="flex cursor-pointer items-center">
              <input
                v-model="bedrockAuthMode"
                type="radio"
                value="apikey"
                class="mr-2 text-primary-600 focus:ring-primary-500"
              />
              <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.providers.bedrockAuthModeApikey') }}</span>
            </label>
          </div>
        </div>

        <!-- SigV4 fields -->
        <template v-if="bedrockAuthMode === 'sigv4'">
          <div>
            <label class="input-label">{{ t('admin.providers.bedrockAccessKeyId') }}</label>
            <input
              v-model="bedrockAccessKeyId"
              type="text"
              required
              class="input font-mono"
              placeholder="AKIA..."
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.providers.bedrockSecretAccessKey') }}</label>
            <input
              v-model="bedrockSecretAccessKey"
              type="password"
              required
              class="input font-mono"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.providers.bedrockSessionToken') }}</label>
            <input
              v-model="bedrockSessionToken"
              type="password"
              class="input font-mono"
            />
            <p class="input-hint">{{ t('admin.providers.bedrockSessionTokenHint') }}</p>
          </div>
        </template>

        <!-- API Key field -->
        <div v-if="bedrockAuthMode === 'apikey'">
          <label class="input-label">{{ t('admin.providers.bedrockApiKeyInput') }}</label>
          <input
            v-model="bedrockApiKeyValue"
            type="password"
            required
            class="input font-mono"
          />
        </div>

        <!-- Shared: Region -->
        <div>
          <label class="input-label">{{ t('admin.providers.bedrockRegion') }}</label>
          <Select v-model="bedrockRegion" :options="bedrockRegionOptions" searchable />
          <p class="input-hint">{{ t('admin.providers.bedrockRegionHint') }}</p>
        </div>

        <!-- Shared: Force Global -->
        <div>
          <label class="flex items-center gap-2 cursor-pointer">
            <input
              v-model="bedrockForceGlobal"
              type="checkbox"
              class="rounded-compact border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-500"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.providers.bedrockForceGlobal') }}</span>
          </label>
          <p class="input-hint mt-1">{{ t('admin.providers.bedrockForceGlobalHint') }}</p>
        </div>

        <!-- Model Restriction Section for Bedrock -->
        <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
          <label class="input-label">{{ t('admin.providers.modelRestriction') }}</label>

          <!-- Mode Toggle -->
          <div class="mb-4 flex gap-2">
            <button
              type="button"
              @click="modelRestrictionMode = 'whitelist'"
              :class="[
                'flex-1 rounded-control px-4 py-2 text-sm font-medium transition-all',
                modelRestrictionMode === 'whitelist'
                  ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
              ]"
            >
              {{ t('admin.providers.modelWhitelist') }}
            </button>
            <button
              type="button"
              @click="modelRestrictionMode = 'mapping'"
              :class="[
                'flex-1 rounded-control px-4 py-2 text-sm font-medium transition-all',
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
          <div v-if="modelRestrictionMode === 'whitelist'">
            <ModelWhitelistSelector :model-value="allowedModels" platform="anthropic" :sync-credentials="syncPreviewCredentials" @update:model-value="setAllowedModels" />
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.selectedModels', { count: allowedModels.length }) }}
              <span v-if="allowedModels.length === 0">{{ t('admin.providers.supportsAllModels') }}</span>
            </p>
          </div>

          <!-- Mapping Mode -->
          <div v-else class="space-y-3">
            <div v-for="(mapping, index) in modelMappings" :key="index" class="flex items-center gap-2">
              <input v-model="mapping.from" type="text" class="input flex-1" :placeholder="t('admin.providers.fromModel')" />
              <span class="text-gray-400">→</span>
              <input v-model="mapping.to" type="text" class="input flex-1" :placeholder="t('admin.providers.toModel')" />
              <button type="button" @click="modelMappings.splice(index, 1)" class="text-red-500 hover:text-red-700">
                <Icon name="trash" size="sm" />
              </button>
            </div>
            <button type="button" @click="modelMappings.push({ from: '', to: '' })" class="btn btn-secondary text-sm">
              + {{ t('admin.providers.addMapping') }}
            </button>
            <!-- Bedrock Preset Mappings -->
            <div class="flex flex-wrap gap-2">
              <button
                v-for="preset in bedrockPresets"
                :key="preset.from"
                type="button"
                @click="addPresetMapping(preset.from, preset.to)"
                :class="['rounded-control px-3 py-1 text-xs transition-colors', preset.color]"
              >
                + {{ preset.label }}
              </button>
            </div>
          </div>
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
          <div v-if="poolModeEnabled" class="rounded-control bg-blue-50 p-3 dark:bg-blue-900/20">
            <p class="text-xs text-blue-700 dark:text-blue-400">
              <Icon name="exclamationCircle" size="sm" class="mr-1 inline" :stroke-width="2" />
              {{ t('admin.providers.poolModeInfo') }}
            </p>
          </div>
          <div v-if="poolModeEnabled" class="mt-3">
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
          <div v-if="poolModeEnabled" class="mt-3">
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
        </div>
      </div>

      <UpstreamUsageConfigEditor
        v-if="form.type === 'apikey'"
        :enabled="upstreamUsageEnabled"
        :adapter="upstreamUsageAdapter"
        :base-url="upstreamUsageBaseUrl"
        :wallet-access-token="upstreamUsageWalletAccessToken"
        :wallet-user-id="upstreamUsageWalletUserId"
        :automatic-adapter="isCNPlatform"
        @update:enabled="upstreamUsageEnabled = $event"
        @update:adapter="upstreamUsageAdapter = $event"
        @update:base-url="upstreamUsageBaseUrl = $event"
        @update:wallet-access-token="upstreamUsageWalletAccessToken = $event"
        @update:wallet-user-id="upstreamUsageWalletUserId = $event"
      />

      <!-- 配额控制 (Anthropic apikey/bedrock: 配额限制 + 亲和) -->
      <div
        v-if="form.platform === 'anthropic' && (form.type === 'apikey' || form.type === 'bedrock')"
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
          :dailyResetMode="editDailyResetMode"
          :dailyResetHour="editDailyResetHour"
          :weeklyResetMode="editWeeklyResetMode"
          :weeklyResetDay="editWeeklyResetDay"
          :weeklyResetHour="editWeeklyResetHour"
          :resetTimezone="editResetTimezone"
          @update:totalLimit="editQuotaLimit = $event"
          @update:dailyLimit="editQuotaDailyLimit = $event"
          @update:weeklyLimit="editQuotaWeeklyLimit = $event"
          @update:quotaNotifyDailyEnabled="quotaNotifyState.daily.enabled = $event"
          @update:quotaNotifyDailyThreshold="quotaNotifyState.daily.threshold = $event"
          @update:quotaNotifyDailyThresholdType="quotaNotifyState.daily.thresholdType = $event"
          @update:quotaNotifyWeeklyEnabled="quotaNotifyState.weekly.enabled = $event"
          @update:quotaNotifyWeeklyThreshold="quotaNotifyState.weekly.threshold = $event"
          @update:quotaNotifyWeeklyThresholdType="quotaNotifyState.weekly.thresholdType = $event"
          @update:quotaNotifyTotalEnabled="quotaNotifyState.total.enabled = $event"
          @update:quotaNotifyTotalThreshold="quotaNotifyState.total.threshold = $event"
          @update:quotaNotifyTotalThresholdType="quotaNotifyState.total.thresholdType = $event"
          @update:dailyResetMode="editDailyResetMode = $event"
          @update:dailyResetHour="editDailyResetHour = $event"
          @update:weeklyResetMode="editWeeklyResetMode = $event"
          @update:weeklyResetDay="editWeeklyResetDay = $event"
          @update:weeklyResetHour="editWeeklyResetHour = $event"
          @update:resetTimezone="editResetTimezone = $event"
        />
      </div>

      <!-- 配额控制 (非 Anthropic apikey/bedrock) -->
      <div
        v-else-if="form.type === 'apikey' || form.type === 'bedrock'"
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
          :dailyResetMode="editDailyResetMode"
          :dailyResetHour="editDailyResetHour"
          :weeklyResetMode="editWeeklyResetMode"
          :weeklyResetDay="editWeeklyResetDay"
          :weeklyResetHour="editWeeklyResetHour"
          :resetTimezone="editResetTimezone"
          @update:totalLimit="editQuotaLimit = $event"
          @update:dailyLimit="editQuotaDailyLimit = $event"
          @update:weeklyLimit="editQuotaWeeklyLimit = $event"
          @update:quotaNotifyDailyEnabled="quotaNotifyState.daily.enabled = $event"
          @update:quotaNotifyDailyThreshold="quotaNotifyState.daily.threshold = $event"
          @update:quotaNotifyDailyThresholdType="quotaNotifyState.daily.thresholdType = $event"
          @update:quotaNotifyWeeklyEnabled="quotaNotifyState.weekly.enabled = $event"
          @update:quotaNotifyWeeklyThreshold="quotaNotifyState.weekly.threshold = $event"
          @update:quotaNotifyWeeklyThresholdType="quotaNotifyState.weekly.thresholdType = $event"
          @update:quotaNotifyTotalEnabled="quotaNotifyState.total.enabled = $event"
          @update:quotaNotifyTotalThreshold="quotaNotifyState.total.threshold = $event"
          @update:quotaNotifyTotalThresholdType="quotaNotifyState.total.thresholdType = $event"
          @update:dailyResetMode="editDailyResetMode = $event"
          @update:dailyResetHour="editDailyResetHour = $event"
          @update:weeklyResetMode="editWeeklyResetMode = $event"
          @update:weeklyResetDay="editWeeklyResetDay = $event"
          @update:weeklyResetHour="editWeeklyResetHour = $event"
          @update:resetTimezone="editResetTimezone = $event"
        />
      </div>

      <!-- Grok OAuth 自定义上游地址（仅改写转发端点，OAuth 授权与刷新不受影响） -->
      <div
        v-if="form.platform === 'grok' && isOAuthFlow"
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
        <div v-if="grokOAuthCustomBaseUrlEnabled" class="space-y-2">
          <input
            v-model="grokOAuthBaseUrl"
            type="text"
            class="input"
            data-testid="grok-custom-base-url-input"
            :placeholder="t('admin.providers.grokCustomBaseUrl.placeholder')"
          />
          <GrokBaseUrlPresets @select="grokOAuthBaseUrl = $event" />
        </div>
      </div>

      <!-- Grok OAuth 请求头覆写（OAuth 类型没有 API Key 容器，需要独立区域） -->
      <div
        v-if="form.platform === 'grok' && isOAuthFlow"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="mb-3 flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{ t('admin.providers.headerOverride.title') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.headerOverride.hint') }}
            </p>
          </div>
          <Toggle v-model="headerOverrideEnabled" variant="flush" off-tone="soft" />
        </div>

        <div v-if="headerOverrideEnabled" class="space-y-3">
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
      </div>

      <!-- OpenAI OAuth Model Mapping (OAuth 类型没有 apikey 容器，需要独立的模型映射区域) -->
      <div
        v-if="['openai', 'grok', 'gemini'].includes(form.platform) && isOAuthFlow"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <label class="input-label">{{ t('admin.providers.modelRestriction') }}</label>

          <!-- Mode Toggle -->
          <div class="mb-4 flex gap-2">
            <button
              type="button"
              @click="modelRestrictionMode = 'whitelist'"
              :class="[
                'flex-1 rounded-control px-4 py-2 text-sm font-medium transition-all',
                modelRestrictionMode === 'whitelist'
                  ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
              ]"
            >
              {{ t('admin.providers.modelWhitelist') }}
            </button>
            <button
              type="button"
              @click="modelRestrictionMode = 'mapping'"
              :class="[
                'flex-1 rounded-control px-4 py-2 text-sm font-medium transition-all',
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
          <div v-if="modelRestrictionMode === 'whitelist'">
            <ModelWhitelistSelector :model-value="allowedModels" :platform="form.platform" :sync-credentials="syncPreviewCredentials" @update:model-value="setAllowedModels" />
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.selectedModels', { count: allowedModels.length }) }}
              <span v-if="allowedModels.length === 0">{{
                t('admin.providers.supportsAllModels')
              }}</span>
            </p>
          </div>

          <!-- Mapping Mode -->
          <div v-else>
            <div class="mb-3 rounded-control bg-purple-50 p-3 dark:bg-purple-900/20">
              <p class="text-xs text-purple-700 dark:text-purple-400">
                {{ t('admin.providers.mapRequestModels') }}
              </p>
            </div>

            <div v-if="modelMappings.length > 0" class="mb-3 space-y-2">
              <div
                v-for="(mapping, index) in modelMappings"
                :key="'oauth-' + getModelMappingKey(mapping)"
                class="flex items-center gap-2"
              >
                <input
                  v-model="mapping.from"
                  type="text"
                  class="input flex-1"
                  :placeholder="t('admin.providers.requestModel')"
                />
                <svg
                  class="h-4 w-4 flex-shrink-0 text-gray-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M14 5l7 7m0 0l-7 7m7-7H3"
                  />
                </svg>
                <input
                  v-model="mapping.to"
                  type="text"
                  class="input flex-1"
                  :placeholder="t('admin.providers.actualModel')"
                />
                <button
                  type="button"
                  @click="removeModelMapping(index)"
                  class="rounded-control p-2 text-red-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
                >
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                    />
                  </svg>
                </button>
              </div>
            </div>

            <button
              type="button"
              @click="addModelMapping"
              class="mb-3 w-full rounded-control border-2 border-dashed border-gray-300 px-4 py-2 text-gray-600 transition-colors hover:border-gray-400 hover:text-gray-700 dark:border-dark-500 dark:text-gray-400 dark:hover:border-dark-400 dark:hover:text-gray-300"
            >
              + {{ t('admin.providers.addMapping') }}
            </button>

            <!-- Quick Add Buttons -->
            <div class="flex flex-wrap gap-2">
              <button
                v-for="preset in presetMappings"
                :key="'oauth-' + preset.label"
                type="button"
                @click="addPresetMapping(preset.from, preset.to)"
                :class="['rounded-control px-3 py-1 text-xs transition-colors', preset.color]"
              >
                + {{ preset.label }}
              </button>
            </div>
          </div>

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

        <div v-if="tempUnschedEnabled" class="space-y-3">
          <div class="rounded-control bg-blue-50 p-3 dark:bg-blue-900/20">
              <p class="text-xs text-blue-700 dark:text-blue-400">
                <Icon name="exclamationTriangle" size="sm" class="mr-1 inline" :stroke-width="2" />
                {{ t('admin.providers.tempUnschedulable.notice') }}
              </p>
            </div>

          <div class="flex flex-wrap gap-2">
            <button
              v-for="preset in tempUnschedPresets"
              :key="preset.label"
              type="button"
              @click="addTempUnschedRule(preset.rule)"
              class="rounded-control bg-gray-100 px-3 py-1.5 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500"
            >
              + {{ preset.label }}
            </button>
          </div>

          <div v-if="tempUnschedRules.length > 0" class="space-y-3">
            <div
              v-for="(rule, index) in tempUnschedRules"
              :key="getTempUnschedRuleKey(rule)"
              class="rounded-control border border-gray-200 p-3 dark:border-dark-600"
            >
              <div class="mb-2 flex items-center justify-between">
                <span class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.providers.tempUnschedulable.ruleIndex', { index: index + 1 }) }}
                </span>
                <div class="flex items-center gap-2">
                  <button
                    type="button"
                    :disabled="index === 0"
                    @click="moveTempUnschedRule(index, -1)"
                    class="rounded-compact p-1 text-gray-400 transition-colors hover:text-gray-600 disabled:cursor-not-allowed disabled:opacity-40 dark:hover:text-gray-200"
                  >
                    <Icon name="chevronUp" size="sm" :stroke-width="2" />
                  </button>
                  <button
                    type="button"
                    :disabled="index === tempUnschedRules.length - 1"
                    @click="moveTempUnschedRule(index, 1)"
                    class="rounded-compact p-1 text-gray-400 transition-colors hover:text-gray-600 disabled:cursor-not-allowed disabled:opacity-40 dark:hover:text-gray-200"
                  >
                    <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                    </svg>
                  </button>
                  <button
                    type="button"
                    @click="removeTempUnschedRule(index)"
                    class="rounded-compact p-1 text-red-500 transition-colors hover:text-red-600"
                  >
                    <Icon name="x" size="sm" :stroke-width="2" />
                  </button>
                </div>
              </div>

              <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div>
                  <label class="input-label">{{ t('admin.providers.tempUnschedulable.errorCode') }}</label>
                  <input
                    v-model.number="rule.error_code"
                    type="number"
                    min="100"
                    max="599"
                    class="input"
                    :placeholder="t('admin.providers.tempUnschedulable.errorCodePlaceholder')"
                  />
                </div>
                <div>
                  <label class="input-label">{{ t('admin.providers.tempUnschedulable.durationMinutes') }}</label>
                  <input
                    v-model.number="rule.duration_minutes"
                    type="number"
                    min="1"
                    class="input"
                    :placeholder="t('admin.providers.tempUnschedulable.durationPlaceholder')"
                  />
                </div>
                <div class="sm:col-span-2">
                  <label class="input-label">{{ t('admin.providers.tempUnschedulable.keywords') }}</label>
                  <input
                    v-model="rule.keywords"
                    type="text"
                    class="input"
                    :placeholder="t('admin.providers.tempUnschedulable.keywordsPlaceholder')"
                  />
                  <p class="input-hint">{{ t('admin.providers.tempUnschedulable.keywordsHint') }}</p>
                </div>
                <div class="sm:col-span-2">
                  <label class="input-label">{{ t('admin.providers.tempUnschedulable.description') }}</label>
                  <input
                    v-model="rule.description"
                    type="text"
                    class="input"
                    :placeholder="t('admin.providers.tempUnschedulable.descriptionPlaceholder')"
                  />
                </div>
              </div>
            </div>
          </div>

          <button
            type="button"
            @click="addTempUnschedRule()"
            class="w-full rounded-control border-2 border-dashed border-gray-300 px-4 py-2 text-sm text-gray-600 transition-colors hover:border-gray-400 hover:text-gray-700 dark:border-dark-500 dark:text-gray-400 dark:hover:border-dark-400 dark:hover:text-gray-300"
          >
            <svg
              class="mr-1 inline h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            {{ t('admin.providers.tempUnschedulable.addRule') }}
          </button>
        </div>
      </div>

      <!-- Intercept Warmup Requests (Anthropic/Antigravity) -->
      <div
        v-if="form.platform === 'anthropic' || form.platform === 'antigravity'"
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

      <!-- 配额控制 (Anthropic OAuth/SetupToken: 亲和 + 窗口费用 + 会话 + RPM 等) -->
      <div
        v-if="form.platform === 'anthropic' && providerCategory === 'oauth-based'"
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

          <div v-if="windowCostEnabled" class="grid grid-cols-2 gap-4">
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

          <div v-if="sessionLimitEnabled" class="grid grid-cols-2 gap-4">
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

          <div v-if="rpmLimitEnabled" class="space-y-4">
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
                    'flex-1 rounded-control px-3 py-2 text-sm font-medium transition-all',
                    rpmStrategy === 'tiered'
                      ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
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
                    'flex-1 rounded-control px-3 py-2 text-sm font-medium transition-all',
                    rpmStrategy === 'sticky_exempt'
                      ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
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

            <div v-if="rpmStrategy === 'tiered'">
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
          <div v-if="cacheTTLOverrideEnabled" class="mt-3">
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
          <div v-if="customBaseUrlEnabled" class="mt-3">
            <input
              v-model="customBaseUrl"
              type="text"
              class="input"
              :placeholder="t('admin.providers.quotaControl.customBaseUrl.urlHint')"
            />
          </div>
        </div>
      </div>

      <div>
        <div class="mb-1 flex items-center gap-2">
          <label class="input-label mb-0">{{ t('admin.providers.proxy') }}</label>
        </div>
        <ProxySelector v-model="form.proxy_id" :proxies="proxies" />
      </div>

      <ProviderProtocolSelector v-model="upstreamProtocols" :platform="form.platform" :type="form.type" :auth-mode="oauthFlowRef?.inputMethod === 'codex_pat' ? 'personalAccessToken' : oauthFlowRef?.inputMethod === 'agent_identity' ? 'agentIdentity' : ''" />

      <UpstreamRequestIdHeaderField
        v-model="upstreamRequestIdHeader"
        :platform="form.platform"
        :type="form.type"
      />

      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <div>
          <label class="input-label">{{ t('admin.providers.concurrency') }}</label>
          <input v-model.number="form.concurrency" type="number" min="1" class="input"
            @input="form.concurrency = Math.max(1, form.concurrency || 1)" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.providers.loadFactor') }}</label>
          <input v-model.number="form.load_factor" type="number" min="1"
            class="input" :placeholder="String(form.concurrency || 1)"
            @input="form.load_factor = (form.load_factor &amp;&amp; form.load_factor >= 1) ? form.load_factor : null" />
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
        v-if="form.platform === 'openai'"
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
        v-if="form.platform === 'openai' && form.type === 'oauth'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between gap-4">
          <div>
            <label class="input-label mb-0">{{ t('admin.providers.openai.flattenNamespaces') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.openai.flattenNamespacesDesc') }}
            </p>
          </div>
          <Toggle v-model="openaiFlattenNamespacesEnabled" variant="flush" off-tone="soft" data-testid="create-openai-flatten-namespaces-toggle" />
        </div>
      </div>

      <!-- OpenAI WS Mode 三态（off/ctx_pool/passthrough） -->
      <div
        v-if="form.platform === 'openai' && (providerCategory === 'oauth-based' || providerCategory === 'apikey')"
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

      <!-- Anthropic API Key 自动透传开关 -->
      <div
        v-if="form.platform === 'anthropic' && providerCategory === 'apikey'"
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
        v-if="form.platform === 'anthropic' && providerCategory === 'apikey'"
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
        v-if="form.platform === 'anthropic' && providerCategory === 'apikey' && webSearchGlobalEnabled"
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

      <!-- OpenAI OAuth 客户端访问策略 -->
      <div
        v-if="form.platform === 'openai' && providerCategory === 'oauth-based'"
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

      <div
        v-if="form.platform === 'openai' && providerCategory === 'oauth-based'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{ t('admin.providers.quotaControl.tlsFingerprint.label') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.quotaControl.tlsFingerprint.hint') }}
            </p>
          </div>
          <Toggle v-model="tlsFingerprintEnabled" variant="flush" off-tone="soft" data-testid="create-openai-tls-fingerprint-toggle" />
        </div>
        <div v-if="tlsFingerprintEnabled" class="mt-3 space-y-3">
          <Select
            v-model="tlsFingerprintProfileId"
            data-testid="create-openai-tls-fingerprint-profile"
            :options="tlsFingerprintProfileOptions"
          />
          <div>
            <Select
              v-model="tlsFingerprintRouterId"
              data-testid="create-openai-tls-fingerprint-router"
              :options="tlsFingerprintRouterOptions"
            />
            <p class="input-hint">{{ t('admin.providers.quotaControl.tlsFingerprint.routerHint') }}</p>
          </div>
        </div>
      </div>

      <!-- Codex 指纹收敛模式（仅 OpenAI OAuth） -->
      <div
        v-if="form.platform === 'openai' && providerCategory === 'oauth-based'"
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
              data-testid="create-codex-fingerprint-mode-select"
              :options="codexFingerprintModeOptions"
            />
          </div>
        </div>
      </div>

      <!-- OpenAI 旧版 Compact 端点能力配置 -->
      <div
        v-if="form.platform === 'openai' && (providerCategory === 'oauth-based' || providerCategory === 'apikey')"
        class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4"
      >
        <OpenAICompactionCheckbox v-model="openAINativeCompactionV2Mode" test-id="create-openai-native-compaction-v2-mode"
          :label="t('admin.providers.openai.nativeCompactV2Mode')" :hint="t('admin.providers.openai.nativeCompactV2ModeDesc')" />
        <OpenAICompactionCheckbox v-model="openAICompactMode" test-id="create-openai-compact-mode"
          :label="t('admin.providers.openai.compactMode')" :hint="t('admin.providers.openai.compactModeDesc')" />
        <div v-if="openAICompactMode !== 'force_off'">
          <label class="input-label">{{ t('admin.providers.openai.compactModelMapping') }}</label>
          <p class="input-hint">{{ t('admin.providers.openai.compactModelMappingDesc') }}</p>
          <div v-if="openAICompactModelMappings.length > 0" class="mb-3 space-y-2">
            <div
              v-for="(mapping, index) in openAICompactModelMappings"
              :key="getOpenAICompactModelMappingKey(mapping)"
              class="flex items-center gap-2"
            >
              <input v-model="mapping.from" type="text" class="input flex-1" :placeholder="t('admin.providers.fromModel')" />
              <span class="text-gray-400">→</span>
              <input v-model="mapping.to" type="text" class="input flex-1" :placeholder="t('admin.providers.toModel')" />
              <button type="button" @click="removeOpenAICompactModelMapping(index)" class="text-red-500 hover:text-red-700">
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </div>
          <button type="button" @click="addOpenAICompactModelMapping" class="btn btn-secondary text-sm">
            + {{ t('admin.providers.addMapping') }}
          </button>
        </div>
      </div>

      <!-- OpenAI API Key 文本工作负载与管理员协议路由 -->
      <div
        v-if="form.platform === 'openai' && providerCategory === 'apikey'"
        class="space-y-5 border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex flex-col gap-3 border-t border-gray-200 pt-4 dark:border-dark-600 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <label class="input-label mb-0" for="create-openai-continuation-supported">
              {{ t('admin.providers.openai.responsesContinuationSupported') }}
            </label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.openai.responsesContinuationSupportedDesc') }}
            </p>
          </div>
          <Toggle
            id="create-openai-continuation-supported"
            v-model="openAIResponsesContinuationSupported"
            data-testid="create-openai-continuation-supported"
            :aria-label="t('admin.providers.openai.responsesContinuationSupported')"
          />
        </div>
        <div class="flex flex-col gap-3 border-t border-gray-200 pt-4 dark:border-dark-600 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <label class="input-label mb-0" for="create-openai-images-url-to-b64-json">
              {{ t('admin.providers.openai.imagesURLToB64JSON') }}
            </label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.providers.openai.imagesURLToB64JSONDesc') }}
            </p>
          </div>
          <Toggle
            id="create-openai-images-url-to-b64-json"
            v-model="openAIImagesURLToB64JSON"
            data-testid="create-openai-images-url-to-b64-json"
            :aria-label="t('admin.providers.openai.imagesURLToB64JSON')"
          />
        </div>

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
        v-if="form.platform === 'openai' && providerCategory === 'oauth-based'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4"
      >
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('admin.providers.autoPause5hDisabled') }}</label>
            <Toggle v-model="autoPause5hDisabled" variant="flush" off-tone="soft" data-testid="create-auto-pause-5h-disabled" />
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
            data-testid="create-auto-pause-5h-threshold"
          />
          <p class="input-hint">{{ t('admin.providers.autoPauseThresholdHint') }}</p>
        </div>
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('admin.providers.autoPause7dDisabled') }}</label>
            <Toggle v-model="autoPause7dDisabled" variant="flush" off-tone="soft" data-testid="create-auto-pause-7d-disabled" />
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
            data-testid="create-auto-pause-7d-threshold"
          />
          <p class="input-hint">{{ t('admin.providers.autoPauseThresholdHint') }}</p>
        </div>
      </div>

      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div v-if="form.platform === 'antigravity'" class="mt-3 flex items-center gap-2">
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
              class="pointer-events-none absolute left-0 top-full z-tooltip mt-1.5 w-72 rounded-compact bg-gray-900 px-3 py-2 text-xs text-white opacity-0 transition-opacity group-hover:opacity-100 dark:bg-gray-700"
            >
              {{ t('admin.providers.allowOveragesTooltip') }}
              <div
                class="absolute bottom-full left-3 border-4 border-transparent border-b-gray-900 dark:border-b-gray-700"
              ></div>
            </div>
          </div>
        </div>

        <!-- 分组选择 -->
        <GroupSelector
          v-model="form.group_ids"
          :groups="groups"
          data-tour="provider-form-groups"
        />
      </div>

    </form>

    <!-- Step 2: OAuth Authorization -->
    <div v-else class="space-y-5">
      <OAuthAuthorizationFlow
        ref="oauthFlowRef"
        :resolved-email="form.platform === 'openai' ? openaiOAuth.detectedEmail?.value || '' : ''"
        :add-method="form.platform === 'anthropic' ? addMethod : 'oauth'"
        :auth-url="currentAuthUrl"
        :session-id="currentSessionId"
        :loading="currentOAuthLoading"
        :error="currentOAuthError"
        :show-help="form.platform === 'anthropic'"
        :show-proxy-warning="form.platform !== 'openai' && form.platform !== 'grok' && !!form.proxy_id"
        :allow-multiple="form.platform === 'anthropic'"
        :show-cookie-option="form.platform === 'anthropic'"
        :show-refresh-token-option="form.platform === 'openai' || form.platform === 'antigravity' || form.platform === 'grok'"
        :show-mobile-refresh-token-option="form.platform === 'openai'"
        :show-session-token-option="false"
        :show-access-token-option="false"
        :show-codex-session-import-option="form.platform === 'openai'"
        :show-agent-identity-option="form.platform === 'openai'"
        :show-codex-pat-option="form.platform === 'openai'"
        :show-sso-option="form.platform === 'grok'"
        :show-email-password-option="false"
        :show-manual-option="true"
        :initial-input-method="'manual'"
        :platform="form.platform"
        :show-project-id="geminiOAuthType === 'code_assist'"
        :auth-sessions="currentOpenAIAuthSessions"
        @generate-url="handleGenerateUrl"
        @remove-auth-session="handleRemoveOpenAIAuthSession"
        @cookie-auth="handleCookieAuth"
        @validate-refresh-token="handleValidateRefreshToken"
        @validate-mobile-refresh-token="handleOpenAIValidateMobileRT"
        @validate-session-token="handleValidateSessionToken"
        @import-codex-session="handleOpenAIImportCodexSession"
        @import-codex-pat="handleOpenAIImportCodexPAT"
        @import-sso="handleGrokImportSSO"
      />

    </div>

    <!-- 仅第一步编辑票据；授权时保留同一草稿，返回和最终创建不能重置为模板。 -->
    <CodexTicketAccountSettings v-if="show && isOpenAIOAuthImportDefaultsTarget" v-show="step === 1" ref="ticketDraft" draft class="mt-5" />

    <template #footer>
      <div v-if="step === 1" class="flex justify-end gap-3">
        <button @click="handleClose" type="button" class="btn btn-secondary">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="create-provider-form"
          :disabled="submitting"
          class="btn btn-primary"
          data-tour="provider-form-submit"
        >
          <svg
            v-if="submitting"
            class="-ml-1 mr-2 h-4 w-4 animate-spin"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              class="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              stroke-width="4"
            ></circle>
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path>
          </svg>
          {{
            isOAuthFlow
              ? t('common.next')
              : submitting
                ? t('admin.providers.creating')
                : t('common.create')
          }}
        </button>
      </div>
      <div v-else class="flex justify-between gap-3">
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="submitting || isQoderOAuthProviderCreating"
          @click="goBackToBasicInfo"
        >
          {{ t('common.back') }}
        </button>
        <button
          v-if="isManualInputMethod"
          type="button"
          :disabled="!canExchangeCode"
          class="btn btn-primary"
          @click="handleExchangeCode"
        >
          <svg
            v-if="currentOAuthLoading"
            class="-ml-1 mr-2 h-4 w-4 animate-spin"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              class="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              stroke-width="4"
            ></circle>
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path>
          </svg>
          {{
            currentOAuthLoading
              ? t('admin.providers.oauth.verifying')
              : t('admin.providers.oauth.completeAuth')
          }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <!-- Gemini Help Dialog -->
  <BaseDialog
    :show="showGeminiHelpDialog"
    :title="t('admin.providers.gemini.helpDialog.title')"
    width="wide"
    @close="showGeminiHelpDialog = false"
  >
    <div class="space-y-6">
      <!-- Setup Guide Section -->
      <div>
        <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.providers.gemini.setupGuide.title') }}
        </h3>
        <div class="space-y-4">
          <div>
            <p class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.providers.gemini.setupGuide.checklistTitle') }}
            </p>
            <ul class="list-inside list-disc space-y-1 text-sm text-gray-600 dark:text-gray-400">
              <li>{{ t('admin.providers.gemini.setupGuide.checklistItems.usIp') }}</li>
              <li>{{ t('admin.providers.gemini.setupGuide.checklistItems.age') }}</li>
            </ul>
          </div>
          <div>
            <p class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.providers.gemini.setupGuide.activationTitle') }}
            </p>
            <ul class="list-inside list-disc space-y-1 text-sm text-gray-600 dark:text-gray-400">
              <li>{{ t('admin.providers.gemini.setupGuide.activationItems.geminiWeb') }}</li>
              <li>{{ t('admin.providers.gemini.setupGuide.activationItems.gcpProject') }}</li>
            </ul>
            <div class="mt-2 flex flex-wrap gap-2">
              <a
                href="https://policies.google.com/terms"
                target="_blank"
                rel="noreferrer"
                class="text-sm text-blue-600 hover:underline dark:text-blue-400"
              >
                {{ t('admin.providers.gemini.setupGuide.links.countryCheck') }}
              </a>
              <span class="text-gray-400">·</span>
              <a
                href="https://policies.google.com/country-association-form"
                target="_blank"
                rel="noreferrer"
                class="text-sm text-blue-600 hover:underline dark:text-blue-400"
              >
                修改归属地
              </a>
              <span class="text-gray-400">·</span>
              <a
                href="https://gemini.google.com/gems/create?hl=en-US&pli=1"
                target="_blank"
                rel="noreferrer"
                class="text-sm text-blue-600 hover:underline dark:text-blue-400"
              >
                {{ t('admin.providers.gemini.setupGuide.links.geminiWebActivation') }}
              </a>
              <span class="text-gray-400">·</span>
              <a
                href="https://console.cloud.google.com"
                target="_blank"
                rel="noreferrer"
                class="text-sm text-blue-600 hover:underline dark:text-blue-400"
              >
                {{ t('admin.providers.gemini.setupGuide.links.gcpProject') }}
              </a>
            </div>
          </div>
        </div>
      </div>

      <!-- Quota Policy Section -->
      <div class="border-t border-gray-200 pt-6 dark:border-dark-600">
        <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.providers.gemini.quotaPolicy.title') }}
        </h3>
        <p class="mb-4 text-xs text-amber-600 dark:text-amber-400">
          {{ t('admin.providers.gemini.quotaPolicy.note') }}
        </p>
        <div class="overflow-x-auto">
          <table class="w-full text-xs">
            <thead class="bg-gray-50 dark:bg-dark-600">
              <tr>
                <th class="px-3 py-2 text-left font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.providers.gemini.quotaPolicy.columns.channel') }}
                </th>
                <th class="px-3 py-2 text-left font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.providers.gemini.quotaPolicy.columns.provider') }}
                </th>
                <th class="px-3 py-2 text-left font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.providers.gemini.quotaPolicy.columns.limits') }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-dark-600">
              <tr>
                <td class="px-3 py-2 text-gray-900 dark:text-white">
                  {{ t('admin.providers.gemini.quotaPolicy.rows.googleOne.channel') }}
                </td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">Free</td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                  {{ t('admin.providers.gemini.quotaPolicy.rows.googleOne.limitsFree') }}
                </td>
              </tr>
              <tr>
                <td class="px-3 py-2 text-gray-900 dark:text-white"></td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">Pro</td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                  {{ t('admin.providers.gemini.quotaPolicy.rows.googleOne.limitsPro') }}
                </td>
              </tr>
              <tr>
                <td class="px-3 py-2 text-gray-900 dark:text-white"></td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">Ultra</td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                  {{ t('admin.providers.gemini.quotaPolicy.rows.googleOne.limitsUltra') }}
                </td>
              </tr>
              <tr>
                <td class="px-3 py-2 text-gray-900 dark:text-white">
                  {{ t('admin.providers.gemini.quotaPolicy.rows.gcp.channel') }}
                </td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">Standard</td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                  {{ t('admin.providers.gemini.quotaPolicy.rows.gcp.limitsStandard') }}
                </td>
              </tr>
              <tr>
                <td class="px-3 py-2 text-gray-900 dark:text-white"></td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">Enterprise</td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                  {{ t('admin.providers.gemini.quotaPolicy.rows.gcp.limitsEnterprise') }}
                </td>
              </tr>
              <tr>
                <td class="px-3 py-2 text-gray-900 dark:text-white">
                  {{ t('admin.providers.gemini.quotaPolicy.rows.aiStudio.channel') }}
                </td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">Free</td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                  {{ t('admin.providers.gemini.quotaPolicy.rows.aiStudio.limitsFree') }}
                </td>
              </tr>
              <tr>
                <td class="px-3 py-2 text-gray-900 dark:text-white"></td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">Paid</td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                  {{ t('admin.providers.gemini.quotaPolicy.rows.aiStudio.limitsPaid') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="mt-4 flex flex-wrap gap-3">
          <a
            :href="geminiQuotaDocs.codeAssist"
            target="_blank"
            rel="noreferrer"
            class="text-sm text-blue-600 hover:underline dark:text-blue-400"
          >
            {{ t('admin.providers.gemini.quotaPolicy.docs.codeAssist') }}
          </a>
          <a
            :href="geminiQuotaDocs.aiStudio"
            target="_blank"
            rel="noreferrer"
            class="text-sm text-blue-600 hover:underline dark:text-blue-400"
          >
            {{ t('admin.providers.gemini.quotaPolicy.docs.aiStudio') }}
          </a>
          <a
            :href="geminiQuotaDocs.vertex"
            target="_blank"
            rel="noreferrer"
            class="text-sm text-blue-600 hover:underline dark:text-blue-400"
          >
            {{ t('admin.providers.gemini.quotaPolicy.docs.vertex') }}
          </a>
        </div>
      </div>

      <!-- API Key Links Section -->
      <div class="border-t border-gray-200 pt-6 dark:border-dark-600">
        <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.providers.gemini.helpDialog.apiKeySection') }}
        </h3>
        <div class="flex flex-wrap gap-3">
          <a
            :href="geminiHelpLinks.apiKey"
            target="_blank"
            rel="noreferrer"
            class="text-sm text-blue-600 hover:underline dark:text-blue-400"
          >
            {{ t('admin.providers.gemini.providerType.apiKeyLink') }}
          </a>
          <a
            :href="geminiHelpLinks.aiStudioPricing"
            target="_blank"
            rel="noreferrer"
            class="text-sm text-blue-600 hover:underline dark:text-blue-400"
          >
            {{ t('admin.providers.gemini.providerType.quotaLink') }}
          </a>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button @click="showGeminiHelpDialog = false" type="button" class="btn btn-primary">
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <!-- Mixed Channel Warning Dialog -->

</template>

<script setup lang="ts">
import OpenCodeGoProtocolRulesEditor from "./OpenCodeGoProtocolRulesEditor.vue"
// 统一协议选择只保存原生集合，不在提供商侧配置转换。
const upstreamProtocols = ref<ProtocolID[] | undefined>(undefined)

import { normalizeLegacyOpenAIExtra, normalizeOpenAICompactMode } from '@/utils/openaiLegacyConfiguration'
import ProviderProtocolSelector from './ProviderProtocolSelector.vue'
import { loadProtocolCatalog, nativeProtocolOptions } from '@/api/admin/protocolCapabilities'
import type { ProtocolID } from '@/types'
import OpenAICompactionCheckbox from './OpenAICompactionCheckbox.vue'
import { ref, reactive, computed, watch, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import {
  claudeModels,
  getPresetMappingsByPlatform,
  getModelsByPlatform,
  commonErrorCodes,
  buildModelMappingObject,
  buildPersistedModelRestriction,
  splitPersistedModelRestriction,
  fetchAntigravityDefaultMappings,
  isValidWildcardPattern
} from '@/composables/useModelWhitelist'
import { adminAPI } from '@/api/admin'
import { useQuotaNotifyState } from '@/composables/useQuotaNotifyState'
import {
  useProviderOAuth,
  type AddMethod,
  type AuthInputMethod
} from '@/composables/useProviderOAuth'
import {
  useOpenAIOAuth,
  type OpenAIOAuthSession,
  type OpenAITokenInfo
} from '@/composables/useOpenAIOAuth'
import { useGeminiOAuth } from '@/composables/useGeminiOAuth'
import { useAntigravityOAuth } from '@/composables/useAntigravityOAuth'
import { useQoderOAuth } from '@/composables/useQoderOAuth'
import type { QoderSite, QoderTokenInfo } from '@/api/admin/qoder'
import { useGrokOAuth } from '@/composables/useGrokOAuth'
import type {
  Proxy,
  AdminGroup,
  ProviderPlatform,
  ProviderType,
  CreateProviderRequest,
  CodexSessionImportMessage,
  OpenAICompactMode,
  OpenAIOAuthClientPolicy,
  UpstreamUsageAdapter
} from '@/types'
import type { OpenAIOAuthImportDefaults } from '@/api/admin/settings'
import CodexTicketAccountSettings from '@/components/admin/provider/CodexTicketAccountSettings.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Toggle from '@/components/common/Toggle.vue'
import UpstreamRequestIdHeaderField from '@/components/provider/UpstreamRequestIdHeaderField.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import ProxySelector from '@/components/common/ProxySelector.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import ModelWhitelistSelector from '@/components/provider/ModelWhitelistSelector.vue'
import QuotaLimitCard from '@/components/provider/QuotaLimitCard.vue'
import GrokBaseUrlPresets from '@/components/provider/GrokBaseUrlPresets.vue'
import CnBaseUrlPresets from '@/components/provider/CnBaseUrlPresets.vue'
import HeaderOverrideEditor from '@/components/provider/HeaderOverrideEditor.vue'
import UpstreamUsageConfigEditor from '@/components/provider/UpstreamUsageConfigEditor.vue'
import {
  applyAntigravityProjectID,
  applyOpenCodeGoProtocolRules,
  cloneOpenCodeGoProtocolRules,
  defaultOpenCodeProtocolRules,
  validOpenCodeGoProtocolRules,
  applyHeaderOverride,
  applyInterceptWarmup,
  cnSupportsNativeResponses,
  defaultCNAdaptiveBaseUrls,
  defaultCNBaseUrl,
  isHeaderOverrideCapable,
  validateHeaderOverrideRows,
  type CnProviderMode,
  type CnApiProtocol,
  type CnNativeApiProtocol,
  type HeaderOverrideRow
} from '@/components/provider/credentialsBuilder'
import { formatDateTimeLocalInput, parseDateTimeLocalInput } from '@/utils/format'
import { createStableObjectKeyResolver } from '@/utils/stableObjectKey'
import {
  BEDROCK_REGION_OPTIONS,
  VERTEX_LOCATION_OPTIONS,
  groupedProviderSelectOptions
} from '@/constants/provider'
import {
  OPENAI_WS_MODE_CTX_POOL,
  OPENAI_WS_MODE_HTTP_BRIDGE,
  OPENAI_WS_MODE_OFF,
  OPENAI_WS_MODE_PASSTHROUGH,
  isOpenAIWSModeEnabled,
  resolveOpenAIWSModeFromExtra,
  resolveOpenAIWSModeConcurrencyHintKey,
  type OpenAIWSMode
} from '@/utils/openaiWsMode'
import OAuthAuthorizationFlow from './OAuthAuthorizationFlow.vue'

// Type for exposed OAuthAuthorizationFlow component
// Note: defineExpose automatically unwraps refs, so we use the unwrapped types
interface OAuthFlowExposed {
  authCode: string
  oauthState: string
  projectId: string
  sessionKey: string
  refreshToken: string
  sessionToken: string
  codexSession: string
  codexPAT: string
  ssoCookie: string
  inputMethod: AuthInputMethod
  reset: () => void
}

const { t } = useI18n()

const oauthStepTitle = computed(() => {
  if (form.platform === 'openai') return t('admin.providers.oauth.openai.title')
  if (form.platform === 'gemini') return t('admin.providers.oauth.gemini.title')
  if (form.platform === 'antigravity') return t('admin.providers.oauth.antigravity.title')
  if (form.platform === 'qoder') return t('admin.providers.oauth.qoder.title')
  if (form.platform === 'grok') return t('admin.providers.oauth.grok.title')
  return t('admin.providers.oauth.title')
})

// Platform-specific hints for API Key type
// 上游ID：直接上游声明请求标识的响应头名，留空不记录。
const upstreamRequestIdHeader = ref('')
const withUpstreamRequestIdHeader = <T extends Record<string, unknown> | undefined>(extra: T): T | Record<string, unknown> => {
  const name = upstreamRequestIdHeader.value.trim()
  if (!name) return extra
  return { ...(extra || {}), upstream_request_id_header: name }
}

const baseUrlHint = computed(() => {
  if (form.platform === 'openai') return t('admin.providers.openai.baseUrlHint')
  if (form.platform === 'gemini' && geminiProviderType.value === 'third_party') {
    return t('admin.providers.gemini.connectionSource.thirdPartyBaseUrlHint')
  }
  if (form.platform === 'gemini') return t('admin.providers.gemini.baseUrlHint')
  if (form.platform === 'grok') return ''
  return t('admin.providers.baseUrlHint')
})

const apiKeyHint = computed(() => {
  if (form.platform === 'openai') return t('admin.providers.openai.apiKeyHint')
  if (form.platform === 'gemini' && geminiProviderType.value === 'third_party') {
    return t('admin.providers.gemini.connectionSource.thirdPartyApiKeyHint')
  }
  if (form.platform === 'gemini') return t('admin.providers.gemini.apiKeyHint')
  if (form.platform === 'grok') return ''
  return t('admin.providers.apiKeyHint')
})

const geminiGoogleOneTierOptions = computed(() => [
  { value: 'google_one_free', label: t('admin.providers.gemini.tier.googleOne.free') },
  { value: 'google_ai_pro', label: t('admin.providers.gemini.tier.googleOne.pro') },
  { value: 'google_ai_ultra', label: t('admin.providers.gemini.tier.googleOne.ultra') }
])

const geminiGCPTierOptions = computed(() => [
  { value: 'gcp_standard', label: t('admin.providers.gemini.tier.gcp.standard') },
  { value: 'gcp_enterprise', label: t('admin.providers.gemini.tier.gcp.enterprise') }
])

const geminiAIStudioTierOptions = computed(() => [
  { value: 'aistudio_free', label: t('admin.providers.gemini.tier.aiStudio.free') },
  { value: 'aistudio_paid', label: t('admin.providers.gemini.tier.aiStudio.paid') }
])

const geminiProviderTypeOptions = computed(() => [
  { value: 'official', label: t('admin.providers.gemini.connectionSource.official') },
  { value: 'third_party', label: t('admin.providers.gemini.connectionSource.thirdParty') }
])

const geminiProviderTypeHint = computed(() =>
  geminiProviderType.value === 'third_party'
    ? t('admin.providers.gemini.connectionSource.thirdPartyHint')
    : t('admin.providers.gemini.connectionSource.officialHint')
)

const isGeminiThirdPartyBaseUrl = (value: string) => {
  const normalized = value.trim()
  if (!normalized) return false
  try {
    return new URL(normalized).hostname.toLowerCase() !== 'generativelanguage.googleapis.com'
  } catch {
    return false
  }
}

const vertexLocationOptions = groupedProviderSelectOptions(VERTEX_LOCATION_OPTIONS)
const bedrockRegionOptions = groupedProviderSelectOptions(BEDROCK_REGION_OPTIONS)

interface Props {
  show: boolean
  initialPlatform?: ProviderPlatform
  proxies: Proxy[]
  groups: AdminGroup[]
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
  created: []
}>()

const appStore = useAppStore()

// OAuth 组合式状态
const oauth = useProviderOAuth() // Anthropic OAuth
const openaiOAuth = useOpenAIOAuth() // OpenAI OAuth
const geminiOAuth = useGeminiOAuth() // Gemini OAuth
const antigravityOAuth = useAntigravityOAuth() // Antigravity OAuth
const qoderOAuth = useQoderOAuth() // Qoder 设备授权
const grokOAuth = useGrokOAuth() // Grok OAuth
let qoderPollTimer: number | null = null
interface QoderAuthPopupLease {
  generation: number
  popup: Window | null
}

let qoderAuthPopupLease: QoderAuthPopupLease | null = null
let qoderAuthPopupGeneration = 0
let qoderPollInFlight = false
let qoderPollGeneration = 0
const qoderFlowGeneration = ref(0)
const qoderProviderCreateGeneration = ref<number | null>(null)
let qoderOAuthCompleted = false

const isQoderOAuthProviderCreating = computed(
  () => qoderProviderCreateGeneration.value !== null
)

// 当前 OAuth 状态用于模板绑定。
const currentAuthUrl = computed(() => {
  if (form.platform === 'openai') return openaiOAuth.authUrl.value
  if (form.platform === 'gemini') return geminiOAuth.authUrl.value
  if (form.platform === 'antigravity') return antigravityOAuth.authUrl.value
  if (form.platform === 'qoder') return qoderOAuth.authUrl.value
  if (form.platform === 'grok') return grokOAuth.authUrl.value
  return oauth.authUrl.value
})

const currentSessionId = computed(() => {
  if (form.platform === 'openai') return openaiOAuth.sessionId.value
  if (form.platform === 'gemini') return geminiOAuth.sessionId.value
  if (form.platform === 'antigravity') return antigravityOAuth.sessionId.value
  if (form.platform === 'qoder') return qoderOAuth.sessionId.value
  if (form.platform === 'grok') return grokOAuth.sessionId.value
  return oauth.sessionId.value
})

const currentOAuthLoading = computed(() => {
  if (form.platform === 'openai') return openaiOAuth.loading.value
  if (form.platform === 'gemini') return geminiOAuth.loading.value
  if (form.platform === 'antigravity') return antigravityOAuth.loading.value
  if (form.platform === 'qoder') {
    return qoderOAuth.loading.value || submitting.value || isQoderOAuthProviderCreating.value
  }
  if (form.platform === 'grok') return grokOAuth.loading.value
  return oauth.loading.value
})

const currentOAuthError = computed(() => {
  if (form.platform === 'openai') return openaiOAuth.error.value
  if (form.platform === 'gemini') return geminiOAuth.error.value
  if (form.platform === 'antigravity') return antigravityOAuth.error.value
  if (form.platform === 'qoder') return qoderOAuth.error.value
  if (form.platform === 'grok') return grokOAuth.error.value
  return oauth.error.value
})

const currentOpenAIAuthSessions = computed(() =>
  form.platform === 'openai' ? openaiOAuth.authSessions.value : []
)

// Refs
const oauthFlowRef = ref<OAuthFlowExposed | null>(null)

// Model mapping type
interface ModelMapping {
  from: string
  to: string
}

interface TempUnschedRuleForm {
  error_code: number | null
  keywords: string
  duration_minutes: number | null
  description: string
}

// State
const step = ref(1)
const submitting = ref(false)
const providerCategory = ref<'oauth-based' | 'apikey' | 'bedrock' | 'service_account'>('oauth-based') // UI selection for provider category
const addMethod = ref<AddMethod>('oauth') // For oauth-based: 'oauth' or 'setup-token'
const apiKeyBaseUrl = ref('https://api.anthropic.com')
const apiKeyValue = ref('')
const upstreamUsageEnabled = ref(true)
const upstreamUsageAdapter = ref<UpstreamUsageAdapter>('sub2api')
const upstreamUsageBaseUrl = ref('')
const upstreamUsageWalletAccessToken = ref('')
const upstreamUsageWalletUserId = ref('')

// 国产供应商提供商的计费模式、协议与默认端点彼此联动。
const providerMode = ref<CnProviderMode>('payg')
const cnModeOptions = computed<CnProviderMode[]>(() => form.platform === 'opencode_go' ? ['zen', 'go'] : form.platform === 'deepseek' ? ['payg'] : ['payg', 'coding'])
const openCodeRules = ref(cloneOpenCodeGoProtocolRules())
// 智谱团队版 Coding Plan 的组织/项目 ID，仅在创建团队提供商时写入凭据。
const zhipuOrganization = ref('')
const zhipuProject = ref('')
// API 协议决定转发端点与格式：cc=现有转换链，anthropic=原生直通（Claude Code），
// responses=deepseek / kimi 原生 Responses 端点（Codex）。与提供商类型正交。
const apiProtocol = ref<CnApiProtocol>('adaptive')
const adaptiveBaseUrls = ref<Record<CnNativeApiProtocol, string>>({
  chat_completions: '',
  anthropic: '',
  responses: ''
})
const isCNPlatform = computed(
  () => form.platform === 'kimi' || form.platform === 'zhipu' || form.platform === 'deepseek' || form.platform === 'minimax' || form.platform === 'opencode_go'
)
// 模板不支持联合类型断言，因此在脚本中收窄预设组件的平台类型。
const cnPresetPlatform = computed<'kimi' | 'zhipu' | 'deepseek' | 'minimax' | 'opencode_go'>(() => {
  if (form.platform === 'kimi' || form.platform === 'zhipu' || form.platform === 'deepseek' || form.platform === 'minimax' || form.platform === 'opencode_go') {
    return form.platform
  }
  return 'kimi'
})
// 当前平台可选的协议档（responses 仅 deepseek / kimi）。
const cnAdaptiveProtocolOptions = computed<Array<{ value: CnNativeApiProtocol; labelKey: string }>>(() => {
  const opts: Array<{ value: CnNativeApiProtocol; labelKey: string }> = [
    { value: 'chat_completions', labelKey: 'chatCompletions' },
    { value: 'anthropic', labelKey: 'anthropic' }
  ]
  if (cnSupportsNativeResponses(form.platform)) opts.push({ value: 'responses', labelKey: 'responses' })
  return opts
})

function resetAdaptiveBaseUrls(platform: 'kimi' | 'zhipu' | 'deepseek' | 'minimax' | 'opencode_go', mode: CnProviderMode) {
  adaptiveBaseUrls.value = defaultCNAdaptiveBaseUrls(platform, mode)
}
// 当前选中平台的品牌色（选中卡片描边 / 图标底色），与 platformColors 取色一致。
const cnAccentActiveClass = computed(() => {
  switch (form.platform) {
    case 'kimi':
      return 'border-pink-500 bg-pink-50 dark:bg-pink-900/20'
    case 'zhipu':
      return 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/20'
    case 'deepseek':
      return 'border-teal-500 bg-teal-50 dark:bg-teal-900/20'
    case 'minimax':
      return 'border-bh-red bg-red-50 dark:bg-red-950/20'
    default:
      return 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
  }
})
const cnAccentIconClass = computed(() => {
  switch (form.platform) {
    case 'kimi':
      return 'bg-pink-500 text-white'
    case 'zhipu':
      return 'bg-indigo-500 text-white'
    case 'deepseek':
      return 'bg-teal-500 text-white'
    case 'minimax':
      return 'bg-bh-red text-white'
    default:
      return 'bg-primary-500 text-white'
  }
})
// 切换国产供应商平台：强制 apikey 类型，deepseek 无 coding 套餐故锁定 payg，
// 协议回落 adaptive，并把 base url 重置为该平台默认端点。
function selectCNPlatform(platform: 'kimi' | 'zhipu' | 'deepseek' | 'minimax' | 'opencode_go') {
  form.platform = platform
  form.type = 'apikey'
  providerCategory.value = 'apikey'
  apiProtocol.value = 'adaptive'
  if (platform === 'opencode_go') {
    providerMode.value = 'go'
    openCodeRules.value = cloneOpenCodeGoProtocolRules()
  } else if (providerMode.value === 'go' || providerMode.value === 'zen') providerMode.value = 'payg'
  if (platform === 'deepseek') {
    providerMode.value = 'payg'
  }
  if (platform !== 'zhipu') {
    zhipuOrganization.value = ''
    zhipuProject.value = ''
  }
  apiKeyBaseUrl.value = defaultCNBaseUrl(platform, providerMode.value, apiProtocol.value)
  resetAdaptiveBaseUrls(platform, providerMode.value)
}
// 提供商类型 / 协议变更时同步默认 base url。
watch(providerMode, (mode, previousMode) => {
  if (!isCNPlatform.value) return
  if (form.platform === 'opencode_go') {
    const previousRules = defaultOpenCodeProtocolRules(previousMode === 'zen' ? 'zen' : 'go')
    if (JSON.stringify(openCodeRules.value) === JSON.stringify(previousRules)) {
      openCodeRules.value = cloneOpenCodeGoProtocolRules(defaultOpenCodeProtocolRules(mode === 'zen' ? 'zen' : 'go'))
    }
  }
  if (apiProtocol.value === 'adaptive') {
    const previousDefaults = defaultCNAdaptiveBaseUrls(cnPresetPlatform.value, previousMode)
    const nextDefaults = defaultCNAdaptiveBaseUrls(cnPresetPlatform.value, mode)
    for (const item of cnAdaptiveProtocolOptions.value) {
      if (!adaptiveBaseUrls.value[item.value] || adaptiveBaseUrls.value[item.value] === previousDefaults[item.value]) {
        adaptiveBaseUrls.value[item.value] = nextDefaults[item.value]
      }
    }
    apiKeyBaseUrl.value = adaptiveBaseUrls.value.chat_completions
    return
  }
  apiKeyBaseUrl.value = defaultCNBaseUrl(form.platform, mode, apiProtocol.value)
})
watch(apiProtocol, protocol => {
  if (!isCNPlatform.value) return
  if (protocol === 'adaptive') {
    const defaults = defaultCNAdaptiveBaseUrls(cnPresetPlatform.value, providerMode.value)
    for (const item of cnAdaptiveProtocolOptions.value) {
      if (!adaptiveBaseUrls.value[item.value]) adaptiveBaseUrls.value[item.value] = defaults[item.value]
    }
    apiKeyBaseUrl.value = adaptiveBaseUrls.value.chat_completions
    return
  }
  apiKeyBaseUrl.value = defaultCNBaseUrl(form.platform, providerMode.value, protocol)
})

// 端点预设同时更新模式和协议，保持表单字段一致。
function onCnPresetSelect(preset: { mode: CnProviderMode; protocol: CnApiProtocol; url: string }) {
  providerMode.value = preset.mode
  apiProtocol.value = 'adaptive'
  if (preset.protocol !== 'adaptive') adaptiveBaseUrls.value[preset.protocol] = preset.url
  apiKeyBaseUrl.value = adaptiveBaseUrls.value.chat_completions
}

const syncPreviewCredentials = computed(() => {
  if (!apiKeyValue.value) return undefined
  const baseUrl = isCNPlatform.value && apiProtocol.value === 'adaptive'
    ? adaptiveBaseUrls.value.chat_completions.trim() || apiKeyBaseUrl.value.trim()
    : apiKeyBaseUrl.value.trim()
  return {
    platform: form.platform,
    type: form.type,
    base_url: baseUrl || undefined,
    api_key: apiKeyValue.value
  }
})

const editQuotaLimit = ref<number | null>(null)
const editQuotaDailyLimit = ref<number | null>(null)
const editQuotaWeeklyLimit = ref<number | null>(null)
const editDailyResetMode = ref<'rolling' | 'fixed' | null>(null)
const editDailyResetHour = ref<number | null>(null)
const editWeeklyResetMode = ref<'rolling' | 'fixed' | null>(null)
const editWeeklyResetDay = ref<number | null>(null)
const editWeeklyResetHour = ref<number | null>(null)
const editResetTimezone = ref<string | null>(null)
const modelMappings = ref<ModelMapping[]>([])
const openAICompactModelMappings = ref<ModelMapping[]>([])
const modelRestrictionMode = ref<'whitelist' | 'mapping'>('whitelist')
const allowedModels = ref<string[]>([])
const qoderModelRestrictionTouched = ref(false)
const qoderModelWhitelistTouched = ref(false)
const DEFAULT_POOL_MODE_RETRY_COUNT = 3
const MAX_POOL_MODE_RETRY_COUNT = 10
const DEFAULT_POOL_MODE_RETRY_STATUS_CODES = [401, 403, 429]
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
const customErrorCodesEnabled = ref(false)
const selectedErrorCodes = ref<number[]>([])
const customErrorCodeInput = ref<number | null>(null)
const headerOverrideEnabled = ref(false)
const headerOverrideRows = ref<HeaderOverrideRow[]>([])

// Grok OAuth：自定义上游地址（base_url 仅改写转发端点，OAuth 授权/刷新不受影响）
const grokOAuthCustomBaseUrlEnabled = ref(false)
const grokOAuthBaseUrl = ref('')

// Grok OAuth 三条创建路径（授权码/RT 批量/SSO 批量）共用的前置校验。
// 授权码路径必须在兑换 code 之前调用，避免校验失败时白白消耗一次性授权码。
const validateGrokOAuthUpstreamConfig = (): boolean => {
  if (grokOAuthCustomBaseUrlEnabled.value) {
    const trimmed = grokOAuthBaseUrl.value.trim()
    if (!trimmed) {
      appStore.showError(t('admin.providers.grokCustomBaseUrl.required'))
      return false
    }
    if (!/^https?:\/\//i.test(trimmed)) {
      appStore.showError(t('admin.providers.grokCustomBaseUrl.invalid'))
      return false
    }
  }
  if (headerOverrideEnabled.value) {
    const headerError = validateHeaderOverrideRows(headerOverrideRows.value)
    if (headerError) {
      appStore.showError(t(`admin.providers.headerOverride.${headerError}`))
      return false
    }
  }
  return true
}

// 把已通过校验的自定义上游地址与请求头覆写写入 credentials
const applyGrokOAuthUpstreamConfig = (credentials: Record<string, unknown>) => {
  if (grokOAuthCustomBaseUrlEnabled.value) {
    credentials.base_url = grokOAuthBaseUrl.value.trim()
  }
  applyHeaderOverride(credentials, headerOverrideEnabled.value, headerOverrideRows.value, 'create')
}
const interceptWarmupRequests = ref(false)
const autoPauseOnExpired = ref(true)
const autoPause5hThreshold = ref<number | null>(null)
const autoPause7dThreshold = ref<number | null>(null)
const autoPause5hDisabled = ref(false)
const autoPause7dDisabled = ref(false)
const openaiPassthroughEnabled = ref(false)
// OpenAI OAuth namespace 工具摊平兼容开关，缺省关闭即原样保留。
const openaiFlattenNamespacesEnabled = ref(false)
const openAICompactMode = ref<OpenAICompactMode>('force_on')
const openAINativeCompactionV2Mode = ref<OpenAICompactMode>('force_on')
// HTTP continuation 默认关闭，避免把中继提供商误判为支持 previous_response_id。
const openAIResponsesContinuationSupported = ref(false)
// 图片回填默认关闭，只对 OpenAI API Key 提供商生效。
const openAIImagesURLToB64JSON = ref(false)
const openaiOAuthResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const openaiAPIKeyResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const codexCLIOnlyAllowClaudeCodeEnabled = ref(false)
const openAIOAuthClientPolicy = ref<OpenAIOAuthClientPolicy>('any')
type CodexFingerprintMode = 'off' | 'device' | 'session' | 'full'
const codexFingerprintMode = ref<CodexFingerprintMode>('off')
const codexFingerprintModeOptions = computed(() => [
  { value: 'off' as CodexFingerprintMode, label: t('admin.providers.openai.codexFingerprintOff') },
  { value: 'device' as CodexFingerprintMode, label: t('admin.providers.openai.codexFingerprintDevice') },
  { value: 'session' as CodexFingerprintMode, label: t('admin.providers.openai.codexFingerprintSession') },
  { value: 'full' as CodexFingerprintMode, label: t('admin.providers.openai.codexFingerprintFull') },
])
type AnthropicAPIKeyAuthScheme = 'x_api_key' | 'authorization_bearer'
const anthropicPassthroughEnabled = ref(false)
const anthropicAPIKeyAuthScheme = ref<AnthropicAPIKeyAuthScheme>('x_api_key')
const webSearchEmulationMode = ref('default')
const webSearchGlobalEnabled = ref(false)

const {
  globalEnabled: quotaNotifyGlobalEnabled,
  state: quotaNotifyState,
  loadGlobalState: loadQuotaNotifyGlobal,
  writeToExtra: writeQuotaNotifyToExtra,
} = useQuotaNotifyState()

// Load global feature states once
adminAPI.settings.getWebSearchEmulationConfig().then(cfg => {
  webSearchGlobalEnabled.value = cfg?.enabled === true && (cfg?.providers?.length ?? 0) > 0
}).catch(() => { webSearchGlobalEnabled.value = false })

loadQuotaNotifyGlobal()
const allowOverages = ref(false) // For antigravity providers: enable AI Credits overages
const antigravityProviderType = ref<'oauth' | 'upstream'>('oauth') // For antigravity: oauth or upstream
const qoderProviderType = ref<'oauth' | 'manual'>('oauth')
const qoderSite = ref<QoderSite>('global')
const qoderPAT = ref('')
const qoderSecurityOauthToken = ref('')
const qoderMachineId = ref('')
const qoderUidAid = ref('')
const qoderRefreshToken = ref('')
const qoderUserType = ref('personal_standard')
const antigravityProjectId = ref('')
const upstreamBaseUrl = ref('') // For upstream type: base URL
const upstreamApiKey = ref('') // For upstream type: API key
const antigravityModelRestrictionMode = ref<'whitelist' | 'mapping'>('whitelist')
const antigravityWhitelistModels = ref<string[]>([])
const antigravityModelMappings = ref<ModelMapping[]>([])
const antigravityPresetMappings = computed(() => getPresetMappingsByPlatform('antigravity'))
const bedrockPresets = computed(() => getPresetMappingsByPlatform('bedrock'))

// Bedrock credentials
const bedrockAuthMode = ref<'sigv4' | 'apikey'>('sigv4')
const bedrockAccessKeyId = ref('')
const bedrockSecretAccessKey = ref('')
const bedrockSessionToken = ref('')
const bedrockRegion = ref('us-east-1')
const bedrockForceGlobal = ref(false)
const bedrockApiKeyValue = ref('')
const vertexServiceAccountFileInput = ref<HTMLInputElement | null>(null)
const vertexServiceAccountJson = ref('')
const vertexProjectId = ref('')
const vertexClientEmail = ref('')
const vertexLocation = ref('global')
const vertexServiceAccountDragActive = ref(false)
const tempUnschedEnabled = ref(false)
const tempUnschedRules = ref<TempUnschedRuleForm[]>([])
const getModelMappingKey = createStableObjectKeyResolver<ModelMapping>('create-model-mapping')
const getOpenAICompactModelMappingKey = createStableObjectKeyResolver<ModelMapping>('create-openai-compact-model-mapping')
const getAntigravityModelMappingKey = createStableObjectKeyResolver<ModelMapping>('create-antigravity-model-mapping')
const getTempUnschedRuleKey = createStableObjectKeyResolver<TempUnschedRuleForm>('create-temp-unsched-rule')
const geminiOAuthType = ref<'code_assist' | 'google_one' | 'ai_studio'>('google_one')
const geminiAIStudioOAuthEnabled = ref(false)

const openAIOAuthClientPolicyOptions = computed(() => [
  { value: 'any', label: t('admin.providers.openai.clientPolicyAny') },
  { value: 'codex_only', label: t('admin.providers.openai.clientPolicyCodexOnly') },
  { value: 'tls_router_matched_only', label: t('admin.providers.openai.clientPolicyTLSRouterMatchedOnly') }
])

function buildAntigravityExtra(): Record<string, unknown> | undefined {
  const extra: Record<string, unknown> = {}
  if (allowOverages.value) extra.allow_overages = true
  return Object.keys(extra).length > 0 ? extra : undefined
}

const buildOpenAICompactModelMapping = () =>
  buildModelMappingObject('mapping', [], openAICompactModelMappings.value)
const showAdvancedOAuth = ref(false)
const showGeminiHelpDialog = ref(false)

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
const webSearchEmulationOptions = computed(() => [
  { value: 'default', label: t('admin.providers.anthropic.webSearchDefault') },
  { value: 'enabled', label: t('admin.providers.anthropic.webSearchEnabled') },
  { value: 'disabled', label: t('admin.providers.anthropic.webSearchDisabled') }
])
const anthropicAPIKeyAuthSchemeOptions = computed(() => [
  { value: 'x_api_key', label: t('admin.providers.anthropic.apiKeyAuthSchemeXApiKey') },
  { value: 'authorization_bearer', label: t('admin.providers.anthropic.apiKeyAuthSchemeBearer') }
])
const customBaseUrlEnabled = ref(false)
const customBaseUrl = ref('')

// Gemini tier selection (used as fallback when auto-detection is unavailable/fails)
type GeminiProviderType = 'official' | 'third_party'
const geminiProviderType = ref<GeminiProviderType>('official')
const geminiTierGoogleOne = ref<'google_one_free' | 'google_ai_pro' | 'google_ai_ultra'>('google_one_free')
const geminiTierGcp = ref<'gcp_standard' | 'gcp_enterprise'>('gcp_standard')
const geminiTierAIStudio = ref<'aistudio_free' | 'aistudio_paid'>('aistudio_free')

const geminiSelectedTier = computed(() => {
  if (form.platform !== 'gemini') return ''
  if (providerCategory.value === 'apikey') {
    return geminiProviderType.value === 'official' ? geminiTierAIStudio.value : ''
  }
  switch (geminiOAuthType.value) {
    case 'google_one':
      return geminiTierGoogleOne.value
    case 'code_assist':
      return geminiTierGcp.value
    default:
      return geminiTierAIStudio.value
  }
})

const openAIWSModeOptions = computed(() => [
  { value: OPENAI_WS_MODE_OFF, label: t('admin.providers.openai.wsModeOff') },
  { value: OPENAI_WS_MODE_CTX_POOL, label: t('admin.providers.openai.wsModeCtxPool') },
  { value: OPENAI_WS_MODE_PASSTHROUGH, label: t('admin.providers.openai.wsModePassthrough') },
  { value: OPENAI_WS_MODE_HTTP_BRIDGE, label: t('admin.providers.openai.wsModeHttpBridge') }
])

const openaiResponsesWebSocketV2Mode = computed({
  get: () => {
    if (form.platform === 'openai' && providerCategory.value === 'apikey') {
      return openaiAPIKeyResponsesWebSocketV2Mode.value
    }
    return openaiOAuthResponsesWebSocketV2Mode.value
  },
  set: (mode: OpenAIWSMode) => {
    if (form.platform === 'openai' && providerCategory.value === 'apikey') {
      openaiAPIKeyResponsesWebSocketV2Mode.value = mode
      return
    }
    openaiOAuthResponsesWebSocketV2Mode.value = mode
  }
})

const openAIWSModeConcurrencyHintKey = computed(() =>
  resolveOpenAIWSModeConcurrencyHintKey(openaiResponsesWebSocketV2Mode.value)
)

const openAIOAuthImportDefaults = ref<OpenAIOAuthImportDefaults | null>(null)
const ticketDraft = ref<InstanceType<typeof CodexTicketAccountSettings>>()
const ticketCreationPatch = async () => {
  const draft = ticketDraft.value
  if (!draft) throw new Error(t('common.loading'))
  await draft.ensureReady()
  if (!props.show || ticketDraft.value !== draft) throw new Error(t('common.loading'))
  return draft.patch()
}
const openAIOAuthImportDefaultsLoaded = ref(false)
const openAIOAuthImportDefaultsApplied = ref(false)
const isOpenAIOAuthImportDefaultsTarget = computed(
  () => form.platform === 'openai' && providerCategory.value === 'oauth-based'
)

const normalizeOpenAITLSFingerprintProfileId = (value: unknown): number | null => {
  // 默认值来自 JSON 配置，兼容数字和数字字符串，非法值回落到内置默认 profile。
  if (typeof value === 'number' && Number.isInteger(value)) {
    return value === 0 ? null : value
  }
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value)
    return Number.isInteger(parsed) && parsed !== 0 ? parsed : null
  }
  return null
}

const normalizeOpenAIOAuthClientPolicy = (policy: unknown, legacyCodexOnly?: unknown): OpenAIOAuthClientPolicy => {
  if (policy === 'codex_only' || policy === 'tls_router_matched_only' || policy === 'any') {
    return policy
  }
  return legacyCodexOnly === true ? 'codex_only' : 'any'
}

const splitDefaultMappingObject = (raw: unknown): ModelMapping[] => {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
    return []
  }

  return Object.entries(raw as Record<string, unknown>)
    .map(([from, to]) => ({ from: from.trim(), to: String(to).trim() }))
    .filter((mapping) => mapping.from && mapping.to)
}

const isSameStringList = (left: string[], right: string[]) => {
  return left.length === right.length && left.every((item, index) => item === right[index])
}

const applyOpenAIOAuthImportDefaultsToForm = () => {
  const defaults = openAIOAuthImportDefaults.value
  if (!defaults || !isOpenAIOAuthImportDefaultsTarget.value || openAIOAuthImportDefaultsApplied.value) {
    return
  }

  const provider = defaults.provider || {}
  if (typeof provider.notes === 'string' && form.notes.trim() === '') {
    form.notes = provider.notes
  }
  if (
    typeof provider.concurrency === 'number' &&
    Number.isFinite(provider.concurrency) &&
    provider.concurrency > 0 &&
    form.concurrency === 10
  ) {
    form.concurrency = provider.concurrency
  }
  if (
    typeof provider.priority === 'number' &&
    Number.isFinite(provider.priority) &&
    provider.priority > 0 &&
    form.priority === 1
  ) {
    form.priority = provider.priority
  }
  if (
    typeof provider.rate_multiplier === 'number' &&
    Number.isFinite(provider.rate_multiplier) &&
    provider.rate_multiplier >= 0 &&
    form.rate_multiplier === 1
  ) {
    form.rate_multiplier = provider.rate_multiplier
  }
  if (
    typeof provider.expires_at === 'number' &&
    Number.isFinite(provider.expires_at) &&
    provider.expires_at >= 0 &&
    form.expires_at === null
  ) {
    form.expires_at = provider.expires_at
  }
  if (typeof provider.auto_pause_on_expired === 'boolean' && autoPauseOnExpired.value === true) {
    autoPauseOnExpired.value = provider.auto_pause_on_expired
  }

  const credentials = defaults.credentials || {}
  const openAIModels = getModelsByPlatform('openai')
  const defaultModelRestriction = splitPersistedModelRestriction(
    credentials.model_mapping && typeof credentials.model_mapping === 'object' && !Array.isArray(credentials.model_mapping)
      ? credentials.model_mapping as Record<string, string>
      : undefined,
    credentials.model_whitelist
  )
  const hasDefaultModelWhitelist =
    Object.prototype.hasOwnProperty.call(credentials, 'model_whitelist') ||
    defaultModelRestriction.allowedModels.length > 0
  if (
    hasDefaultModelWhitelist &&
    modelRestrictionMode.value === 'whitelist' &&
    modelMappings.value.length === 0 &&
    isSameStringList(allowedModels.value, openAIModels)
  ) {
    allowedModels.value = defaultModelRestriction.allowedModels
  }
  if (defaultModelRestriction.modelMappings.length > 0 && modelMappings.value.length === 0) {
    modelMappings.value = defaultModelRestriction.modelMappings
    modelRestrictionMode.value = 'mapping'
  }
  const defaultCompactMappings = splitDefaultMappingObject(credentials.compact_model_mapping)
  if (defaultCompactMappings.length > 0 && openAICompactModelMappings.value.length === 0) {
    openAICompactModelMappings.value = defaultCompactMappings
  }

  const extra = defaults.extra || {}
  if (extra.openai_passthrough === true || extra.openai_oauth_passthrough === true) {
    openaiPassthroughEnabled.value = true
  }
  openAIOAuthClientPolicy.value = normalizeOpenAIOAuthClientPolicy(extra.openai_oauth_client_policy, extra.codex_cli_only)
  if (
    Array.isArray(extra.codex_cli_only_allowed_clients) &&
    extra.codex_cli_only_allowed_clients.includes('claude_code')
  ) {
    codexCLIOnlyAllowClaudeCodeEnabled.value = true
  }
  if (
    typeof extra.auto_pause_5h_threshold === 'number' &&
    Number.isFinite(extra.auto_pause_5h_threshold) &&
    autoPause5hThreshold.value == null
  ) {
    autoPause5hThreshold.value = extra.auto_pause_5h_threshold * 100
  }
  if (
    typeof extra.auto_pause_7d_threshold === 'number' &&
    Number.isFinite(extra.auto_pause_7d_threshold) &&
    autoPause7dThreshold.value == null
  ) {
    autoPause7dThreshold.value = extra.auto_pause_7d_threshold * 100
  }
  if (extra.auto_pause_5h_disabled === true) {
    autoPause5hDisabled.value = true
  }
  if (extra.auto_pause_7d_disabled === true) {
    autoPause7dDisabled.value = true
  }
  const defaultWSMode = resolveOpenAIWSModeFromExtra(extra, {
    modeKey: 'openai_oauth_responses_websockets_v2_mode',
    enabledKey: 'openai_oauth_responses_websockets_v2_enabled',
    fallbackEnabledKeys: ['responses_websockets_v2_enabled', 'openai_ws_enabled'],
    defaultMode: OPENAI_WS_MODE_OFF
  })
  if (openaiOAuthResponsesWebSocketV2Mode.value === OPENAI_WS_MODE_OFF) {
    openaiOAuthResponsesWebSocketV2Mode.value = defaultWSMode
  }
  if (openAICompactMode.value === 'force_on') {
    openAICompactMode.value = normalizeOpenAICompactMode(extra.openai_compact_mode)
  }
  if (openAINativeCompactionV2Mode.value === 'force_on') {
    openAINativeCompactionV2Mode.value = normalizeOpenAICompactMode(extra.openai_native_compaction_v2_mode)
  }
  if (extra.enable_tls_fingerprint === true) {
    tlsFingerprintEnabled.value = true
    tlsFingerprintProfileId.value = normalizeOpenAITLSFingerprintProfileId(extra.tls_fingerprint_profile_id)
    tlsFingerprintRouterId.value = normalizeOpenAITLSFingerprintProfileId(extra.tls_fingerprint_router_id)
  }

  openAIOAuthImportDefaultsApplied.value = true
}

const loadOpenAIOAuthImportDefaults = async () => {
  if (openAIOAuthImportDefaultsLoaded.value) {
    applyOpenAIOAuthImportDefaultsToForm()
    return
  }
  try {
    openAIOAuthImportDefaults.value = await adminAPI.settings.getOpenAIOAuthImportDefaults()
    openAIOAuthImportDefaultsLoaded.value = true
    applyOpenAIOAuthImportDefaultsToForm()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.providers.openAIOAuthImportDefaultsLoadFailed'))
  }
}

const applyOpenAIOAuthCredentialDefaults = (credentials: Record<string, unknown>) => {
  if (!isOpenAIOAuthImportDefaultsTarget.value) {
    return
  }

  const defaults = openAIOAuthImportDefaults.value?.credentials || {}
  for (const [key, value] of Object.entries(defaults)) {
    if (key === 'model_whitelist' || key === 'model_mapping' || key === 'compact_model_mapping') {
      continue
    }
    if (!Object.prototype.hasOwnProperty.call(credentials, key)) {
      credentials[key] = value
    }
  }
}

const geminiQuotaDocs = {
  codeAssist: 'https://developers.google.com/gemini-code-assist/resources/quotas',
  aiStudio: 'https://ai.google.dev/pricing',
  vertex: 'https://cloud.google.com/vertex-ai/generative-ai/docs/quotas'
}

const geminiHelpLinks = {
  apiKey: 'https://aistudio.google.com/app/apikey',
  aiStudioPricing: 'https://ai.google.dev/pricing',
  gcpProject: 'https://console.cloud.google.com/welcome/new',
  geminiWebActivation: 'https://gemini.google.com/gems/create?hl=en-US&pli=1',
  countryCheck: 'https://policies.google.com/terms',
  countryChange: 'https://policies.google.com/country-association-form'
}

// Computed: current preset mappings based on platform
const presetMappings = computed(() =>
  getPresetMappingsByPlatform(form.platform, form.platform === 'qoder' ? qoderSite.value : undefined)
)
const qoderAvailableModels = computed(() => getModelsByPlatform('qoder', qoderSite.value))
const tempUnschedPresets = computed(() => [
  {
    label: t('admin.providers.tempUnschedulable.presets.overloadLabel'),
    rule: {
      error_code: 529,
      keywords: 'overloaded, too many',
      duration_minutes: 60,
      description: t('admin.providers.tempUnschedulable.presets.overloadDesc')
    }
  },
  {
    label: t('admin.providers.tempUnschedulable.presets.rateLimitLabel'),
    rule: {
      error_code: 429,
      keywords: 'rate limit, too many requests',
      duration_minutes: 10,
      description: t('admin.providers.tempUnschedulable.presets.rateLimitDesc')
    }
  },
  {
    label: t('admin.providers.tempUnschedulable.presets.unavailableLabel'),
    rule: {
      error_code: 503,
      keywords: 'unavailable, maintenance',
      duration_minutes: 30,
      description: t('admin.providers.tempUnschedulable.presets.unavailableDesc')
    }
  }
])

const form = reactive({
  name: '',
  notes: '',
  platform: 'anthropic' as ProviderPlatform,
  type: 'oauth' as ProviderType, // Will be 'oauth', 'setup-token', or 'apikey'
  credentials: {} as Record<string, unknown>,
  proxy_id: null as number | null,
  concurrency: 10,
  load_factor: null as number | null,
  priority: 1,
  rate_multiplier: 1,
  group_ids: [] as number[],
  expires_at: null as number | null
})

// Helper to check if current type needs OAuth flow
const isOAuthFlow = computed(() => {
  // Antigravity upstream 类型不需要 OAuth 流程
  if (form.platform === 'antigravity' && antigravityProviderType.value === 'upstream') {
    return false
  }
  if (form.platform === 'qoder' && qoderProviderType.value === 'manual') {
    return false
  }
  // Bedrock 类型不需要 OAuth 流程
  if (form.platform === 'anthropic' && providerCategory.value === 'bedrock') {
    return false
  }
  return providerCategory.value === 'oauth-based'
})

const isGrokSSOInputMethod = computed(() => form.platform === 'grok' && oauthFlowRef.value?.inputMethod === 'sso_cookie')

const isManualInputMethod = computed(() => {
  return oauthFlowRef.value?.inputMethod === 'manual'
})

const expiresAtInput = computed({
  get: () => formatDateTimeLocal(form.expires_at),
  set: (value: string) => {
    form.expires_at = parseDateTimeLocal(value)
  }
})

const canExchangeCode = computed(() => {
  const authCode = oauthFlowRef.value?.authCode || ''
  if (form.platform === 'openai') {
    return authCode.trim() && openaiOAuth.authSessions.value.length > 0 && !openaiOAuth.loading.value
  }
  if (form.platform === 'gemini') {
    return authCode.trim() && geminiOAuth.sessionId.value && !geminiOAuth.loading.value
  }
  if (form.platform === 'antigravity') {
    return authCode.trim() && antigravityOAuth.sessionId.value && !antigravityOAuth.loading.value
  }
  if (form.platform === 'qoder') {
    return qoderOAuth.sessionId.value &&
      !qoderOAuth.loading.value &&
      !submitting.value &&
      !isQoderOAuthProviderCreating.value
  }
  if (form.platform === 'grok') {
    return authCode.trim() && grokOAuth.sessionId.value && !grokOAuth.loading.value
  }
  return authCode.trim() && oauth.sessionId.value && !oauth.loading.value
})

// Watchers
watch(
  () => props.show,
  (newVal) => {
    if (newVal) {
      form.platform = props.initialPlatform || 'anthropic'
      // Load TLS fingerprint profiles
      adminAPI.tlsFingerprintProfiles.list()
        .then(profiles => { tlsFingerprintProfiles.value = profiles.map(p => ({ id: p.id, name: p.name })) })
        .catch(() => { tlsFingerprintProfiles.value = [] })
      adminAPI.tlsFingerprintRouters.list()
        .then(routers => { tlsFingerprintRouters.value = routers.map(router => ({ id: router.id, name: router.name })) })
        .catch(() => { tlsFingerprintRouters.value = [] })
      // Modal opened - fill related models
      allowedModels.value = []
      if (isOpenAIOAuthImportDefaultsTarget.value) {
        void loadOpenAIOAuthImportDefaults()
      }
      // Antigravity: 默认使用映射模式并填充默认映射
      if (form.platform === 'antigravity') {
        antigravityModelRestrictionMode.value = 'mapping'
        fetchAntigravityDefaultMappings().then(mappings => {
          antigravityModelMappings.value = [...mappings]
        })
        antigravityWhitelistModels.value = []
      } else {
        antigravityWhitelistModels.value = []
        antigravityModelMappings.value = []
        antigravityModelRestrictionMode.value = 'mapping'
      }
    } else {
      resetForm()
    }
  }
)

// Sync form.type based on providerCategory, addMethod, and platform-specific type
watch(
  [providerCategory, addMethod, antigravityProviderType, qoderProviderType, () => form.platform],
  ([category, method, agType]) => {
    if (form.platform === 'qoder') {
      form.type = 'cosy'
      return
    }
    // Antigravity upstream 类型（实际创建为 apikey）
    if (form.platform === 'antigravity' && agType === 'upstream') {
      form.type = 'apikey'
      return
    }
    // Bedrock 类型
    if (form.platform === 'anthropic' && category === 'bedrock') {
      form.type = 'bedrock' as ProviderType
      return
    }
    if ((form.platform === 'gemini' || form.platform === 'anthropic') && category === 'service_account') {
      form.type = 'service_account' as ProviderType
    } else if (category === 'oauth-based') {
      form.type = form.platform === 'anthropic' ? method as ProviderType : 'oauth'
    } else {
      form.type = 'apikey'
    }
  },
  { immediate: true }
)

watch(
  isOpenAIOAuthImportDefaultsTarget,
  (enabled) => {
    if (!enabled) {
      openAIOAuthImportDefaultsApplied.value = false
      return
    }
    void loadOpenAIOAuthImportDefaults()
  }
)

// Reset platform-specific settings when platform changes
watch(
  () => form.platform,
  (newPlatform) => {
    // Reset base URL based on platform
    apiKeyBaseUrl.value =
      (newPlatform === 'openai')
        ? 'https://api.openai.com'
        : newPlatform === 'gemini'
          ? geminiProviderType.value === 'third_party'
            ? ''
            : 'https://generativelanguage.googleapis.com'
          : newPlatform === 'grok'
            ? 'https://api.x.ai/v1'
            : 'https://api.anthropic.com'
    // 切换平台时旧平台模型不再适用。Qoder 由提供商 model_mapping
    // 配置展示/请求模型，默认不填充会过期的前端硬编码白名单。
    allowedModels.value = newPlatform === 'qoder' ? [] : [...getModelsByPlatform(newPlatform)]
    modelMappings.value = []
    modelRestrictionMode.value = (newPlatform === 'qoder' || newPlatform === 'grok') ? 'mapping' : 'whitelist'
    qoderModelRestrictionTouched.value = false
    qoderModelWhitelistTouched.value = false
    // Antigravity: 默认使用映射模式并填充默认映射
    if (newPlatform === 'antigravity') {
      antigravityModelRestrictionMode.value = 'mapping'
      fetchAntigravityDefaultMappings().then(mappings => {
        antigravityModelMappings.value = [...mappings]
      })
      antigravityWhitelistModels.value = []
      providerCategory.value = 'oauth-based'
      antigravityProviderType.value = 'oauth'
    } else {
      allowOverages.value = false
      antigravityWhitelistModels.value = []
      antigravityModelMappings.value = []
      antigravityModelRestrictionMode.value = 'mapping'
    }
    if (newPlatform === 'qoder') {
      providerCategory.value = 'oauth-based'
      qoderProviderType.value = 'oauth'
      qoderSite.value = 'global'
    } else {
      qoderProviderType.value = 'oauth'
      qoderPAT.value = ''
      qoderSecurityOauthToken.value = ''
      qoderMachineId.value = ''
      qoderUidAid.value = ''
      qoderRefreshToken.value = ''
      qoderUserType.value = 'personal_standard'
    }
    if (newPlatform === 'grok') {
      providerCategory.value = 'oauth-based'
      addMethod.value = 'oauth'
      modelRestrictionMode.value = 'mapping'
      form.concurrency = 1
      form.load_factor = null
    }
    if (newPlatform !== 'gemini' && newPlatform !== 'anthropic' && providerCategory.value === 'service_account') {
      providerCategory.value = 'oauth-based'
    }
    if (newPlatform !== 'anthropic' && providerCategory.value === 'bedrock') {
      providerCategory.value = 'oauth-based'
    }
    // Reset Bedrock fields when switching platforms
    bedrockAccessKeyId.value = ''
    bedrockSecretAccessKey.value = ''
    bedrockSessionToken.value = ''
    bedrockRegion.value = 'us-east-1'
    bedrockForceGlobal.value = false
    bedrockAuthMode.value = 'sigv4'
    bedrockApiKeyValue.value = ''
    vertexServiceAccountJson.value = ''
    vertexProjectId.value = ''
    vertexClientEmail.value = ''
    vertexLocation.value = 'global'
    // Reset Anthropic/Antigravity-specific settings when switching to other platforms
    if (newPlatform !== 'anthropic' && newPlatform !== 'antigravity') {
      interceptWarmupRequests.value = false
    }
    if (newPlatform !== 'openai') {
      openaiPassthroughEnabled.value = false
      openaiFlattenNamespacesEnabled.value = false
      openAIResponsesContinuationSupported.value = false
      openAIImagesURLToB64JSON.value = false
      openaiOAuthResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      openaiAPIKeyResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      codexCLIOnlyAllowClaudeCodeEnabled.value = false
      openAIOAuthClientPolicy.value = 'any'
    }
    if (newPlatform !== 'anthropic') {
      anthropicPassthroughEnabled.value = false
      anthropicAPIKeyAuthScheme.value = 'x_api_key'
      webSearchEmulationMode.value = 'default'
    }
    // 请求头覆写为平台相关配置（常用头集合不同），切换平台时清空，
    // 避免上一平台的配置行被提交到新平台提供商
    headerOverrideEnabled.value = false
    headerOverrideRows.value = []
    grokOAuthCustomBaseUrlEnabled.value = false
    grokOAuthBaseUrl.value = ''
    // Reset OAuth states
    oauth.resetState()
    openaiOAuth.resetState()

    geminiOAuth.resetState()
    antigravityOAuth.resetState()
    qoderOAuth.resetState()
    grokOAuth.resetState()
  }
)

watch(geminiProviderType, (providerType) => {
  if (form.platform !== 'gemini' || providerType !== 'third_party') return
  // 切换到第三方来源时，不能把官方默认端点误当成第三方地址提交。
  if (!isGeminiThirdPartyBaseUrl(apiKeyBaseUrl.value)) {
    apiKeyBaseUrl.value = ''
  }
})

watch(qoderSite, (newSite, oldSite) => {
  if (newSite === oldSite || form.platform !== 'qoder') return
  // OAuth 会话冻结站点和代理；切站后必须销毁旧会话，手动输入保持不变。
  stopQoderPolling()
  closeQoderAuthPopup()
  resetQoderOAuthCompletionState()
  qoderOAuth.resetState()
})

// Gemini AI Studio OAuth availability (requires operator-configured OAuth client)
watch(
  [providerCategory, () => form.platform],
  ([category, platform]) => {
    if (platform === 'openai' && category !== 'oauth-based') {
      codexCLIOnlyAllowClaudeCodeEnabled.value = false
      openAIOAuthClientPolicy.value = 'any'
      tlsFingerprintRouterId.value = null
    }
    if (platform !== 'anthropic' || category !== 'apikey') {
      anthropicPassthroughEnabled.value = false
      anthropicAPIKeyAuthScheme.value = 'x_api_key'
      webSearchEmulationMode.value = 'default'
    }
  }
)

watch(
  [() => props.show, () => form.platform, providerCategory],
  async ([show, platform, category]) => {
    if (!show || platform !== 'gemini' || category !== 'oauth-based') {
      geminiAIStudioOAuthEnabled.value = false
      return
    }
    const caps = await geminiOAuth.getCapabilities()
    geminiAIStudioOAuthEnabled.value = !!caps?.ai_studio_oauth_enabled
    if (!geminiAIStudioOAuthEnabled.value && geminiOAuthType.value === 'ai_studio') {
      geminiOAuthType.value = 'code_assist'
    }
  },
  { immediate: true }
)

const handleSelectGeminiOAuthType = (oauthType: 'code_assist' | 'google_one' | 'ai_studio') => {
  if (oauthType === 'ai_studio' && !geminiAIStudioOAuthEnabled.value) {
    appStore.showError(t('admin.providers.oauth.gemini.aiStudioNotConfigured'))
    return
  }
  geminiOAuthType.value = oauthType
}

watch(
  [antigravityModelRestrictionMode, () => form.platform],
  ([, platform]) => {
    if (platform !== 'antigravity') return
    // Antigravity 默认不做限制：白名单留空表示允许所有（包含未来新增模型）。
    // 如果需要快速填充常用模型，可在组件内点“填充相关模型”。
  }
)

// Model mapping helpers
const touchQoderModelRestriction = () => {
  if (form.platform === 'qoder') {
    qoderModelRestrictionTouched.value = true
  }
}

const setAllowedModels = (models: string[]) => {
  touchQoderModelRestriction()
  if (form.platform === 'qoder') {
    qoderModelWhitelistTouched.value = true
  }
  allowedModels.value = models
}

const addModelMapping = () => {
  touchQoderModelRestriction()
  modelMappings.value.push({ from: '', to: '' })
}

const addOpenAICompactModelMapping = () => {
  openAICompactModelMappings.value.push({ from: '', to: '' })
}

const removeOpenAICompactModelMapping = (index: number) => {
  openAICompactModelMappings.value.splice(index, 1)
}

const removeModelMapping = (index: number) => {
  touchQoderModelRestriction()
  modelMappings.value.splice(index, 1)
}

const applyPersistedModelRestriction = (credentials: Record<string, unknown>) => {
  // 普通提供商将请求侧映射与最终白名单拆开持久化。
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
  if (!qoderModelRestrictionTouched.value) {
    delete credentials.model_mapping
    delete credentials.model_whitelist
    return
  }
  const persisted = buildPersistedModelRestriction(
    qoderModelWhitelistTouched.value ? allowedModels.value : [],
    modelMappings.value
  )
  if (persisted.modelMapping) {
    credentials.model_mapping = persisted.modelMapping
  } else {
    delete credentials.model_mapping
  }
  credentials.model_whitelist = persisted.modelWhitelist
}

const addPresetMapping = (from: string, to: string) => {
  touchQoderModelRestriction()
  if (modelMappings.value.some((m) => m.from === from)) {
    appStore.showInfo(t('admin.providers.mappingExists', { model: from }))
    return
  }
  modelMappings.value.push({ from, to })
}

const addAntigravityModelMapping = () => {
  antigravityModelMappings.value.push({ from: '', to: '' })
}

const removeAntigravityModelMapping = (index: number) => {
  antigravityModelMappings.value.splice(index, 1)
}

const addAntigravityPresetMapping = (from: string, to: string) => {
  if (antigravityModelMappings.value.some((m) => m.from === from)) {
    appStore.showInfo(t('admin.providers.mappingExists', { model: from }))
    return
  }
  antigravityModelMappings.value.push({ from, to })
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

const addTempUnschedRule = (preset?: TempUnschedRuleForm) => {
  if (preset) {
    tempUnschedRules.value.push({ ...preset })
    return
  }
  tempUnschedRules.value.push({
    error_code: null,
    keywords: '',
    duration_minutes: 30,
    description: ''
  })
}

const removeTempUnschedRule = (index: number) => {
  tempUnschedRules.value.splice(index, 1)
}

const moveTempUnschedRule = (index: number, direction: number) => {
  const target = index + direction
  if (target < 0 || target >= tempUnschedRules.value.length) return
  const rules = tempUnschedRules.value
  const current = rules[index]
  rules[index] = rules[target]
  rules[target] = current
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

const splitTempUnschedKeywords = (value: string) => {
  return value
    .split(/[,;]/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

// 普通提交、批量授权与导入入口共用此边界，不能遗漏原生集合。
async function createProtocolProvider(payload: CreateProviderRequest) {
  if (payload.platform === 'openai' && payload.type === 'oauth' && String(payload.credentials?.auth_mode || '').toLowerCase() !== 'agentidentity') {
    payload.codex_ticket = await ticketCreationPatch()
  }
  if (payload.platform === 'antigravity') {
    payload.credentials = { ...payload.credentials, model_whitelist: [...antigravityWhitelistModels.value] }
  } else if (payload.platform === 'gemini') {
    applyPersistedModelRestriction(payload.credentials)
  }
  await loadProtocolCatalog()
  const options = nativeProtocolOptions(payload.platform, payload.type, String(payload.credentials?.auth_mode ?? ''))
  payload.credentials = { ...payload.credentials, upstream_protocols: (upstreamProtocols.value ?? options).filter(id => options.includes(id)) }
  delete payload.credentials.api_protocol
  delete payload.credentials.openai_workload_capabilities
  if (payload.extra) delete payload.extra.openai_text_route_mode
  return adminAPI.providers.create(payload)
}

type ProviderCreateGuard = () => boolean

const submitCreateProvider = async (
  payload: CreateProviderRequest,
  isCurrent: ProviderCreateGuard = () => true
): Promise<boolean> => {
  submitting.value = true
  try {
    await createProtocolProvider(payload)
    if (!isCurrent()) return false
    appStore.showSuccess(t('admin.providers.providerCreated'))
    emit('created')
    finishClose()
    return true
  } catch (error: any) {
    if (!isCurrent()) return false
    appStore.showError(error.response?.data?.message || error.response?.data?.detail || t('admin.providers.failedToCreate'))
    return false
  } finally {
    // 旧流程结束时不能解除新流程的提交锁。
    if (isCurrent()) {
      submitting.value = false
    }
  }
}

// Methods
const resetForm = () => {
  stopQoderPolling()
  closeQoderAuthPopup()
  resetQoderOAuthCompletionState()
  step.value = 1
  form.name = ''
  form.notes = ''
  form.platform = 'anthropic'
  form.type = 'oauth'
  form.credentials = {}
  upstreamProtocols.value = undefined
  form.proxy_id = null
  form.concurrency = 10
  form.load_factor = null
  form.priority = 1
  form.rate_multiplier = 1
  form.group_ids = []
  form.expires_at = null
  providerCategory.value = 'oauth-based'
  addMethod.value = 'oauth'
  providerMode.value = 'payg'
  apiProtocol.value = 'adaptive'
  zhipuOrganization.value = ''
  zhipuProject.value = ''
  adaptiveBaseUrls.value = { chat_completions: '', anthropic: '', responses: '' }
  apiKeyBaseUrl.value = 'https://api.anthropic.com'
  apiKeyValue.value = ''
  upstreamUsageEnabled.value = true
  upstreamUsageAdapter.value = 'sub2api'
  upstreamUsageBaseUrl.value = ''
  upstreamUsageWalletAccessToken.value = ''
  upstreamUsageWalletUserId.value = ''
  upstreamRequestIdHeader.value = ''
  editQuotaLimit.value = null
  editQuotaDailyLimit.value = null
  editQuotaWeeklyLimit.value = null
  editDailyResetMode.value = null
  editDailyResetHour.value = null
  editWeeklyResetMode.value = null
  editWeeklyResetDay.value = null
  editWeeklyResetHour.value = null
  editResetTimezone.value = null
  modelMappings.value = []
  openAICompactModelMappings.value = []
  modelRestrictionMode.value = 'whitelist'
  allowedModels.value = [...claudeModels] // Default fill related models
  qoderModelRestrictionTouched.value = false
  qoderModelWhitelistTouched.value = false

  antigravityModelRestrictionMode.value = 'mapping'
  antigravityWhitelistModels.value = []
  antigravityProjectId.value = ''
  fetchAntigravityDefaultMappings().then(mappings => {
    antigravityModelMappings.value = [...mappings]
  })
  poolModeEnabled.value = false
  poolModeRetryCount.value = DEFAULT_POOL_MODE_RETRY_COUNT
  poolModeRetryStatusCodesInput.value = ''
  customErrorCodesEnabled.value = false
  selectedErrorCodes.value = []
  customErrorCodeInput.value = null
  headerOverrideEnabled.value = false
  headerOverrideRows.value = []
  grokOAuthCustomBaseUrlEnabled.value = false
  grokOAuthBaseUrl.value = ''
  interceptWarmupRequests.value = false
  autoPauseOnExpired.value = true
  autoPause5hThreshold.value = null
  autoPause7dThreshold.value = null
  autoPause5hDisabled.value = false
  autoPause7dDisabled.value = false
  openaiPassthroughEnabled.value = false
  openaiFlattenNamespacesEnabled.value = false
  openAICompactMode.value = 'force_on'
  openAINativeCompactionV2Mode.value = 'force_on'
  openAIResponsesContinuationSupported.value = false
  openAIImagesURLToB64JSON.value = false
  openaiOAuthResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
  openaiAPIKeyResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
  codexCLIOnlyAllowClaudeCodeEnabled.value = false
  openAIOAuthClientPolicy.value = 'any'
  codexFingerprintMode.value = 'off'
  anthropicPassthroughEnabled.value = false
  anthropicAPIKeyAuthScheme.value = 'x_api_key'
  webSearchEmulationMode.value = 'default'
  // Reset quota control state
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
  allowOverages.value = false
  antigravityProviderType.value = 'oauth'
  qoderProviderType.value = 'oauth'
  qoderSite.value = 'global'
  qoderPAT.value = ''
  qoderSecurityOauthToken.value = ''
  qoderMachineId.value = ''
  qoderUidAid.value = ''
  qoderRefreshToken.value = ''
  qoderUserType.value = 'personal_standard'
  antigravityProjectId.value = ''
  upstreamBaseUrl.value = ''
  upstreamApiKey.value = ''
  vertexServiceAccountJson.value = ''
  vertexProjectId.value = ''
  vertexClientEmail.value = ''
  vertexLocation.value = 'global'
  tempUnschedEnabled.value = false
  tempUnschedRules.value = []
  geminiOAuthType.value = 'code_assist'
  geminiProviderType.value = 'official'
  geminiTierGoogleOne.value = 'google_one_free'
  geminiTierGcp.value = 'gcp_standard'
  geminiTierAIStudio.value = 'aistudio_free'
  oauth.resetState()
  openaiOAuth.resetState()
  geminiOAuth.resetState()
  antigravityOAuth.resetState()
  qoderOAuth.resetState()
  grokOAuth.resetState()
  oauthFlowRef.value?.reset()
  openAIOAuthImportDefaults.value = null
  openAIOAuthImportDefaultsLoaded.value = false
  openAIOAuthImportDefaultsApplied.value = false
}

const finishClose = () => {
  stopQoderPolling()
  resetQoderOAuthCompletionState()
  closeQoderAuthPopup()
  emit('close')
}

const handleClose = () => {
  // Qoder 创建请求不可取消；等待服务端响应，避免提供商已创建但页面丢弃成功事件。
  if (form.platform === 'qoder' && submitting.value) return
  finishClose()
}

const buildOpenAIExtra = (base?: Record<string, unknown>): Record<string, unknown> | undefined => {
  if (form.platform !== 'openai') {
    return base
  }

  const defaultsExtra =
    providerCategory.value === 'oauth-based'
      ? openAIOAuthImportDefaults.value?.extra
      : undefined
  const extra: Record<string, unknown> = { ...(defaultsExtra || {}), ...(base || {}) }
  if (providerCategory.value === 'oauth-based') {
    extra.openai_oauth_responses_websockets_v2_mode = openaiOAuthResponsesWebSocketV2Mode.value
    extra.openai_oauth_responses_websockets_v2_enabled = isOpenAIWSModeEnabled(openaiOAuthResponsesWebSocketV2Mode.value)
  } else if (providerCategory.value === 'apikey') {
    extra.openai_apikey_responses_websockets_v2_mode = openaiAPIKeyResponsesWebSocketV2Mode.value
    extra.openai_apikey_responses_websockets_v2_enabled = isOpenAIWSModeEnabled(openaiAPIKeyResponsesWebSocketV2Mode.value)
  }
  // 清理兼容旧键，统一改用分类型开关。
  delete extra.responses_websockets_v2_enabled
  delete extra.openai_ws_enabled
  delete extra.openai_long_context_billing_enabled
  if (openaiPassthroughEnabled.value) {
    extra.openai_passthrough = true
  } else {
    delete extra.openai_passthrough
    delete extra.openai_oauth_passthrough
  }
  // 关闭时删除默认项，避免提供商 extra 堆积无意义的 false。
  if (form.type === 'oauth' && openaiFlattenNamespacesEnabled.value) {
    extra.openai_responses_flatten_namespaces = true
  } else {
    delete extra.openai_responses_flatten_namespaces
  }
  if (providerCategory.value === 'oauth-based') {
    extra.openai_oauth_client_policy = openAIOAuthClientPolicy.value
    if (openAIOAuthClientPolicy.value === 'codex_only') {
      extra.codex_cli_only = true
    } else {
      delete extra.codex_cli_only
    }
    if (openAIOAuthClientPolicy.value === 'codex_only' && codexCLIOnlyAllowClaudeCodeEnabled.value) {
      extra.codex_cli_only_allowed_clients = ['claude_code']
    } else {
      delete extra.codex_cli_only_allowed_clients
    }
  } else {
    delete extra.openai_oauth_client_policy
    delete extra.codex_cli_only
    delete extra.codex_cli_only_allowed_clients
  }

  if (providerCategory.value === 'oauth-based') {
    if (autoPause5hThreshold.value != null && autoPause5hThreshold.value > 0) {
      extra.auto_pause_5h_threshold = autoPause5hThreshold.value / 100
    } else {
      delete extra.auto_pause_5h_threshold
    }
    if (autoPause7dThreshold.value != null && autoPause7dThreshold.value > 0) {
      extra.auto_pause_7d_threshold = autoPause7dThreshold.value / 100
    } else {
      delete extra.auto_pause_7d_threshold
    }
    if (autoPause5hDisabled.value) {
      extra.auto_pause_5h_disabled = true
    } else {
      delete extra.auto_pause_5h_disabled
    }
    if (autoPause7dDisabled.value) {
      extra.auto_pause_7d_disabled = true
    } else {
      delete extra.auto_pause_7d_disabled
    }
  } else {
    delete extra.auto_pause_5h_threshold
    delete extra.auto_pause_7d_threshold
    delete extra.auto_pause_5h_disabled
    delete extra.auto_pause_7d_disabled
  }
  if (providerCategory.value === 'oauth-based' && tlsFingerprintEnabled.value) {
    extra.enable_tls_fingerprint = true
    if (tlsFingerprintProfileId.value) {
      extra.tls_fingerprint_profile_id = tlsFingerprintProfileId.value
    } else {
      delete extra.tls_fingerprint_profile_id
    }
    if (form.platform === 'openai' && providerCategory.value === 'oauth-based' && tlsFingerprintRouterId.value) {
      extra.tls_fingerprint_router_id = tlsFingerprintRouterId.value
    } else {
      delete extra.tls_fingerprint_router_id
    }
  } else {
    delete extra.enable_tls_fingerprint
    delete extra.tls_fingerprint_profile_id
    delete extra.tls_fingerprint_router_id
  }

  // 收敛是显式 opt-in；off 为默认值，不写入 extra。
  if (providerCategory.value === 'oauth-based' && codexFingerprintMode.value !== 'off') {
    extra.codex_fingerprint_mode = codexFingerprintMode.value
  } else {
    delete extra.codex_fingerprint_mode
  }
  extra.openai_compact_mode = openAICompactMode.value
  extra.openai_native_compaction_v2_mode = openAINativeCompactionV2Mode.value

  if (providerCategory.value === 'apikey' && openAIImagesURLToB64JSON.value) {
    extra.images_url_to_b64_json = true
  } else {
    delete extra.images_url_to_b64_json
  }

  if (providerCategory.value === 'apikey') {
    delete extra.openai_responses_mode
    extra.openai_responses_continuation_supported = openAIResponsesContinuationSupported.value
  }

  return Object.keys(extra).length > 0 ? normalizeLegacyOpenAIExtra(extra) : undefined
}

const buildQoderExtra = (): Record<string, unknown> | undefined => {
  const extra: Record<string, unknown> = {}
  if (tlsFingerprintEnabled.value) {
    extra.enable_tls_fingerprint = true
    if (tlsFingerprintProfileId.value) {
      extra.tls_fingerprint_profile_id = tlsFingerprintProfileId.value
    }
  }
  return Object.keys(extra).length > 0 ? extra : undefined
}

const buildAnthropicExtra = (base?: Record<string, unknown>): Record<string, unknown> | undefined => {
  if (form.platform !== 'anthropic' || providerCategory.value !== 'apikey') {
    return base
  }

  const extra: Record<string, unknown> = { ...(base || {}) }
  if (anthropicPassthroughEnabled.value) {
    extra.anthropic_passthrough = true
  } else {
    delete extra.anthropic_passthrough
  }
  if (anthropicAPIKeyAuthScheme.value === 'authorization_bearer') {
    extra.anthropic_apikey_auth_scheme = 'authorization_bearer'
  } else {
    delete extra.anthropic_apikey_auth_scheme
  }
  if (webSearchEmulationMode.value === 'default') {
    delete extra.web_search_emulation
  } else {
    extra.web_search_emulation = webSearchEmulationMode.value
  }

  return Object.keys(extra).length > 0 ? extra : undefined
}

const doCreateProvider = async (payload: CreateProviderRequest, isCurrent: ProviderCreateGuard = () => true): Promise<boolean> => {
  if (!isCurrent()) return false
  return submitCreateProvider(payload, isCurrent)
}

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

const applyVertexServiceAccountJson = (value: string) => {
  const raw = value.trim()
  if (!raw) {
    vertexProjectId.value = ''
    vertexClientEmail.value = ''
    return false
  }
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>
    const projectId = typeof parsed.project_id === 'string' ? parsed.project_id.trim() : ''
    const clientEmail = typeof parsed.client_email === 'string' ? parsed.client_email.trim() : ''
    const privateKey = typeof parsed.private_key === 'string' ? parsed.private_key.trim() : ''
    if (!projectId || !clientEmail || !privateKey) {
      appStore.showError(t('admin.providers.vertexSaJsonMissingFields'))
      return false
    }
    vertexProjectId.value = projectId
    vertexClientEmail.value = clientEmail
    vertexServiceAccountJson.value = JSON.stringify(parsed)
    return true
  } catch {
    appStore.showError(t('admin.providers.vertexSaJsonInvalid'))
    return false
  }
}

const parseVertexServiceAccountJson = () => applyVertexServiceAccountJson(vertexServiceAccountJson.value)

const handleVertexServiceAccountFile = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  try {
    applyVertexServiceAccountJson(await file.text())
  } finally {
    input.value = ''
  }
}

const handleVertexServiceAccountDrop = async (event: DragEvent) => {
  vertexServiceAccountDragActive.value = false
  const file = event.dataTransfer?.files?.[0]
  if (!file) return
  applyVertexServiceAccountJson(await file.text())
}

const handleSubmit = async () => {
  if (form.platform === 'opencode_go' && !validOpenCodeGoProtocolRules(openCodeRules.value)) {
    appStore.showError(t('admin.accounts.opencodeGo.protocolRules.invalid'))
    return
  }
  // For OAuth-based type, handle OAuth flow (goes to step 2)
  if (isOAuthFlow.value) {
    if (submitting.value) return
    if (!isGrokSSOInputMethod.value && !form.name.trim()) {
      appStore.showError(t('admin.providers.pleaseEnterProviderName'))
      return
    }
    step.value = 2
    return
  }

  // For Bedrock type, create directly
  if (form.platform === 'anthropic' && providerCategory.value === 'bedrock') {
    if (!form.name.trim()) {
      appStore.showError(t('admin.providers.pleaseEnterProviderName'))
      return
    }

    const credentials: Record<string, unknown> = {
      auth_mode: bedrockAuthMode.value,
      aws_region: bedrockRegion.value.trim() || 'us-east-1',
    }

    if (bedrockAuthMode.value === 'sigv4') {
      if (!bedrockAccessKeyId.value.trim()) {
        appStore.showError(t('admin.providers.bedrockAccessKeyIdRequired'))
        return
      }
      if (!bedrockSecretAccessKey.value.trim()) {
        appStore.showError(t('admin.providers.bedrockSecretAccessKeyRequired'))
        return
      }
      credentials.aws_access_key_id = bedrockAccessKeyId.value.trim()
      credentials.aws_secret_access_key = bedrockSecretAccessKey.value.trim()
      if (bedrockSessionToken.value.trim()) {
        credentials.aws_session_token = bedrockSessionToken.value.trim()
      }
    } else {
      if (!bedrockApiKeyValue.value.trim()) {
        appStore.showError(t('admin.providers.bedrockApiKeyRequired'))
        return
      }
      credentials.api_key = bedrockApiKeyValue.value.trim()
    }

    if (bedrockForceGlobal.value) {
      credentials.aws_force_global = 'true'
    }

    // Model restriction
    applyPersistedModelRestriction(credentials)

    // Pool mode
    if (poolModeEnabled.value) {
      credentials.pool_mode = true
      credentials.pool_mode_retry_count = normalizePoolModeRetryCount(poolModeRetryCount.value)
      const parsedRetryStatusCodes = parsePoolModeRetryStatusCodes(poolModeRetryStatusCodesInput.value)
      if (parsedRetryStatusCodes.length > 0) {
        credentials.pool_mode_retry_status_codes = parsedRetryStatusCodes
      }
    }

    applyInterceptWarmup(credentials, interceptWarmupRequests.value, 'create')

    await createProviderAndFinish('anthropic', 'bedrock' as ProviderType, credentials)
    return
  }

  // For Antigravity upstream type, create directly
  if (form.platform === 'antigravity' && antigravityProviderType.value === 'upstream') {
    if (!form.name.trim()) {
      appStore.showError(t('admin.providers.pleaseEnterProviderName'))
      return
    }
    if (!upstreamBaseUrl.value.trim()) {
      appStore.showError(t('admin.providers.upstream.pleaseEnterBaseUrl'))
      return
    }
    if (!upstreamApiKey.value.trim()) {
      appStore.showError(t('admin.providers.upstream.pleaseEnterApiKey'))
      return
    }

    // Build upstream credentials (and optional model restriction)
    const credentials: Record<string, unknown> = {
      base_url: upstreamBaseUrl.value.trim(),
      api_key: upstreamApiKey.value.trim()
    }

    // Antigravity 只使用映射模式
    const antigravityModelMapping = buildModelMappingObject(
      'mapping',
      [],
      antigravityModelMappings.value
    )
    if (antigravityModelMapping) {
      credentials.model_mapping = antigravityModelMapping
    }
    credentials.model_whitelist = [...antigravityWhitelistModels.value]

    applyInterceptWarmup(credentials, interceptWarmupRequests.value, 'create')

    const extra = buildAntigravityExtra()
    await createProviderAndFinish(form.platform, 'apikey', credentials, extra)
    return
  }

  if (form.platform === 'qoder' && qoderProviderType.value === 'manual') {
    if (!form.name.trim()) {
      appStore.showError(t('admin.providers.pleaseEnterProviderName'))
      return
    }
    const credentials: Record<string, unknown> = {
      site: qoderSite.value,
      refresh_mode: 'cosy'
    }
    if (qoderPAT.value.trim()) {
      credentials.pat = qoderPAT.value.trim()
    } else {
      if (!qoderSecurityOauthToken.value.trim()) {
        appStore.showError(t('admin.providers.qoder.pleaseEnterSecurityOauthToken'))
        return
      }
      if (!qoderMachineId.value.trim()) {
        appStore.showError(t('admin.providers.qoder.pleaseEnterMachineId'))
        return
      }
      if (!qoderUidAid.value.trim()) {
        appStore.showError(t('admin.providers.qoder.pleaseEnterUidAid'))
        return
      }

      const uidAid = qoderUidAid.value.trim()
      credentials.security_oauth_token = qoderSecurityOauthToken.value.trim()
      credentials.machine_id = qoderMachineId.value.trim()
      credentials.uid = uidAid
      credentials.aid = uidAid
      credentials.user_type = qoderUserType.value.trim() || 'personal_standard'
      if (qoderRefreshToken.value.trim()) {
        credentials.refresh_token = qoderRefreshToken.value.trim()
      }
    }
    applyQoderModelRestriction(credentials)
    applyInterceptWarmup(credentials, interceptWarmupRequests.value, 'create')
    await createProviderAndFinish('qoder', 'cosy', credentials, buildQoderExtra())
    return
  }

  if ((form.platform === 'gemini' || form.platform === 'anthropic') && providerCategory.value === 'service_account') {
    if (!form.name.trim()) {
      appStore.showError(t('admin.providers.pleaseEnterProviderName'))
      return
    }
    if (!parseVertexServiceAccountJson()) {
      return
    }
    if (!vertexLocation.value.trim()) {
      appStore.showError(t('admin.providers.vertexLocationRequired'))
      return
    }
    const credentials: Record<string, unknown> = {
      service_account_json: vertexServiceAccountJson.value.trim(),
      project_id: vertexProjectId.value.trim(),
      client_email: vertexClientEmail.value.trim(),
      location: vertexLocation.value.trim(),
      tier_id: 'vertex'
    }
    await createProviderAndFinish(form.platform, 'service_account' as ProviderType, credentials)
    return
  }

  // For apikey type, create directly
  if (!apiKeyValue.value.trim()) {
    appStore.showError(t('admin.providers.pleaseEnterApiKey'))
    return
  }

  const enteredBaseUrl = apiKeyBaseUrl.value.trim()
  if (
    form.platform === 'gemini' &&
    geminiProviderType.value === 'third_party' &&
    !isGeminiThirdPartyBaseUrl(enteredBaseUrl)
  ) {
    appStore.showError(t('admin.providers.gemini.connectionSource.thirdPartyBaseUrlRequired'))
    return
  }

  // Determine default base URL based on platform
  const defaultBaseUrl =
    form.platform === 'openai'
      ? 'https://api.openai.com'
      : form.platform === 'gemini'
        ? 'https://generativelanguage.googleapis.com'
        : form.platform === 'grok'
          ? 'https://api.x.ai/v1'
          : 'https://api.anthropic.com'

  // Build credentials with optional model mapping
  const credentials: Record<string, unknown> = {
    base_url: enteredBaseUrl || defaultBaseUrl,
    api_key: apiKeyValue.value.trim()
  }
  // New API 钱包是用户级余额；访问令牌只写入 Credentials，不进入 Extra。
  if (upstreamUsageAdapter.value === 'new_api') {
    if (upstreamUsageWalletAccessToken.value.trim()) {
      credentials.new_api_user_access_token = upstreamUsageWalletAccessToken.value.trim()
    }
    if (upstreamUsageWalletUserId.value.trim()) {
      credentials.new_api_user_id = upstreamUsageWalletUserId.value.trim()
    }
  }
  if (form.platform === 'gemini') {
    credentials.provider_type = geminiProviderType.value
    if (geminiProviderType.value === 'official') {
      credentials.tier_id = geminiTierAIStudio.value
    }
  }

  // 国产供应商：提供商模式 + 协议 + 对应端点写入凭据；后端按 provider_mode 路由
  // 额度/余额探测，按 api_protocol 路由转发端点与格式。注意 CN apikey 走本函数
  // 的通用路径（直接 doCreateProvider），不经过 createProviderAndFinish。
  if (form.platform === 'kimi' || form.platform === 'zhipu' || form.platform === 'deepseek' || form.platform === 'minimax' || form.platform === 'opencode_go') {
    credentials.provider_mode = providerMode.value
    credentials.api_protocol = apiProtocol.value
    if (form.platform === 'opencode_go') applyOpenCodeGoProtocolRules(credentials, openCodeRules.value, 'create')
    if (apiProtocol.value === 'adaptive') {
      const defaults = defaultCNAdaptiveBaseUrls(form.platform, providerMode.value)
      const protocolBaseUrls: Record<string, string> = {}
      for (const item of cnAdaptiveProtocolOptions.value) {
        protocolBaseUrls[item.value] = (adaptiveBaseUrls.value[item.value] || defaults[item.value]).trim()
      }
      credentials.api_base_urls = protocolBaseUrls
      credentials.base_url = protocolBaseUrls.chat_completions
    }
    const resolvedCNBase = (
      apiKeyBaseUrl.value.trim() || defaultCNBaseUrl(form.platform, providerMode.value, apiProtocol.value)
    ).trim()
    if (apiProtocol.value !== 'adaptive' && resolvedCNBase) {
      credentials.base_url = resolvedCNBase
    }
    if (form.platform === 'zhipu' && providerMode.value === 'coding') {
      const organization = zhipuOrganization.value.trim()
      const project = zhipuProject.value.trim()
      if (organization) {
        credentials.zhipu_organization = organization
        if (project) credentials.zhipu_project = project
      }
    }
  }

  // Add model mapping if configured（OpenAI 开启自动透传时不应用）
  if (true) {
    applyPersistedModelRestriction(credentials)
  }
  if (form.platform === 'openai') {
    const compactModelMapping = buildOpenAICompactModelMapping()
    if (compactModelMapping) {
      credentials.compact_model_mapping = compactModelMapping
    }
  }

  // Add pool mode if enabled
  if (poolModeEnabled.value) {
    credentials.pool_mode = true
    credentials.pool_mode_retry_count = normalizePoolModeRetryCount(poolModeRetryCount.value)
    const parsedRetryStatusCodes = parsePoolModeRetryStatusCodes(poolModeRetryStatusCodesInput.value)
    if (parsedRetryStatusCodes.length > 0) {
      credentials.pool_mode_retry_status_codes = parsedRetryStatusCodes
    }
  }

  // Add custom error codes if enabled
  if (customErrorCodesEnabled.value) {
    credentials.custom_error_codes_enabled = true
    credentials.custom_error_codes = [...selectedErrorCodes.value]
  }

  // 为支持该功能的平台 API Key 提供商写入请求头覆写。
  if (isHeaderOverrideCapable(form.platform, 'apikey')) {
    if (headerOverrideEnabled.value) {
      const headerError = validateHeaderOverrideRows(headerOverrideRows.value)
      if (headerError) {
        appStore.showError(t(`admin.providers.headerOverride.${headerError}`))
        return
      }
    }
    applyHeaderOverride(credentials, headerOverrideEnabled.value, headerOverrideRows.value, 'create')
  }

  applyInterceptWarmup(credentials, interceptWarmupRequests.value, 'create')

  form.credentials = credentials
  const extra = buildAnthropicExtra(buildOpenAIExtra())

  // API Key 统一经过构造器，确保配额和上游用量查询配置一起写入请求。
  await createProviderAndFinish(form.platform, 'apikey', credentials, extra)
}

const goBackToBasicInfo = () => {
  stopQoderPolling()
  closeQoderAuthPopup()
  resetQoderOAuthCompletionState()
  step.value = 1
  oauth.resetState()
  openaiOAuth.resetState()
  geminiOAuth.resetState()
  antigravityOAuth.resetState()
  qoderOAuth.resetState()
  grokOAuth.resetState()
  oauthFlowRef.value?.reset()
}

const getQoderPopupFeatures = () => {
  const width = Math.min(1100, (window.screen?.availWidth || 1100) - 40)
  const height = Math.min(820, (window.screen?.availHeight || 820) - 40)
  const left = Math.max(0, Math.floor(((window.screen?.availWidth || width) - width) / 2))
  const top = Math.max(0, Math.floor(((window.screen?.availHeight || height) - height) / 2))
  return `width=${width},height=${height},left=${left},top=${top},scrollbars=yes,resizable=yes`
}

const stopQoderPolling = () => {
  qoderPollGeneration += 1
  qoderOAuth.invalidatePendingRequests()
  if (qoderPollTimer) {
    window.clearInterval(qoderPollTimer)
    qoderPollTimer = null
  }
  qoderPollInFlight = false
}

const closeQoderAuthPopup = (lease?: QoderAuthPopupLease) => {
  const currentLease = qoderAuthPopupLease
  if (!currentLease || (lease && currentLease.generation !== lease.generation)) return

  // 先解绑再关闭，旧请求恢复执行时只能处理自己仍持有的弹窗。
  qoderAuthPopupLease = null
  currentLease.popup?.close()
}

const resetQoderOAuthCompletionState = () => {
  // 流程代数与轮询代数分离，停止一次轮询不会误伤同一授权流程的兑换或创建。
  qoderFlowGeneration.value += 1
  qoderOAuthCompleted = false
}

interface QoderFlowContext {
  generation: number
  site: QoderSite
}

const captureQoderFlowContext = (): QoderFlowContext => ({
  generation: qoderFlowGeneration.value,
  site: qoderSite.value
})

const isCurrentQoderFlow = (context: QoderFlowContext) =>
  context.generation === qoderFlowGeneration.value &&
  context.site === qoderSite.value &&
  form.platform === 'qoder' &&
  props.show

const createQoderOAuthProvider = async (
  tokenInfo: QoderTokenInfo | undefined,
  context: QoderFlowContext
): Promise<boolean> => {
  if (
    !tokenInfo ||
    !isCurrentQoderFlow(context) ||
    qoderProviderCreateGeneration.value !== null ||
    qoderOAuthCompleted
  ) return false

  qoderProviderCreateGeneration.value = context.generation
  try {
    if (!isCurrentQoderFlow(context)) return false
    const credentials = qoderOAuth.buildCredentials(tokenInfo)
    applyQoderModelRestriction(credentials)
    applyInterceptWarmup(credentials, interceptWarmupRequests.value, 'create')
    const created = await createProviderAndFinish(
      'qoder',
      'cosy',
      credentials,
      buildQoderExtra(),
      () => isCurrentQoderFlow(context)
    )
    if (created && isCurrentQoderFlow(context)) {
      qoderOAuthCompleted = true
    }
    return created
  } finally {
    // 只释放自己持有的锁，旧流程的 finally 不能解除新流程的创建锁。
    if (qoderProviderCreateGeneration.value === context.generation) {
      qoderProviderCreateGeneration.value = null
      submitting.value = false
    }
  }
}

const pollQoderAuthorizationOnce = async () => {
  if (qoderPollInFlight || qoderOAuthCompleted || !qoderOAuth.sessionId.value || !qoderOAuth.state.value)
    return
  const generation = qoderPollGeneration
  const flowContext = captureQoderFlowContext()
  const sessionId = qoderOAuth.sessionId.value
  const state = qoderOAuth.state.value
  if (!isCurrentQoderFlow(flowContext)) return
  qoderPollInFlight = true

  try {
    const result = await qoderOAuth.pollAuthorization({
      sessionId,
      state
    })
    if (generation !== qoderPollGeneration || !isCurrentQoderFlow(flowContext)) return
    if (!result && qoderOAuth.error.value) {
      stopQoderPolling()
      return
    }
    if (result?.status !== 'completed' || !result.token_info) return

    stopQoderPolling()
    closeQoderAuthPopup()
    await createQoderOAuthProvider(result.token_info, flowContext)
  } finally {
    if (generation === qoderPollGeneration) {
      qoderPollInFlight = false
    }
  }
}

const startQoderPolling = (intervalSeconds = 2) => {
  stopQoderPolling()
  const generation = qoderPollGeneration
  void pollQoderAuthorizationOnce()
  const intervalMs = Math.max(1, intervalSeconds) * 1000
  qoderPollTimer = window.setInterval(() => {
    if (generation !== qoderPollGeneration) return
    void pollQoderAuthorizationOnce()
  }, intervalMs)
}

const handleGenerateUrl = async () => {
  if (form.platform === 'openai') {
    await openaiOAuth.appendAuthUrl(form.proxy_id)
  } else if (form.platform === 'gemini') {
    await geminiOAuth.generateAuthUrl(
      form.proxy_id,
      oauthFlowRef.value?.projectId,
      geminiOAuthType.value,
      geminiSelectedTier.value
    )
  } else if (form.platform === 'antigravity') {
    await antigravityOAuth.generateAuthUrl(form.proxy_id)
  } else if (form.platform === 'qoder') {
    // 创建请求不可取消；完成前禁止重置授权世代，否则成功响应会被当作旧流程丢弃。
    if (submitting.value || isQoderOAuthProviderCreating.value) return
    stopQoderPolling()
    closeQoderAuthPopup()
    resetQoderOAuthCompletionState()
    const flowContext = captureQoderFlowContext()
    const authPopup = window.open('about:blank', 'qoderAuthPopup', getQoderPopupFeatures())
    const popupLease: QoderAuthPopupLease = {
      generation: ++qoderAuthPopupGeneration,
      popup: authPopup
    }
    qoderAuthPopupLease = popupLease
    const ok = await qoderOAuth.generateAuthUrl(form.proxy_id, flowContext.site)
    if (!isCurrentQoderFlow(flowContext)) {
      closeQoderAuthPopup(popupLease)
      return
    }
    if (!ok) {
      closeQoderAuthPopup(popupLease)
      return
    }
    if (authPopup) {
      authPopup.location.href = qoderOAuth.authUrl.value
      authPopup.focus()
    } else {
      appStore.showWarning(t('admin.providers.oauth.qoder.popupBlocked'))
    }
    startQoderPolling(qoderOAuth.pollInterval.value)
  } else if (form.platform === 'grok') {
    await grokOAuth.generateAuthUrl(form.proxy_id)
  } else {
    await oauth.generateAuthUrl(addMethod.value, form.proxy_id)
  }
}

const handleRemoveOpenAIAuthSession = (sessionId: string) => {
  openaiOAuth.removeAuthSession(sessionId)
}

const handleValidateRefreshToken = (rt: string) => {
  if (form.platform === 'openai') {
    handleOpenAIValidateRT(rt)
  } else if (form.platform === 'antigravity') {
    handleAntigravityValidateRT(rt)
  } else if (form.platform === 'grok') {
    handleGrokValidateRT(rt)
  }
}

const handleValidateSessionToken = (_sessionToken: string) => {
  // Session token validation removed
}

const formatDateTimeLocal = formatDateTimeLocalInput
const parseDateTimeLocal = parseDateTimeLocalInput

// Create provider and handle success/failure
const createProviderAndFinish = async (
  platform: ProviderPlatform,
  type: ProviderType,
  credentials: Record<string, unknown>,
  extra?: Record<string, unknown>,
  isCurrent: ProviderCreateGuard = () => true
): Promise<boolean> => {
  if (!applyTempUnschedConfig(credentials)) {
    return false
  }
  // Inject quota limits for apikey/bedrock providers
  let finalExtra = withUpstreamRequestIdHeader(extra)
  if (type === 'apikey' || type === 'bedrock') {
    const quotaExtra: Record<string, unknown> = { ...(finalExtra || {}) }
    if (editQuotaLimit.value != null && editQuotaLimit.value > 0) {
      quotaExtra.quota_limit = editQuotaLimit.value
    }
    if (editQuotaDailyLimit.value != null && editQuotaDailyLimit.value > 0) {
      quotaExtra.quota_daily_limit = editQuotaDailyLimit.value
    }
    if (editQuotaWeeklyLimit.value != null && editQuotaWeeklyLimit.value > 0) {
      quotaExtra.quota_weekly_limit = editQuotaWeeklyLimit.value
    }
    // Quota reset mode config
    if (editDailyResetMode.value === 'fixed') {
      quotaExtra.quota_daily_reset_mode = 'fixed'
      quotaExtra.quota_daily_reset_hour = editDailyResetHour.value ?? 0
    }
    if (editWeeklyResetMode.value === 'fixed') {
      quotaExtra.quota_weekly_reset_mode = 'fixed'
      quotaExtra.quota_weekly_reset_day = editWeeklyResetDay.value ?? 1
      quotaExtra.quota_weekly_reset_hour = editWeeklyResetHour.value ?? 0
    }
    if (editDailyResetMode.value === 'fixed' || editWeeklyResetMode.value === 'fixed') {
      quotaExtra.quota_reset_timezone = editResetTimezone.value || 'UTC'
    }
    // Quota notify config
    writeQuotaNotifyToExtra(quotaExtra, 'create')
    if (type === 'apikey') {
      const upstreamConfig: Record<string, unknown> = {
        enabled: upstreamUsageEnabled.value,
        adapter: upstreamUsageAdapter.value
      }
      if (upstreamUsageBaseUrl.value.trim()) {
        upstreamConfig.base_url = upstreamUsageBaseUrl.value.trim()
      }
      quotaExtra.upstream_usage_query = upstreamConfig
    }
    if (Object.keys(quotaExtra).length > 0) {
      finalExtra = quotaExtra
    }
  }
  if (platform === 'openai') {
    const compactModelMapping = buildOpenAICompactModelMapping()
    if (compactModelMapping) {
      credentials.compact_model_mapping = compactModelMapping
    } else {
      delete credentials.compact_model_mapping
    }
  }
  if (platform === 'grok') {
    if (!credentials.base_url) {
      credentials.base_url = apiKeyBaseUrl.value.trim() || 'https://api.x.ai/v1'
    }
    applyPersistedModelRestriction(credentials)
  }
  if (!isCurrent()) return false
  return doCreateProvider({
    name: form.name,
    notes: form.notes,
    platform,
    type,
    credentials,
    extra: finalExtra,
    proxy_id: form.proxy_id,
    concurrency: form.concurrency,
    load_factor: form.load_factor ?? undefined,
    priority: form.priority,
    rate_multiplier: form.rate_multiplier,
    group_ids: form.group_ids,
    expires_at: form.expires_at,
    auto_pause_on_expired: autoPauseOnExpired.value
  }, isCurrent)
}

interface OpenAIAuthCodeEntry {
  lineNumber: number
  code: string
  state: string
}

const extractOpenAIAuthParam = (value: string, param: 'code' | 'state') => {
  try {
    const parsed = new URL(value)
    return (parsed.searchParams.get(param) || '').trim()
  } catch {
    const match = value.match(new RegExp(`(?:^|[?&])${param}=([^&#\\s]+)`))
    if (!match?.[1]) {
      return ''
    }
    try {
      return decodeURIComponent(match[1].replace(/\+/g, ' ')).trim()
    } catch {
      return match[1].trim()
    }
  }
}

const parseOpenAIAuthCodeEntries = (input: string): OpenAIAuthCodeEntry[] => {
  return input
    .split(/\r?\n/)
    .map((line, index) => {
      const trimmed = line.trim()
      const code = extractOpenAIAuthParam(trimmed, 'code')
      const state = extractOpenAIAuthParam(trimmed, 'state')
      const hasOAuthParam = /(?:^|[?&])(?:code|state)=/.test(trimmed)
      const looksLikeURL = /^[a-z][a-z\d+.-]*:\/\//i.test(trimmed)
      const plainCode = !hasOAuthParam && !looksLikeURL ? trimmed : ''
      return {
        lineNumber: index + 1,
        code: code || plainCode,
        state
      }
    })
    .filter((entry) => entry.code || entry.state)
}

const getOpenAIAuthSessions = (): OpenAIOAuthSession[] => {
  if (openaiOAuth.authSessions.value.length > 0) {
    return [...openaiOAuth.authSessions.value]
  }
  if (!openaiOAuth.sessionId.value) {
    return []
  }
  return [{
    authUrl: openaiOAuth.authUrl.value,
    sessionId: openaiOAuth.sessionId.value,
    state: openaiOAuth.oauthState.value
  }]
}

const findOpenAIAuthSession = (
  entry: OpenAIAuthCodeEntry,
  sessions: OpenAIOAuthSession[],
  usedSessionIds: Set<string>
) => {
  if (entry.state) {
    return sessions.find((session) =>
      session.state === entry.state && !usedSessionIds.has(session.sessionId)
    ) || null
  }
  return sessions.find((session) => !usedSessionIds.has(session.sessionId)) || null
}

const ensureOpenAITempUnschedConfigReady = () => {
  if (!tempUnschedEnabled.value) {
    return true
  }
  if (buildTempUnschedRules(tempUnschedRules.value).length > 0) {
    return true
  }
  const message = t('admin.providers.tempUnschedulable.rulesInvalid')
  openaiOAuth.error.value = message
  appStore.showError(message)
  return false
}

const buildOpenAIOAuthProviderRequest = (
  tokenInfo: OpenAITokenInfo,
  providerName: string,
  clientId?: string
): CreateProviderRequest | null => {
  const credentials = openaiOAuth.buildCredentials(tokenInfo)
  if (clientId) {
    credentials.client_id = clientId
  }
  const oauthExtra = openaiOAuth.buildExtraInfo(tokenInfo) as Record<string, unknown> | undefined
  const extra = buildOpenAIExtra(oauthExtra)

  applyOpenAIOAuthCredentialDefaults(credentials)
  // OpenAI OAuth 透传模式下不应用模型限制。
  if (true) {
    applyPersistedModelRestriction(credentials)
  }
  const compactModelMapping = buildOpenAICompactModelMapping()
  if (compactModelMapping) {
    credentials.compact_model_mapping = compactModelMapping
  }
  if (!applyTempUnschedConfig(credentials)) {
    openaiOAuth.error.value = t('admin.providers.tempUnschedulable.rulesInvalid')
    return null
  }

  return {
    name: providerName,
    notes: form.notes,
    platform: 'openai',
    type: 'oauth',
    credentials,
    extra,
    proxy_id: form.proxy_id,
    concurrency: form.concurrency,
    load_factor: form.load_factor ?? undefined,
    priority: form.priority,
    rate_multiplier: form.rate_multiplier,
    group_ids: form.group_ids,
    expires_at: form.expires_at,
    auto_pause_on_expired: autoPauseOnExpired.value
  }
}

const createOpenAIOAuthProviderFromToken = async (
  tokenInfo: OpenAITokenInfo,
  providerName: string,
  clientId?: string
) => {
  const payload = buildOpenAIOAuthProviderRequest(tokenInfo, providerName, clientId)
  if (!payload) {
    return false
  }
  await createProtocolProvider(payload)
  return true
}

const buildOpenAIOAuthProviderName = (
  tokenInfo: OpenAITokenInfo,
  index: number,
  total: number
) => {
  const baseName = form.name || tokenInfo.email || 'OpenAI OAuth Provider'
  return total > 1 ? `${baseName} #${index + 1}` : baseName
}

// OpenAI OAuth token 请求只在启用 TLS 指纹路由器时携带路由器 ID。
const selectedOpenAITokenTLSRouterId = () => {
  return tlsFingerprintEnabled.value ? tlsFingerprintRouterId.value : null
}

// Grok 手动 RT 批量验证和创建
const handleGrokValidateRT = async (refreshTokenInput: string) => {
  if (!refreshTokenInput.trim()) return

  const refreshTokens = refreshTokenInput
    .split('\n')
    .map((rt) => rt.trim())
    .filter((rt) => rt)

  if (refreshTokens.length === 0) {
    grokOAuth.error.value = t('admin.providers.oauth.grok.pleaseEnterRefreshToken')
    return
  }
  if (!validateGrokOAuthUpstreamConfig()) return

  grokOAuth.loading.value = true
  grokOAuth.error.value = ''

  let successCount = 0
  let failedCount = 0
  const errors: string[] = []

  try {
    for (let i = 0; i < refreshTokens.length; i++) {
      try {
        const tokenInfo = await grokOAuth.validateRefreshToken(refreshTokens[i], form.proxy_id)
        if (!tokenInfo) {
          failedCount++
          errors.push(`#${i + 1}: ${grokOAuth.error.value || 'Validation failed'}`)
          grokOAuth.error.value = ''
          continue
        }

        const credentials = grokOAuth.buildCredentials(tokenInfo)
        applyGrokOAuthUpstreamConfig(credentials)
        const extra = grokOAuth.buildExtraInfo(tokenInfo)
        const providerName = refreshTokens.length > 1 ? `${form.name || tokenInfo.email || 'Grok OAuth Provider'} #${i + 1}` : (form.name || tokenInfo.email || 'Grok OAuth Provider')

        applyPersistedModelRestriction(credentials)
        if (!applyTempUnschedConfig(credentials)) {
          failedCount++
          errors.push(`#${i + 1}: ${t('admin.providers.tempUnschedulable.rulesInvalid')}`)
          continue
        }

        await createProtocolProvider({
          name: providerName,
          notes: form.notes,
          platform: 'grok',
          type: 'oauth',
          credentials,
          extra: withUpstreamRequestIdHeader(extra),
          proxy_id: form.proxy_id,
          concurrency: form.concurrency,
          load_factor: form.load_factor ?? undefined,
          priority: form.priority,
          rate_multiplier: form.rate_multiplier,
          group_ids: form.group_ids,
          expires_at: form.expires_at,
          auto_pause_on_expired: autoPauseOnExpired.value
        })
        successCount++
      } catch (error: any) {
        failedCount++
        const errMsg = error.response?.data?.detail || error.message || 'Unknown error'
        errors.push(`#${i + 1}: ${errMsg}`)
      }
    }

    if (successCount > 0 && failedCount === 0) {
      appStore.showSuccess(
        refreshTokens.length > 1
          ? t('admin.providers.oauth.batchSuccess', { count: successCount })
          : t('admin.providers.providerCreated')
      )
      emit('created')
      handleClose()
    } else if (successCount > 0) {
      appStore.showWarning(t('admin.providers.oauth.batchPartialSuccess', { success: successCount, failed: failedCount }))
      grokOAuth.error.value = errors.join('\n')
      emit('created')
    } else {
      grokOAuth.error.value = errors.join('\n')
      appStore.showError(t('admin.providers.oauth.batchFailed'))
    }
  } finally {
    grokOAuth.loading.value = false
  }
}

const handleGrokImportSSO = async (ssoInput: string) => {
  // 与 OpenAI/Grok RT 批量导入保持一致：每行一个令牌，前端不去重。
  const ssoTokens = ssoInput
    .split('\n')
    .map((token) => token.trim())
    .filter((token) => token)
  if (ssoTokens.length === 0) return
  if (!validateGrokOAuthUpstreamConfig()) return

  grokOAuth.loading.value = true
  grokOAuth.error.value = ''

  const credentials: Record<string, unknown> = {}
  applyGrokOAuthUpstreamConfig(credentials)
  applyPersistedModelRestriction(credentials)
  if (!applyTempUnschedConfig(credentials)) {
    grokOAuth.loading.value = false
    return
  }

  try {
    const result = await adminAPI.grok.createFromSSO({
      sso_tokens: ssoTokens,
      name: form.name || undefined,
      notes: form.notes || undefined,
      proxy_id: form.proxy_id,
      group_ids: form.group_ids,
      credentials,
      concurrency: form.concurrency,
      load_factor: form.load_factor ?? undefined,
      priority: form.priority,
      rate_multiplier: form.rate_multiplier,
      expires_at: form.expires_at,
      auto_pause_on_expired: autoPauseOnExpired.value
    })

    const successCount = result.created?.length || 0
    const failedCount = result.failed?.length || 0
    if (successCount > 0 && failedCount === 0) {
      appStore.showSuccess(
        ssoTokens.length > 1
          ? t('admin.providers.oauth.batchSuccess', { count: successCount })
          : t('admin.providers.providerCreated')
      )
      emit('created')
      handleClose()
    } else if (successCount > 0 && failedCount > 0) {
      // 与 OpenAI/Grok RT 一致：保留输入、显示失败项并刷新列表。
      appStore.showWarning(
        t('admin.providers.oauth.batchPartialSuccess', { success: successCount, failed: failedCount })
      )
      grokOAuth.error.value = (result.failed || [])
        .map((item) => `#${item.index}: ${item.error || 'Unknown error'}`)
        .join('\n')
      emit('created')
    } else {
      grokOAuth.error.value = (result.failed || [])
        .map((item) => `#${item.index}: ${item.error || 'Unknown error'}`)
        .join('\n') || t('admin.providers.oauth.grok.failedToConvertSSO')
      appStore.showError(t('admin.providers.oauth.batchFailed'))
    }
  } catch (error: any) {
    grokOAuth.error.value = error.response?.data?.detail || error.message || t('admin.providers.oauth.grok.failedToConvertSSO')
    appStore.showError(grokOAuth.error.value)
  } finally {
    grokOAuth.loading.value = false
  }
}

// OpenAI OAuth 授权码批量兑换和创建
const handleOpenAIExchange = async (authCodeInput: string) => {
  const oauthClient = openaiOAuth
  if (!authCodeInput.trim()) return

  const entries = parseOpenAIAuthCodeEntries(authCodeInput)
  if (entries.length === 0) {
    oauthClient.error.value = t('admin.providers.oauth.openai.pleaseEnterAuthCode')
    return
  }

  const sessions = getOpenAIAuthSessions()
  if (sessions.length === 0) {
    oauthClient.error.value = t('admin.providers.oauth.openai.pleaseGenerateAuthUrl')
    appStore.showError(oauthClient.error.value)
    return
  }
  if (!ensureOpenAITempUnschedConfigReady()) {
    return
  }

  oauthClient.loading.value = true
  oauthClient.error.value = ''

  let successCount = 0
  let failedCount = 0
  const errors: string[] = []
  const usedSessionIds = new Set<string>()

  try {
    await loadOpenAIOAuthImportDefaults()

    for (let i = 0; i < entries.length; i++) {
      const entry = entries[i]
      const session = findOpenAIAuthSession(entry, sessions, usedSessionIds)
      if (!entry.code) {
        failedCount++
        errors.push(`#${entry.lineNumber}: ${t('admin.providers.oauth.openai.pleaseEnterAuthCode')}`)
        continue
      }
      if (!session) {
        failedCount++
        errors.push(`#${entry.lineNumber}: ${t('admin.providers.oauth.openai.noMatchingAuthUrl')}`)
        continue
      }

      usedSessionIds.add(session.sessionId)
      const stateToUse = entry.state || session.state
      if (!stateToUse) {
        failedCount++
        errors.push(`#${entry.lineNumber}: ${t('admin.providers.oauth.openai.missingState')}`)
        continue
      }

      try {
        const tokenInfo = await oauthClient.exchangeAuthCode(
          entry.code,
          session.sessionId,
          stateToUse,
          form.proxy_id,
          selectedOpenAITokenTLSRouterId()
        )
        oauthClient.loading.value = true
        if (!tokenInfo) {
          failedCount++
          errors.push(`#${entry.lineNumber}: ${oauthClient.error.value || t('admin.providers.oauth.openai.failedToExchangeCode')}`)
          oauthClient.error.value = ''
          continue
        }

        oauthClient.removeAuthSession(session.sessionId)
        const providerName = buildOpenAIOAuthProviderName(tokenInfo, i, entries.length)
        const created = await createOpenAIOAuthProviderFromToken(tokenInfo, providerName)
        if (!created) {
          failedCount++
          errors.push(`#${entry.lineNumber}: ${oauthClient.error.value || t('admin.providers.failedToCreate')}`)
          oauthClient.error.value = ''
          continue
        }

        successCount++
      } catch (error: any) {
        failedCount++
        const errMsg = error.response?.data?.detail || error.message || t('admin.providers.oauth.authFailed')
        errors.push(`#${entry.lineNumber}: ${errMsg}`)
      }
    }

    if (successCount > 0 && failedCount === 0) {
      appStore.showSuccess(
        entries.length > 1
          ? t('admin.providers.oauth.batchSuccess', { count: successCount })
          : t('admin.providers.providerCreated')
      )
      emit('created')
      handleClose()
    } else if (successCount > 0 && failedCount > 0) {
      appStore.showWarning(
        t('admin.providers.oauth.batchPartialSuccess', { success: successCount, failed: failedCount })
      )
      oauthClient.error.value = errors.join('\n')
      emit('created')
    } else {
      oauthClient.error.value = errors.join('\n')
      appStore.showError(t('admin.providers.oauth.batchFailed'))
    }
  } finally {
    oauthClient.loading.value = false
  }
}

// OpenAI 手动 RT 批量验证和创建
// OpenAI Mobile RT client_id
const OPENAI_MOBILE_RT_CLIENT_ID = 'app_LlGpXReQgckcGGUo2JrYvtJK'

const buildOpenAICodexImportCredentialExtras = (): Record<string, unknown> | null => {
  const credentials: Record<string, unknown> = {}
  if (true) {
    // 与其他 OpenAI OAuth 创建方式保持一致：映射和最终白名单分别保存。
    applyPersistedModelRestriction(credentials)
  }

  const compactModelMapping = buildOpenAICompactModelMapping()
  if (compactModelMapping) {
    credentials.compact_model_mapping = compactModelMapping
  }

  if (!applyTempUnschedConfig(credentials)) {
    return null
  }
  return credentials
}

const formatCodexImportMessages = (messages?: CodexSessionImportMessage[]) => {
  return (messages || [])
    .map((item) => {
      const name = item.name ? ` ${item.name}` : ''
      return `#${item.index}${name}: ${item.message}`
    })
    .join('\n')
}

const isAgentIdentityImportContent = (content: string) => {
  const isAgentIdentityValue = (value: unknown): boolean => {
    if (Array.isArray(value)) return value.length > 0 && value.every(isAgentIdentityValue)
    if (!value || typeof value !== 'object') return false
    const record = value as Record<string, unknown>
    const authMode = record.auth_mode ?? record.authMode
    const agentIdentity = record.agent_identity ?? record.agentIdentity
    return (typeof authMode === 'string' && authMode.toLowerCase() === 'agentidentity')
      || (!!agentIdentity && typeof agentIdentity === 'object')
  }

  try {
    return isAgentIdentityValue(JSON.parse(content))
  } catch {
    const lines = content.split('\n').map((line) => line.trim()).filter(Boolean)
    if (lines.length === 0) return false
    try {
      return lines.every((line) => isAgentIdentityValue(JSON.parse(line)))
    } catch {
      return false
    }
  }
}

const handleOpenAIImportCodexSession = async (content: string) => {
  const oauthClient = openaiOAuth
  const trimmed = content.trim()
  if (!trimmed) {
    oauthClient.error.value = t('admin.providers.oauth.openai.codexSessionEmpty')
    return
  }
  if (oauthFlowRef.value?.inputMethod === 'agent_identity' && !isAgentIdentityImportContent(trimmed)) {
    oauthClient.error.value = t('admin.providers.oauth.openai.agentIdentityInvalid')
    return
  }

  oauthClient.loading.value = true
  oauthClient.error.value = ''

  try {
    await loadOpenAIOAuthImportDefaults()
    await loadProtocolCatalog()
    // 等默认配置加载完成后再生成凭据快照，避免异步竞态覆盖管理员的模型限制。
    const credentialExtras = buildOpenAICodexImportCredentialExtras()
    if (credentialExtras === null) {
      return
    }
    const extra = buildOpenAIExtra()
    const result = await adminAPI.providers.importCodexSession({
      content: trimmed,
      name: form.name,
      notes: form.notes || null,
      proxy_id: form.proxy_id,
      concurrency: form.concurrency,
      load_factor: form.load_factor ?? undefined,
      priority: form.priority,
      rate_multiplier: form.rate_multiplier,
      group_ids: form.group_ids,
      expires_at: form.expires_at,
      auto_pause_on_expired: autoPauseOnExpired.value,
      credential_extras: { ...credentialExtras, upstream_protocols: (upstreamProtocols.value ?? nativeProtocolOptions('openai', 'oauth', 'personalAccessToken')).filter(id => nativeProtocolOptions('openai', 'oauth', 'personalAccessToken').includes(id)) },
      extra: withUpstreamRequestIdHeader(extra),
      update_existing: true
    })

    const successCount = result.created + result.updated
    const params = {
      created: result.created,
      updated: result.updated,
      skipped: result.skipped,
      failed: result.failed
    }

    if (successCount > 0 && result.failed === 0) {
      appStore.showSuccess(t('admin.providers.oauth.openai.codexSessionImportSuccess', params))
      emit('created')
      handleClose()
      return
    }

    const errorText = formatCodexImportMessages(result.errors)
    const warningText = formatCodexImportMessages(result.warnings)
    oauthClient.error.value = [errorText, warningText].filter(Boolean).join('\n')

    if (result.failed === 0) {
      appStore.showWarning(t('admin.providers.oauth.openai.codexSessionImportSuccess', params))
      return
    }

    if (successCount > 0) {
      appStore.showWarning(t('admin.providers.oauth.openai.codexSessionImportPartial', params))
      emit('created')
      return
    }

    appStore.showError(t('admin.providers.oauth.openai.codexSessionImportFailed'))
  } catch (error: any) {
    oauthClient.error.value =
      error.response?.data?.detail ||
      error.response?.data?.message ||
      error.message ||
      t('admin.providers.oauth.openai.codexSessionImportFailed')
    appStore.showError(oauthClient.error.value)
  } finally {
    oauthClient.loading.value = false
  }
}

const handleOpenAIImportCodexPAT = async (accessToken: string) => {
  const oauthClient = openaiOAuth
  const trimmed = accessToken.trim()
  if (!trimmed) {
    oauthClient.error.value = t('admin.providers.oauth.openai.codexPatEmpty')
    return
  }

  oauthClient.loading.value = true
  oauthClient.error.value = ''

  try {
    await loadOpenAIOAuthImportDefaults()
    // 等默认配置加载完成后再生成凭据快照，避免异步竞态覆盖管理员的模型限制。
    const credentialExtras = buildOpenAICodexImportCredentialExtras()
    if (credentialExtras === null) {
      return
    }
    const extra = buildOpenAIExtra()
    await adminAPI.providers.createOpenAICodexPAT({
      codex_ticket: await ticketCreationPatch(),
      access_token: trimmed,
      name: form.name,
      notes: form.notes || null,
      proxy_id: form.proxy_id,
      concurrency: form.concurrency,
      load_factor: form.load_factor ?? undefined,
      priority: form.priority,
      rate_multiplier: form.rate_multiplier,
      group_ids: form.group_ids,
      expires_at: form.expires_at,
      auto_pause_on_expired: autoPauseOnExpired.value,
      credential_extras: Object.keys(credentialExtras).length > 0 ? credentialExtras : undefined,
      extra: withUpstreamRequestIdHeader(extra)
    })
    appStore.showSuccess(t('admin.providers.providerCreated'))
    emit('created')
    handleClose()
  } catch (error: any) {
    oauthClient.error.value =
      error.response?.data?.detail ||
      error.response?.data?.message ||
      error.message ||
      t('admin.providers.oauth.openai.codexPatImportFailed')
    appStore.showError(oauthClient.error.value)
  } finally {
    oauthClient.loading.value = false
  }
}

// OpenAI RT 批量验证和创建（共享逻辑）
const handleOpenAIBatchRT = async (refreshTokenInput: string, clientId?: string) => {
  const oauthClient = openaiOAuth
  if (!refreshTokenInput.trim()) return

  const refreshTokens = refreshTokenInput
    .split('\n')
    .map((rt) => rt.trim())
    .filter((rt) => rt)

  if (refreshTokens.length === 0) {
    oauthClient.error.value = t('admin.providers.oauth.openai.pleaseEnterRefreshToken')
    return
  }
  if (!ensureOpenAITempUnschedConfigReady()) {
    return
  }

  oauthClient.loading.value = true
  oauthClient.error.value = ''

  let successCount = 0
  let failedCount = 0
  const errors: string[] = []

  try {
    await loadOpenAIOAuthImportDefaults()

    for (let i = 0; i < refreshTokens.length; i++) {
      try {
        const tokenInfo = await oauthClient.validateRefreshToken(
          refreshTokens[i],
          form.proxy_id,
          clientId,
          selectedOpenAITokenTLSRouterId()
        )
        oauthClient.loading.value = true
        if (!tokenInfo) {
          failedCount++
          errors.push(`#${i + 1}: ${oauthClient.error.value || 'Validation failed'}`)
          oauthClient.error.value = ''
          continue
        }

        const providerName = buildOpenAIOAuthProviderName(tokenInfo, i, refreshTokens.length)
        const created = await createOpenAIOAuthProviderFromToken(tokenInfo, providerName, clientId)
        if (!created) {
          failedCount++
          errors.push(`#${i + 1}: ${oauthClient.error.value || t('admin.providers.failedToCreate')}`)
          oauthClient.error.value = ''
          continue
        }

        successCount++
      } catch (error: any) {
        failedCount++
        const errMsg = error.response?.data?.detail || error.message || 'Unknown error'
        errors.push(`#${i + 1}: ${errMsg}`)
      }
    }

    // Show results
    if (successCount > 0 && failedCount === 0) {
      appStore.showSuccess(
        refreshTokens.length > 1
          ? t('admin.providers.oauth.batchSuccess', { count: successCount })
          : t('admin.providers.providerCreated')
      )
      emit('created')
      handleClose()
    } else if (successCount > 0 && failedCount > 0) {
      appStore.showWarning(
        t('admin.providers.oauth.batchPartialSuccess', { success: successCount, failed: failedCount })
      )
      oauthClient.error.value = errors.join('\n')
      emit('created')
    } else {
      oauthClient.error.value = errors.join('\n')
      appStore.showError(t('admin.providers.oauth.batchFailed'))
    }
  } finally {
    oauthClient.loading.value = false
  }
}

// 手动输入 RT（Codex CLI client_id，默认）
const handleOpenAIValidateRT = (rt: string) => handleOpenAIBatchRT(rt)

// 手动输入 Mobile RT
const handleOpenAIValidateMobileRT = (rt: string) => handleOpenAIBatchRT(rt, OPENAI_MOBILE_RT_CLIENT_ID)

// Antigravity 手动 RT 批量验证和创建
const handleAntigravityValidateRT = async (refreshTokenInput: string) => {
  if (!refreshTokenInput.trim()) return

  // Parse multiple refresh tokens (one per line)
  const refreshTokens = refreshTokenInput
    .split('\n')
    .map((rt) => rt.trim())
    .filter((rt) => rt)

  if (refreshTokens.length === 0) {
    antigravityOAuth.error.value = t('admin.providers.oauth.antigravity.pleaseEnterRefreshToken')
    return
  }

  antigravityOAuth.loading.value = true
  antigravityOAuth.error.value = ''

  let successCount = 0
  let failedCount = 0
  const errors: string[] = []

  try {
    for (let i = 0; i < refreshTokens.length; i++) {
      try {
        const tokenInfo = await antigravityOAuth.validateRefreshToken(
          refreshTokens[i],
          form.proxy_id
        )
        if (!tokenInfo) {
          failedCount++
          errors.push(`#${i + 1}: ${antigravityOAuth.error.value || 'Validation failed'}`)
          antigravityOAuth.error.value = ''
          continue
        }

        const credentials = antigravityOAuth.buildCredentials(tokenInfo, refreshTokens[i])
        applyAntigravityProjectID(credentials, antigravityProjectId.value, 'create')

        // Generate provider name with index for batch
        const providerName = refreshTokens.length > 1 ? `${form.name} #${i + 1}` : form.name

        // Note: Antigravity doesn't have buildExtraInfo, so we pass empty extra or rely on credentials
        const createPayload: CreateProviderRequest = {
          name: providerName,
          notes: form.notes,
          platform: 'antigravity',
          type: 'oauth',
          credentials,
          extra: withUpstreamRequestIdHeader({}),
          proxy_id: form.proxy_id,
          concurrency: form.concurrency,
          load_factor: form.load_factor ?? undefined,
          priority: form.priority,
          rate_multiplier: form.rate_multiplier,
          group_ids: form.group_ids,
          expires_at: form.expires_at,
          auto_pause_on_expired: autoPauseOnExpired.value
        }
        await createProtocolProvider(createPayload)
        successCount++
      } catch (error: any) {
        failedCount++
        const errMsg = error.response?.data?.detail || error.message || 'Unknown error'
        errors.push(`#${i + 1}: ${errMsg}`)
      }
    }

    // Show results
    if (successCount > 0 && failedCount === 0) {
      appStore.showSuccess(
        refreshTokens.length > 1
          ? t('admin.providers.oauth.batchSuccess', { count: successCount })
          : t('admin.providers.providerCreated')
      )
      emit('created')
      handleClose()
    } else if (successCount > 0 && failedCount > 0) {
      appStore.showWarning(
        t('admin.providers.oauth.batchPartialSuccess', { success: successCount, failed: failedCount })
      )
      antigravityOAuth.error.value = errors.join('\n')
      emit('created')
    } else {
      antigravityOAuth.error.value = errors.join('\n')
      appStore.showError(t('admin.providers.oauth.batchFailed'))
    }
  } finally {
    antigravityOAuth.loading.value = false
  }
}

// Gemini OAuth 授权码兑换
const handleGeminiExchange = async (authCode: string) => {
  if (!authCode.trim() || !geminiOAuth.sessionId.value) return

  geminiOAuth.loading.value = true
  geminiOAuth.error.value = ''

  try {
    const stateFromInput = oauthFlowRef.value?.oauthState || ''
    const stateToUse = stateFromInput || geminiOAuth.state.value
    if (!stateToUse) {
      geminiOAuth.error.value = t('admin.providers.oauth.authFailed')
      appStore.showError(geminiOAuth.error.value)
      return
    }

    const tokenInfo = await geminiOAuth.exchangeAuthCode({
      code: authCode.trim(),
      sessionId: geminiOAuth.sessionId.value,
      state: stateToUse,
      proxyId: form.proxy_id,
      oauthType: geminiOAuthType.value,
      tierId: geminiSelectedTier.value
    })
    if (!tokenInfo) return

    const credentials = geminiOAuth.buildCredentials(tokenInfo)
    const extra = geminiOAuth.buildExtraInfo(tokenInfo)
    await createProviderAndFinish('gemini', 'oauth', credentials, extra)
  } catch (error: any) {
    geminiOAuth.error.value = error.response?.data?.detail || t('admin.providers.oauth.authFailed')
    appStore.showError(geminiOAuth.error.value)
  } finally {
    geminiOAuth.loading.value = false
  }
}

// Antigravity OAuth 授权码兑换
const handleAntigravityExchange = async (authCode: string) => {
  if (!authCode.trim() || !antigravityOAuth.sessionId.value) return

  antigravityOAuth.loading.value = true
  antigravityOAuth.error.value = ''

  try {
    const stateFromInput = oauthFlowRef.value?.oauthState || ''
    const stateToUse = stateFromInput || antigravityOAuth.state.value
    if (!stateToUse) {
      antigravityOAuth.error.value = t('admin.providers.oauth.authFailed')
      appStore.showError(antigravityOAuth.error.value)
      return
    }

    const tokenInfo = await antigravityOAuth.exchangeAuthCode({
      code: authCode.trim(),
      sessionId: antigravityOAuth.sessionId.value,
      state: stateToUse,
      proxyId: form.proxy_id
    })
		if (!tokenInfo) return

		const credentials = antigravityOAuth.buildCredentials(tokenInfo)
		applyAntigravityProjectID(credentials, antigravityProjectId.value, 'create')
		applyInterceptWarmup(credentials, interceptWarmupRequests.value, 'create')
		// Antigravity 只使用映射模式
		const antigravityModelMapping = buildModelMappingObject(
			'mapping',
			[],
			antigravityModelMappings.value
		)
		if (antigravityModelMapping) {
			credentials.model_mapping = antigravityModelMapping
		}
    credentials.model_whitelist = [...antigravityWhitelistModels.value]
		const extra = buildAntigravityExtra()
		await createProviderAndFinish('antigravity', 'oauth', credentials, extra)
  } catch (error: any) {
    antigravityOAuth.error.value = error.response?.data?.detail || t('admin.providers.oauth.authFailed')
    appStore.showError(antigravityOAuth.error.value)
  } finally {
    antigravityOAuth.loading.value = false
  }
}

const handleQoderExchange = async (authCode: string) => {
  const flowContext = captureQoderFlowContext()
  if (
    !qoderOAuth.sessionId.value ||
    !isCurrentQoderFlow(flowContext) ||
    qoderOAuthCompleted ||
    qoderProviderCreateGeneration.value !== null
  ) return

  const shouldResumePolling = qoderPollTimer !== null
  const sessionId = qoderOAuth.sessionId.value
  stopQoderPolling()
  qoderOAuth.loading.value = true
  qoderOAuth.error.value = ''
  const resumePollingIfNeeded = () => {
    if (
      isCurrentQoderFlow(flowContext) &&
      shouldResumePolling &&
      !qoderPollTimer &&
      qoderOAuth.sessionId.value &&
      qoderOAuth.state.value
    ) {
      startQoderPolling(qoderOAuth.pollInterval.value)
    }
  }

  let exchanged = false
  try {
    const stateFromInput = oauthFlowRef.value?.oauthState || ''
    const stateToUse = stateFromInput || qoderOAuth.state.value
    if (!stateToUse) {
      qoderOAuth.error.value = t('admin.providers.oauth.authFailed')
      appStore.showError(qoderOAuth.error.value)
      resumePollingIfNeeded()
      return
    }

    const rawInput = authCode.trim()
    const tokenInfo = await qoderOAuth.exchangeAuthCode({
      code: rawInput,
      callbackUrl: rawInput,
      sessionId,
      state: stateToUse
    })
    if (!isCurrentQoderFlow(flowContext)) return
    if (!tokenInfo) {
      resumePollingIfNeeded()
      return
    }

    exchanged = true
    await createQoderOAuthProvider(tokenInfo, flowContext)
  } catch (error: any) {
    if (!isCurrentQoderFlow(flowContext)) return
    qoderOAuth.error.value = error.response?.data?.detail || t('admin.providers.oauth.authFailed')
    appStore.showError(qoderOAuth.error.value)
    if (!exchanged) {
      resumePollingIfNeeded()
    }
  } finally {
    if (isCurrentQoderFlow(flowContext)) {
      qoderOAuth.loading.value = false
    }
  }
}

// Grok OAuth 授权码兑换
const handleGrokExchange = async (authCode: string) => {
  if (!authCode.trim() || !grokOAuth.sessionId.value) return
  if (!validateGrokOAuthUpstreamConfig()) return

  grokOAuth.loading.value = true
  grokOAuth.error.value = ''

  try {
    const stateFromInput = oauthFlowRef.value?.oauthState || ''
    const stateToUse = stateFromInput || grokOAuth.state.value
    if (!stateToUse) {
      grokOAuth.error.value = t('admin.providers.oauth.authFailed')
      appStore.showError(grokOAuth.error.value)
      return
    }

    const tokenInfo = await grokOAuth.exchangeAuthCode({
      code: authCode.trim(),
      sessionId: grokOAuth.sessionId.value,
      state: stateToUse,
      proxyId: form.proxy_id
    })
    if (!tokenInfo) return

    const credentials = grokOAuth.buildCredentials(tokenInfo)
    applyGrokOAuthUpstreamConfig(credentials)
    const extra = grokOAuth.buildExtraInfo(tokenInfo)
    await createProviderAndFinish('grok', 'oauth', credentials, extra)
  } catch (error: any) {
    grokOAuth.error.value = error.response?.data?.detail || t('admin.providers.oauth.authFailed')
    appStore.showError(grokOAuth.error.value)
  } finally {
    grokOAuth.loading.value = false
  }
}

// Anthropic OAuth 授权码兑换
const handleAnthropicExchange = async (authCode: string) => {
  if (!authCode.trim() || !oauth.sessionId.value) return

  oauth.loading.value = true
  oauth.error.value = ''

  try {
    const proxyConfig = form.proxy_id ? { proxy_id: form.proxy_id } : {}
    const endpoint =
      addMethod.value === 'oauth'
        ? '/admin/providers/exchange-code'
        : '/admin/providers/exchange-setup-token-code'

    const tokenInfo = await adminAPI.providers.exchangeCode(endpoint, {
      session_id: oauth.sessionId.value,
      code: authCode.trim(),
      ...proxyConfig
    })

    // Build extra with quota control settings
    const baseExtra = oauth.buildExtraInfo(tokenInfo) || {}
    const extra: Record<string, unknown> = { ...baseExtra }

    // Add window cost limit settings
    if (windowCostEnabled.value && windowCostLimit.value != null && windowCostLimit.value > 0) {
      extra.window_cost_limit = windowCostLimit.value
      extra.window_cost_sticky_reserve = windowCostStickyReserve.value ?? 10
    }

    // Add session limit settings
    if (sessionLimitEnabled.value && maxSessions.value != null && maxSessions.value > 0) {
      extra.max_sessions = maxSessions.value
      extra.session_idle_timeout_minutes = sessionIdleTimeout.value ?? 5
    }

    // Add RPM limit settings
    if (rpmLimitEnabled.value) {
      const DEFAULT_BASE_RPM = 15
      extra.base_rpm = (baseRpm.value != null && baseRpm.value > 0)
        ? baseRpm.value
        : DEFAULT_BASE_RPM
      extra.rpm_strategy = rpmStrategy.value
      if (rpmStickyBuffer.value != null && rpmStickyBuffer.value > 0) {
        extra.rpm_sticky_buffer = rpmStickyBuffer.value
      }
    }

    // UMQ mode（独立于 RPM）
    if (userMsgQueueMode.value) {
      extra.user_msg_queue_mode = userMsgQueueMode.value
    }

    // Add TLS fingerprint settings
    if (tlsFingerprintEnabled.value) {
      extra.enable_tls_fingerprint = true
      if (tlsFingerprintProfileId.value) {
        extra.tls_fingerprint_profile_id = tlsFingerprintProfileId.value
      }
      if (form.platform === 'openai' && providerCategory.value === 'oauth-based' && tlsFingerprintRouterId.value) {
        extra.tls_fingerprint_router_id = tlsFingerprintRouterId.value
      }
    }

    // Add session ID masking settings
    if (sessionIdMaskingEnabled.value) {
      extra.session_id_masking_enabled = true
    }

    // Add cache TTL override settings
    if (cacheTTLOverrideEnabled.value) {
      extra.cache_ttl_override_enabled = true
      extra.cache_ttl_override_target = cacheTTLOverrideTarget.value
    }

    // Add custom base URL settings
    if (customBaseUrlEnabled.value && customBaseUrl.value.trim()) {
      extra.custom_base_url_enabled = true
      extra.custom_base_url = customBaseUrl.value.trim()
    }

    const credentials: Record<string, unknown> = { ...tokenInfo }
    applyInterceptWarmup(credentials, interceptWarmupRequests.value, 'create')
    await createProviderAndFinish(form.platform, addMethod.value as ProviderType, credentials, extra)
  } catch (error: any) {
    oauth.error.value = error.response?.data?.detail || t('admin.providers.oauth.authFailed')
    appStore.showError(oauth.error.value)
  } finally {
    oauth.loading.value = false
  }
}

// 主入口：根据平台路由到对应处理函数
const handleExchangeCode = async () => {
  const authCode = oauthFlowRef.value?.authCode || ''

  switch (form.platform) {
    case 'openai':
      return handleOpenAIExchange(authCode)
    case 'gemini':
      return handleGeminiExchange(authCode)
    case 'antigravity':
      return handleAntigravityExchange(authCode)
    case 'qoder':
      return handleQoderExchange(authCode)
    case 'grok':
      return handleGrokExchange(authCode)
    default:
      return handleAnthropicExchange(authCode)
  }
}

const handleCookieAuth = async (sessionKey: string) => {
  oauth.loading.value = true
  oauth.error.value = ''

  try {
    const proxyConfig = form.proxy_id ? { proxy_id: form.proxy_id } : {}
    const keys = oauth.parseSessionKeys(sessionKey)

    if (keys.length === 0) {
      oauth.error.value = t('admin.providers.oauth.pleaseEnterSessionKey')
      return
    }

    const tempUnschedPayload = tempUnschedEnabled.value
      ? buildTempUnschedRules(tempUnschedRules.value)
      : []
    if (tempUnschedEnabled.value && tempUnschedPayload.length === 0) {
      appStore.showError(t('admin.providers.tempUnschedulable.rulesInvalid'))
      return
    }

    const endpoint =
      addMethod.value === 'oauth'
        ? '/admin/providers/cookie-auth'
        : '/admin/providers/setup-token-cookie-auth'

    let successCount = 0
    let failedCount = 0
    const errors: string[] = []

    for (let i = 0; i < keys.length; i++) {
      try {
        const tokenInfo = await adminAPI.providers.exchangeCode(endpoint, {
          session_id: '',
          code: keys[i],
          ...proxyConfig
        })

        // Build extra with quota control settings
        const baseExtra = oauth.buildExtraInfo(tokenInfo) || {}
        const extra: Record<string, unknown> = { ...baseExtra }

        // Add window cost limit settings
        if (windowCostEnabled.value && windowCostLimit.value != null && windowCostLimit.value > 0) {
          extra.window_cost_limit = windowCostLimit.value
          extra.window_cost_sticky_reserve = windowCostStickyReserve.value ?? 10
        }

        // Add session limit settings
        if (sessionLimitEnabled.value && maxSessions.value != null && maxSessions.value > 0) {
          extra.max_sessions = maxSessions.value
          extra.session_idle_timeout_minutes = sessionIdleTimeout.value ?? 5
        }

        // Add RPM limit settings
        if (rpmLimitEnabled.value) {
          const DEFAULT_BASE_RPM = 15
          extra.base_rpm = (baseRpm.value != null && baseRpm.value > 0)
            ? baseRpm.value
            : DEFAULT_BASE_RPM
          extra.rpm_strategy = rpmStrategy.value
          if (rpmStickyBuffer.value != null && rpmStickyBuffer.value > 0) {
            extra.rpm_sticky_buffer = rpmStickyBuffer.value
          }
        }

        // UMQ mode（独立于 RPM）
        if (userMsgQueueMode.value) {
          extra.user_msg_queue_mode = userMsgQueueMode.value
        }

        // Add TLS fingerprint settings
        if (tlsFingerprintEnabled.value) {
          extra.enable_tls_fingerprint = true
          if (tlsFingerprintProfileId.value) {
            extra.tls_fingerprint_profile_id = tlsFingerprintProfileId.value
          }
          if (form.platform === 'openai' && providerCategory.value === 'oauth-based' && tlsFingerprintRouterId.value) {
            extra.tls_fingerprint_router_id = tlsFingerprintRouterId.value
          }
        }

        // Add session ID masking settings
        if (sessionIdMaskingEnabled.value) {
          extra.session_id_masking_enabled = true
        }

        // Add cache TTL override settings
        if (cacheTTLOverrideEnabled.value) {
          extra.cache_ttl_override_enabled = true
          extra.cache_ttl_override_target = cacheTTLOverrideTarget.value
        }

        // Add custom base URL settings
        if (customBaseUrlEnabled.value && customBaseUrl.value.trim()) {
          extra.custom_base_url_enabled = true
          extra.custom_base_url = customBaseUrl.value.trim()
        }

        const providerName = keys.length > 1 ? `${form.name} #${i + 1}` : form.name

        const credentials: Record<string, unknown> = { ...tokenInfo }
        applyInterceptWarmup(credentials, interceptWarmupRequests.value, 'create')
        if (tempUnschedEnabled.value) {
          credentials.temp_unschedulable_enabled = true
          credentials.temp_unschedulable_rules = tempUnschedPayload
        }

        await createProtocolProvider({
          name: providerName,
          notes: form.notes,
          platform: form.platform,
          type: addMethod.value, // Use addMethod as type: 'oauth' or 'setup-token'
          credentials,
          extra: withUpstreamRequestIdHeader(extra),
          proxy_id: form.proxy_id,
          concurrency: form.concurrency,
          load_factor: form.load_factor ?? undefined,
          priority: form.priority,
          rate_multiplier: form.rate_multiplier,
          group_ids: form.group_ids,
          expires_at: form.expires_at,
          auto_pause_on_expired: autoPauseOnExpired.value
        })

        successCount++
      } catch (error: any) {
        failedCount++
        errors.push(
          t('admin.providers.oauth.keyAuthFailed', {
            index: i + 1,
            error: error.response?.data?.detail || t('admin.providers.oauth.authFailed')
          })
        )
      }
    }

    if (successCount > 0) {
      appStore.showSuccess(t('admin.providers.oauth.successCreated', { count: successCount }))
      if (failedCount === 0) {
        emit('created')
        handleClose()
      } else {
        emit('created')
      }
    }

    if (failedCount > 0) {
      oauth.error.value = errors.join('\n')
    }
  } catch (error: any) {
    oauth.error.value = error.response?.data?.detail || t('admin.providers.oauth.cookieAuthFailed')
  } finally {
    oauth.loading.value = false
  }
}

onBeforeUnmount(() => {
  stopQoderPolling()
  closeQoderAuthPopup()
})
</script>
