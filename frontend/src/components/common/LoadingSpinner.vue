<template>
  <!-- 包豪斯加载装置：红圆 / 蓝方 / 黄三角 交替跳动 -->
  <div
    class="bh-loader"
    :class="[`bh-loader-${size}`, colorClass]"
    role="status"
    :aria-label="t('common.loading')"
  >
    <i class="bh-loader-shape bh-loader-circle"></i>
    <i class="bh-loader-shape bh-loader-square"></i>
    <i class="bh-loader-shape bh-loader-triangle"></i>
    <span class="sr-only">{{ t('common.loading') }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

type SpinnerSize = 'sm' | 'md' | 'lg' | 'xl'
type SpinnerColor = 'primary' | 'secondary' | 'white' | 'gray'

interface Props {
  size?: SpinnerSize
  color?: SpinnerColor
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
  color: 'primary'
})

// primary 使用三原色；secondary/gray 保留低调中性色，white 强制纸白以兼容深色底。
const colorClass = computed(() => {
  const colors: Record<SpinnerColor, string> = {
    primary: '',
    secondary: 'bh-loader-muted',
    white: 'bh-loader-mono',
    gray: 'bh-loader-muted'
  }
  return colors[props.color]
})
</script>

<style scoped>
.bh-loader {
  display: inline-flex;
  align-items: flex-end;
  gap: 0.5em;
}

.bh-loader-sm { font-size: 5px; }
.bh-loader-md { font-size: 8px; }
.bh-loader-lg { font-size: 12px; }
.bh-loader-xl { font-size: 16px; }

.bh-loader-shape {
  display: block;
  animation: bh-loader-jump 1.05s ease-in-out infinite;
}

.bh-loader-circle {
  width: 1.6em;
  height: 1.6em;
  border-radius: 50%;
  background: var(--bh-red);
}

.bh-loader-square {
  width: 1.6em;
  height: 1.6em;
  background: var(--bh-blue);
  animation-delay: 0.175s;
}

.bh-loader-triangle {
  width: 0;
  height: 0;
  border-left: 0.9em solid transparent;
  border-right: 0.9em solid transparent;
  border-bottom: 1.6em solid var(--bh-yellow);
  animation-delay: 0.35s;
}

.bh-loader-mono .bh-loader-circle { background: currentColor; }
.bh-loader-mono .bh-loader-square { background: currentColor; }
.bh-loader-mono .bh-loader-triangle { border-bottom-color: currentColor; }
.bh-loader-mono { color: #ffffff; }

.bh-loader-muted { color: #736f63; }
.dark .bh-loader-muted { color: #a39e8f; }

@keyframes bh-loader-jump {
  0%, 55%, 100% {
    transform: translateY(0) scale(1, 1);
  }
  20% {
    transform: translateY(-1.4em) scale(0.96, 1.06);
  }
  40% {
    transform: translateY(0) scale(1.08, 0.9);
  }
}

@media (prefers-reduced-motion: reduce) {
  .bh-loader-shape {
    animation: bh-loader-fade 1.4s ease-in-out infinite;
  }

  @keyframes bh-loader-fade {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.35; }
  }
}
</style>
