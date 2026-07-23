<script setup lang="ts">
import { computed, useId } from 'vue'

defineOptions({ inheritAttrs: false })

interface Props {
  modelValue: string | number
  label?: string
  type?: string
  placeholder?: string
  error?: string
  required?: boolean
  id?: string
}

const props = withDefaults(defineProps<Props>(), {
  type: 'text',
  placeholder: '',
  required: false,
  id: undefined,
})

defineEmits<{
  (e: 'update:modelValue', value: string | number): void
}>()

// Math.random() produced a different id on the server vs. the client,
// causing an SSR hydration mismatch on every unlabeled BaseInput —
// useId() is Vue 3.5's SSR-stable id generator, built for exactly this.
const autoId = useId()
const inputId = computed(() => props.id ?? `input-${autoId}`)
</script>

<template>
  <div class="w-full">
    <label
      v-if="label"
      :for="inputId"
      class="mb-1 block text-sm font-medium text-fg-secondary"
    >
      {{ label }}
      <span v-if="required" class="text-fg-danger">*</span>
    </label>
    <input
      v-bind="$attrs"
      :id="inputId"
      :type="type"
      :value="modelValue"
      :placeholder="placeholder"
      class="block w-full rounded-control border border-stroke-subtle bg-surface-sunken px-3 py-2 text-fg-primary shadow-sm focus:border-stroke-focus focus:ring-stroke-focus sm:text-sm"
      :class="{ 'border-fg-danger focus:border-fg-danger focus:ring-fg-danger': error }"
      @input="
        $emit('update:modelValue', ($event.target as HTMLInputElement).value)
      "
    />
    <p v-if="error" class="mt-1 text-sm text-fg-danger">{{ error }}</p>
  </div>
</template>
