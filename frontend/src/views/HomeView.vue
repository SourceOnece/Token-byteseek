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

  <!-- 默认首页：包豪斯构成 -->
  <div v-else class="ba-theme-shell relative flex min-h-screen flex-col overflow-hidden text-gray-950 dark:text-white">
    <div class="ba-theme-backdrop pointer-events-none fixed inset-0"></div>

    <!-- 顶部导航：纸色 + 黑色硬边 + 三原色条 -->
    <header class="relative z-20 border-b-[3px] border-gray-950 bg-bh-paper px-4 dark:border-dark-100 dark:bg-dark-900 sm:px-6">
      <div class="bh-stripe absolute inset-x-0 top-0 !h-1" aria-hidden="true"><i></i><i></i><i></i></div>
      <nav class="mx-auto flex h-14 max-w-7xl items-center justify-between gap-4 pt-1">
        <router-link to="/home" class="flex min-w-0 items-center gap-2.5">
          <span class="h-8 w-8 shrink-0 overflow-hidden border-2 border-gray-950 bg-white dark:border-dark-100">
            <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
          </span>
          <span class="truncate text-base font-extrabold tracking-tight text-gray-950 dark:text-white">{{ siteName }}<span class="text-bh-red">.</span></span>
        </router-link>

        <div class="flex items-center gap-2 sm:gap-3">
          <div class="hidden items-center gap-5 text-sm font-bold text-gray-800 dark:text-dark-100 md:flex">
            <router-link to="/models" class="bh-navlink">
              {{ t('home.nav.models') }}
            </router-link>
            <a
              v-if="docUrl"
              :href="docUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="bh-navlink"
            >
              {{ t('home.docs') }}
            </a>
          </div>

          <LocaleSwitcher />

          <button
            @click="toggleTheme"
            class="flex h-9 w-9 items-center justify-center text-gray-900 transition-colors hover:bg-bh-yellow hover:text-gray-950 dark:text-dark-100"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>

          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="bh-pressable inline-flex items-center gap-1.5 border-2 border-gray-950 bg-primary-600 py-1.5 pl-1.5 pr-3 text-xs font-extrabold text-white transition hover:bg-primary-700 dark:border-dark-100"
          >
            <UserAvatar
              :user-id="authStore.user?.id"
              :avatar-url="authStore.user?.avatar_url || ''"
              :alt="authStore.user?.username || authStore.user?.email || ''"
              size-class="h-5 w-5"
            />
            {{ t('home.dashboard') }}
          </router-link>
          <router-link
            v-else
            to="/login"
            class="inline-flex items-center border-2 border-gray-950 bg-bh-red px-4 py-2 text-xs font-extrabold text-white shadow-sm transition hover:translate-x-[-1px] hover:translate-y-[-1px] dark:border-dark-100"
          >
            {{ t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="relative z-10 flex-1 pb-0">
      <!-- ===== Hero：不对称构成 + 几何装饰 ===== -->
      <section class="relative mx-auto max-w-7xl px-4 pb-16 pt-16 sm:px-6 lg:px-8 lg:pb-24 lg:pt-24">
        <div class="relative grid items-center gap-14 lg:grid-cols-[minmax(0,0.92fr)_minmax(360px,1.08fr)] lg:gap-20">
          <div class="max-w-3xl">
            <span class="bh-kicker animate-bh-rise">{{ siteName }} · AI API GATEWAY</span>

            <h1 class="mt-6 animate-bh-rise text-5xl font-extrabold leading-[1.02] tracking-tighter text-gray-950 [animation-delay:0.08s] dark:text-white sm:text-6xl lg:text-7xl">
              {{ homeHeroTitle }}<span class="text-bh-red">.</span>
            </h1>

            <p class="mt-6 max-w-xl animate-bh-rise text-lg font-bold leading-8 text-gray-800 [animation-delay:0.16s] dark:text-dark-100">
              {{ homeHeroSubtitle }}
            </p>

            <div class="mt-10 flex flex-wrap items-center gap-5 animate-bh-rise [animation-delay:0.24s]">
              <router-link
                :to="isAuthenticated ? dashboardPath : '/login'"
                class="bh-cta bh-cta-red"
              >
                {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
                <Icon name="arrowRight" size="sm" :stroke-width="2.5" />
              </router-link>
              <router-link
                to="/models"
                class="bh-cta bh-cta-yellow"
              >
                {{ t('home.exploreMarketplace') }}
                <span class="relative flex h-5 w-5 items-center justify-center overflow-hidden">
                  <Transition name="home-marketplace-icon" mode="out-in">
                    <ProviderIcon
                      v-if="homeMarketplaceButtonBrand"
                      :key="homeMarketplaceButtonBrand"
                      :brand="homeMarketplaceButtonBrand"
                      size="18px"
                    />
                    <Icon v-else key="marketplace-fallback" name="sparkles" size="sm" />
                  </Transition>
                </span>
              </router-link>
            </div>
          </div>

          <!-- 独立构成舞台：三种基本形与多家模型供应商在轨道上运行。 -->
          <div class="home-stage">
            <div class="home-stage-grid"></div>
            <div class="home-stage-sun"></div>
            <div class="home-stage-ring home-stage-ring-outer"></div>
            <div class="home-stage-ring home-stage-ring-inner"></div>
            <div class="home-stage-geometry-orbit">
              <span class="home-stage-geometry home-stage-geometry-square"></span>
              <span class="home-stage-geometry home-stage-geometry-triangle"></span>
              <span class="home-stage-geometry home-stage-geometry-circle"></span>
            </div>
            <div class="home-stage-icon-cloud">
              <span
                v-for="node in homeOrbitNodes"
                :key="`orbit-${node.brand}`"
                class="home-stage-icon-node"
                :style="{ left: node.left, top: node.top }"
              >
                <ProviderIcon :brand="node.brand" size="16px" />
              </span>
            </div>
            <router-link
              to="/models"
              class="home-stage-panel"
              :aria-label="t('home.exploreMarketplace')"
              :title="t('home.exploreMarketplace')"
            >
              <div class="flex items-center justify-between border-b-2 border-gray-950 pb-3 dark:border-dark-100">
                <span class="font-mono text-[10px] font-extrabold uppercase tracking-[0.24em] text-gray-600 dark:text-dark-200">LIVE ROUTE</span>
                <span class="home-stage-live-dot"></span>
              </div>
              <p class="mt-4 truncate font-mono text-sm font-extrabold text-gray-950 dark:text-white">{{ homeRouteLabel }}</p>
              <div class="mt-5 flex items-center gap-2">
                <span v-for="brand in homeRouteProviderBrands" :key="`panel-${brand}`" class="home-stage-node">
                  <ProviderIcon :brand="brand" size="16px" />
                </span>
                <span class="home-stage-route-link ml-auto font-mono text-lg font-extrabold text-bh-red" aria-hidden="true">→</span>
              </div>
            </router-link>
            <span class="home-stage-stamp">24<span>/</span>7</span>
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
      <section class="bh-marquee relative z-10" aria-label="ByteSeek capabilities">
        <div class="bh-marquee-track">
          <!-- 轨道复制一份用于无缝滚动，单轮内每个关键词只出现一次。 -->
          <template v-for="copy in 2" :key="`marquee-copy-${copy}`">
            <template v-for="(keyword, index) in homeMarqueeKeywords" :key="`${copy}-${keyword}-${index}`">
              <span class="home-marquee-word" :class="`home-marquee-word-${index % 4}`">{{ keyword }}</span>
              <span class="home-marquee-separator" aria-hidden="true">◆</span>
            </template>
          </template>
        </div>
      </section>

      <!-- ===== 功能区：色块头卡片 ===== -->
      <section class="mx-auto max-w-7xl px-4 pt-20 sm:px-6 lg:px-8">
        <h2 class="bh-section-title">{{ t('home.features.unifiedGateway') }}</h2>

        <div class="grid gap-8 sm:grid-cols-2 xl:grid-cols-4">
          <!-- 卡片 1：统一网关 -->
          <article class="bh-block bh-block-hover group">
            <div class="flex items-center justify-between border-b-[3px] border-gray-950 bg-bh-red px-5 py-3.5 dark:border-dark-100">
              <h3 class="text-base font-extrabold text-white">{{ t('home.features.unifiedGateway') }}</h3>
              <span class="bh-head-shape h-5 w-5 border-2 border-gray-950 bg-bh-yellow transition-transform duration-300 group-hover:rotate-[135deg]"></span>
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
              <router-link to="/models" class="bh-card-cta">
                {{ t('home.features.browseAll') }}
                <Icon name="arrowRight" size="xs" :stroke-width="2.5" />
              </router-link>
            </div>
          </article>

          <!-- 卡片 2：多账号智能调度 -->
          <article class="bh-block bh-block-hover group">
            <div class="flex items-center justify-between border-b-[3px] border-gray-950 bg-bh-blue px-5 py-3.5 dark:border-dark-100">
              <h3 class="text-base font-extrabold text-white">{{ t('home.features.multiAccount') }}</h3>
              <span class="bh-head-shape h-5 w-5 rounded-full bg-bh-yellow transition-transform duration-300 group-hover:rotate-[135deg]"></span>
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
              <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="bh-card-cta">
                {{ t('home.features.learnMore') }}
                <Icon name="arrowRight" size="xs" :stroke-width="2.5" />
              </router-link>
            </div>
          </article>

          <!-- 卡片 3：额度用量 -->
          <article class="bh-block bh-block-hover group">
            <div class="flex items-center justify-between border-b-[3px] border-gray-950 bg-bh-yellow px-5 py-3.5 dark:border-dark-100">
              <h3 class="text-base font-extrabold text-gray-950">{{ t('home.features.balanceQuota') }}</h3>
              <span class="bh-head-shape bh-tri-sm transition-transform duration-300 group-hover:rotate-[135deg]"></span>
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
              <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="bh-card-cta">
                {{ t('home.features.viewUsage') }}
                <Icon name="arrowRight" size="xs" :stroke-width="2.5" />
              </router-link>
            </div>
          </article>

          <!-- 卡片 4：数据与策略 -->
          <article class="bh-block bh-block-hover group">
            <div class="flex items-center justify-between border-b-[3px] border-gray-950 bg-gray-950 px-5 py-3.5 dark:border-dark-100 dark:bg-dark-950">
              <h3 class="text-base font-extrabold text-bh-paper">{{ t('home.features.dataPolicies') }}</h3>
              <span class="bh-head-shape h-5 w-5 rounded-full bg-bh-red transition-transform duration-300 group-hover:scale-125"></span>
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
                class="bh-card-cta"
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
              <h2 class="bh-section-title !mb-0">{{ t('home.providers.title') }}</h2>
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
              class="bh-block bh-block-hover p-6"
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
              class="bh-block bh-block-hover p-6"
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
        <h2 class="bh-section-title">{{ t('home.steps.signup.title') }} → {{ t('home.steps.apiKey.title') }}</h2>
        <div class="grid gap-8 md:grid-cols-3">
          <article
            v-for="(step, sIndex) in homeSteps"
            :key="step.key"
            class="bh-block flex min-h-[220px] flex-col p-6"
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
                class="bh-cta bh-cta-ink"
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
          <!-- Brand -->
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
              <i class="bh-foot-tri block"></i>
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

          <!-- Link columns -->
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

        <div class="bh-stripe mt-10" aria-hidden="true"><i></i><i></i><i></i></div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import GitHubMark from '@/components/auth/GitHubMark.vue'
import GoogleOneTap from '@/components/auth/GoogleOneTap.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import UserAvatar from '@/components/common/UserAvatar.vue'
import ProviderIcon from '@/components/common/ProviderIcon.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { useTheme } from '@/composables/useTheme'
import { getMarketplaceModels, getMarketplaceStats } from '@/api/marketplace'
import type { MarketplaceGroup, MarketplaceModel, MarketplaceStats } from '@/types'
import { sanitizeUrl } from '@/utils/url'
import { hasAcceptedLoginAgreement } from '@/utils/loginAgreement'
import { isGoogleOneTapEligible, isGoogleOneTapOriginSupported } from '@/utils/googleIdentity'
import {
  providerBrandDisplayName,
  providerBrandFilterKey,
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

interface HomeOrbitNode {
  brand: string
  left: string
  top: string
}

// 首页精选卡片：单个模型及其所属分组（分组提供品牌与折扣上下文）
interface HomeFeaturedModel {
  model: MarketplaceModel
  group: MarketplaceGroup
  // 相对官方价的折扣文案（如 "95.6% off"），分组无有效折扣时为 null
  discountOff: string | null
}

const { t, locale } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// 站点设置直接读取已注入或已缓存的公开配置。
const siteName = computed(() => appStore.siteName || 'Sub2API')
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

const { isDark, toggleTheme } = useTheme()

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

const homeOrbitNodes = computed<HomeOrbitNode[]>(() =>
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

  switch (group.platform) {
    case 'anthropic':
      return { key: 'claude', label: t('home.providers.claude'), iconBrand: 'Claude' }
    case 'openai':
      return { key: 'gpt', label: t('home.providers.gpt'), iconBrand: 'OpenAI' }
    case 'gemini':
      return { key: 'gemini', label: t('home.providers.gemini'), iconBrand: 'Gemini' }
    case 'antigravity':
      return { key: 'antigravity', label: t('home.providers.antigravity'), iconBrand: 'Antigravity' }
  }

  const fallbackLabel = brandSource || group.platform
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
    const normalizedKey = normalizedBrand.toLocaleLowerCase()
    if (!normalizedBrand || seen.has(normalizedKey)) {
      return
    }
    seen.add(normalizedKey)
    merged.push(normalizedBrand)
  })

  return merged
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
/* 导航链接：黄色下划线滑入 */
.bh-navlink {
  position: relative;
  padding-bottom: 2px;
  transition: color 0.15s ease;
}

.bh-navlink::after {
  content: '';
  position: absolute;
  left: 0;
  bottom: -2px;
  height: 3px;
  width: 0;
  background: var(--bh-red);
  transition: width 0.2s ease;
}

.bh-navlink:hover::after {
  width: 100%;
}

/* CTA 按钮：黑框 + 硬阴影 + 位移 */
.bh-cta {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  min-height: 48px;
  min-width: 180px;
  padding: 0.8rem 1.9rem;
  font-size: 0.95rem;
  font-weight: 800;
  border: 3px solid var(--bh-ink);
  box-shadow: var(--bh-shadow-sm);
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}

.bh-cta:hover {
  transform: translate(-3px, -3px);
  box-shadow: var(--bh-shadow-sm);
}

.bh-cta:active {
  transform: translate(3px, 3px);
  box-shadow: 1px 1px 0 0 var(--bh-shadow-ink);
}

.bh-cta-red {
  background: var(--bh-red);
  color: #ffffff;
}

.bh-cta-yellow {
  background: var(--bh-yellow);
  color: #141414;
}

.bh-cta-ink {
  background: #141414;
  color: #f4f0e6;
  border-color: #141414;
  box-shadow: var(--bh-shadow-sm);
}

/* 区块标题：蓝色菱形前缀 */
.bh-section-title {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 2rem;
  font-size: clamp(24px, 3.6vw, 34px);
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--bh-ink);
}

.bh-section-title::before {
  content: '';
  width: 20px;
  height: 20px;
  background: var(--bh-blue);
  transform: rotate(45deg);
  flex: none;
}

/* 卡片底部 CTA */
.bh-card-cta {
  margin-top: 1rem;
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.875rem;
  font-weight: 800;
  color: var(--bh-ink);
  text-decoration: none;
}

.bh-card-cta:hover {
  text-decoration: underline;
  text-decoration-thickness: 3px;
  text-underline-offset: 4px;
  text-decoration-color: var(--bh-red);
}

/* 卡片头部小几何 */
.bh-tri-sm {
  width: 0;
  height: 0;
  border-left: 11px solid transparent;
  border-right: 11px solid transparent;
  border-bottom: 19px solid var(--bh-red);
}

/* Hero 构成舞台：轨道动态表达多账号路由，结构与传统居中营销页区分开。 */
.home-stage {
  position: relative;
  min-height: 390px;
  overflow: hidden;
  isolation: isolate;
  border: 3px solid var(--bh-ink);
  background: var(--bh-paper);
  box-shadow: var(--bh-shadow-sm);
}

.home-stage-grid {
  position: absolute;
  inset: 0;
  z-index: -1;
  background-image:
    linear-gradient(rgba(20, 20, 20, 0.08) 1px, transparent 1px),
    linear-gradient(90deg, rgba(20, 20, 20, 0.08) 1px, transparent 1px);
  background-size: 28px 28px;
  opacity: 0.7;
}

.dark .home-stage-grid {
  background-image:
    linear-gradient(rgba(244, 240, 230, 0.1) 1px, transparent 1px),
    linear-gradient(90deg, rgba(244, 240, 230, 0.1) 1px, transparent 1px);
}

.home-stage-sun {
  position: absolute;
  top: 44px;
  right: 40px;
  width: 150px;
  height: 150px;
  border: 3px solid var(--bh-ink);
  border-radius: 50%;
  background: var(--bh-yellow);
  animation: bh-stage-pulse 4.5s ease-in-out infinite;
}

.home-stage-ring {
  position: absolute;
  border: 3px solid var(--bh-blue);
  border-radius: 50%;
  pointer-events: none;
}

.home-stage-ring-outer {
  top: 24px;
  right: 2px;
  width: 285px;
  height: 285px;
  animation: bh-stage-spin 18s linear infinite;
}

.home-stage-ring-inner {
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
.home-stage-geometry-orbit,
.home-stage-icon-cloud {
  position: absolute;
  top: 26px;
  right: 5px;
  width: 282px;
  height: 282px;
  border-radius: 50%;
  pointer-events: none;
  animation: bh-stage-spin 22s linear infinite;
}

.home-stage-geometry-orbit {
  top: 69px;
  right: 48px;
  z-index: 2;
  width: 194px;
  height: 194px;
  animation-duration: 12s;
  animation-direction: reverse;
}

.home-stage-geometry {
  position: absolute;
  display: block;
  filter: drop-shadow(2px 2px 0 var(--bh-ink));
}

.home-stage-geometry-square {
  top: -10px;
  left: calc(50% - 10px);
  width: 20px;
  height: 20px;
  border: 2px solid var(--bh-ink);
  background: var(--bh-red);
}

.home-stage-geometry-triangle {
  right: -11px;
  bottom: 24px;
  width: 0;
  height: 0;
  border-right: 12px solid transparent;
  border-bottom: 22px solid var(--bh-yellow);
  border-left: 12px solid transparent;
}

.home-stage-geometry-circle {
  bottom: 21px;
  left: -9px;
  width: 21px;
  height: 21px;
  border: 2px solid var(--bh-ink);
  border-radius: 50%;
  background: var(--bh-blue);
}

.home-stage-icon-cloud {
  z-index: 2;
}

.home-stage-icon-node {
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

.home-stage-orbit {
  position: absolute;
  top: 50px;
  right: 8px;
  width: 272px;
  height: 240px;
  pointer-events: none;
}

.home-stage-orbit-a {
  animation: bh-stage-spin 18s linear infinite;
}

.home-stage-orbit-b {
  top: 74px;
  right: 34px;
  width: 220px;
  height: 190px;
  animation: bh-stage-spin-reverse 12s linear infinite;
}

.home-stage-orbit span {
  position: absolute;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  border: 2px solid var(--bh-ink);
  background: var(--bh-surface);
  color: var(--bh-ink);
}

.home-stage-orbit-b span {
  width: 32px;
  height: 32px;
  border-color: var(--bh-red);
}

.home-stage-orbit span:nth-child(1) { top: 0; left: 50%; transform: translateX(-50%); }
.home-stage-orbit span:nth-child(2) { right: 0; bottom: 4px; }
.home-stage-orbit span:nth-child(3) { bottom: 4px; left: 0; }

.home-stage-panel {
  position: absolute;
  bottom: 28px;
  left: 28px;
  z-index: 3;
  width: min(250px, calc(100% - 56px));
  padding: 18px;
  border: 3px solid var(--bh-ink);
  background: var(--bh-surface);
  box-shadow: var(--bh-shadow-sm);
  color: inherit;
  text-decoration: none;
  cursor: pointer;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
  animation: bh-stage-panel-in 0.8s 0.28s ease-out both;
}

.home-stage-panel:hover {
  transform: translate(-3px, -3px);
  box-shadow: 7px 7px 0 0 var(--bh-shadow-ink);
}

.home-stage-panel:active {
  transform: translate(3px, 3px);
  box-shadow: 1px 1px 0 0 var(--bh-shadow-ink);
}

.home-stage-panel:focus-visible {
  outline: 3px solid var(--bh-blue);
  outline-offset: 3px;
}

.home-stage-live-dot {
  width: 10px;
  height: 10px;
  border: 2px solid var(--bh-ink);
  border-radius: 50%;
  background: #10b981;
  animation: bh-stage-blink 1.8s ease-in-out infinite;
}

.home-stage-node {
  display: flex;
  width: 34px;
  height: 34px;
  align-items: center;
  justify-content: center;
  border: 2px solid var(--bh-ink);
  background: var(--bh-paper);
}

.home-stage-route-link {
  display: inline-flex;
  width: 34px;
  height: 34px;
  align-items: center;
  justify-content: center;
  border: 2px solid transparent;
  transition: transform 0.15s ease, background-color 0.15s ease;
}

.home-stage-route-link:hover {
  background: var(--bh-yellow);
  transform: translate(-2px, -2px);
}

.home-stage-route-link:active {
  transform: translate(2px, 2px);
}

.home-stage-stamp {
  position: absolute;
  top: 22px;
  left: 24px;
  z-index: 3;
  padding: 4px 8px;
  border: 2px solid var(--bh-ink);
  background: var(--bh-red);
  color: #fff;
  font-family: 'Geist Mono Variable', ui-monospace, monospace;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.12em;
}

.home-stage-stamp span {
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
.home-marquee-word {
  padding-inline: clamp(1.1rem, 3vw, 2.8rem);
  color: #f4f0e6;
  font-family: 'Archivo Black', 'Plus Jakarta Sans Variable', system-ui, sans-serif;
  font-size: clamp(0.95rem, 2.1vw, 1.35rem);
  font-weight: 900;
  letter-spacing: 0.08em;
  white-space: nowrap;
}

.home-marquee-word-1 { color: var(--bh-yellow); }
.home-marquee-word-2 { color: #10b981; }
.home-marquee-word-3 { color: #78aaf0; }

.home-marquee-separator {
  color: var(--bh-red);
  font-size: 0.8rem;
}

/* 页脚三角 */
.bh-foot-tri {
  width: 0;
  height: 0;
  border-left: 8px solid transparent;
  border-right: 8px solid transparent;
  border-bottom: 14px solid var(--bh-yellow);
}

/* 模型广场按钮图标轮换过渡 */
.home-marketplace-icon-enter-active,
.home-marketplace-icon-leave-active {
  transition: opacity 220ms ease, transform 220ms ease;
}

.home-marketplace-icon-enter-from {
  opacity: 0;
  transform: translateY(-70%);
}

.home-marketplace-icon-leave-to {
  opacity: 0;
  transform: translateY(70%);
}

@media (max-width: 639px) {
  .home-stage {
    min-height: 310px;
  }

  .home-stage-sun {
    top: 34px;
    right: 24px;
    width: 112px;
    height: 112px;
  }

  .home-stage-ring-outer {
    top: 18px;
    right: -22px;
    width: 220px;
    height: 220px;
  }

  .home-stage-ring-inner {
    top: 52px;
    right: 14px;
    width: 155px;
    height: 155px;
  }

  .home-stage-icon-cloud {
    top: 21px;
    right: -19px;
    width: 218px;
    height: 218px;
  }

  .home-stage-geometry-orbit {
    top: 55px;
    right: 16px;
    width: 150px;
    height: 150px;
  }

  .home-stage-icon-node {
    width: 24px;
    height: 24px;
  }

  .home-stage-orbit {
    top: 34px;
    right: -14px;
    width: 212px;
    height: 192px;
  }

  .home-stage-orbit-b {
    top: 54px;
    right: 8px;
    width: 174px;
    height: 150px;
  }

  .home-stage-panel {
    bottom: 18px;
    left: 18px;
    width: min(228px, calc(100% - 36px));
    padding: 14px;
  }

  .home-stage-stamp {
    top: 16px;
    left: 16px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .home-stage-sun,
  .home-stage-ring,
  .home-stage-orbit,
  .home-stage-geometry-orbit,
  .home-stage-icon-cloud,
  .home-stage-icon-node,
  .home-stage-live-dot,
  .home-stage-panel,
  .animate-bh-rise,
  .home-marketplace-icon-enter-active,
  .home-marketplace-icon-leave-active {
    animation: none !important;
    transition: none !important;
  }

}
</style>
