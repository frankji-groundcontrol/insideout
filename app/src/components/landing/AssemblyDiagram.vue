<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

// The Assembly diagram — the landing's signature device, in the build-
// instruction grammar: four ink icons (idea spark → PRD document → roadmap
// tree → shipped seal) joined by dashed guide arrows.
//   `hero`     renders the whole journey in ink with the seal in vermilion and
//              plays the click-in sequence (spark, doc, tree, then the seal
//              stamps down) via CSS keyframes.
//   `progress` renders a step-wizard — dim past / solid present / ghosted
//              future — reused as each section's "you are here" mini-map.
interface Props {
  mode?: 'hero' | 'progress'
  currentStep?: number
  animated?: boolean
}
const props = withDefaults(defineProps<Props>(), {
  mode: 'progress',
  currentStep: 4,
  animated: false,
})

const { t } = useI18n()

const KEYS = ['spark', 'doc', 'tree', 'seal'] as const
const CX = [45, 150, 255, 345]
const CY = 65
const ARROWS = [
  { x1: 82, x2: 113 },
  { x1: 187, x2: 218 },
  { x1: 292, x2: 322 },
]

type State = 'heroInk' | 'heroSeal' | 'done' | 'current' | 'upcoming'
const stateOf = (i: number): State => {
  if (props.mode === 'hero') return i === KEYS.length - 1 ? 'heroSeal' : 'heroInk'
  if (i < props.currentStep - 1) return 'done'
  if (i === props.currentStep - 1) return 'current'
  return 'upcoming'
}
const COLOR: Record<State, string> = {
  heroInk: 'text-fg-primary',
  heroSeal: 'text-seal',
  done: 'text-fg-primary',
  current: 'text-fg-primary',
  upcoming: 'text-stroke-subtle',
}
const OPACITY: Record<State, number> = {
  heroInk: 1,
  heroSeal: 1,
  done: 0.38,
  current: 1,
  upcoming: 0.55,
}
const nodeColor = computed(() => KEYS.map((_, i) => COLOR[stateOf(i)]))
const nodeOpacity = computed(() => KEYS.map((_, i) => OPACITY[stateOf(i)]))

// Arrow `a` (0-based) feeds node a+1; it is "lit" once that node is reached.
const arrowClass = (a: number) =>
  props.mode === 'hero' || a + 1 <= props.currentStep - 1 ? 'text-fg-primary' : 'text-stroke-subtle'
const arrowOpacity = (a: number) =>
  props.mode === 'hero' || a + 1 <= props.currentStep - 1 ? 0.7 : 0.5
</script>

<template>
  <svg viewBox="0 0 400 130" role="img" :aria-label="t('landing.diagramAlt')" class="block h-auto w-full">
    <!-- dashed guide arrows (behind the nodes) -->
    <g v-for="(a, i) in ARROWS" :key="'arrow' + i" :transform="`translate(${a.x1}, ${CY})`">
      <g
        :class="[arrowClass(i), { 'a-arrow-anim': animated }]"
        :style="{ opacity: arrowOpacity(i), '--i': i * 2 + 1 }"
        fill="none"
        stroke="currentColor"
        stroke-width="2.5"
        stroke-linecap="round"
      >
        <line x1="0" y1="0" :x2="a.x2 - a.x1 - 6" y2="0" stroke-dasharray="2.5 5" />
        <path :d="`M ${a.x2 - a.x1 - 6},-4.5 L ${a.x2 - a.x1},0 L ${a.x2 - a.x1 - 6},4.5`" />
      </g>
    </g>

    <!-- the four pieces -->
    <g v-for="(key, i) in KEYS" :key="key" :transform="`translate(${CX[i]}, ${CY})`">
      <g
        :class="[nodeColor[i], { 'a-node-anim': animated, 'a-seal-anim': animated && i === KEYS.length - 1 }]"
        :style="{ opacity: nodeOpacity[i], '--i': i * 2 }"
      >
        <!-- idea spark -->
        <template v-if="key === 'spark'">
          <circle cx="0" cy="0" r="4.5" fill="currentColor" />
          <g stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
            <line x1="9" y1="0" x2="17" y2="0" />
            <line x1="6.4" y1="-6.4" x2="12" y2="-12" />
            <line x1="0" y1="-9" x2="0" y2="-17" />
            <line x1="-6.4" y1="-6.4" x2="-12" y2="-12" />
            <line x1="-9" y1="0" x2="-17" y2="0" />
            <line x1="-6.4" y1="6.4" x2="-12" y2="12" />
            <line x1="0" y1="9" x2="0" y2="17" />
            <line x1="6.4" y1="6.4" x2="12" y2="12" />
          </g>
        </template>

        <!-- PRD document -->
        <template v-else-if="key === 'doc'">
          <g fill="none" stroke="currentColor" stroke-width="3" stroke-linejoin="round" stroke-linecap="round">
            <path d="M -15,-21 H 5 L 15,-11 V 21 H -15 Z" />
            <path d="M 5,-21 V -11 H 15" />
            <path d="M -9,-3 H 6" stroke-width="2.5" />
            <path d="M -9,4 H 6" stroke-width="2.5" />
            <path d="M -9,11 H 1" stroke-width="2.5" />
          </g>
        </template>

        <!-- roadmap tree -->
        <template v-else-if="key === 'tree'">
          <g fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
            <path d="M 0,-26 L -20,-8 M 0,-26 L 20,-8 M -20,-8 L -30,14 M -20,-8 L -10,14 M 20,-8 L 10,14 M 20,-8 L 30,14" />
            <path d="M -34,28 H 34" />
          </g>
          <g fill="currentColor">
            <circle cx="0" cy="-26" r="4" />
            <circle cx="-20" cy="-8" r="4" />
            <circle cx="20" cy="-8" r="4" />
            <circle cx="-30" cy="14" r="4" />
            <circle cx="-10" cy="14" r="4" />
            <circle cx="10" cy="14" r="4" />
            <circle cx="30" cy="14" r="4" />
          </g>
        </template>

        <!-- shipped seal -->
        <template v-else>
          <g fill="none" stroke="currentColor" stroke-width="3.5" stroke-linecap="round" stroke-linejoin="round">
            <rect x="-19" y="-19" width="38" height="38" rx="7" />
            <path d="M -9,1 L -2,8 L 10,-7" />
          </g>
        </template>
      </g>
    </g>
  </svg>
</template>

<style scoped>
/* The assembly click-in. Outer <g> carries the translate; the inner <g> gets
   the scale so the two transforms never fight (attribute vs CSS). */
.a-node-anim {
  transform-box: fill-box;
  transform-origin: center;
  animation: assemble-pop 0.5s cubic-bezier(0.16, 1, 0.3, 1) both;
  animation-delay: calc(var(--i) * 0.14s);
}
.a-seal-anim {
  animation-name: seal-stamp;
  animation-duration: 0.65s;
}
.a-arrow-anim {
  transform-box: fill-box;
  transform-origin: left center;
  animation: arrow-draw 0.35s ease-out both;
  animation-delay: calc(var(--i) * 0.14s);
}
@keyframes assemble-pop {
  0% { opacity: 0; transform: scale(0.25); }
  60% { opacity: 1; transform: scale(1.12); }
  100% { opacity: 1; transform: scale(1); }
}
@keyframes seal-stamp {
  0% { opacity: 0; transform: scale(1.6); }
  55% { opacity: 1; transform: scale(0.88); }
  78% { transform: scale(1.07); }
  100% { opacity: 1; transform: scale(1); }
}
@keyframes arrow-draw {
  from { opacity: 0; transform: scaleX(0); }
  to { opacity: 1; transform: scaleX(1); }
}
@media (prefers-reduced-motion: reduce) {
  .a-node-anim,
  .a-arrow-anim {
    animation: none !important;
  }
}
</style>
