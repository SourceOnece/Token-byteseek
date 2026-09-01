<template>
  <Teleport to="body">
    <div
      class="pointer-events-none fixed right-4 top-4 z-[9999] space-y-3"
      aria-live="polite"
      aria-atomic="true"
    >
      <TransitionGroup
        enter-active-class="transition ease-out duration-300"
        enter-from-class="opacity-0 translate-x-full"
        enter-to-class="opacity-100 translate-x-0"
        leave-active-class="transition ease-in duration-200"
        leave-from-class="opacity-100 translate-x-0"
        leave-to-class="opacity-0 translate-x-full"
      >
        <div
          v-for="toast in toasts"
          :key="toast.id"
          class="bh-toast pointer-events-auto flex min-w-[320px] max-w-md overflow-hidden"
        >
          <!-- 信号色板：色块 + 白色图标，一眼分辨消息类型 -->
          <div
            :class="['flex w-12 flex-shrink-0 items-center justify-center border-r-2 border-gray-950 dark:border-dark-100', getPlateClass(toast.type)]"
            aria-hidden="true"
          >
            <Icon
              :name="getToastIconName(toast.type)"
              size="md"
              :stroke-width="2.5"
              :class="toast.type === 'warning' ? 'text-gray-950' : 'text-white'"
            />
          </div>

          <div class="min-w-0 flex-1 bg-white dark:bg-dark-800">
            <div class="p-3.5">
              <div class="flex items-start gap-3">
                <!-- Content -->
                <div class="min-w-0 flex-1">
                  <p v-if="toast.title" class="text-sm font-extrabold text-gray-950 dark:text-white">
                    {{ toast.title }}
                  </p>
                  <p
                    :class="[
                      'text-sm font-medium leading-relaxed',
                      toast.title
                        ? 'mt-1 text-gray-700 dark:text-dark-100'
                        : 'font-semibold text-gray-950 dark:text-white'
                    ]"
                  >
                    {{ toast.message }}
                  </p>
                </div>

                <!-- Close button -->
                <button
                  @click="removeToast(toast.id)"
                  class="-m-1 flex-shrink-0 border-2 border-transparent p-1 text-gray-500 transition-colors hover:border-gray-950 hover:bg-bh-yellow hover:text-gray-950 dark:text-dark-300 dark:hover:border-dark-100"
                  aria-label="Close notification"
                >
                  <Icon name="x" size="sm" :stroke-width="2.5" />
                </button>
              </div>
            </div>

            <!-- Progress bar -->
            <div v-if="toast.duration" class="h-1.5 border-t border-gray-950/30 bg-transparent dark:border-dark-200/30">
              <div
                :class="['h-full toast-progress', getProgressBarColor(toast.type)]"
                :style="{ animationDuration: `${toast.duration}ms` }"
              ></div>
            </div>
          </div>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const toasts = computed(() => appStore.toasts)

const getToastIconName = (type: string): 'checkCircle' | 'xCircle' | 'exclamationTriangle' | 'infoCircle' => {
  switch (type) {
    case 'success':
      return 'checkCircle'
    case 'error':
      return 'xCircle'
    case 'warning':
      return 'exclamationTriangle'
    case 'info':
    default:
      return 'infoCircle'
  }
}

const getPlateClass = (type: string): string => {
  const plates: Record<string, string> = {
    success: 'bg-emerald-600',
    error: 'bg-bh-red',
    warning: 'bg-bh-yellow',
    info: 'bg-bh-blue'
  }
  return plates[type] || plates.info
}

const getProgressBarColor = (type: string): string => {
  const colors: Record<string, string> = {
    success: 'bg-emerald-600',
    error: 'bg-bh-red',
    warning: 'bg-bh-yellow',
    info: 'bg-bh-blue'
  }
  return colors[type] || colors.info
}

const removeToast = (id: string) => {
  appStore.hideToast(id)
}
</script>

<style scoped>
.bh-toast {
  border: 2px solid var(--bh-ink);
  box-shadow: var(--bh-shadow-sm);
}

.toast-progress {
  width: 100%;
  animation-name: toast-progress-shrink;
  animation-timing-function: linear;
  animation-fill-mode: forwards;
}

@keyframes toast-progress-shrink {
  from {
    width: 100%;
  }
  to {
    width: 0%;
  }
}
</style>
