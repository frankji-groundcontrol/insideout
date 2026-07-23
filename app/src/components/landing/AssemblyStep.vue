<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { motion } from 'motion-v'
import AssemblyDiagram from '@/components/landing/AssemblyDiagram.vue'
import StepPeek from '@/components/landing/StepPeek.vue'
import { useReducedMotion } from '@/composables/useReducedMotion'

// One build-instruction "page": the Assembly mini-map advanced to this step,
// the step's seal chop + label, its title and copy, and a 1:1 peek at the real
// artifact. `flip` alternates which side the copy sits on for reading rhythm.
interface Props {
  n: number
  chop: string
  seal: string
  title: string
  body: string
  kind: 'idea' | 'prd' | 'roadmap' | 'shipped'
  flip?: boolean
}
withDefaults(defineProps<Props>(), { flip: false })

const { t } = useI18n()
const reduce = useReducedMotion()
const ease = [0.16, 1, 0.3, 1] as const
</script>

<template>
  <section class="relative mx-auto max-w-6xl px-4 py-12 sm:px-6 sm:py-16 lg:px-8">
    <!-- "you are here" mini-map, advanced one step per section -->
    <motion.div
      class="mb-10 flex justify-center"
      :initial="reduce ? false : { opacity: 0, y: 20 }"
      :while-in-view="{ opacity: 1, y: 0 }"
      :viewport="{ once: true, margin: '-90px' }"
      :transition="{ duration: 0.6, ease }"
    >
      <AssemblyDiagram mode="progress" :current-step="n" class="w-full max-w-md" />
    </motion.div>

    <div class="grid items-center gap-10 lg:grid-cols-2 lg:gap-16">
      <motion.div
        :class="flip ? 'lg:order-2' : ''"
        :initial="reduce ? false : { opacity: 0, y: 28 }"
        :while-in-view="{ opacity: 1, y: 0 }"
        :viewport="{ once: true, margin: '-90px' }"
        :transition="{ duration: 0.65, ease }"
      >
        <div class="flex items-center gap-3">
          <img
            :src="seal"
            :alt="chop"
            aria-hidden="true"
            class="h-10 w-10 select-none object-contain"
            loading="lazy"
            decoding="async"
          />
          <span class="text-sm font-medium text-fg-muted">{{ t('landing.stepLabel', { n }) }}</span>
        </div>
        <h2 class="mt-5 font-serif text-3xl font-bold tracking-tight text-fg-primary sm:text-4xl">
          {{ t(title) }}
        </h2>
        <p class="mt-4 max-w-xl text-base leading-relaxed text-fg-secondary sm:text-lg">
          {{ t(body) }}
        </p>
      </motion.div>

      <motion.div
        :class="flip ? 'lg:order-1' : ''"
        :initial="reduce ? false : { opacity: 0, y: 28 }"
        :while-in-view="{ opacity: 1, y: 0 }"
        :viewport="{ once: true, margin: '-90px' }"
        :transition="{ duration: 0.65, delay: 0.08, ease }"
      >
        <StepPeek :kind="kind" />
      </motion.div>
    </div>
  </section>
</template>
