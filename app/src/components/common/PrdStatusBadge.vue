<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseBadge from './BaseBadge.vue'
import type { PrdStatus } from '@/types'

const props = defineProps<{ status: PrdStatus }>()
const { t } = useI18n()

// 'info' (seal wash) is reserved for 评审中 per tokens.css's own comment —
// the design system pre-assigned this status to that exact wash.
const toneByStatus: Record<PrdStatus, 'neutral' | 'info' | 'success' | 'danger'> = {
  draft: 'neutral',
  reviewing: 'info',
  approved: 'success',
  rejected: 'danger',
}

const tone = computed(() => toneByStatus[props.status])
</script>

<template>
  <BaseBadge :tone="tone">{{ t(`prd.status.${status}`) }}</BaseBadge>
</template>
