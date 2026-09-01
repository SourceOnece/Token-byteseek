<template>
  <div class="empty-state">
    <!-- 几何构成画框：无数据时的包豪斯小海报 -->
    <div class="bh-empty-canvas mb-6" aria-hidden="true">
      <slot name="icon">
        <template v-if="icon">
          <component :is="icon" class="empty-state-icon h-10 w-10" />
        </template>
        <template v-else>
          <span class="bh-empty-hatch"></span>
          <span class="bh-empty-circle"></span>
          <span class="bh-empty-square"></span>
          <span class="bh-empty-triangle"></span>
          <span class="bh-empty-bar"></span>
        </template>
      </slot>
    </div>

    <!-- Title -->
    <h3 class="empty-state-title">
      {{ displayTitle }}
    </h3>

    <!-- Description -->
    <p class="empty-state-description">
      {{ description }}
    </p>

    <!-- Action -->
    <div v-if="actionText || $slots.action" class="mt-6">
      <slot name="action">
        <component
          :is="actionTo ? 'RouterLink' : 'button'"
          v-if="actionText"
          :to="actionTo"
          @click="!actionTo && $emit('action')"
          class="btn btn-primary"
        >
          <Icon v-if="actionIcon" name="plus" size="md" class="mr-2" />
          {{ actionText }}
        </component>
      </slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Component } from 'vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

interface Props {
  icon?: Component | string
  title?: string
  description?: string
  actionText?: string
  actionTo?: string | object
  actionIcon?: boolean
  message?: string
}

const props = withDefaults(defineProps<Props>(), {
  description: '',
  actionIcon: true
})

const displayTitle = computed(() => props.title || t('common.noData'))

defineEmits(['action'])
</script>

<style scoped>
.bh-empty-canvas {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 132px;
  height: 96px;
  border: 3px solid var(--bh-ink);
  background: var(--bh-surface);
  box-shadow: var(--bh-shadow-sm);
  overflow: hidden;
}

.bh-empty-hatch {
  position: absolute;
  inset: 0;
  background: repeating-linear-gradient(
    -45deg,
    rgba(20, 20, 20, 0.05) 0 5px,
    transparent 5px 14px
  );
}

.dark .bh-empty-hatch {
  background: repeating-linear-gradient(
    -45deg,
    rgba(244, 240, 230, 0.06) 0 5px,
    transparent 5px 14px
  );
}

.bh-empty-circle {
  position: absolute;
  left: 20px;
  top: 18px;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: var(--bh-red);
  opacity: 0.9;
}

.bh-empty-square {
  position: absolute;
  right: 24px;
  top: 26px;
  width: 26px;
  height: 26px;
  background: var(--bh-blue);
  transform: rotate(12deg);
}

.bh-empty-triangle {
  position: absolute;
  left: 52px;
  bottom: 12px;
  width: 0;
  height: 0;
  border-left: 15px solid transparent;
  border-right: 15px solid transparent;
  border-bottom: 26px solid var(--bh-yellow);
}

.bh-empty-bar {
  position: absolute;
  left: 14px;
  bottom: 20px;
  width: 24px;
  height: 5px;
  background: var(--bh-ink);
}
</style>
