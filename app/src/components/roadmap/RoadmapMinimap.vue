<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { CARD_W, CARD_H, type LaidOutNode } from '@/utils/tidyTree'

// Workstream D orientation aid: a miniature of the whole tree with the current
// viewport framed; click or drag to recenter. Full route only (the canvas mounts
// it when `!embedded`). It is an *aid*, not a primary control: keyboard users
// keep the toolbar's zoom/fit buttons, so this is role="img" (an orientation
// picture) with pointer navigation as a mouse enhancement, not a focusable
// widget. ponytail: no keyboard pan grid — the fit button already answers
// "where is everything?".
const props = defineProps<{
  placed: LaidOutNode[]
  contentW: number
  contentH: number
  viewW: number
  viewH: number
  panX: number
  panY: number
  scale: number
}>()
const emit = defineEmits<{ center: [wx: number, wy: number] }>()
const { t } = useI18n()

const W = 176
const H = 120
const PAD = 8

// Fit the content bbox into the box, letterboxed; content top-left at (ox, oy).
const fit = computed(() => {
  const cw = Math.max(1, props.contentW)
  const ch = Math.max(1, props.contentH)
  const k = Math.min((W - PAD * 2) / cw, (H - PAD * 2) / ch)
  return { k, ox: (W - cw * k) / 2, oy: (H - ch * k) / 2 }
})

// The visible world rect, mapped into the minimap and clamped to the content so
// an off-content pan pins the frame to the nearest edge instead of vanishing.
const viewRect = computed(() => {
  const { k, ox, oy } = fit.value
  if (!props.scale || !props.viewW || !props.viewH) return null
  const cw = props.contentW * k
  const ch = props.contentH * k
  const l = ox + (-props.panX / props.scale) * k
  const tp = oy + (-props.panY / props.scale) * k
  const w = (props.viewW / props.scale) * k
  const h = (props.viewH / props.scale) * k
  const cl = Math.min(Math.max(l, ox), ox + cw)
  const ct = Math.min(Math.max(tp, oy), oy + ch)
  return { x: cl, y: ct, w: Math.min(w, ox + cw - cl), h: Math.min(h, oy + ch - ct) }
})

function recenter(e: PointerEvent) {
  const r = (e.currentTarget as SVGSVGElement).getBoundingClientRect()
  const { k, ox, oy } = fit.value
  const wx = Math.min(Math.max((e.clientX - r.left - ox) / k, 0), props.contentW)
  const wy = Math.min(Math.max((e.clientY - r.top - oy) / k, 0), props.contentH)
  emit('center', wx, wy)
}

let dragging = false
function onDown(e: PointerEvent) {
  if (e.button !== 0) return
  dragging = true
  ;(e.currentTarget as SVGSVGElement).setPointerCapture?.(e.pointerId)
  recenter(e)
}
function onMove(e: PointerEvent) {
  if (dragging) recenter(e)
}
function onUp() {
  dragging = false
}
</script>

<template>
  <svg
    :width="W"
    :height="H"
    role="img"
    :aria-label="t('roadmap.canvas.minimap')"
    class="cursor-crosshair touch-none rounded-control border border-stroke-subtle bg-surface-raised/95 shadow-card backdrop-blur-sm"
    @pointerdown="onDown"
    @pointermove="onMove"
    @pointerup="onUp"
    @pointercancel="onUp"
  >
    <!-- Node slivers: a faint silhouette of the tree. -->
    <g aria-hidden="true">
      <rect
        v-for="p in placed"
        :key="p.node.id"
        class="rm-mini-node"
        :x="fit.ox + p.x * fit.k"
        :y="fit.oy + p.y * fit.k"
        :width="Math.max(2, CARD_W * fit.k)"
        :height="Math.max(2, CARD_H * fit.k)"
        rx="1"
      />
    </g>
    <!-- The current viewport, framed. Neutral ink — never vermilion (One Seal). -->
    <rect
      v-if="viewRect"
      class="rm-mini-view"
      :x="viewRect.x"
      :y="viewRect.y"
      :width="viewRect.w"
      :height="viewRect.h"
      rx="2"
    />
  </svg>
</template>

<style scoped>
.rm-mini-node {
  fill: rgb(var(--color-stroke-strong) / 0.55);
}
.rm-mini-view {
  fill: rgb(var(--color-surface-sunken) / 0.15);
  stroke: rgb(var(--color-fg-secondary));
  stroke-width: 1.25;
}
</style>
