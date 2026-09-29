<template>
  <!-- 404 包豪斯海报：巨型编号 + 几何构成 -->
  <div
    class="relative flex min-h-screen items-center justify-center overflow-hidden bg-bh-paper px-4 dark:bg-dark-900"
  >
    <!-- 顶部三原色条 -->
    <div class="bh-stripe absolute inset-x-0 top-0" aria-hidden="true"><i></i><i></i><i></i></div>

    <!-- 几何装饰 -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden" aria-hidden="true">
      <div class="bh-404-geo bh-404-ring"></div>
      <div class="bh-404-geo bh-404-square"></div>
      <div class="bh-404-geo bh-404-hatch"></div>
    </div>

    <div class="relative z-10 w-full max-w-md text-center">
      <!-- 404 Display -->
      <div class="mb-8">
        <div class="relative inline-block">
          <span class="text-[12rem] font-bold leading-none text-gray-100 dark:text-dark-800"
            >404</span
          >
          <div class="absolute inset-0 flex items-center justify-center">
            <div
              class="flex h-24 w-24 items-center justify-center rounded-surface bg-gradient-to-br from-primary-500 to-primary-600 shadow-lg shadow-primary-500/30"
            >
              <svg
                class="h-12 w-12 text-white"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z"
                />
              </svg>
            </div>
          </div>
        </div>
      </div>

      <!-- Text Content -->
      <div class="mb-10">
        <span class="bh-kicker mb-4">LOST IN COMPOSITION</span>
        <h1 class="mb-3 mt-4 text-3xl font-extrabold tracking-tight text-gray-950 dark:text-white">
          {{ t('errors.pageNotFound') }}
        </h1>
        <p class="font-semibold text-gray-700 dark:text-dark-200">
          The page you are looking for doesn't exist or has been moved.
        </p>
      </div>

      <!-- Action Buttons -->
      <div class="flex flex-col justify-center gap-4 sm:flex-row">
        <button @click="goBack" class="btn btn-secondary">
          <Icon name="arrowLeft" size="md" class="mr-2" />
          Go Back
        </button>
        <router-link to="/dashboard" class="btn btn-primary">
          <Icon name="home" size="md" class="mr-2" />
          Go to Dashboard
        </router-link>
      </div>
    </div>

    <!-- 底部三原色条 -->
    <div class="bh-stripe absolute inset-x-0 bottom-0" aria-hidden="true"><i></i><i></i><i></i></div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const router = useRouter()

function goBack(): void {
  router.back()
}
</script>

<style scoped>
.bh-404-digit {
  display: flex;
  align-items: center;
  justify-content: center;
  width: clamp(84px, 18vw, 130px);
  height: clamp(104px, 22vw, 160px);
  border: 4px solid var(--bh-ink);
  box-shadow: var(--bh-shadow-sm);
  font-family: 'Archivo Black', 'Plus Jakarta Sans Variable', system-ui, sans-serif;
  font-size: clamp(56px, 12vw, 96px);
  line-height: 1;
  animation: bh-404-rise 0.55s ease-out both;
}

.bh-404-digit:nth-child(2) {
  animation-delay: 0.12s;
  transform-origin: bottom center;
}

.bh-404-digit:nth-child(3) {
  animation-delay: 0.24s;
}

/* 中间的 0 用圆形（圆也是基本形） */
.bh-404-digit-circle {
  border-radius: 50%; /* check-ui-allow: 包豪斯圆形或半圆装饰，不是控件圆角。 */
  height: clamp(84px, 18vw, 130px);
  margin-bottom: clamp(10px, 2vw, 15px);
}

@keyframes bh-404-rise {
  from {
    opacity: 0;
    transform: translateY(34px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.bh-404-geo {
  position: absolute;
}

.bh-404-ring {
  top: 12%;
  left: 8%;
  width: 120px;
  height: 120px;
  border-radius: 50%; /* check-ui-allow: 包豪斯圆形或半圆装饰，不是控件圆角。 */
  border: 15px solid var(--bh-blue);
  opacity: 0.85;
  animation: bh-404-float 6s ease-in-out infinite;
}

.bh-404-square {
  bottom: 14%;
  right: 9%;
  width: 90px;
  height: 90px;
  background: var(--bh-yellow);
  border: 3px solid var(--bh-ink);
  animation: bh-404-spin 16s linear infinite;
}

.bh-404-hatch {
  top: 18%;
  right: 14%;
  width: 140px;
  height: 140px;
  transform: rotate(45deg);
  background: repeating-linear-gradient(
    0deg,
    var(--bh-ink) 0 4px,
    transparent 4px 12px
  );
  opacity: 0.12;
}

@keyframes bh-404-float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-20px); }
}

@keyframes bh-404-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@media (prefers-reduced-motion: reduce) {
  .bh-404-digit,
  .bh-404-geo {
    animation: none !important;
  }
}
</style>
