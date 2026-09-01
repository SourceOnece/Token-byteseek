<template>
  <!-- 包豪斯几何背景：三原色基本形的漂浮构成 -->
  <div
    class="pointer-events-none absolute inset-0 z-0 select-none overflow-hidden"
    aria-hidden="true"
  >
    <!-- 顶部三原色条 -->
    <div class="bh-stripe absolute inset-x-0 top-0"><i></i><i></i><i></i></div>

    <!-- 红色实心圆 -->
    <div class="bh-geo bh-geo-circle"></div>

    <!-- 蓝色圆环 -->
    <div class="bh-geo bh-geo-ring"></div>

    <!-- 黄色方块（缓慢旋转） -->
    <div class="bh-geo bh-geo-square"></div>

    <!-- 墨色三角 -->
    <div class="bh-geo bh-geo-triangle"></div>

    <!-- 蓝色小方 -->
    <div class="bh-geo bh-geo-square-sm"></div>

    <!-- 红色半圆（左下） -->
    <div class="bh-geo bh-geo-half"></div>

    <!-- 斜纹装饰（右下角） -->
    <div class="bh-geo bh-geo-hatch"></div>
  </div>
</template>

<script setup lang="ts">
// 纯 CSS 几何构成，无需任何运行时逻辑。
</script>

<style scoped>
.bh-geo {
  position: absolute;
}

.bh-geo-circle {
  top: 8%;
  right: 10%;
  width: 200px;
  height: 200px;
  border-radius: 50%;
  background: var(--bh-red);
  animation: bh-float 6s ease-in-out infinite;
}

.bh-geo-ring {
  top: 30%;
  left: 7%;
  width: 130px;
  height: 130px;
  border-radius: 50%;
  border: 16px solid var(--bh-blue);
  animation: bh-float 7s 0.8s ease-in-out infinite;
}

.bh-geo-square {
  bottom: 12%;
  right: 16%;
  width: 110px;
  height: 110px;
  background: var(--bh-yellow);
  border: 3px solid var(--bh-ink);
  animation: bh-spin-slow 16s linear infinite;
}

.bh-geo-triangle {
  bottom: 18%;
  left: 12%;
  width: 0;
  height: 0;
  border-left: 52px solid transparent;
  border-right: 52px solid transparent;
  border-bottom: 90px solid var(--bh-ink);
  animation: bh-bob 4.4s ease-in-out infinite;
}

.bh-geo-square-sm {
  top: 14%;
  left: 22%;
  width: 42px;
  height: 42px;
  background: var(--bh-blue);
  animation: bh-bob 3.6s 0.5s ease-in-out infinite;
}

.bh-geo-half {
  bottom: -60px;
  left: 42%;
  width: 120px;
  height: 60px;
  border-radius: 120px 120px 0 0;
  background: var(--bh-red);
  opacity: 0.85;
}

.bh-geo-hatch {
  right: -30px;
  bottom: -30px;
  width: 160px;
  height: 160px;
  transform: rotate(45deg);
  background: repeating-linear-gradient(
    0deg,
    var(--bh-ink) 0 4px,
    transparent 4px 12px
  );
  opacity: 0.16;
}

@keyframes bh-float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-24px); }
}

@keyframes bh-bob {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-12px); }
}

@keyframes bh-spin-slow {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@media (max-width: 720px) {
  .bh-geo-ring,
  .bh-geo-square-sm,
  .bh-geo-half {
    display: none;
  }

  .bh-geo-circle {
    width: 120px;
    height: 120px;
    right: -30px;
  }

  .bh-geo-triangle {
    border-left-width: 34px;
    border-right-width: 34px;
    border-bottom-width: 58px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .bh-geo {
    animation: none !important;
  }
}
</style>
