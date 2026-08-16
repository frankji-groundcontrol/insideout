<script setup lang="ts">
import { onMounted, ref, useId } from 'vue'
import { motion } from 'motion-v'
import { XMarkIcon } from '@heroicons/vue/24/outline'
import { useI18n } from 'vue-i18n'
import { useDialogA11y } from '@/composables/useDialogA11y'
import { useReducedMotion } from '@/composables/useReducedMotion'

// A prompted floating modal: the auth middleware's redirect to /login is the
// "prompt", and this shell floats the Ink & Seal door open over a dimmed scrim.
// The seal stamps in, the wordmark rises, the paper panel surfaces beneath.
interface Props {
  greeting?: string
}
withDefaults(defineProps<Props>(), { greeting: undefined })
const emit = defineEmits<{ (e: 'close'): void }>()
const { t } = useI18n()
const reduce = useReducedMotion()
const ease = [0.16, 1, 0.3, 1] as const

const panelRef = ref<HTMLElement | null>(null)
const wordmarkId = `auth-door-title-${useId()}`

// Open post-mount so the first client render matches SSR (nothing rendered)
// and the enter motion plays after hydration — also keeps motion-v's `initial`
// resolution client-only, avoiding the hydration-mismatch warning.
const open = ref(false)
onMounted(() => {
  open.value = true
})

useDialogA11y(panelRef, open, () => emit('close'))
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-modal flex items-center justify-center overflow-y-auto bg-surface-base p-4"
      role="presentation"
      @click.self="emit('close')"
    >
      <!-- the modal carries its own world-ground (bg-surface-base above) so the
           door always floats over celadon / ink-night, never a void. This layer
           is an ambient vignette — transparent over the seal+panel, dimming only
           the periphery for focus — not a flat black wall (which would erase the
           Ink & Seal ground in light mode). pointer-events-none so clicks fall
           through to the container's @click.self above. -->
      <div
        aria-hidden="true"
        class="pointer-events-none absolute inset-0 bg-[radial-gradient(120%_120%_at_50%_38%,transparent_40%,rgb(var(--color-surface-overlay)/0.5)_100%)]"
      />
      <!-- a faint vermilion wash bleeding into the scrim behind the seal —
           a bleed, not a solid element (One Seal Rule) -->
      <div
        aria-hidden="true"
        class="pointer-events-none absolute inset-0 bg-[radial-gradient(36rem_24rem_at_50%_32%,rgb(var(--color-seal)/0.06),transparent_70%)]"
      />

      <!-- the dialog landmark wraps BOTH the header and the panel: with
           aria-modal="true" the UA hides everything outside it from the
           accessibility tree, so every control (close button, form) and the
           focus-trap root must live inside this one element -->
      <div
        role="dialog"
        aria-modal="true"
        :aria-labelledby="wordmarkId"
        class="relative w-full max-w-sm"
      >
        <!-- the door header: real baiwen seal, serif wordmark, optional greeting -->
        <div class="mb-8 flex flex-col items-center text-center">
          <motion.img
            src="/seals/yin.webp"
            alt=""
            aria-hidden="true"
            width="80"
            height="80"
            class="mb-5 h-20 w-20 select-none object-contain"
            decoding="async"
            :initial="reduce ? false : { opacity: 0, scale: 1.6 }"
            :animate="{ opacity: 1, scale: reduce ? 1 : [1.6, 0.88, 1.07, 1] }"
            :transition="{ duration: 0.65, delay: 0.15, ease }"
          />
          <motion.h1
            :id="wordmarkId"
            class="font-serif text-2xl font-bold text-fg-primary"
            :initial="reduce ? false : { opacity: 0, y: 16 }"
            :animate="{ opacity: 1, y: 0 }"
            :transition="{ duration: 0.55, delay: 0.3, ease }"
          >
            {{ t('nav.brand') }}
          </motion.h1>
          <motion.p
            v-if="greeting"
            class="mt-2 text-sm text-fg-muted"
            :initial="reduce ? false : { opacity: 0, y: 12 }"
            :animate="{ opacity: 1, y: 0 }"
            :transition="{ duration: 0.5, delay: 0.4, ease }"
          >
            {{ greeting }}
          </motion.p>
        </div>

        <!-- the paper panel surfacing beneath the seal -->
        <motion.div
          class="relative rounded-hero border border-stroke-subtle bg-surface-raised p-8 shadow-modal"
          :initial="reduce ? false : { opacity: 0, y: 24, scale: 0.98 }"
          :animate="{ opacity: 1, y: 0, scale: 1 }"
          :transition="{ duration: 0.5, delay: 0.35, ease }"
        >
          <!-- panelRef must land on a real HTMLElement for the focus trap; a
               ref on motion.div yields its component proxy, so we wrap. -->
          <div ref="panelRef" tabindex="-1" class="focus:outline-none">
            <button
              type="button"
              :aria-label="t('common.close')"
              class="absolute right-3 top-3 rounded-control p-1 text-fg-muted transition-colors hover:bg-surface-sunken hover:text-fg-primary"
              @click="emit('close')"
            >
              <XMarkIcon class="h-5 w-5" />
            </button>
            <slot />
          </div>
        </motion.div>
      </div>
    </div>
  </Teleport>
</template>
