<template>
  <div class="ba-theme-shell relative flex min-h-screen items-center justify-center overflow-hidden p-4">
    <!-- Background -->
    <div class="ba-theme-backdrop pointer-events-none fixed inset-0"></div>
    <AuthBackground />

    <!-- Content Container -->
    <div class="relative z-10 w-full max-w-md">
      <!-- Logo/Brand -->
      <div class="mb-8 text-center">
        <!-- Custom Logo or Default Logo -->
        <template v-if="settingsLoaded">
          <div
            class="mb-5 inline-flex h-16 w-16 items-center justify-center overflow-hidden border-[3px] border-gray-950 bg-white shadow dark:border-dark-100 dark:bg-dark-800"
          >
            <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <!-- 品牌标题：厚重几何 + 红色句点 -->
          <h1 class="mb-2 text-4xl font-extrabold tracking-tight text-gray-950 dark:text-white">
            {{ siteName }}<span class="text-bh-red">.</span>
          </h1>
          <p class="inline-block text-sm font-bold text-gray-700 dark:text-dark-200">
            {{ siteSubtitle }}
          </p>
        </template>
      </div>

      <!-- Card Container：硬边框 + 大位移阴影 + 三原色顶条 -->
      <div class="relative border-[3px] border-gray-950 bg-white shadow-xl dark:border-dark-100 dark:bg-dark-800">
        <div class="bh-stripe" aria-hidden="true"><i></i><i></i><i></i></div>
        <div class="p-8">
          <slot />
        </div>
      </div>

      <!-- Footer Links -->
      <div class="mt-6 text-center text-sm font-semibold">
        <slot name="footer" />
      </div>

      <!-- Copyright -->
      <div class="mt-8 text-center font-mono text-xs font-bold uppercase tracking-widest text-gray-500 dark:text-dark-300">
        &copy; {{ currentYear }} {{ siteName }} · FORM &amp; FUNKTION
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AuthBackground from '@/components/auth/AuthBackground.vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()
const { locale } = useI18n()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => {
  const settings = appStore.cachedPublicSettings
  const isZh = String(locale.value).toLowerCase().startsWith('zh')
  const primary = isZh ? settings?.site_subtitle_zh : settings?.site_subtitle_en
  const secondary = isZh ? settings?.site_subtitle_en : settings?.site_subtitle_zh
  return firstConfiguredText(primary, secondary, settings?.site_subtitle, 'Subscription to API Conversion Platform')
})
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

function firstConfiguredText(...values: Array<string | undefined>): string {
  for (const value of values) {
    const normalized = value?.trim()
    if (normalized) {
      return normalized
    }
  }
  return ''
}

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>
