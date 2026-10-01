<template>
  <AppLayout>
    <template #page-heading-actions>
      <button
        type="button"
        class="btn btn-secondary shrink-0 btn-icon"
        :disabled="loadingCharts || loading"
        :title="t('common.refresh')"
        @click="refreshAll"
      >
        <Icon name="refresh" size="md" :class="(loadingCharts || loading) ? 'animate-spin' : ''" />
      </button>
    </template>

    <div class="space-y-6" :style="dashboardMotionVars">
      <div v-if="loading" class="flex items-center justify-center py-12"><LoadingSpinner /></div>
      <template v-else-if="stats">
        <UserDashboardStats :stats="stats" :balance="user?.balance || 0" />
        <UserDashboardLiveStats ref="liveStatsRef" @update="value => { if (value) stats = value }" />
        <UserDashboardCharts :start-date="usage.pickerStart.value" :end-date="usage.pickerEnd.value" :granularity="usage.activeRange.value.granularity" :loading="loadingCharts" :trend="usage.current.value" :models="modelStats">
          <template #toolbar><UserDashboardUsageToolbar :refreshing="loadingCharts || loading" @refresh="refreshAll" /></template>
          <template #trend><UserDashboardUsageChart ref="usageChartRef" /></template>
        </UserDashboardCharts>
        <UserDashboardHeatmap ref="heatmapRef" @select-day="revealChart" />
        <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <div class="lg:col-span-2"><UserDashboardAnnouncements /></div>
          <div class="lg:col-span-1"><UserDashboardQuickActions /></div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAnnouncementStore } from '@/stores/announcements'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardHeatmap from '@/components/user/dashboard/UserDashboardHeatmap.vue'
import UserDashboardAnnouncements from '@/components/user/dashboard/UserDashboardAnnouncements.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import Icon from '@/components/icons/Icon.vue'
import { provideUsageChartState } from '@/components/user/dashboard/usageChartState'
import { dashboardMotionVars } from '@/components/user/dashboard/dashboardMotion'
import UserDashboardLiveStats from '@/components/user/dashboard/UserDashboardLiveStats.vue'
import UserDashboardUsageChart from '@/components/user/dashboard/UserDashboardUsageChart.vue'
import UserDashboardUsageToolbar from '@/components/user/dashboard/UserDashboardUsageToolbar.vue'

const authStore = useAuthStore()
const { t } = useI18n()
const announcementStore = useAnnouncementStore()
const user = computed(() => authStore.user)
const stats = ref<UserStatsType | null>(null)
const loading = ref(false)
const usage = provideUsageChartState()
const loadingCharts = computed(() => usage.loading.value)
const modelStats = usage.models
const liveStatsRef = ref<InstanceType<typeof UserDashboardLiveStats> | null>(null)
const usageChartRef = ref<InstanceType<typeof UserDashboardUsageChart> | null>(null)
const revealChart = () => usageChartRef.value?.chartCardRef?.scrollIntoView({ block: "center", behavior: "auto" })

const loadStats = async () => {
  loading.value = true
  try {
    const [, nextStats] = await Promise.all([
      authStore.refreshUser(),
      usageAPI.getDashboardStats(),
    ])
    stats.value = nextStats
  } catch (error) {
    console.error('Failed to load dashboard stats:', error)
  } finally {
    loading.value = false
  }
}

// 趋势和模型分布复用同一份筛选结果，不重复查询。
const loadCharts = async () => { await usage.load() }

// App 负责首次预加载；用户主动刷新时同时绕过公告节流获取最新内容。
const heatmapRef = ref<InstanceType<typeof UserDashboardHeatmap> | null>(null)
const refreshAll = () => {
  void loadStats()
  void loadCharts()
  void liveStatsRef.value?.reload()
  void heatmapRef.value?.reload()
  void announcementStore.fetchAnnouncements(true)
}

onMounted(() => {
  void loadStats()
  void loadCharts()
  void usage.loadFilterOptions()
})
</script>
