<template>
  <Select
    class="visual-theme-selector w-14 shrink-0"
    data-testid="visual-theme-selector"
    :model-value="visualTheme"
    :options="options"
    :searchable="false"
    :aria-label="t('nav.visualTheme')"
    :title="`${t('nav.visualTheme')} · ${options.find(option => option.value === visualTheme)?.label}`"
    @update:model-value="selectTheme"
  >
    <template #selected><Icon name="creative" size="sm" /></template>
  </Select>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import Select from './Select.vue'
import { useVisualTheme } from '@/composables/useVisualTheme'

const { t } = useI18n()
const { visualTheme, setVisualTheme } = useVisualTheme()
const options = computed(() => [
  { value: 'tokenflux', label: t('nav.tokenfluxTheme') },
  { value: 'bauhaus', label: t('nav.bauhausTheme') },
])
// 复用 Select 的键盘、浮层、外部点击处理；选择皮肤不会触发路由或业务请求。
function selectTheme(value: unknown) {
  if (value === 'tokenflux' || value === 'bauhaus') setVisualTheme(value)
}
</script>

<style scoped>
.visual-theme-selector :deep(.input-trigger) { gap: .25rem; padding-left: .5rem; padding-right: .5rem; }
</style>
