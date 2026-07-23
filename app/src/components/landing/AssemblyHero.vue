<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { motion } from 'motion-v'
import BaseButton from '@/components/common/BaseButton.vue'
import AssemblyDiagram from '@/components/landing/AssemblyDiagram.vue'
import { useReducedMotion } from '@/composables/useReducedMotion'

// The first viewport: a quiet celadon field, the Song-serif claim, the ink
// CTA, and the Assembly diagram assembling itself piece by piece below.
const { t } = useI18n()
const reduce = useReducedMotion()
const ease = [0.16, 1, 0.3, 1] as const
</script>

<template>
  <div class="relative mx-auto max-w-5xl px-4 pb-12 pt-20 text-center sm:px-6 sm:pt-24 lg:px-8">
    <motion.h1
      class="font-serif text-4xl font-bold leading-[1.15] tracking-tight text-fg-primary sm:text-6xl"
      :initial="reduce ? false : { opacity: 0, y: 24 }"
      :animate="{ opacity: 1, y: 0 }"
      :transition="{ duration: 0.7, ease }"
    >
      {{ t('landing.heroTitle') }}
    </motion.h1>
    <motion.p
      class="mx-auto mt-6 max-w-2xl text-lg leading-relaxed text-fg-secondary"
      :initial="reduce ? false : { opacity: 0, y: 20 }"
      :animate="{ opacity: 1, y: 0 }"
      :transition="{ duration: 0.6, delay: 0.15, ease }"
    >
      {{ t('landing.heroSubtitle') }}
    </motion.p>
    <motion.div
      class="mt-10 flex flex-wrap items-center justify-center gap-4"
      :initial="reduce ? false : { opacity: 0, y: 20 }"
      :animate="{ opacity: 1, y: 0 }"
      :transition="{ duration: 0.6, delay: 0.3, ease }"
    >
      <BaseButton to="/register" size="lg">{{ t('landing.ctaPrimary') }}</BaseButton>
      <BaseButton to="/login" size="lg" variant="outline">{{ t('landing.ctaSecondary') }}</BaseButton>
    </motion.div>

    <!-- the signature device, assembling itself -->
    <div class="mx-auto mt-16 max-w-3xl sm:mt-20">
      <AssemblyDiagram mode="hero" animated />
    </div>
  </div>
</template>
