<script setup lang="ts">
import { ArrowUpTrayIcon } from '@heroicons/vue/24/outline'
import BaseButton from '@/components/common/BaseButton.vue'

interface Props {
  exportText: string
  submitText: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  export: []
  submit: []
}>()

function handleExport() {
  emit('export')
}

function handleSubmit() {
  emit('submit')
}
</script>

<template>
  <div class="flex items-center gap-2">
    <BaseButton size="sm" variant="outline" @click="handleExport">
      <span class="inline-flex items-center gap-1.5">
        <ArrowUpTrayIcon class="h-4 w-4" />
        {{ props.exportText }}
      </span>
    </BaseButton>
    <BaseButton size="sm" @click="handleSubmit">
      {{ props.submitText }}
    </BaseButton>
  </div>
</template>
