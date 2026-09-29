<template>
  <div class="card">
    <div class="border-b-2 border-gray-950 px-6 py-4 dark:border-dark-200/60">
      <h2 class="bh-card-title">{{ t('dashboard.quickActions') }}</h2>
    </div>
    <div class="space-y-3 p-4">
      <button @click="router.push('/keys')" class="bh-action group flex w-full items-center gap-4 p-3.5 text-left">
        <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center bh-action-plate bg-bh-blue">
          <Icon name="key" size="lg" class="text-white" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-extrabold text-gray-950 dark:text-white">{{ t('dashboard.createApiKey') }}</p>
          <p class="text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('dashboard.generateNewKey') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="md"
          class="bh-action-arrow text-gray-800 dark:text-dark-100"
        />
      </button>

      <button @click="router.push('/usage')" class="bh-action group flex w-full items-center gap-4 p-3.5 text-left">
        <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center bh-action-plate bg-bh-red">
          <Icon name="chart" size="lg" class="text-white" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-extrabold text-gray-950 dark:text-white">{{ t('dashboard.viewUsage') }}</p>
          <p class="text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('dashboard.checkDetailedLogs') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="md"
          class="bh-action-arrow text-gray-800 dark:text-dark-100"
        />
      </button>

      <button v-if="canUseBatchImage" @click="router.push('/batch-image')" class="bh-action group flex w-full items-center gap-4 p-3.5 text-left">
        <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center bh-action-plate bg-bh-yellow">
          <Icon name="sparkles" size="lg" class="text-gray-950" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-extrabold text-gray-950 dark:text-white">{{ t('dashboard.batchImageAgent') }}</p>
          <p class="text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('dashboard.batchImageAgentDesc') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="md"
          class="bh-action-arrow text-gray-800 dark:text-dark-100"
        />
      </button>

      <button
        v-if="paymentEnabled"
        data-testid="purchase-quick-action"
        @click="router.push('/purchase')"
        class="bh-action group flex w-full items-center gap-4 p-3.5 text-left"
      >
        <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center bh-action-plate bg-gray-950 dark:bg-dark-100">
          <Icon name="creditCard" size="lg" class="text-white dark:text-gray-950" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-extrabold text-gray-950 dark:text-white">{{ t('nav.buySubscription') }}</p>
          <p class="text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('dashboard.purchasePlanOrRecharge') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="md"
          class="bh-action-arrow text-gray-800 dark:text-dark-100"
        />
      </button>

      <button @click="router.push('/redeem')" class="bh-action group flex w-full items-center gap-4 p-3.5 text-left">
        <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center bh-action-plate bg-emerald-600">
          <Icon name="gift" size="lg" class="text-white" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-extrabold text-gray-950 dark:text-white">{{ t('dashboard.redeemCode') }}</p>
          <p class="text-xs font-medium text-gray-600 dark:text-dark-300">{{ t('dashboard.addBalanceWithCode') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="md"
          class="bh-action-arrow text-gray-800 dark:text-dark-100"
        />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
import { useAppStore } from '@/stores/app'

const router = useRouter()
const { t } = useI18n()
const appStore = useAppStore()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
// 公共设置明确启用支付时才展示入口，避免加载失败时暴露不可用路由。
const paymentEnabled = computed(() => appStore.cachedPublicSettings?.payment_enabled === true)

onMounted(() => {
  void refreshBatchImageAccess()
})
</script>

<style scoped>
.bh-card-title {
  font-size: 1.05rem;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--bh-ink);
  position: relative;
  padding-left: 16px;
}

.bh-card-title::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 8px;
  height: 1em;
  background: var(--bh-blue);
}

.bh-action {
  border: 2px solid var(--bh-ink);
  background: var(--bh-surface);
  transition: transform 0.14s ease, box-shadow 0.14s ease;
}

.bh-action:hover {
  transform: translate(-2px, -2px);
  box-shadow: 4px 4px 0 0 var(--bh-shadow-ink);
}

.bh-action:active {
  transform: translate(1px, 1px);
  box-shadow: none;
}

.bh-action-plate {
  display: flex;
  height: 2.75rem;
  width: 2.75rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border: 2px solid var(--bh-ink);
}

.bh-action-arrow {
  transition: transform 0.14s ease;
}

.bh-action:hover .bh-action-arrow {
  transform: translateX(5px);
}
</style>
