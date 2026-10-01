<template>
  <GoogleOneTap
    :enabled="googleOneTapEligible"
    :client-id="googleOneTapClientID"
  />

  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- 默认首页 -->
  <div v-else class="ba-theme-shell relative flex min-h-screen flex-col overflow-hidden pt-[var(--header-h)] text-gray-950 dark:text-white">
    <div class="ba-theme-backdrop pointer-events-none fixed inset-0"></div>

    <!-- 首页与控制台共用顶栏，外壳为固定顶栏预留高度。 -->
    <AppHeader public-page />

    <!-- 两套首页只区分构成，公开配置、市场数据和登录状态共用下方逻辑。 -->
    <template v-if="visualTheme === 'bauhaus'">
    <main data-testid="bauhaus-home" v-content-reveal="motionRoute?.path" class="relative z-10 flex-1 pb-0">
      <!-- ===== Hero：不对称构成 + 几何装饰 ===== -->
      <section class="relative mx-auto max-w-7xl px-4 pb-16 pt-16 sm:px-6 lg:px-8 lg:pb-24 lg:pt-24">
        <div class="relative grid items-center gap-14 lg:grid-cols-[minmax(0,0.92fr)_minmax(360px,1.08fr)] lg:gap-20">
          <div class="max-w-3xl">
            <span class="bh-home-kicker animate-bh-rise">{{ siteName }} · AI API GATEWAY</span>

            <h1 class="mt-6 animate-bh-rise text-5xl font-extrabold leading-[1.02] tracking-tighter text-gray-950 [animation-delay:0.08s] dark:text-white sm:text-6xl lg:text-7xl">
              {{ homeHeroTitle }}<span class="text-bh-red">.</span>
            </h1>

            <p class="mt-6 max-w-xl animate-bh-rise text-lg font-bold leading-8 text-gray-800 [animation-delay:0.16s] dark:text-dark-100">
              {{ homeHeroSubtitle }}
            </p>

            <div class="mt-10 flex flex-wrap items-center gap-5 animate-bh-rise [animation-delay:0.24s]">
              <router-link
                :to="isAuthenticated ? dashboardPath : '/login'"
                class="bh-home-cta bh-home-cta-red"
              >
                {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
                <Icon name="arrowRight" size="sm" :stroke-width="2.5" />
              </router-link>
              <router-link
                to="/models"
                class="bh-home-cta bh-home-cta-yellow"
              >
                {{ t('home.exploreMarketplace') }}
                <span class="relative flex h-5 w-5 items-center justify-center overflow-hidden">
                  <MotionTransition name="bauhaus-home-marketplace-icon" mode="out-in">
                    <ProviderIcon
                      v-if="homeMarketplaceButtonBrand"
                      :key="homeMarketplaceButtonBrand"
                      :brand="homeMarketplaceButtonBrand"
                      size="18px"
                    />
                    <Icon v-else key="marketplace-fallback" name="sparkles" size="sm" />
                  </MotionTransition>
                </span>
              </router-link>
            </div>
          </div>

          <!-- 独立构成舞台：三种基本形与多家模型供应商在轨道上运行。 -->
          <div class="bauhaus-home-stage" data-testid="bauhaus-home-orbit">
            <div class="bauhaus-home-stage-grid"></div>
            <div class="bauhaus-home-stage-sun"></div>
            <div class="bauhaus-home-stage-ring bauhaus-home-stage-ring-outer"></div>
            <div class="bauhaus-home-stage-ring bauhaus-home-stage-ring-inner"></div>
            <div class="bauhaus-home-stage-geometry-orbit">
              <span class="bauhaus-home-stage-geometry bauhaus-home-stage-geometry-square"></span>
              <span class="bauhaus-home-stage-geometry bauhaus-home-stage-geometry-triangle"></span>
              <span class="bauhaus-home-stage-geometry bauhaus-home-stage-geometry-circle"></span>
            </div>
            <div class="bauhaus-home-stage-icon-cloud">
              <span
                v-for="node in homeOrbitNodes"
                :key="`orbit-${node.brand}`"
                class="bauhaus-home-stage-icon-node"
                :style="{ left: node.left, top: node.top }"
              >
                <ProviderIcon :brand="node.brand" size="16px" />
              </span>
            </div>
            <router-link
              to="/models"
              class="bauhaus-home-stage-panel"
              :aria-label="t('home.exploreMarketplace')"
              :title="t('home.exploreMarketplace')"
            >
              <div class="flex items-center justify-between border-b-2 border-gray-950 pb-3 dark:border-dark-100">
                <span class="font-mono text-xs font-extrabold uppercase tracking-[0.24em] text-gray-600 dark:text-dark-200">LIVE ROUTE</span>
                <span class="bauhaus-home-stage-live-dot"></span>
              </div>
              <p class="mt-4 truncate font-mono text-sm font-extrabold text-gray-950 dark:text-white">{{ homeRouteLabel }}</p>
              <div class="mt-5 flex items-center gap-2">
                <span v-for="brand in homeRouteProviderBrands" :key="`panel-${brand}`" class="bauhaus-home-stage-node">
                  <ProviderIcon :brand="brand" size="16px" />
                </span>
                <span class="bauhaus-home-stage-route-link ml-auto font-mono text-lg font-extrabold text-bh-red" aria-hidden="true">→</span>
              </div>
            </router-link>
            <span class="bauhaus-home-stage-stamp">24<span>/</span>7</span>
          </div>
        </div>

        <!-- ===== 数据统计：色顶方块 ===== -->
        <div class="mt-16 grid grid-cols-2 gap-5 md:grid-cols-4 lg:mt-24">
          <div
            v-for="(card, index) in homeStatsCards"
            :key="card.key"
            class="border-[3px] border-gray-950 bg-white shadow transition-transform hover:-translate-x-0.5 hover:-translate-y-0.5 dark:border-dark-100 dark:bg-dark-800"
          >
            <div class="h-2.5" :class="['bg-bh-red', 'bg-bh-yellow', 'bg-bh-blue', 'bg-gray-950 dark:bg-dark-100'][index % 4]"></div>
            <div class="px-5 pb-5 pt-4">
              <p class="min-h-[1.1em] text-3xl font-extrabold tabular-nums tracking-tight text-gray-950 dark:text-white md:text-4xl">
                {{ card.value }}
              </p>
              <p class="mt-1.5 text-xs font-extrabold uppercase tracking-widest text-gray-600 dark:text-dark-200">{{ card.label }}</p>
            </div>
          </div>
        </div>
        <p v-if="homeStatsError" class="mt-4 text-xs font-bold text-gray-500 dark:text-dark-300">
          {{ t('home.stats.unavailable') }}
        </p>
      </section>

      <!-- ===== 品牌走马灯：黑底承载核心关键词 ===== -->
      <section class="bh-home-marquee relative z-10" :aria-label="t('home.features.unifiedGateway')">
        <div class="bh-home-marquee-track">
          <!-- 轨道复制一份用于无缝滚动，单轮内每个关键词只出现一次。 -->
          <span v-for="copy in 2" class="bh-home-marquee-copy" :aria-hidden="copy === 2 ? true : undefined" :key="`marquee-copy-${copy}`">
            <template v-for="(keyword, index) in homeMarqueeKeywords" :key="`${copy}-${keyword}-${index}`">
              <span class="bauhaus-home-marquee-word" :class="`bauhaus-home-marquee-word-${index % 4}`">{{ keyword }}</span>
              <span class="bauhaus-home-marquee-separator" aria-hidden="true">◆</span>
            </template>
          </span>
        </div>
      </section>

      <!-- ===== 功能区：色块头卡片 ===== -->
      <section class="mx-auto max-w-7xl px-4 pt-20 sm:px-6 lg:px-8">
        <h2 class="bh-home-section-title">{{ t('home.features.unifiedGateway') }}</h2>

        <div class="grid gap-8 sm:grid-cols-2 xl:grid-cols-4">
          <!-- 卡片 1：统一网关 -->
          <article class="bh-home-block bh-home-block-hover group">
            <div class="flex items-center justify-between border-b-[3px] border-gray-950 bg-bh-red px-5 py-3.5 dark:border-dark-100">
              <h3 class="text-base font-extrabold text-white">{{ t('home.features.unifiedGateway') }}</h3>
              <span class="bh-home-head-shape h-5 w-5 border-2 border-gray-950 bg-bh-yellow transition-transform duration-300 group-hover:rotate-[135deg]"></span>
            </div>
            <div class="relative h-40 overflow-hidden border-b-[3px] border-gray-950 bg-bh-paper dark:border-dark-100 dark:bg-dark-900">
              <div class="absolute inset-0 transition-transform duration-500 ease-out group-hover:scale-110">
                <span
                  v-for="(icon, index) in homeProviderCloudIcons"
                  :key="`${icon.brand}-${index}`"
                  class="absolute flex h-7 w-7 items-center justify-center border border-gray-950/70 bg-white text-gray-700 dark:border-dark-200/60 dark:bg-dark-800 dark:text-dark-100"
                  :style="{
                    left: icon.left,
                    top: icon.top,
                    opacity: icon.opacity,
                    transform: `translate(-50%, -50%) scale(${icon.scale})`,
                  }"
                >
                  <ProviderIcon :brand="icon.brand" size="14px" />
                </span>
              </div>
            </div>
            <div class="p-5">
              <p class="text-sm font-medium leading-6 text-gray-700 dark:text-dark-100">
                {{ t('home.features.unifiedGatewayDesc') }}
              </p>
              <router-link to="/models" class="bh-home-card-cta">
                {{ t('home.features.browseAll') }}
                <Icon name="arrowRight" size="xs" :stroke-width="2.5" />
              </router-link>
            </div>
          </article>

          <!-- 卡片 2：多账号智能调度 -->
          <article class="bh-home-block bh-home-block-hover group">
            <div class="flex items-center justify-between border-b-[3px] border-gray-950 bg-bh-blue px-5 py-3.5 dark:border-dark-100">
              <h3 class="text-base font-extrabold text-white">{{ t('home.features.multiAccount') }}</h3>
              <span class="bh-home-head-shape h-5 w-5 rounded-full bg-bh-yellow transition-transform duration-300 group-hover:rotate-[135deg]"></span>
            </div>
            <div class="relative flex h-40 items-center justify-center overflow-hidden border-b-[3px] border-gray-950 bg-bh-paper dark:border-dark-100 dark:bg-dark-900">
              <div class="relative h-full w-full transition-transform duration-500 ease-out group-hover:scale-110">
                <div class="absolute left-1/2 top-5 z-10 max-w-[82%] -translate-x-1/2 truncate border-2 border-gray-950 bg-bh-yellow px-3.5 py-1 font-mono text-xs font-bold text-gray-950 dark:border-dark-100">
                  {{ homeRouteLabel }}
                </div>
                <svg
                  class="absolute left-1/2 top-10 h-24 w-[220px] -translate-x-1/2 text-gray-950 dark:text-dark-200"
                  viewBox="0 0 220 110"
                  fill="none"
                  aria-hidden="true"
                >
                  <path d="M110 0V30" stroke="currentColor" stroke-width="2.5" />
                  <path
                    d="M110 30C110 60 28 52 28 84M110 30C110 55 110 64 110 84M110 30C110 60 192 52 192 84"
                    stroke="currentColor"
                    stroke-width="2.5"
                  />
                </svg>
                <div class="absolute bottom-5 left-1/2 flex w-[190px] -translate-x-1/2 justify-between">
                  <span
                    v-for="brand in homeRouteProviderBrands"
                    :key="brand"
                    class="flex h-9 w-9 items-center justify-center border-2 border-gray-950 bg-white text-gray-700 dark:border-dark-100 dark:bg-dark-800 dark:text-dark-100"
                  >
                    <ProviderIcon :brand="brand" size="17px" />
                  </span>
                </div>
              </div>
            </div>
            <div class="p-5">
              <p class="text-sm font-medium leading-6 text-gray-700 dark:text-dark-100">
                {{ t('home.features.multiAccountDesc') }}
              </p>
              <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="bh-home-card-cta">
                {{ t('home.features.learnMore') }}
                <Icon name="arrowRight" size="xs" :stroke-width="2.5" />
              </router-link>
            </div>
          </article>

          <!-- 卡片 3：额度用量 -->
          <article class="bh-home-block bh-home-block-hover group">
            <div class="flex items-center justify-between border-b-[3px] border-gray-950 bg-bh-yellow px-5 py-3.5 dark:border-dark-100">
              <h3 class="text-base font-extrabold text-gray-950">{{ t('home.features.balanceQuota') }}</h3>
              <span class="bh-home-head-shape bh-home-tri-sm transition-transform duration-300 group-hover:rotate-[135deg]"></span>
            </div>
            <div class="flex h-40 items-center justify-center border-b-[3px] border-gray-950 bg-bh-paper p-6 dark:border-dark-100 dark:bg-dark-900">
              <div class="w-full max-w-[200px] border-2 border-gray-950 bg-white p-4 shadow-sm transition-transform duration-500 ease-out group-hover:scale-110 dark:border-dark-100 dark:bg-dark-800">
                <div class="mb-4 flex items-center justify-between text-xs font-bold text-gray-700 dark:text-dark-200">
                  <span>{{ t('home.features.usageChart') }}</span>
                  <Icon name="chart" size="sm" />
                </div>
                <div class="space-y-3">
                  <div class="h-2.5 w-11/12 bg-bh-red"></div>
                  <div class="h-2.5 w-2/3 bg-bh-yellow"></div>
                  <div class="h-2.5 w-5/6 bg-bh-blue"></div>
                  <div class="h-2.5 w-1/2 bg-gray-950 dark:bg-dark-100"></div>
                </div>
              </div>
            </div>
            <div class="p-5">
              <p class="text-sm font-medium leading-6 text-gray-700 dark:text-dark-100">
                {{ t('home.features.balanceQuotaDesc') }}
              </p>
              <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="bh-home-card-cta">
                {{ t('home.features.viewUsage') }}
                <Icon name="arrowRight" size="xs" :stroke-width="2.5" />
              </router-link>
            </div>
          </article>

          <!-- 卡片 4：数据与策略 -->
          <article class="bh-home-block bh-home-block-hover group">
            <div class="flex items-center justify-between border-b-[3px] border-gray-950 bg-gray-950 px-5 py-3.5 dark:border-dark-100 dark:bg-dark-950">
              <h3 class="text-base font-extrabold text-bh-paper">{{ t('home.features.dataPolicies') }}</h3>
              <span class="bh-home-head-shape h-5 w-5 rounded-full bg-bh-red transition-transform duration-300 group-hover:scale-125"></span>
            </div>
            <div class="flex h-40 items-center justify-center border-b-[3px] border-gray-950 bg-bh-paper dark:border-dark-100 dark:bg-dark-900">
              <div class="relative flex h-24 w-24 items-center justify-center rounded-full bg-bh-blue transition-transform duration-500 ease-out group-hover:scale-110">
                <Icon name="shield" size="xl" class="text-white" :stroke-width="2" />
                <span class="absolute -right-2 -top-2 flex h-9 w-9 items-center justify-center border-2 border-gray-950 bg-bh-yellow text-gray-950">
                  <Icon name="check" size="md" :stroke-width="2.5" />
                </span>
              </div>
            </div>
            <div class="p-5">
              <p class="text-sm font-medium leading-6 text-gray-700 dark:text-dark-100">
                {{ t('home.features.dataPoliciesDesc') }}
              </p>
              <a
                v-if="docUrl"
                :href="docUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="bh-home-card-cta"
              >
                {{ t('home.docs') }}
                <Icon name="externalLink" size="xs" :stroke-width="2.5" />
              </a>
            </div>
          </article>
        </div>
      </section>

      <!-- ===== 服务商 / 精选模型 ===== -->
      <section class="mx-auto max-w-7xl px-4 pt-20 sm:px-6 lg:px-8">
        <div class="mb-8 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <router-link to="/models" class="group inline-flex items-center gap-2">
              <h2 class="bh-home-section-title !mb-0">{{ t('home.providers.title') }}</h2>
              <Icon name="chevronRight" size="md" class="text-bh-red transition-transform group-hover:translate-x-1" :stroke-width="2.5" />
            </router-link>
            <p class="mt-3 text-sm font-bold text-gray-700 dark:text-dark-200">
              {{ formatMarketplaceStat(totalModelCount) }} {{ t('marketplace.modelsStat') }}
              ·
              {{ formatMarketplaceStat(supportedProviders.length) }} {{ t('home.stats.providerTypes') }}
            </p>
          </div>
          <router-link to="/models" class="inline-flex items-center gap-1 border-2 border-gray-950 bg-white px-3 py-1.5 text-sm font-extrabold text-gray-950 shadow-sm transition hover:bg-bh-yellow dark:border-dark-100 dark:bg-dark-800 dark:text-white dark:hover:bg-dark-700">
            {{ t('home.viewAll') }}
            <Icon name="arrowRight" size="xs" :stroke-width="2.5" />
          </router-link>
        </div>

        <div class="grid gap-7 sm:grid-cols-2 lg:grid-cols-3">
          <div
            v-if="homeMarketplaceLoading"
            class="border-[3px] border-gray-950 bg-white px-5 py-4 text-center text-sm font-bold text-gray-600 dark:border-dark-100 dark:bg-dark-800 dark:text-dark-200 sm:col-span-2 lg:col-span-3"
          >
            {{ t('common.loading') }}
          </div>

          <div
            v-else-if="supportedProviders.length === 0"
            class="border-[3px] border-gray-950 bg-white px-5 py-4 text-center text-sm font-bold text-gray-600 dark:border-dark-100 dark:bg-dark-800 dark:text-dark-200 sm:col-span-2 lg:col-span-3"
          >
            {{ homeMarketplaceError ? t('home.providers.unavailable') : t('home.providers.empty') }}
          </div>

          <!-- 管理员配置了首页展示模型时，渲染单模型卡片 -->
          <template v-else-if="featuredModels.length > 0">
            <article
              v-for="(featured, fIndex) in featuredModels"
              :key="featured.model.id"
              class="bh-home-block bh-home-block-hover p-6"
            >
              <div class="flex items-start gap-4">
                <span class="relative flex h-12 w-12 shrink-0 items-center justify-center border-2 border-gray-950 bg-white dark:border-dark-100 dark:bg-dark-900">
                  <ModelIcon :model="featured.model.id" size="28px" />
                  <i class="absolute -left-[2px] -top-[2px] block h-2.5 w-2.5" :class="['bg-bh-red', 'bg-bh-blue', 'bg-bh-yellow'][fIndex % 3]"></i>
                </span>
                <div class="min-w-0 flex-1">
                  <h3 class="truncate text-lg font-extrabold text-gray-950 dark:text-white">
                    {{ featured.model.display_name || featured.model.id }}
                  </h3>
                  <p class="truncate text-sm font-semibold text-gray-600 dark:text-dark-200">
                    {{ t('home.featured.byProvider', { provider: homeProviderCategory(featured.group).label }) }}
                  </p>
                </div>
              </div>
              <div v-if="featured.discountOff" class="mt-5 border-t-2 border-gray-950 pt-4 dark:border-dark-200/50">
                <span class="inline-block border-2 border-gray-950 bg-bh-yellow px-2.5 py-1 text-sm font-extrabold tabular-nums text-gray-950">
                  {{ featured.discountOff }}
                </span>
              </div>
            </article>
          </template>

          <template v-else>
            <article
              v-for="provider in supportedProviders.slice(0, 6)"
              :key="provider.key"
              class="bh-home-block bh-home-block-hover p-6"
            >
              <div class="flex items-start gap-4">
                <span class="relative flex h-12 w-12 shrink-0 items-center justify-center border-2 border-gray-950 bg-white dark:border-dark-100 dark:bg-dark-900">
                  <ProviderIcon :brand="provider.iconBrand" size="22px" />
                  <i class="absolute -left-[2px] -top-[2px] block h-2.5 w-2.5 rounded-full border border-gray-950 bg-emerald-500 dark:border-dark-100"></i>
                </span>
                <div class="min-w-0 flex-1">
                  <h3 class="truncate text-lg font-extrabold text-gray-950 dark:text-white">
                    {{ provider.label }}
                  </h3>
                  <p class="text-sm font-semibold text-gray-600 dark:text-dark-200">
                    {{ provider.groupCount }} {{ t('home.providers.groups') }}
                  </p>
                </div>
              </div>
              <div class="mt-5 border-t-2 border-gray-950 pt-4 dark:border-dark-200/50">
                <div class="flex items-end justify-between gap-4">
                  <div>
                    <p class="text-xs font-extrabold uppercase tracking-widest text-gray-600 dark:text-dark-200">{{ t('home.providers.modelCount') }}</p>
                    <p class="mt-1 text-2xl font-extrabold tabular-nums text-gray-950 dark:text-white">
                      {{ provider.modelCount }}
                    </p>
                  </div>
                  <span
                    v-if="provider.officialPriceRatio"
                    class="inline-block max-w-[180px] border-2 border-gray-950 bg-bh-yellow px-2.5 py-1 text-right text-sm font-extrabold text-gray-950"
                  >
                    {{ formatOfficialPriceRatio(provider.officialPriceRatio) }}
                  </span>
                  <span v-else class="text-sm font-extrabold text-primary-600 dark:text-primary-300">
                    {{ t('home.providers.supported') }}
                  </span>
                </div>
              </div>
            </article>
          </template>
        </div>
      </section>

      <!-- ===== 三步上手 ===== -->
      <section class="mx-auto max-w-7xl px-4 pt-20 sm:px-6 lg:px-8">
        <h2 class="bh-home-section-title">{{ t('home.steps.signup.title') }} → {{ t('home.steps.apiKey.title') }}</h2>
        <div class="grid gap-8 md:grid-cols-3">
          <article
            v-for="(step, sIndex) in homeSteps"
            :key="step.key"
            class="bh-home-block flex min-h-[220px] flex-col p-6"
          >
            <div class="flex items-center gap-4">
              <span
                class="flex h-12 w-12 shrink-0 items-center justify-center border-2 border-gray-950 font-display text-xl dark:border-dark-100"
                :class="[
                  'bg-bh-red text-white',
                  'bg-bh-blue text-white',
                  'bg-bh-yellow text-gray-950'
                ][sIndex % 3]"
              >
                {{ step.index }}
              </span>
              <h3 class="text-lg font-extrabold tracking-tight text-gray-950 dark:text-white">{{ step.title }}</h3>
            </div>
            <p class="mt-4 max-w-sm text-sm font-medium leading-6 text-gray-700 dark:text-dark-100">{{ step.description }}</p>

            <div v-if="step.key === 'signup'" class="mt-auto pt-6">
              <div class="grid max-w-[156px] grid-cols-3 gap-3">
                <span class="flex h-10 w-10 items-center justify-center border-2 border-gray-950 bg-white dark:border-dark-100 dark:bg-dark-900">
                  <ProviderIcon brand="Google" size="20px" />
                </span>
                <span class="flex h-10 w-10 items-center justify-center border-2 border-gray-950 bg-white text-gray-800 dark:border-dark-100 dark:bg-dark-900 dark:text-gray-100">
                  <GitHubMark class="h-5 w-5" />
                </span>
                <span class="flex h-10 w-10 items-center justify-center border-2 border-gray-950 bg-bh-yellow text-gray-950 dark:border-dark-100">
                  <Icon name="mail" size="md" :stroke-width="2" />
                </span>
              </div>
            </div>

            <div v-else-if="step.key === 'browse'" class="mt-auto max-w-[270px] pt-6">
              <div class="space-y-2">
                <div class="flex items-center gap-2 border-2 border-gray-950 bg-white px-3 py-2 text-gray-800 dark:border-dark-100 dark:bg-dark-900 dark:text-dark-100">
                  <span class="w-14 text-xs font-extrabold">Claude</span>
                  <span class="h-2 flex-1 bg-bh-red"></span>
                  <span class="h-2 w-12 bg-bh-yellow"></span>
                </div>
                <div class="flex items-center gap-2 border-2 border-gray-950 bg-white px-3 py-2 text-gray-800 dark:border-dark-100 dark:bg-dark-900 dark:text-dark-100">
                  <span class="w-14 text-xs font-extrabold">GPT</span>
                  <span class="h-2 flex-1 bg-bh-blue"></span>
                  <span class="h-2 w-12 bg-gray-950 dark:bg-dark-100"></span>
                </div>
              </div>
            </div>

            <div v-else class="mt-auto max-w-[270px] pt-6">
              <div class="flex items-center gap-3">
                <span class="flex h-9 w-9 shrink-0 items-center justify-center border-2 border-gray-950 bg-bh-blue text-white dark:border-dark-100">
                  <Icon name="key" size="sm" :stroke-width="2" />
                </span>
                <div class="flex-1 border-2 border-gray-950 bg-white px-3 py-2 font-mono text-xs font-bold text-gray-700 dark:border-dark-100 dark:bg-dark-900 dark:text-dark-200">
                  TOKENFLUX_API_KEY
                </div>
              </div>
              <div class="mt-3 border-2 border-gray-950 bg-gray-950 px-3 py-2 font-mono text-sm tracking-[0.2em] text-bh-yellow dark:border-dark-100">
                ••••••••••••••••
              </div>
            </div>
          </article>
        </div>
      </section>

      <!-- ===== CTA：黄色横幅 ===== -->
      <section class="mx-auto max-w-7xl px-4 pb-24 pt-20 sm:px-6 lg:px-8">
        <div class="relative overflow-hidden border-[3px] border-gray-950 bg-bh-yellow px-6 py-12 shadow-lg dark:border-dark-100 sm:px-12">
          <span class="pointer-events-none absolute -right-10 -top-10 h-40 w-40 rounded-full bg-bh-red opacity-90" aria-hidden="true"></span>
          <span class="pointer-events-none absolute -bottom-12 right-24 h-32 w-32 bg-bh-blue" aria-hidden="true"></span>
          <span class="bh-home-cta-triangle" aria-hidden="true"></span>
          <div class="relative max-w-2xl">
            <h2 class="text-3xl font-extrabold tracking-tight text-gray-950 sm:text-4xl">
              {{ t('home.cta.title') }}
            </h2>
            <p class="mt-4 max-w-xl text-base font-bold leading-7 text-gray-900">
              {{ t('home.cta.description') }}
            </p>
            <div class="mt-8">
              <router-link
                :to="isAuthenticated ? dashboardPath : '/login'"
                class="bh-home-cta bh-home-cta-ink"
              >
                {{ isAuthenticated ? t('home.goToDashboard') : t('home.cta.button') }}
                <Icon name="arrowRight" size="sm" :stroke-width="2.5" />
              </router-link>
            </div>
          </div>
        </div>
      </section>
    </main>

    <!-- ===== 页脚 ===== -->
    <footer class="relative z-10 border-t-[3px] border-gray-950 bg-bh-paper px-6 pb-4 pt-12 dark:border-dark-100 dark:bg-dark-900">
      <div class="mx-auto max-w-7xl">
        <div class="grid grid-cols-2 gap-x-8 gap-y-10 sm:grid-cols-3 lg:flex lg:justify-between lg:gap-8">
          <!-- 站点品牌 -->
          <div class="col-span-2 sm:col-span-3 lg:col-auto lg:max-w-[240px] lg:shrink-0">
            <div class="flex items-center gap-2.5">
              <span class="h-8 w-8 shrink-0 overflow-hidden border-2 border-gray-950 bg-white dark:border-dark-100">
                <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
              </span>
              <span class="text-sm font-extrabold text-gray-950 dark:text-white">{{ siteName }}<span class="text-bh-red">.</span></span>
            </div>
            <div class="mt-4 flex items-center gap-2.5" aria-hidden="true">
              <i class="block h-3.5 w-3.5 rounded-full bg-bh-red"></i>
              <i class="block h-3.5 w-3.5 bg-bh-blue"></i>
              <i class="bh-home-foot-tri block"></i>
            </div>
            <p class="mt-4 text-sm font-bold text-gray-700 dark:text-dark-200">
              &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
            </p>
            <p
              v-for="(line, index) in footerTextLines"
              :key="index"
              class="mt-1 text-xs font-medium text-gray-600 dark:text-dark-300"
            >
              {{ line }}
            </p>
          </div>

          <!-- 链接分组 -->
          <div v-for="column in footerColumns" :key="column.title" class="lg:min-w-[140px]">
            <h3 class="inline-block border-b-[3px] border-bh-red pb-1 text-sm font-extrabold uppercase tracking-widest text-gray-950 dark:text-white">{{ column.title }}</h3>
            <ul class="mt-4 space-y-2.5">
              <li v-for="link in column.links" :key="link.label">
                <router-link
                  v-if="link.url.startsWith('/')"
                  :to="link.url"
                  class="text-sm font-semibold text-gray-700 transition hover:text-bh-red dark:text-dark-200 dark:hover:text-bh-yellow"
                >
                  {{ link.label }}
                </router-link>
                <a
                  v-else
                  :href="link.url"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-sm font-semibold text-gray-700 transition hover:text-bh-red dark:text-dark-200 dark:hover:text-bh-yellow"
                >
                  {{ link.label }}
                </a>
              </li>
            </ul>
          </div>
        </div>

        <div class="bh-home-stripe mt-10" aria-hidden="true"><i></i><i></i><i></i></div>
      </div>
    </footer>
    </template>
    <template v-else>
    <main v-content-reveal="motionRoute?.path" class="relative z-10 flex-1 px-4 pb-20 pt-16 sm:px-6 lg:px-8">
      <section class="mx-auto max-w-5xl text-center">
        <h1 class="mx-auto max-w-4xl text-4xl font-bold leading-[1.05] tracking-tight text-gray-950 dark:text-white sm:text-5xl md:text-6xl lg:text-7xl">
          {{ homeHeroTitle }}
        </h1>
        <p class="mx-auto mt-6 max-w-2xl text-lg leading-8 text-gray-600 dark:text-dark-300">
          {{ homeHeroSubtitle }}
        </p>

        <div class="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-[44px] min-w-[180px] items-center justify-center gap-2 rounded-control bg-primary-600 px-6 py-3 text-sm font-semibold text-white shadow-none transition hover:bg-primary-700"
          >
            {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
            <Icon name="arrowRight" size="sm" :stroke-width="2" />
          </router-link>
          <router-link
            to="/models"
            class="inline-flex min-h-[44px] min-w-[180px] items-center justify-center gap-2 rounded-control border border-gray-200 bg-white px-6 py-3 text-sm font-semibold text-gray-900 shadow-sm transition hover:border-black/20 hover:text-primary-600 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-100 dark:hover:border-primary-500"
          >
            {{ t('home.exploreMarketplace') }}
            <span class="relative flex h-5 w-5 items-center justify-center overflow-hidden">
              <MotionTransition name="home-marketplace-icon" mode="out-in">
                <ProviderIcon
                  v-if="homeMarketplaceButtonBrand"
                  :key="homeMarketplaceButtonBrand"
                  :brand="homeMarketplaceButtonBrand"
                  size="18px"
                />
                <Icon v-else key="marketplace-fallback" name="sparkles" size="sm" class="text-primary-500" />
              </MotionTransition>
            </span>
          </router-link>
        </div>
      </section>

      <section class="mx-auto mt-16 grid max-w-4xl grid-cols-2 gap-x-6 gap-y-8 md:grid-cols-4">
        <div v-for="card in homeStatsCards" :key="card.key" class="text-center">
          <p class="min-h-[1.1em] text-3xl font-bold tabular-nums tracking-tight text-gray-950 dark:text-white md:text-4xl">
            {{ card.value }}
          </p>
          <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ card.label }}</p>
        </div>
      </section>
      <p v-if="homeStatsError" class="mt-4 text-center text-xs text-gray-500 dark:text-dark-400">
        {{ t('home.stats.unavailable') }}
      </p>

      <!-- Provider icon marquee -->
      <section class="mx-auto mt-14 max-w-5xl" aria-hidden="true">
        <div class="home-marquee relative overflow-hidden">
          <div class="home-marquee-track flex w-max items-center gap-8">
            <span
              v-for="(brand, index) in homeMarqueeBrands"
              :key="`${brand}-${index}`"
              class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-gray-200/80 bg-white text-gray-700 shadow-sm dark:border-dark-700 dark:bg-dark-900 dark:text-dark-100"
            >
              <ProviderIcon :brand="brand" size="17px" />
            </span>
          </div>
        </div>
      </section>

      <section class="mx-auto mt-20 grid max-w-7xl gap-5 sm:grid-cols-2 xl:grid-cols-4">
        <article class="group overflow-hidden rounded-surface border border-gray-200 bg-white shadow-sm ring-1 ring-transparent transition duration-layout hover:-translate-y-1 hover:border-black/20 hover:shadow-[0_12px_32px_rgba(0,0,0,0.1)] focus-within:border-black/20 dark:border-dark-800 dark:bg-dark-900 dark:hover:border-dark-600 dark:hover:shadow-[0_12px_32px_rgba(0,0,0,0.4)]">
          <div class="relative h-44 overflow-hidden border-b border-gray-200 bg-white dark:border-dark-800 dark:bg-dark-950">
            <div class="absolute inset-0 transition-transform duration-500 ease-out group-hover:scale-110">
              <span
                v-for="(icon, index) in homeProviderCloudIcons"
                :key="`${icon.brand}-${index}`"
                class="absolute flex h-7 w-7 items-center justify-center rounded-full border border-gray-100 bg-white/95 text-gray-700 shadow-[0_5px_16px_rgba(0,0,0,0.13)] ring-1 ring-black/[0.02] dark:border-dark-700 dark:bg-dark-900 dark:text-dark-100 dark:ring-white/[0.04]"
                :style="{
                  left: icon.left,
                  top: icon.top,
                  opacity: icon.opacity,
                  transform: `translate(-50%, -50%) scale(${icon.scale})`,
                }"
              >
                <ProviderIcon :brand="icon.brand" size="14px" />
              </span>
            </div>
            <span
              class="pointer-events-none absolute inset-x-0 bottom-0 h-8 bg-gradient-to-t from-white via-white/35 to-transparent dark:from-dark-950 dark:via-dark-950/35"
            ></span>
          </div>
          <div class="p-5">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">
              {{ t('home.features.unifiedGateway') }}
            </h2>
            <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
              {{ t('home.features.unifiedGatewayDesc') }}
            </p>
            <router-link to="/models" class="mt-4 inline-flex items-center gap-1 text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-300">
              {{ t('home.features.browseAll') }}
              <Icon name="arrowRight" size="xs" />
            </router-link>
          </div>
        </article>

        <article class="group overflow-hidden rounded-surface border border-gray-200 bg-white shadow-sm ring-1 ring-transparent transition duration-layout hover:-translate-y-1 hover:border-black/20 hover:shadow-[0_12px_32px_rgba(0,0,0,0.1)] focus-within:border-black/20 dark:border-dark-800 dark:bg-dark-900 dark:hover:border-dark-600 dark:hover:shadow-[0_12px_32px_rgba(0,0,0,0.4)]">
          <div class="relative flex h-44 items-center justify-center overflow-hidden border-b border-gray-200 bg-white dark:border-dark-800 dark:bg-dark-950">
            <div class="relative h-full w-full transition-transform duration-500 ease-out group-hover:scale-110">
              <div class="absolute left-1/2 top-7 z-10 max-w-[82%] -translate-x-1/2 truncate rounded-control bg-gray-100 px-3.5 py-1.5 text-xs font-medium text-gray-800 shadow-sm dark:bg-dark-900 dark:text-dark-100">
                {{ homeRouteLabel }}
              </div>
              <svg
                class="absolute left-1/2 top-12 h-24 w-[220px] -translate-x-1/2 text-gray-300 dark:text-dark-700"
                viewBox="0 0 220 110"
                fill="none"
                aria-hidden="true"
              >
                <path
                  d="M110 0V30"
                  stroke="currentColor"
                  stroke-width="1.35"
                  stroke-linecap="round"
                />
                <path
                  d="M110 30C110 60 28 52 28 84M110 30C110 55 110 64 110 84M110 30C110 60 192 52 192 84"
                  stroke="currentColor"
                  stroke-width="1.35"
                  stroke-linecap="round"
                />
              </svg>
              <div class="absolute bottom-6 left-1/2 flex w-[190px] -translate-x-1/2 justify-between">
                <span
                  v-for="brand in homeRouteProviderBrands"
                  :key="brand"
                  class="flex h-9 w-9 items-center justify-center rounded-control border border-gray-100 bg-white text-gray-700 shadow-[0_5px_16px_rgba(0,0,0,0.13)] dark:border-dark-700 dark:bg-dark-900 dark:text-dark-100"
                >
                  <ProviderIcon :brand="brand" size="17px" />
                </span>
              </div>
            </div>
          </div>
          <div class="p-5">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">
              {{ t('home.features.multiProvider') }}
            </h2>
            <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
              {{ t('home.features.multiProviderDesc') }}
            </p>
            <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="mt-4 inline-flex items-center gap-1 text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-300">
              {{ t('home.features.learnMore') }}
              <Icon name="arrowRight" size="xs" />
            </router-link>
          </div>
        </article>

        <article class="group overflow-hidden rounded-surface border border-gray-200 bg-white shadow-sm ring-1 ring-transparent transition duration-layout hover:-translate-y-1 hover:border-black/20 hover:shadow-[0_12px_32px_rgba(0,0,0,0.1)] focus-within:border-black/20 dark:border-dark-800 dark:bg-dark-900 dark:hover:border-dark-600 dark:hover:shadow-[0_12px_32px_rgba(0,0,0,0.4)]">
          <div class="flex h-44 items-center justify-center border-b border-gray-200 bg-gray-50 p-6 dark:border-dark-800 dark:bg-dark-950">
            <div class="w-full max-w-[200px] rounded-surface border border-gray-200 bg-white p-4 shadow-sm transition-transform duration-500 ease-out group-hover:scale-110 dark:border-dark-700 dark:bg-dark-900">
              <div class="mb-4 flex items-center justify-between text-xs text-gray-500 dark:text-dark-400">
                <span>{{ t('home.features.usageChart') }}</span>
                <Icon name="chart" size="sm" />
              </div>
              <div class="space-y-3">
                <div class="h-2 w-11/12 rounded-full bg-sky-300"></div>
                <div class="h-2 w-2/3 rounded-full bg-amber-300"></div>
                <div class="h-2 w-5/6 rounded-full bg-emerald-300"></div>
                <div class="h-2 w-1/2 rounded-full bg-violet-300"></div>
              </div>
            </div>
          </div>
          <div class="p-5">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">
              {{ t('home.features.balanceQuota') }}
            </h2>
            <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
              {{ t('home.features.balanceQuotaDesc') }}
            </p>
            <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="mt-4 inline-flex items-center gap-1 text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-300">
              {{ t('home.features.viewUsage') }}
              <Icon name="arrowRight" size="xs" />
            </router-link>
          </div>
        </article>

        <article class="group overflow-hidden rounded-surface border border-gray-200 bg-white shadow-sm ring-1 ring-transparent transition duration-layout hover:-translate-y-1 hover:border-black/20 hover:shadow-[0_12px_32px_rgba(0,0,0,0.1)] focus-within:border-black/20 dark:border-dark-800 dark:bg-dark-900 dark:hover:border-dark-600 dark:hover:shadow-[0_12px_32px_rgba(0,0,0,0.4)]">
          <div class="flex h-44 items-center justify-center border-b border-gray-200 bg-gray-50 dark:border-dark-800 dark:bg-dark-950">
            <div class="relative flex h-24 w-24 items-center justify-center rounded-full border border-gray-200 bg-white shadow-sm transition-transform duration-500 ease-out group-hover:scale-110 dark:border-dark-700 dark:bg-dark-900">
              <Icon name="shield" size="xl" class="text-gray-400 dark:text-dark-300" />
              <span class="absolute -right-1 -top-1 flex h-9 w-9 items-center justify-center rounded-full bg-emerald-100 text-emerald-600 dark:bg-emerald-500/15 dark:text-emerald-300">
                <Icon name="check" size="md" :stroke-width="2" :animate-on-hover="false" />
              </span>
            </div>
          </div>
          <div class="p-5">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">
              {{ t('home.features.dataPolicies') }}
            </h2>
            <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
              {{ t('home.features.dataPoliciesDesc') }}
            </p>
            <a
              v-if="docUrl"
              :href="docUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="mt-4 inline-flex items-center gap-1 text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-300"
            >
              {{ t('home.docs') }}
              <Icon name="externalLink" size="xs" />
            </a>
          </div>
        </article>
      </section>

      <section class="mx-auto mt-20 max-w-7xl">
        <div class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <router-link to="/models" class="inline-flex items-center gap-2 text-2xl font-bold text-gray-950 hover:text-primary-600 dark:text-white dark:hover:text-primary-300">
              {{ t('home.providers.title') }}
              <Icon name="chevronRight" size="md" :animate-on-hover="false" />
            </router-link>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
              {{ formatMarketplaceStat(totalModelCount) }} {{ t('marketplace.modelsStat') }}
              ·
              {{ formatMarketplaceStat(supportedProviders.length) }} {{ t('home.stats.providerTypes') }}
            </p>
          </div>
          <router-link to="/models" class="text-sm font-medium text-gray-500 transition hover:text-primary-600 dark:text-dark-400 dark:hover:text-primary-300">
            {{ t('home.viewAll') }}
            <Icon name="arrowRight" size="xs" class="inline-block" />
          </router-link>
        </div>

        <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          <div
            v-if="homeMarketplaceLoading"
            class="rounded-surface border border-gray-200 bg-white px-5 py-4 text-center text-sm text-gray-500 dark:border-dark-800 dark:bg-dark-900 dark:text-dark-400 sm:col-span-2 lg:col-span-3"
          >
            {{ t('common.loading') }}
          </div>

          <div
            v-else-if="supportedProviders.length === 0"
            class="rounded-surface border border-gray-200 bg-white px-5 py-4 text-center text-sm text-gray-500 dark:border-dark-800 dark:bg-dark-900 dark:text-dark-400 sm:col-span-2 lg:col-span-3"
          >
            {{ homeMarketplaceError ? t('home.providers.unavailable') : t('home.providers.empty') }}
          </div>

          <!-- 管理员配置了首页展示模型时，按 OpenRouter Featured Models 风格渲染单模型卡片 -->
          <template v-else-if="featuredModels.length > 0">
            <article
              v-for="featured in featuredModels"
              :key="featured.model.id"
              class="rounded-surface border border-gray-200 bg-white p-6 shadow-sm ring-1 ring-transparent transition duration-layout hover:-translate-y-1 hover:border-black/20 hover:shadow-[0_12px_32px_rgba(0,0,0,0.1)] focus-within:border-black/20 dark:border-dark-800 dark:bg-dark-900 dark:hover:border-dark-600 dark:hover:shadow-[0_12px_32px_rgba(0,0,0,0.4)]"
            >
              <div class="flex items-start gap-4">
                <!-- 图标与模型广场保持一致：模型图标体系 + 白底圆角方形 -->
                <span class="flex h-12 w-12 shrink-0 items-center justify-center rounded-surface border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-950">
                  <ModelIcon :model="featured.model.id" size="28px" />
                </span>
                <div class="min-w-0 flex-1">
                  <h3 class="truncate text-lg font-semibold text-gray-950 dark:text-white">
                    {{ featured.model.display_name || featured.model.id }}
                  </h3>
                  <p class="truncate text-sm text-gray-500 dark:text-dark-400">
                    {{ t('home.featured.byProvider', { provider: homeProviderCategory(featured.group).label }) }}
                  </p>
                </div>
              </div>
              <!-- 左下角展示相对官方价的折扣，右下角留空；无折扣数据时整块底部区域不渲染 -->
              <div v-if="featured.discountOff" class="mt-5 border-t border-gray-200 pt-5 dark:border-dark-800">
                <p class="text-sm font-semibold tabular-nums text-gray-950 dark:text-white">
                  {{ featured.discountOff }}
                </p>
              </div>
            </article>
          </template>

          <template v-else>
            <article
              v-for="provider in supportedProviders.slice(0, 6)"
              :key="provider.key"
              class="rounded-surface border border-gray-200 bg-white p-6 shadow-sm ring-1 ring-transparent transition duration-layout hover:-translate-y-1 hover:border-black/20 hover:shadow-[0_12px_32px_rgba(0,0,0,0.1)] focus-within:border-black/20 dark:border-dark-800 dark:bg-dark-900 dark:hover:border-dark-600 dark:hover:shadow-[0_12px_32px_rgba(0,0,0,0.4)]"
            >
              <div class="flex items-start gap-4">
                <span
                  class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-950"
                  :class="providerIconWrapClass(provider)"
                >
                  <ProviderIcon :brand="provider.iconBrand" size="22px" />
                </span>
                <div class="min-w-0 flex-1">
                  <h3 class="truncate text-lg font-semibold text-gray-950 dark:text-white">
                    {{ provider.label }}
                  </h3>
                  <p class="text-sm text-gray-500 dark:text-dark-400">
                    {{ provider.groupCount }} {{ t('home.providers.groups') }}
                  </p>
                </div>
              </div>
              <div class="mt-5 border-t border-gray-200 pt-5 dark:border-dark-800">
                <div class="flex items-end justify-between gap-4">
                  <div>
                    <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('home.providers.modelCount') }}</p>
                    <p class="mt-1 text-lg font-semibold text-gray-950 dark:text-white">
                      {{ provider.modelCount }}
                    </p>
                  </div>
                  <p
                    v-if="provider.officialPriceRatio"
                    class="max-w-[180px] text-right text-sm font-semibold text-emerald-600 dark:text-emerald-300"
                  >
                    {{ formatOfficialPriceRatio(provider.officialPriceRatio) }}
                  </p>
                  <p v-else class="text-sm font-medium text-primary-600 dark:text-primary-300">
                    {{ t('home.providers.supported') }}
                  </p>
                </div>
              </div>
            </article>
          </template>
        </div>
      </section>

      <section class="mx-auto mt-16 max-w-7xl px-2 sm:px-0">
        <div class="grid gap-8 md:grid-cols-3">
          <article
            v-for="step in homeSteps"
            :key="step.key"
            class="flex min-h-[190px] flex-col"
          >
            <div class="flex items-center gap-3">
              <span class="flex h-8 w-8 items-center justify-center rounded-full bg-primary-50 text-base font-semibold text-primary-600 dark:bg-primary-500/10 dark:text-primary-300">
                {{ step.index }}
              </span>
              <h2 class="text-lg font-semibold tracking-tight text-gray-950 dark:text-white">{{ step.title }}</h2>
            </div>
            <p class="mt-4 max-w-sm text-sm leading-6 text-gray-600 dark:text-dark-300">{{ step.description }}</p>

            <div v-if="step.key === 'signup'" class="mt-8">
              <div class="flex items-center gap-3 text-primary-500">
                <Icon name="user" size="md" :stroke-width="1.8" />
                <div class="space-y-1.5">
                  <div class="h-1.5 w-7 rounded-full bg-primary-100 dark:bg-primary-400/20"></div>
                  <div class="h-1.5 w-20 rounded-full bg-primary-100 dark:bg-primary-400/20"></div>
                </div>
              </div>
              <div class="mt-4 grid max-w-[156px] grid-cols-3 gap-3">
                <span class="flex h-10 w-10 items-center justify-center rounded-control bg-white/90 shadow-sm ring-1 ring-gray-100 dark:bg-dark-950 dark:ring-dark-800">
                  <ProviderIcon brand="Google" size="20px" />
                </span>
                <span class="flex h-10 w-10 items-center justify-center rounded-control bg-white/90 text-gray-800 shadow-sm ring-1 ring-gray-100 dark:bg-dark-950 dark:text-gray-100 dark:ring-dark-800">
                  <GitHubMark class="h-5 w-5" />
                </span>
                <span class="flex h-10 w-10 items-center justify-center rounded-control bg-white/90 text-primary-500 shadow-sm ring-1 ring-gray-100 dark:bg-dark-950 dark:ring-dark-800">
                  <Icon name="mail" size="md" :stroke-width="1.8" />
                </span>
              </div>
            </div>

            <div v-else-if="step.key === 'browse'" class="mt-auto max-w-[270px] pt-6">
              <div class="flex items-center gap-3 text-primary-500">
                <Icon name="grid" size="md" :stroke-width="1.8" />
                <div class="grid flex-1 grid-cols-4 gap-2">
                  <div class="h-1 rounded-full bg-primary-100 dark:bg-primary-400/20"></div>
                  <div class="h-1 rounded-full bg-primary-100 dark:bg-primary-400/20"></div>
                  <div class="h-1 rounded-full bg-primary-100 dark:bg-primary-400/20"></div>
                  <div class="h-1 rounded-full bg-primary-100 dark:bg-primary-400/20"></div>
                </div>
              </div>
              <div class="mt-4 space-y-2">
                <div class="flex items-center gap-2 rounded-control bg-white/90 px-3 py-2 text-gray-700 shadow-sm ring-1 ring-gray-100 dark:bg-dark-950 dark:text-dark-200 dark:ring-dark-800">
                  <span class="w-14 text-xs font-medium">Claude</span>
                  <span class="h-2 flex-1 rounded-full bg-primary-100 dark:bg-primary-400/20"></span>
                  <span class="h-2 w-12 rounded-full bg-primary-100 dark:bg-primary-400/20"></span>
                </div>
                <div class="flex items-center gap-2 rounded-control bg-white/90 px-3 py-2 text-gray-700 shadow-sm ring-1 ring-gray-100 dark:bg-dark-950 dark:text-dark-200 dark:ring-dark-800">
                  <span class="w-14 text-xs font-medium">GPT</span>
                  <span class="h-2 flex-1 rounded-full bg-primary-100 dark:bg-primary-400/20"></span>
                  <span class="h-2 w-12 rounded-full bg-primary-100 dark:bg-primary-400/20"></span>
                </div>
              </div>
            </div>

            <div v-else class="mt-8 max-w-[270px]">
              <div class="flex items-center gap-3 text-primary-500">
                <Icon name="key" size="md" :stroke-width="1.8" />
                <div class="flex-1 rounded-control bg-white/90 px-3 py-2 font-mono text-xs text-gray-600 shadow-sm ring-1 ring-gray-100 dark:bg-dark-950 dark:text-dark-300 dark:ring-dark-800">
                  TOKENFLUX_API_KEY
                </div>
              </div>
              <div class="mt-3 rounded-control bg-white/90 px-3 py-2 font-mono text-sm tracking-[0.2em] text-gray-950 shadow-sm ring-1 ring-gray-100 dark:bg-dark-950 dark:text-white dark:ring-dark-800">
                ••••••••••••••••
              </div>
            </div>
          </article>
        </div>
      </section>

      <!-- CTA -->
      <section class="mx-auto mt-24 max-w-3xl text-center">
        <h2 class="text-3xl font-bold tracking-tight text-gray-950 dark:text-white sm:text-4xl">
          {{ t('home.cta.title') }}
        </h2>
        <p class="mx-auto mt-4 max-w-xl text-base leading-7 text-gray-600 dark:text-dark-300">
          {{ t('home.cta.description') }}
        </p>
        <div class="mt-8">
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-[44px] min-w-[180px] items-center justify-center gap-2 rounded-control bg-primary-600 px-8 py-3 text-sm font-semibold text-white transition hover:bg-primary-700"
          >
            {{ isAuthenticated ? t('home.goToDashboard') : t('home.cta.button') }}
            <Icon name="arrowRight" size="sm" :stroke-width="2" />
          </router-link>
        </div>
      </section>
    </main>

    <footer class="relative z-10 border-t border-gray-200 bg-white/90 px-6 py-12 backdrop-blur dark:border-dark-800 dark:bg-dark-950/90">
      <div class="mx-auto max-w-7xl">
        <div class="grid grid-cols-2 gap-x-8 gap-y-10 sm:grid-cols-3 lg:flex lg:justify-between lg:gap-8">
          <!-- Brand -->
          <div class="col-span-2 sm:col-span-3 lg:col-auto lg:max-w-[240px] lg:shrink-0">
            <div class="flex items-center gap-2.5">
              <span class="h-8 w-8 shrink-0 overflow-hidden rounded-control shadow-sm">
                <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
              </span>
              <span class="text-sm font-semibold text-gray-950 dark:text-white">{{ siteName }}</span>
            </div>
            <p class="mt-4 text-sm text-gray-500 dark:text-dark-400">
              &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
            </p>
            <p
              v-for="(line, index) in footerTextLines"
              :key="index"
              class="mt-1 text-xs text-gray-400 dark:text-dark-500"
            >
              {{ line }}
            </p>
          </div>

          <!-- Link columns -->
          <div v-for="column in footerColumns" :key="column.title" class="lg:min-w-[140px]">
            <h3 class="text-sm font-semibold text-gray-950 dark:text-white">{{ column.title }}</h3>
            <ul class="mt-4 space-y-2.5">
              <li v-for="link in column.links" :key="link.label">
                <router-link
                  v-if="link.url.startsWith('/')"
                  :to="link.url"
                  class="text-sm text-gray-500 transition hover:text-gray-950 dark:text-dark-400 dark:hover:text-white"
                >
                  {{ link.label }}
                </router-link>
                <a
                  v-else
                  :href="link.url"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-sm text-gray-500 transition hover:text-gray-950 dark:text-dark-400 dark:hover:text-white"
                >
                  {{ link.label }}
                </a>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </footer>
    </template>
  </div>
</template>

<script setup lang="ts">
import { vContentReveal } from '@/directives/contentReveal'
import { useRoute as useMotionRoute } from 'vue-router'
const motionRoute = useMotionRoute()

import MotionTransition from '@/components/common/MotionTransition.vue'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import GitHubMark from '@/components/auth/GitHubMark.vue'
import GoogleOneTap from '@/components/auth/GoogleOneTap.vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import { useVisualTheme } from '@/composables/useVisualTheme'
import ProviderIcon from '@/components/common/ProviderIcon.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { getMarketplaceModels, getMarketplaceStats } from '@/api/marketplace'
import type { MarketplaceGroup, MarketplaceModel, MarketplaceStats } from '@/types'
import { sanitizeUrl } from '@/utils/url'
import { hasAcceptedLoginAgreement } from '@/utils/loginAgreement'
import { isGoogleOneTapEligible, isGoogleOneTapOriginSupported } from '@/utils/googleIdentity'
import {
  providerBrandDisplayName,
  providerBrandFilterKey,
  resolveProviderBrand,
  resolveProviderBrandKey,
} from '@/utils/providerBrand'

type HomeStatsKey = 'today-tokens' | 'total-tokens' | 'total-users' | 'supported-models'
type HomeStatsIcon = 'bolt' | 'database' | 'users' | 'grid'
type HomeStepIcon = 'userPlus' | 'grid' | 'key'
type HomeStatFormat = 'compact' | 'number'

interface HomeProviderCategory {
  key: string
  label: string
  iconBrand: string
}

interface HomeProviderSummary extends HomeProviderCategory {
  modelCount: number
  groupCount: number
  officialPriceRatio?: number
  sortOrder: number
  firstIndex: number
}

interface HomeStatsCard {
  key: HomeStatsKey
  label: string
  value: string
  icon: HomeStatsIcon
  iconWrapClass: string
  iconClass: string
}

interface HomeStep {
  key: string
  index: number
  title: string
  description: string
  icon: HomeStepIcon
}

interface HomeProviderCloudIcon {
  brand: string
  left: string
  top: string
  opacity: number
  scale: number
}

// 首页精选卡片：单个模型及其所属分组（分组提供品牌与折扣上下文）
interface HomeFeaturedModel {
  model: MarketplaceModel
  group: MarketplaceGroup
  // 相对官方价的折扣文案（如 "95.6% off"），分组无有效折扣时为 null
  discountOff: string | null
}

const { t, locale } = useI18n()
const { visualTheme } = useVisualTheme()

const authStore = useAuthStore()
const appStore = useAppStore()

// 站点设置直接读取已注入或已缓存的公开配置。
const siteName = computed(() => appStore.siteName || 'ByteSeek')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const currentLanguage = computed(() => String(locale.value).toLowerCase().startsWith('zh') ? 'zh' : 'en')
const numberLocale = computed(() => currentLanguage.value === 'zh' ? 'zh-CN' : 'en-US')
const homeHeroTitle = computed(() => {
  const settings = appStore.cachedPublicSettings
  return localizedHomeCopy(
    settings?.site_title_zh,
    settings?.site_title_en,
    t('home.heroTitle')
  )
})
const homeHeroSubtitle = computed(() => {
  const settings = appStore.cachedPublicSettings
  return localizedHomeCopy(
    settings?.site_subtitle_zh,
    settings?.site_subtitle_en,
    t('home.heroDescription')
  )
})

// 自定义首页支持 URL iframe 和 HTML 两种模式。
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')

const googleOneTapClientID = computed(
  () => appStore.cachedPublicSettings?.google_oauth_client_id || ''
)
const googleOneTapEligible = computed(() => {
  const settings = appStore.cachedPublicSettings
  if (!settings) return false
  const agreementEnabled = settings.login_agreement_enabled === true
  return isGoogleOneTapEligible({
    publicSettingsLoaded: appStore.publicSettingsLoaded,
    isAuthenticated: isAuthenticated.value,
    oneTapEnabled: settings.google_one_tap_enabled === true,
    clientID: googleOneTapClientID.value,
    backendModeEnabled: settings.backend_mode_enabled,
    tencentCaptchaEnabled: settings.tencent_captcha_enabled === true,
    aliyunCaptchaEnabled: settings.aliyun_captcha_enabled === true,
    loginAgreementEnabled: agreementEnabled,
    loginAgreementAccepted: !agreementEnabled || hasAcceptedLoginAgreement(settings.login_agreement_revision || ''),
    originSupported: isGoogleOneTapOriginSupported()
  })
})

const currentYear = computed(() => new Date().getFullYear())

// 底栏:管理员配置的链接分组 + 内置"快速链接"列;附加文本按行渲染
const footerTextLines = computed<string[]>(() => {
  const raw = appStore.cachedPublicSettings?.footer_text || ''
  return raw.split('\n').map(line => line.trim()).filter(Boolean)
})

const footerColumns = computed(() => {
  const configured = (appStore.cachedPublicSettings?.footer_links || [])
    .filter(group => group.title && Array.isArray(group.links) && group.links.length > 0)
    .map(group => ({
      title: group.title,
      links: group.links.filter(link => link.label && link.url),
    }))
    .filter(group => group.links.length > 0)

  // 管理员已配置分组时以配置为准;未配置时回退到内置"快速链接"列
  if (configured.length > 0) {
    return configured
  }

  const quickLinks: Array<{ label: string; url: string }> = [
    { label: t('home.nav.models'), url: '/models' },
    { label: t('keyUsage.title'), url: '/key-usage' },
  ]
  if (docUrl.value) {
    quickLinks.push({ label: t('home.docs'), url: docUrl.value })
  }

  return [{ title: t('home.footer.quickLinks'), links: quickLinks }]
})

// 服务商图标滚动条:图标列表复制一份实现无缝循环
const homeMarqueeBrands = computed(() => {
  const brands = homeProviderVisuals.value.slice(0, 20)
  return [...brands, ...brands]
})

// 黑色走马灯随语言切换文案，单轮不重复；模板复制轨道保证无缝循环。
const homeMarqueeKeywords = computed(() => {
  const keywords = currentLanguage.value === 'zh'
    ? [
        '一站接入',
        '统一管理',
        '灵活切换',
        '智能中枢',
        '透明计费',
        '智能分发',
        '接口统一',
        '全链加速',
        '企业稳定',
      ]
    : [
        'ONE-STOP ACCESS',
        'UNIFIED MANAGEMENT',
        'FLEXIBLE SWITCHING',
        'INTELLIGENT HUB',
        'TRANSPARENT BILLING',
        'SMART DISTRIBUTION',
        'UNIFIED INTERFACE',
        'FULL-CHAIN ACCELERATION',
        'ENTERPRISE RELIABILITY',
      ]
  return keywords
})

const marketplaceGroups = ref<MarketplaceGroup[]>([])
const homeStats = ref<MarketplaceStats | null>(null)
const homeMarketplaceLoading = ref(true)
const homeStatsLoading = ref(true)
const homeMarketplaceError = ref(false)
const homeStatsError = ref(false)
const homeMarketplaceButtonIconIndex = ref(0)
let homeMarketplaceButtonIconTimer: number | null = null
const homeAnimatedStats = ref<Record<HomeStatsKey, number>>({
  'today-tokens': 0,
  'total-tokens': 0,
  'total-users': 0,
  'supported-models': 0,
})
const homeAnimatedStatKeys = new Set<HomeStatsKey>()
const homeStatAnimationFrames = new Map<HomeStatsKey, number>()
const homeStatAnimationDurationMs = 3200

const providerVisualFallbacks = [
  'Google',
  'Meta',
  'Gemini',
  'OpenAI',
  'Qwen',
  'DeepSeek',
  'Mistral',
  'Moonshot',
  'Claude',
  'xAI',
  'Antigravity',
  'Zhipu',
  'Cohere',
  'Perplexity',
  'Minimax',
  'Doubao',
  'Baidu',
  'Tencent',
  'Cloudflare',
  'OpenRouter',
]

const providerCloudLayout = [
  { left: '7%', top: '13%', opacity: 0.72, scale: 0.94 },
  { left: '25%', top: '13%', opacity: 0.8, scale: 0.94 },
  { left: '43%', top: '13%', opacity: 0.9, scale: 1 },
  { left: '62%', top: '13%', opacity: 0.8, scale: 0.94 },
  { left: '81%', top: '13%', opacity: 0.72, scale: 0.94 },
  { left: '17%', top: '35%', opacity: 0.86, scale: 0.98 },
  { left: '35%', top: '35%', opacity: 0.92, scale: 1 },
  { left: '53%', top: '35%', opacity: 0.96, scale: 1.04 },
  { left: '72%', top: '35%', opacity: 0.9, scale: 0.98 },
  { left: '91%', top: '35%', opacity: 0.76, scale: 0.94 },
  { left: '7%', top: '57%', opacity: 0.82, scale: 0.94 },
  { left: '25%', top: '57%', opacity: 0.9, scale: 0.98 },
  { left: '43%', top: '57%', opacity: 1, scale: 1.06 },
  { left: '62%', top: '57%', opacity: 0.9, scale: 0.98 },
  { left: '81%', top: '57%', opacity: 0.82, scale: 0.94 },
  { left: '17%', top: '79%', opacity: 0.72, scale: 0.94 },
  { left: '35%', top: '79%', opacity: 0.78, scale: 0.96 },
  { left: '53%', top: '79%', opacity: 0.82, scale: 0.98 },
  { left: '72%', top: '79%', opacity: 0.78, scale: 0.96 },
  { left: '91%', top: '79%', opacity: 0.66, scale: 0.92 },
  { left: '7%', top: '98%', opacity: 0.6, scale: 0.9 },
  { left: '25%', top: '98%', opacity: 0.66, scale: 0.92 },
  { left: '43%', top: '98%', opacity: 0.7, scale: 0.94 },
  { left: '62%', top: '98%', opacity: 0.66, scale: 0.92 },
  { left: '81%', top: '98%', opacity: 0.6, scale: 0.9 },
  { left: '99%', top: '98%', opacity: 0.48, scale: 0.86 },
] as const

// 图标只出现一次，并沿主轨道均匀分布，避免同一模型品牌重复堆叠。
const providerOrbitLayout = [
  { left: '50%', top: '1%' },
  { left: '66%', top: '4%' },
  { left: '80%', top: '12%' },
  { left: '91%', top: '24%' },
  { left: '98%', top: '40%' },
  { left: '99%', top: '57%' },
  { left: '93%', top: '73%' },
  { left: '82%', top: '86%' },
  { left: '67%', top: '95%' },
  { left: '50%', top: '99%' },
  { left: '33%', top: '95%' },
  { left: '18%', top: '86%' },
  { left: '7%', top: '73%' },
  { left: '1%', top: '57%' },
  { left: '2%', top: '40%' },
  { left: '9%', top: '24%' },
  { left: '20%', top: '12%' },
  { left: '34%', top: '4%' },
  { left: '72%', top: '50%' },
  { left: '28%', top: '50%' },
] as const

const totalModelCount = computed(() =>
  marketplaceGroups.value.reduce((total, group) => total + group.models.length, 0)
)

// 管理员在「系统设置 - 通用设置 - 首页模型展示」配置的模型 ID 列表（公开设置注入或接口返回）
const homeFeaturedModelIds = computed<string[]>(() => {
  const configured = appStore.cachedPublicSettings?.home_featured_models
  return Array.isArray(configured) ? configured : []
})

// 按配置顺序在市场分组中解析模型，解析不到的 ID 直接跳过
const featuredModels = computed<HomeFeaturedModel[]>(() => {
  const resolved: HomeFeaturedModel[] = []
  for (const modelId of homeFeaturedModelIds.value) {
    for (const group of marketplaceGroups.value) {
      const model = group.models.find(item => item.id === modelId)
      if (model) {
        resolved.push({
          model,
          group,
          discountOff: formatFeaturedDiscountOff(group.official_price_ratio),
        })
        break
      }
    }
  }
  return resolved
})

const homeStatAnimationTargets = computed<Record<HomeStatsKey, number | null>>(() => ({
  'today-tokens': homeStatsLoading.value ? null : normalizedHomeStatTarget(homeStats.value?.today_tokens),
  'total-tokens': homeStatsLoading.value ? null : normalizedHomeStatTarget(homeStats.value?.total_tokens),
  'total-users': homeStatsLoading.value ? null : normalizedHomeStatTarget(homeStats.value?.total_users),
  'supported-models': homeMarketplaceLoading.value ? null : normalizedHomeStatTarget(totalModelCount.value),
}))

const supportedProviders = computed<HomeProviderSummary[]>(() => {
  const summaries = new Map<string, HomeProviderSummary>()
  const sortedGroups = [...marketplaceGroups.value].sort((left, right) => {
    const sortDiff = (left.sort_order ?? 0) - (right.sort_order ?? 0)
    if (sortDiff !== 0) {
      return sortDiff
    }
    return left.id - right.id
  })

  sortedGroups.forEach((group, index) => {
    const modelCount = group.models.length
    if (modelCount === 0) {
      return
    }

    const category = homeProviderCategory(group)
    const existing = summaries.get(category.key)
    const ratio = validOfficialPriceRatio(group.official_price_ratio)
    if (!existing) {
      summaries.set(category.key, {
        ...category,
        modelCount,
        groupCount: 1,
        officialPriceRatio: ratio ?? undefined,
        sortOrder: group.sort_order ?? 0,
        firstIndex: index,
      })
      return
    }

    existing.modelCount += modelCount
    existing.groupCount += 1
    existing.sortOrder = Math.min(existing.sortOrder, group.sort_order ?? 0)
    existing.firstIndex = Math.min(existing.firstIndex, index)
    if (ratio && (!existing.officialPriceRatio || ratio < existing.officialPriceRatio)) {
      existing.officialPriceRatio = ratio
    }
  })

  return [...summaries.values()].sort((left, right) => {
    const priorityDiff = homeProviderPriority(left.key) - homeProviderPriority(right.key)
    if (priorityDiff !== 0) {
      return priorityDiff
    }
    const sortDiff = left.sortOrder - right.sortOrder
    if (sortDiff !== 0) {
      return sortDiff
    }
    return left.firstIndex - right.firstIndex
  })
})

const homeStatsCards = computed<HomeStatsCard[]>(() => [
  {
    key: 'today-tokens',
    label: t('home.stats.todayTokens'),
    value: formatAnimatedHomeStat('today-tokens', homeStatAnimationTargets.value['today-tokens'], homeStatsLoading.value),
    icon: 'bolt',
    iconWrapClass: 'bg-sky-100 dark:bg-sky-500/15',
    iconClass: 'text-sky-600 dark:text-sky-300',
  },
  {
    key: 'total-tokens',
    label: t('home.stats.totalTokens'),
    value: formatAnimatedHomeStat('total-tokens', homeStatAnimationTargets.value['total-tokens'], homeStatsLoading.value),
    icon: 'database',
    iconWrapClass: 'bg-emerald-100 dark:bg-emerald-500/15',
    iconClass: 'text-emerald-600 dark:text-emerald-300',
  },
  {
    key: 'total-users',
    label: t('home.stats.totalUsers'),
    value: formatAnimatedHomeStat('total-users', homeStatAnimationTargets.value['total-users'], homeStatsLoading.value, 'number'),
    icon: 'users',
    iconWrapClass: 'bg-violet-100 dark:bg-violet-500/15',
    iconClass: 'text-violet-600 dark:text-violet-300',
  },
  {
    key: 'supported-models',
    label: t('home.stats.supportedModels'),
    value: formatAnimatedHomeStat('supported-models', homeStatAnimationTargets.value['supported-models'], homeMarketplaceLoading.value, 'number'),
    icon: 'grid',
    iconWrapClass: 'bg-primary-100 dark:bg-primary-500/15',
    iconClass: 'text-primary-600 dark:text-primary-300',
  },
])

const homeProviderVisuals = computed(() => {
  const brands = supportedProviders.value.map(provider => provider.iconBrand)
  return mergeProviderVisualBrands(brands)
})

const homeMarketplaceButtonBrands = computed(() => supportedProviders.value.map(provider => provider.iconBrand))

const homeMarketplaceButtonBrand = computed(() => {
  const brands = homeMarketplaceButtonBrands.value
  if (brands.length === 0) {
    return ''
  }
  return brands[homeMarketplaceButtonIconIndex.value % brands.length]
})

const homeProviderCloudIcons = computed<HomeProviderCloudIcon[]>(() => {
  const brands = homeProviderVisuals.value
  return providerCloudLayout.map((layout, index) => ({
    brand: brands[index % brands.length],
    ...layout,
  }))
})

const homeRouteProviderBrands = computed(() => homeProviderVisuals.value.slice(0, 3))

const homeOrbitNodes = computed(() =>
  homeProviderVisuals.value.slice(0, providerOrbitLayout.length).map((brand, index) => ({
    brand,
    ...providerOrbitLayout[index],
  })),
)

const homeRouteLabel = computed(() => {
  return 'OpenAI/GPT-5.4'
})

const homeSteps = computed<HomeStep[]>(() => [
  {
    key: 'signup',
    index: 1,
    title: t('home.steps.signup.title'),
    description: t('home.steps.signup.description'),
    icon: 'userPlus',
  },
  {
    key: 'browse',
    index: 2,
    title: t('home.steps.browse.title'),
    description: t('home.steps.browse.description'),
    icon: 'grid',
  },
  {
    key: 'api-key',
    index: 3,
    title: t('home.steps.apiKey.title'),
    description: t('home.steps.apiKey.description'),
    icon: 'key',
  },
])

function homeProviderCategory(group: MarketplaceGroup): HomeProviderCategory {
  const brandSource = group.display_brand?.trim() || group.name.trim()
  const brandKey = resolveProviderBrandKey(brandSource)
  if (brandKey && brandKey !== 'unknown') {
    return homeProviderCategoryFromBrand(brandKey, brandSource)
  }

  const fallbackLabel = brandSource
  return {
    key: providerBrandFilterKey(fallbackLabel),
    label: fallbackLabel,
    iconBrand: fallbackLabel,
  }
}

function homeProviderCategoryFromBrand(brandKey: string, source: string): HomeProviderCategory {
  switch (brandKey) {
    case 'anthropic':
      return { key: 'claude', label: t('home.providers.claude'), iconBrand: 'Claude' }
    case 'openai':
      return { key: 'gpt', label: t('home.providers.gpt'), iconBrand: 'OpenAI' }
    case 'google':
      return { key: 'gemini', label: t('home.providers.gemini'), iconBrand: 'Gemini' }
    default: {
      const label = providerBrandDisplayName(source)
      return { key: brandKey || providerBrandFilterKey(source), label, iconBrand: label }
    }
  }
}

function homeProviderPriority(key: string): number {
  const priorities = ['claude', 'gpt', 'deepseek', 'gemini', 'antigravity']
  const index = priorities.indexOf(key)
  return index === -1 ? priorities.length : index
}

function validOfficialPriceRatio(value?: number): number | null {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : null
}

function formatOfficialPriceRatio(ratio: number): string {
  const discount = new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
  }).format(ratio * 10)

  return t('marketplace.officialPriceDiscount', { discount })
}

// 相对官方价的折扣百分比（分组级），如 0.044 返回 "95.6% off"；
// 无有效倍率或价格不低于官方价时返回 null，卡片底部整块不渲染
function formatFeaturedDiscountOff(ratio?: number): string | null {
  const valid = validOfficialPriceRatio(ratio)
  if (valid === null || valid >= 1) {
    return null
  }
  const percent = new Intl.NumberFormat(undefined, {
    maximumFractionDigits: 1,
  }).format((1 - valid) * 100)
  return t('home.featured.discountOff', { percent })
}

function formatAnimatedHomeStat(
  key: HomeStatsKey,
  target: number | null,
  loading: boolean,
  format: HomeStatFormat = 'compact'
): string {
  if (loading) {
    return '...'
  }
  if (target === null) {
    return '-'
  }

  const value = homeAnimatedStats.value[key]
  const formatted = format === 'compact'
    ? formatAnimatedCompactNumber(value, target)
    : formatWholeNumber(value)
  return `${formatted}+`
}

function formatMarketplaceStat(value: number): string {
  if (homeMarketplaceLoading.value) {
    return '...'
  }
  return new Intl.NumberFormat(numberLocale.value).format(value)
}

function formatAnimatedCompactNumber(value: number, target: number): string {
  const targetParts = new Intl.NumberFormat(numberLocale.value, {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).formatToParts(target)
  const compactPart = targetParts.find(part => part.type === 'compact')?.value ?? ''
  const scaledValue = compactPart ? scaleCompactValue(value, target) : value
  const decimalDigits = compactPart && targetParts.some(part => part.type === 'fraction') ? 1 : 0
  const numberText = new Intl.NumberFormat(numberLocale.value, {
    minimumFractionDigits: decimalDigits,
    maximumFractionDigits: decimalDigits,
    useGrouping: false,
  }).format(scaledValue)

  return `${numberText}${compactPart}`
}

function scaleCompactValue(value: number, target: number): number {
  const compactTargetNumber = Number(new Intl.NumberFormat(numberLocale.value, {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).formatToParts(target)
    .filter(part => part.type === 'integer' || part.type === 'decimal' || part.type === 'fraction')
    .map(part => part.value)
    .join(''))
  if (!Number.isFinite(compactTargetNumber) || compactTargetNumber <= 0) {
    return value
  }

  return value / (target / compactTargetNumber)
}

function formatWholeNumber(value: number): string {
  return new Intl.NumberFormat(numberLocale.value, {
    maximumFractionDigits: 0,
    useGrouping: false,
  }).format(Math.round(value))
}

function normalizedHomeStatTarget(value?: number): number | null {
  return typeof value === 'number' && Number.isFinite(value) ? Math.max(0, value) : null
}

function startHomeStatAnimation(key: HomeStatsKey, target: number) {
  homeAnimatedStatKeys.add(key)
  if (homeStatAnimationFrames.has(key)) {
    cancelAnimationFrame(homeStatAnimationFrames.get(key)!)
    homeStatAnimationFrames.delete(key)
  }

  const reduceMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false
  if (reduceMotion || target === 0) {
    homeAnimatedStats.value = { ...homeAnimatedStats.value, [key]: target }
    return
  }

  const startTime = performance.now()
  const tick = (now: number) => {
    const progress = Math.min((now - startTime) / homeStatAnimationDurationMs, 1)
    // 四次缓出配合更长时长，让计数器前段启动快，尾段明显慢下来。
    const easedProgress = 1 - Math.pow(1 - progress, 4)
    homeAnimatedStats.value = {
      ...homeAnimatedStats.value,
      [key]: target * easedProgress,
    }

    if (progress < 1) {
      homeStatAnimationFrames.set(key, requestAnimationFrame(tick))
      return
    }

    homeAnimatedStats.value = { ...homeAnimatedStats.value, [key]: target }
    homeStatAnimationFrames.delete(key)
  }

  homeStatAnimationFrames.set(key, requestAnimationFrame(tick))
}

watch(
  homeStatAnimationTargets,
  (targets) => {
    const statEntries = Object.entries(targets) as Array<[HomeStatsKey, number | null]>
    statEntries.forEach(([key, target]) => {
      if (target === null || homeAnimatedStatKeys.has(key)) {
        return
      }
      startHomeStatAnimation(key, target)
    })
  },
  { immediate: true }
)

watch(
  homeMarketplaceButtonBrands,
  (brands) => {
    if (brands.length === 0) {
      homeMarketplaceButtonIconIndex.value = 0
      return
    }
    homeMarketplaceButtonIconIndex.value %= brands.length
  },
  { immediate: true }
)

function localizedHomeCopy(zhText: string | undefined, enText: string | undefined, fallback: string): string {
  const primary = currentLanguage.value === 'zh' ? zhText : enText
  const secondary = currentLanguage.value === 'zh' ? enText : zhText
  return firstConfiguredText(primary, secondary, fallback)
}

function firstConfiguredText(...values: Array<string | undefined>): string {
  for (const value of values) {
    const normalized = value?.trim()
    if (normalized) {
      return normalized
    }
  }
  return ''
}

function mergeProviderVisualBrands(brands: string[]): string[] {
  const seen = new Set<string>()
  const merged: string[] = []

  ;[...brands, ...providerVisualFallbacks].forEach((brand) => {
    const normalizedBrand = brand.trim()
    if (!normalizedBrand || seen.has(normalizedBrand)) {
      return
    }
    seen.add(normalizedBrand)
    merged.push(normalizedBrand)
  })

  return merged
}

function providerIconWrapClass(provider: Pick<HomeProviderSummary, 'key' | 'iconBrand'>): string {
  if (provider.key === 'antigravity') {
    return 'bg-rose-50 text-rose-700 ring-rose-200 dark:bg-rose-500/15 dark:text-rose-200 dark:ring-rose-400/30'
  }
  return resolveProviderBrand(provider.iconBrand).iconWrapClass
}

async function fetchHomeMarketplace() {
  homeMarketplaceLoading.value = true
  homeMarketplaceError.value = false

  try {
    marketplaceGroups.value = await getMarketplaceModels()
  } catch (error) {
    console.error('Failed to load home marketplace models:', error)
    marketplaceGroups.value = []
    homeMarketplaceError.value = true
  } finally {
    homeMarketplaceLoading.value = false
  }
}

async function fetchHomeStats() {
  homeStatsLoading.value = true
  homeStatsError.value = false

  try {
    homeStats.value = await getMarketplaceStats()
  } catch (error) {
    console.error('Failed to load home marketplace stats:', error)
    homeStats.value = null
    homeStatsError.value = true
  } finally {
    homeStatsLoading.value = false
  }
}

onMounted(async () => {
  authStore.checkAuth()
  homeMarketplaceButtonIconTimer = window.setInterval(() => {
    if (homeMarketplaceButtonBrands.value.length <= 1) {
      return
    }
    homeMarketplaceButtonIconIndex.value += 1
  }, 1800)

  if (!appStore.publicSettingsLoaded) {
    try {
      await appStore.fetchPublicSettings()
    } catch (error) {
      console.error('Failed to load public settings:', error)
    }
  }

  if (!homeContent.value) {
    await Promise.all([fetchHomeMarketplace(), fetchHomeStats()])
  }
})

onUnmounted(() => {
  homeStatAnimationFrames.forEach(frameId => cancelAnimationFrame(frameId))
  homeStatAnimationFrames.clear()
  if (homeMarketplaceButtonIconTimer) {
    window.clearInterval(homeMarketplaceButtonIconTimer)
    homeMarketplaceButtonIconTimer = null
  }
})
</script>

<style scoped>
.home-marketplace-icon-enter-active,
.home-marketplace-icon-leave-active {
  transition: opacity var(--motion-layout) var(--motion-ease), transform var(--motion-layout) var(--motion-ease);
}

.home-marketplace-icon-enter-from {
  opacity: 0;
  transform: translateY(-70%);
}

.home-marketplace-icon-leave-to {
  opacity: 0;
  transform: translateY(70%);
}

/* 服务商图标无缝滚动条,两端用渐隐遮罩 */
.home-marquee {
  -webkit-mask-image: linear-gradient(to right, transparent, black 12%, black 88%, transparent);
  mask-image: linear-gradient(to right, transparent, black 12%, black 88%, transparent);
}

.home-marquee-track {
  animation: home-marquee-scroll 48s linear infinite;
}

.home-marquee:hover .home-marquee-track {
  animation-play-state: paused;
}

@keyframes home-marquee-scroll {
  0% {
    transform: translateX(0);
  }
  100% {
    transform: translateX(-50%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .home-marquee-track {
    animation: none;
  }
}

/* CTA 按钮：黑框 + 硬阴影 + 位移 */
.bh-home-cta {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  min-height: 48px;
  min-width: 180px;
  padding: 0.75rem 2rem;
  font-size: 0.95rem;
  font-weight: 800;
  border: 3px solid var(--bh-ink);
  box-shadow: var(--bh-shadow-sm);
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}

.bh-home-cta:hover {
  transform: translate(-3px, -3px);
  box-shadow: var(--bh-shadow-sm);
}

.bh-home-cta:active {
  transform: translate(3px, 3px);
  box-shadow: 1px 1px 0 0 var(--bh-shadow-ink);
}

.bh-home-cta-red {
  background: var(--bh-red);
  color: #ffffff;
}

.bh-home-cta-yellow {
  background: var(--bh-yellow);
  color: #141414;
}

.bh-home-cta-ink {
  background: #141414;
  color: #f4f0e6;
  border-color: #141414;
  box-shadow: var(--bh-shadow-sm);
}

/* 区块标题：蓝色菱形前缀 */
.bh-home-section-title {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 2rem;
  font-size: clamp(24px, 3.6vw, 34px);
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--bh-ink);
}

.bh-home-section-title::before {
  content: '';
  width: 20px;
  height: 20px;
  background: var(--bh-blue);
  transform: rotate(45deg);
  flex: none;
}

/* 卡片底部 CTA */
.bh-home-card-cta {
  margin-top: 1rem;
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.875rem;
  font-weight: 800;
  color: var(--bh-ink);
  text-decoration: none;
}

.bh-home-card-cta:hover {
  text-decoration: underline;
  text-decoration-thickness: 3px;
  text-underline-offset: 4px;
  text-decoration-color: var(--bh-red);
}

/* 卡片头部小几何 */
.bh-home-tri-sm {
  width: 0;
  height: 0;
  border-left: 11px solid transparent;
  border-right: 11px solid transparent;
  border-bottom: 19px solid var(--bh-red);
}

/* Hero 构成舞台：轨道动态表达多账号路由，结构与传统居中营销页区分开。 */
.bauhaus-home-stage {
  position: relative;
  min-height: 390px;
  overflow: hidden;
  isolation: isolate;
  border: 3px solid var(--bh-ink);
  background: var(--bh-paper);
  box-shadow: var(--bh-shadow-sm);
}

.bauhaus-home-stage-grid {
  position: absolute;
  inset: 0;
  z-index: -1;
  background-image:
    linear-gradient(rgba(20, 20, 20, 0.08) 1px, transparent 1px),
    linear-gradient(90deg, rgba(20, 20, 20, 0.08) 1px, transparent 1px);
  background-size: 28px 28px;
  opacity: 0.7;
}

.dark .bauhaus-home-stage-grid {
  background-image:
    linear-gradient(rgba(244, 240, 230, 0.1) 1px, transparent 1px),
    linear-gradient(90deg, rgba(244, 240, 230, 0.1) 1px, transparent 1px);
}

.bauhaus-home-stage-sun {
  position: absolute;
  top: 44px;
  right: 40px;
  width: 150px;
  height: 150px;
  border: 3px solid var(--bh-ink);
  border-radius: 50%; /* check-ui-allow: 包豪斯圆形或半圆装饰，不是控件圆角。 */
  background: var(--bh-yellow);
  animation: bh-stage-pulse 4.5s ease-in-out infinite;
}

.bauhaus-home-stage-ring {
  position: absolute;
  border: 3px solid var(--bh-blue);
  border-radius: 50%; /* check-ui-allow: 包豪斯圆形或半圆装饰，不是控件圆角。 */
  pointer-events: none;
}

.bauhaus-home-stage-ring-outer {
  top: 24px;
  right: 2px;
  width: 285px;
  height: 285px;
  animation: bh-stage-spin 18s linear infinite;
}

.bauhaus-home-stage-ring-inner {
  top: 66px;
  right: 44px;
  width: 200px;
  height: 200px;
  border-width: 2px;
  border-color: var(--bh-red);
  border-style: dashed;
  animation: bh-stage-spin-reverse 12s linear infinite;
}

/* 三种基础形在内轨道运行，模型图标在外轨道保持正向阅读。 */
.bauhaus-home-stage-geometry-orbit,
.bauhaus-home-stage-icon-cloud {
  position: absolute;
  top: 26px;
  right: 5px;
  width: 282px;
  height: 282px;
  border-radius: 50%; /* check-ui-allow: 包豪斯圆形或半圆装饰，不是控件圆角。 */
  pointer-events: none;
  animation: bh-stage-spin 22s linear infinite;
}

.bauhaus-home-stage-geometry-orbit {
  top: 69px;
  right: 48px;
  z-index: 2; /* check-ui-allow: 本地几何层叠，不属于全局浮层。 */
  width: 194px;
  height: 194px;
  animation-duration: 12s;
  animation-direction: reverse;
}

.bauhaus-home-stage-geometry {
  position: absolute;
  display: block;
  filter: drop-shadow(2px 2px 0 var(--bh-ink));
}

.bauhaus-home-stage-geometry-square {
  top: -10px;
  left: calc(50% - 10px);
  width: 20px;
  height: 20px;
  border: 2px solid var(--bh-ink);
  background: var(--bh-red);
}

.bauhaus-home-stage-geometry-triangle {
  right: -11px;
  bottom: 24px;
  width: 0;
  height: 0;
  border-right: 12px solid transparent;
  border-bottom: 22px solid var(--bh-yellow);
  border-left: 12px solid transparent;
}

.bauhaus-home-stage-geometry-circle {
  bottom: 21px;
  left: -9px;
  width: 21px;
  height: 21px;
  border: 2px solid var(--bh-ink);
  border-radius: 50%; /* check-ui-allow: 包豪斯圆形或半圆装饰，不是控件圆角。 */
  background: var(--bh-blue);
}

.bauhaus-home-stage-icon-cloud {
  z-index: 2; /* check-ui-allow: 本地几何层叠，不属于全局浮层。 */
}

.bauhaus-home-stage-icon-node {
  position: absolute;
  display: flex;
  width: 28px;
  height: 28px;
  align-items: center;
  justify-content: center;
  border: 2px solid var(--bh-ink);
  background: var(--bh-surface);
  box-shadow: 2px 2px 0 var(--bh-shadow-ink);
  transform: translate(-50%, -50%);
  animation: bh-stage-icon-counter-spin 22s linear infinite;
}

.bauhaus-home-stage-panel {
  position: absolute;
  bottom: 28px;
  left: 28px;
  z-index: 3; /* check-ui-allow: 本地几何层叠，不属于全局浮层。 */
  width: min(250px, calc(100% - 56px));
  padding: 1rem;
  border: 3px solid var(--bh-ink);
  background: var(--bh-surface);
  box-shadow: var(--bh-shadow-sm);
  color: inherit;
  text-decoration: none;
  cursor: pointer;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
  animation: bh-stage-panel-in 0.8s 0.28s ease-out backwards;
}

.bauhaus-home-stage-panel:hover {
  transform: translate(-3px, -3px);
  box-shadow: var(--bh-shadow);
}

.bauhaus-home-stage-panel:active {
  transform: translate(3px, 3px);
  box-shadow: 1px 1px 0 0 var(--bh-shadow-ink);
}

.bauhaus-home-stage-panel:focus-visible {
  outline: 3px solid var(--bh-blue);
  outline-offset: 3px;
}

.bauhaus-home-stage-live-dot {
  width: 10px;
  height: 10px;
  border: 2px solid var(--bh-ink);
  border-radius: 50%; /* check-ui-allow: 包豪斯圆形或半圆装饰，不是控件圆角。 */
  background: #10b981;
  animation: bh-stage-blink 1.8s ease-in-out infinite;
}

.bauhaus-home-stage-node {
  display: flex;
  width: 34px;
  height: 34px;
  align-items: center;
  justify-content: center;
  border: 2px solid var(--bh-ink);
  background: var(--bh-paper);
}

.bauhaus-home-stage-route-link {
  display: inline-flex;
  width: 34px;
  height: 34px;
  align-items: center;
  justify-content: center;
  border: 2px solid transparent;
  transition: transform 0.15s ease, background-color 0.15s ease;
}

.bauhaus-home-stage-route-link:hover {
  background: var(--bh-yellow);
  transform: translate(-2px, -2px);
}

.bauhaus-home-stage-route-link:active {
  transform: translate(2px, 2px);
}

.bauhaus-home-stage-stamp {
  position: absolute;
  top: 22px;
  left: 24px;
  z-index: 3; /* check-ui-allow: 本地几何层叠，不属于全局浮层。 */
  padding: 4px 8px;
  border: 2px solid var(--bh-ink);
  background: var(--bh-red);
  color: #fff;
  font-family: 'Geist Mono Variable', ui-monospace, monospace;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.12em;
}

.bauhaus-home-stage-stamp span {
  color: var(--bh-yellow);
}

@keyframes bh-stage-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@keyframes bh-stage-spin-reverse {
  from { transform: rotate(360deg); }
  to { transform: rotate(0deg); }
}

@keyframes bh-stage-icon-counter-spin {
  from { transform: translate(-50%, -50%) rotate(0deg); }
  to { transform: translate(-50%, -50%) rotate(-360deg); }
}

@keyframes bh-stage-pulse {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.05); }
}

@keyframes bh-stage-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.35; }
}

@keyframes bh-stage-panel-in {
  from { opacity: 0; transform: translateY(12px); }
  to { opacity: 1; transform: translateY(0); }
}

/* 黑带走马灯将关键信息做成高对比排版，而不是图标列表。 */
.bauhaus-home-marquee-word {
  padding-inline: clamp(1.1rem, 3vw, 2.8rem);
  color: #f4f0e6;
  font-family: 'Archivo Black', 'Plus Jakarta Sans Variable', system-ui, sans-serif;
  font-size: clamp(0.95rem, 2.1vw, 1.35rem);
  font-weight: 900;
  letter-spacing: 0.08em;
  white-space: nowrap;
}

.bauhaus-home-marquee-word-1 { color: var(--bh-yellow); }
.bauhaus-home-marquee-word-2 { color: #10b981; }
.bauhaus-home-marquee-word-3 { color: #78aaf0; }

.bauhaus-home-marquee-separator {
  color: var(--bh-red);
  font-size: 0.8rem;
}

/* 页脚三角 */
.bh-home-foot-tri {
  width: 0;
  height: 0;
  border-left: 8px solid transparent;
  border-right: 8px solid transparent;
  border-bottom: 14px solid var(--bh-yellow);
}

/* 模型广场按钮图标轮换过渡 */
.bauhaus-home-marketplace-icon-enter-active,
.bauhaus-home-marketplace-icon-leave-active {
  transition: opacity 220ms ease, transform 220ms ease;
}

.bauhaus-home-marketplace-icon-enter-from {
  opacity: 0;
  transform: translateY(-70%);
}

.bauhaus-home-marketplace-icon-leave-to {
  opacity: 0;
  transform: translateY(70%);
}

@media (max-width: 639px) {
  .bauhaus-home-stage {
    min-height: 310px;
  }

  .bauhaus-home-stage-sun {
    top: 34px;
    right: 24px;
    width: 112px;
    height: 112px;
  }

  .bauhaus-home-stage-ring-outer {
    top: 18px;
    right: -22px;
    width: 220px;
    height: 220px;
  }

  .bauhaus-home-stage-ring-inner {
    top: 52px;
    right: 14px;
    width: 155px;
    height: 155px;
  }

  .bauhaus-home-stage-icon-cloud {
    top: 21px;
    right: -19px;
    width: 218px;
    height: 218px;
  }

  .bauhaus-home-stage-geometry-orbit {
    top: 55px;
    right: 16px;
    width: 150px;
    height: 150px;
  }

  .bauhaus-home-stage-icon-node {
    width: 24px;
    height: 24px;
  }

  .bauhaus-home-stage-panel {
    bottom: 18px;
    left: 18px;
    width: min(228px, calc(100% - 36px));
    padding: 1rem;
  }

  .bauhaus-home-stage-stamp {
    top: 16px;
    left: 16px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .bauhaus-home-stage-sun,
  .bauhaus-home-stage-ring,
  .bauhaus-home-stage-geometry-orbit,
  .bauhaus-home-stage-icon-cloud,
  .bauhaus-home-stage-icon-node,
  .bauhaus-home-stage-live-dot,
  .bauhaus-home-stage-panel,
  .animate-bh-rise,
  .bauhaus-home-marketplace-icon-enter-active,
  .bauhaus-home-marketplace-icon-leave-active {
    animation: none !important;
    transition: none !important;
  }

}

/* 旧首页专属样式不影响 TokenFlux 分支和其它页面。 */
  /* 黑底标语标签（kicker） */
  .bh-home-kicker {
    @apply inline-block px-4 py-2 text-xs font-extrabold uppercase;
    letter-spacing: 0.28em;
    background: var(--bh-ink);
    color: var(--bh-paper);
  }

  /* 三原色条纹 */
  .bh-home-stripe {
    display: flex;
    height: 8px;
    width: 100%;
  }

  .bh-home-stripe > i {
    flex: 1;
    display: block;
  }

  .bh-home-stripe > i:nth-child(1) { background: var(--bh-red); }
  .bh-home-stripe > i:nth-child(2) { background: var(--bh-yellow); }
  .bh-home-stripe > i:nth-child(3) { background: var(--bh-blue); }

  /* 硬边框块（白底卡片，带交互位移） */
  .bh-home-block {
    @apply block bg-white dark:bg-dark-800;
    border: 3px solid var(--bh-ink);
    box-shadow: var(--bh-shadow);
    transition: transform 0.18s ease, box-shadow 0.18s ease;
  }

  a.bh-home-block:hover,
  .bh-home-block-hover:hover {
    transform: translate(-5px, -5px);
    box-shadow: var(--bh-shadow-sm);
  }

  /* 跑马灯黑带 */
  .bh-home-marquee {
    background: #141414;
    color: #f4f0e6;
    overflow: hidden;
    border-top: 3px solid var(--bh-ink);
    border-bottom: 3px solid var(--bh-ink);
    padding: 12px 0;
  }

  .bh-home-marquee-track {
    display: flex;
    width: max-content;
    align-items: center;
    animation: bhMarquee 26s linear infinite;
    will-change: transform;
  }

  /* 走马灯移动半个复制轨道，回到同一首词时不会出现跳帧。 */
  @keyframes bhMarquee {
    from { transform: translate3d(0, 0, 0); }
    to { transform: translate3d(-50%, 0, 0); }
  }

  .bh-home-marquee:hover .bh-home-marquee-track {
    animation-play-state: paused;
  }

  @media (prefers-reduced-motion: reduce) {
    .bh-home-marquee-track {
      animation: none;
    }
  }
.bh-home-marquee-copy { display: flex; align-items: center; flex: none; }
.bh-home-cta:focus-visible, .bh-home-card-cta:focus-visible { outline: 3px solid var(--bh-blue); outline-offset: 4px; }
.bh-home-cta-triangle { position: absolute; right: 15%; top: 1rem; width: 0; height: 0; border-inline: 2rem solid transparent; border-bottom: 3.5rem solid var(--bh-blue); transform: rotate(15deg); pointer-events: none; }
.animate-bh-rise { animation: bh-home-rise .6s ease-out both; }
@keyframes bh-home-rise { from { opacity: 0; transform: translateY(12px); } to { opacity: 1; transform: translateY(0); } }
@media (prefers-reduced-motion: reduce) {
 .animate-bh-rise { animation: none; }
 .bh-home-marquee-track { width: 100%; }
 .bh-home-marquee-copy { flex-wrap: wrap; justify-content: center; row-gap: 1rem; }
 .bh-home-marquee-copy[aria-hidden] { display: none; }
}

</style>
