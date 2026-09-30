<template>
  <svg viewBox="0 0 100 32" preserveAspectRatio="none" aria-hidden="true" focusable="false">
    <path
      v-if="path"
      :d="path"
      fill="none"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
      vector-effect="non-scaling-stroke"
    />
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  // 按时间排列的数值，null 表示该时段没有可计算的值，跳过后与前后时段相连，与主图一致。
  values: Array<number | null>
}>()

const WIDTH = 100
const HEIGHT = 32
// 上下各留一点边距，避免线条贴边被裁掉。
const PADDING = 2

// path 以 0 为底线按最大值等比缩放，全为 0 时画一条贴底的平线。
const path = computed(() => {
  const values = props.values
  if (values.length === 0) return ''
  const max = Math.max(0, ...values.filter((value): value is number => value !== null))
  const toY = (value: number) => {
    const ratio = max > 0 ? value / max : 0
    return (HEIGHT - PADDING - ratio * (HEIGHT - PADDING * 2)).toFixed(2)
  }
  // 只有一个时段时画成横线，否则只剩一个看不见的点。
  if (values.length === 1) {
    const value = values[0]
    return value === null ? '' : `M0 ${toY(value)} L${WIDTH} ${toY(value)}`
  }
  const step = WIDTH / (values.length - 1)
  const points = values
    .map((value, index) => (value === null ? null : `${(index * step).toFixed(2)} ${toY(value)}`))
    .filter((point): point is string => point !== null)
  return points.length > 0 ? `M${points.join(' L')}` : ''
})
</script>
