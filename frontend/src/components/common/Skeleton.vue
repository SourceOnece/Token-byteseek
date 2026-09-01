<template>
  <!-- 包豪斯骨架屏：斜纹底 + 高光扫过，直角硬边 -->
  <div
    :class="[
      'bh-skeleton',
      variant === 'circle' ? 'rounded-full' : 'rounded-none',
      customClass
    ]"
    :style="style"
  ></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  variant?: 'rect' | 'circle' | 'text'
  width?: string | number
  height?: string | number
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'rect',
  width: '100%'
})

const customClass = computed(() => props.class || '')

const style = computed(() => {
  const s: Record<string, string> = {}

  if (props.width) {
    s.width = typeof props.width === 'number' ? `${props.width}px` : props.width
  }

  if (props.height) {
    s.height = typeof props.height === 'number' ? `${props.height}px` : props.height
  } else if (props.variant === 'text') {
    s.height = '1em'
    s.marginTop = '0.25em'
    s.marginBottom = '0.25em'
  }

  return s
})
</script>

<style scoped>
.bh-skeleton {
  position: relative;
  overflow: hidden;
  background-image: repeating-linear-gradient(
    -45deg,
    rgba(20, 20, 20, 0.1) 0 6px,
    rgba(20, 20, 20, 0.045) 6px 12px
  );
  background-color: rgba(20, 20, 20, 0.03);
}

.dark .bh-skeleton {
  background-image: repeating-linear-gradient(
    -45deg,
    rgba(244, 240, 230, 0.12) 0 6px,
    rgba(244, 240, 230, 0.05) 6px 12px
  );
  background-color: rgba(244, 240, 230, 0.04);
}

.bh-skeleton::after {
  content: '';
  position: absolute;
  inset: 0;
  transform: translateX(-100%);
  background: linear-gradient(
    90deg,
    transparent 0%,
    rgba(255, 204, 0, 0.28) 50%,
    transparent 100%
  );
  animation: bh-skeleton-sweep 1.6s ease-in-out infinite;
}

@keyframes bh-skeleton-sweep {
  to {
    transform: translateX(100%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .bh-skeleton::after {
    animation: none;
  }
}
</style>
