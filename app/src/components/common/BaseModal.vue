<script setup lang="ts">
import { ref, watch, nextTick, onBeforeUnmount, useId } from 'vue'
import { XMarkIcon } from '@heroicons/vue/24/outline'
import { useI18n } from 'vue-i18n'

interface Props {
  open: boolean
  title?: string
  size?: 'sm' | 'md' | 'lg'
}
const props = withDefaults(defineProps<Props>(), { title: undefined, size: 'md' })
const emit = defineEmits<{ (e: 'close'): void }>()
const { t } = useI18n()

const panelRef = ref<HTMLElement | null>(null)
const titleId = `modal-title-${useId()}`
let previouslyFocused: HTMLElement | null = null

const FOCUSABLE =
  'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'

function visibleFocusables(): HTMLElement[] {
  if (!panelRef.value) return []
  return Array.from(panelRef.value.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
    (el) => el.offsetParent !== null,
  )
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.stopPropagation()
    emit('close')
    return
  }
  if (e.key !== 'Tab') return
  const focusables = visibleFocusables()
  if (!focusables.length) return
  const first = focusables[0]!
  const last = focusables[focusables.length - 1]!
  const active = document.activeElement as HTMLElement | null
  if (e.shiftKey && active === first) {
    e.preventDefault()
    last.focus()
  } else if (!e.shiftKey && active === last) {
    e.preventDefault()
    first.focus()
  }
}

watch(
  () => props.open,
  async (isOpen) => {
    if (!import.meta.client) return
    if (isOpen) {
      previouslyFocused = document.activeElement as HTMLElement | null
      document.body.style.overflow = 'hidden'
      document.addEventListener('keydown', onKeydown, true)
      await nextTick()
      const target = visibleFocusables()[0] ?? panelRef.value
      target?.focus()
    } else {
      document.body.style.overflow = ''
      document.removeEventListener('keydown', onKeydown, true)
      previouslyFocused?.focus?.()
      previouslyFocused = null
    }
  },
)

onBeforeUnmount(() => {
  if (!import.meta.client) return
  document.body.style.overflow = ''
  document.removeEventListener('keydown', onKeydown, true)
})

const maxWidth = { sm: 'max-w-sm', md: 'max-w-md', lg: 'max-w-lg' }
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="open"
        class="fixed inset-0 z-modal flex items-center justify-center p-4"
        role="presentation"
        @click.self="emit('close')"
      >
        <div class="absolute inset-0 bg-surface-overlay" aria-hidden="true" />
        <div
          ref="panelRef"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="title ? titleId : undefined"
          tabindex="-1"
          class="relative w-full rounded-card border border-stroke-subtle bg-surface-raised shadow-modal focus:outline-none"
          :class="maxWidth[size]"
        >
          <div class="flex items-start justify-between gap-4 border-b border-stroke-subtle px-5 py-4">
            <h2 v-if="title" :id="titleId" class="font-serif text-lg font-semibold text-fg-primary">{{ title }}</h2>
            <span v-else class="flex-1" aria-hidden="true" />
            <button
              type="button"
              :aria-label="t('common.close')"
              class="-mr-1 rounded-control p-1 text-fg-muted transition-colors hover:bg-surface-sunken hover:text-fg-primary"
              @click="emit('close')"
            >
              <XMarkIcon class="h-5 w-5" />
            </button>
          </div>
          <div class="px-5 py-4">
            <slot />
          </div>
          <div v-if="$slots.footer" class="flex items-center justify-end gap-3 border-t border-stroke-subtle px-5 py-4">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
