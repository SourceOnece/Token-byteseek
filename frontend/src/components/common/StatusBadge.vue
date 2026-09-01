<template>
  <!-- 形状语义状态标：方块=正常运行 / 三角=警告停用 / 圆=错误告警 -->
  <div class="flex items-center gap-1.5">
    <span :class="['bh-status-shape', shapeClass]" aria-hidden="true"></span>
    <span class="text-sm font-semibold text-gray-800 dark:text-gray-200">
      {{ label }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  status: string
  label: string
}>()

const shapeClass = computed(() => {
  switch (props.status) {
    case 'active':
    case 'success':
      return 'bh-status-square'
    case 'disabled':
    case 'inactive':
    case 'warning':
      return 'bh-status-triangle'
    case 'error':
    case 'danger':
      return 'bh-status-circle'
    default:
      return 'bh-status-neutral'
  }
})
</script>

<style scoped>
.bh-status-shape {
  display: inline-block;
  flex-shrink: 0;
}

/* 方块：正常 / 启用 */
.bh-status-square {
  width: 9px;
  height: 9px;
  background: #059669;
}

/* 三角：警告 / 停用 */
.bh-status-triangle {
  width: 0;
  height: 0;
  border-left: 6px solid transparent;
  border-right: 6px solid transparent;
  border-bottom: 10px solid var(--bh-yellow);
  filter: drop-shadow(0 1px 0 rgba(20, 20, 20, 0.55));
}

/* 圆：错误 / 告警 */
.bh-status-circle {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--bh-red);
}

/* 未知状态：空心方 */
.bh-status-neutral {
  width: 9px;
  height: 9px;
  border: 2px solid var(--bh-ink);
  background: transparent;
}
</style>
