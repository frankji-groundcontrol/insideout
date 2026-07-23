<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { motion } from 'motion-v'
import BaseButton from '@/components/common/BaseButton.vue'
import AssemblyHero from '@/components/landing/AssemblyHero.vue'
import AssemblyStep from '@/components/landing/AssemblyStep.vue'
import { useReducedMotion } from '@/composables/useReducedMotion'

// 首页（公开）：产品落地页 — 「一步步 / The Assembly」。
// A build-instruction narrative in the Ink & Seal world: idea spark → PRD →
// branched roadmap → shipped seal, each piece clicking into place. The
// AssemblyDiagram recurs as a hero device and as each step's "you are here"
// mini-map. 路由元信息 public:true 供 middleware/auth.global.ts 放行未认证访问。
definePageMeta({ public: true })

const { t } = useI18n()
const reduce = useReducedMotion()
const ease = [0.16, 1, 0.3, 1] as const

const steps = [
  { n: 1, chop: '落墨', title: 'landing.step1Title', body: 'landing.step1Body', kind: 'idea' as const },
  { n: 2, chop: '成文', title: 'landing.step2Title', body: 'landing.step2Body', kind: 'prd' as const },
  { n: 3, chop: '分枝', title: 'landing.step3Title', body: 'landing.step3Body', kind: 'roadmap' as const },
  { n: 4, chop: '盖印', title: 'landing.step4Title', body: 'landing.step4Body', kind: 'shipped' as const },
]
</script>

<template>
  <div class="relative w-full overflow-hidden">
    <AssemblyHero />

    <main>
      <AssemblyStep v-for="(s, i) in steps" :key="s.n" v-bind="s" :flip="i % 2 === 1" />
    </main>

    <!-- close: press your first seal -->
    <section class="relative mx-auto max-w-3xl px-4 py-20 text-center sm:px-6 sm:py-24 lg:px-8">
      <motion.div
        :initial="reduce ? false : { opacity: 0, y: 24 }"
        :while-in-view="{ opacity: 1, y: 0 }"
        :viewport="{ once: true, margin: '-80px' }"
        :transition="{ duration: 0.7, ease }"
      >
        <span
          aria-hidden="true"
          class="mx-auto flex h-14 w-14 select-none items-center justify-center rounded-lg bg-seal font-serif text-xl font-bold text-carve"
        >印</span>
        <h2 class="mt-6 font-serif text-3xl font-bold tracking-tight text-fg-primary sm:text-4xl">
          {{ t('landing.ctaCloseTitle') }}
        </h2>
        <p class="mx-auto mt-4 max-w-xl text-lg leading-relaxed text-fg-secondary">
          {{ t('landing.ctaCloseBody') }}
        </p>
        <div class="mt-8 flex justify-center">
          <BaseButton to="/register" size="lg">{{ t('landing.ctaCloseButton') }}</BaseButton>
        </div>
      </motion.div>
    </section>
  </div>
</template>
