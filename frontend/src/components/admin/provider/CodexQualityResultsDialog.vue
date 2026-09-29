<template>
  <BaseDialog :show="show" :title="title" width="wide" :z-index="70" @close="$emit('close')">
    <p v-if="!results.length" class="text-sm text-gray-500">{{ t('admin.accounts.quality.emptyCategory') }}</p>
    <div v-if="show" class="space-y-4"><CodexQualityResultCard v-for="result in pageResults" :key="result.account_id" :result="result" /></div>
    <Pagination v-if="results.length > 20" :page="page" :page-size="20" :total="results.length" :show-page-size-selector="false" @update:page="page = $event" />
  </BaseDialog>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import CodexQualityResultCard from './CodexQualityResult.vue'
import type { CodexQualityResult } from '@/api/admin/codexQuality'
const props = defineProps<{ show: boolean; title: string; results: CodexQualityResult[] }>()
defineEmits<{ close: [] }>()
const { t } = useI18n()
const page = ref(1)
watch(() => [props.show, props.title], () => { page.value = 1 })
const pageResults = computed(() => props.results.slice((page.value - 1) * 20, page.value * 20))
</script>
