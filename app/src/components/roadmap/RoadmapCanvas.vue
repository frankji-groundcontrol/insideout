<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useServices } from '@/composables/useServices'
import { usePanZoom } from '@/composables/usePanZoom'
import { layoutForest, isDescendant, siblingBands, CARD_W, CARD_H, type LaidOutNode } from '@/utils/tidyTree'
import type { RoadmapNode } from '@/types'
import RoadmapCanvasNode from './RoadmapCanvasNode.vue'
import RoadmapMinimap from './RoadmapMinimap.vue'
import { PlusIcon, MinusIcon, ArrowsPointingOutIcon, ArrowTopRightOnSquareIcon, MapIcon, EyeIcon, EyeSlashIcon } from '@heroicons/vue/24/outline'

// The roadmap as a spatial tree on a pan/zoom canvas. One component, two
// placements: `embedded` (project page, fixed-height card shell) and the
// full-viewport route. Hand-rolled — SVG edges + absolutely-positioned HTML
// cards inside a transformed world container (see docs/plans/2026-07-24).
const props = withDefaults(defineProps<{ projectId: string; embedded?: boolean }>(), { embedded: false })
const { t } = useI18n()

const loading = ref(true)
const nodes = ref<RoadmapNode[]>([])
const addingGoal = ref(false)
const newGoalTitle = ref('')
const busy = ref(false)
// A failed load must show an error, not masquerade as an empty roadmap. Held as
// the message to surface (empty = no error); reused by node mutations too, whose
// recovery is also "reload the tree".
const loadError = ref('')

// B1 review/present mode — a view-state deep-link, NOT access control (D11). Under
// the flat invite model every recipient is still an editor; `?review=1` only seeds
// a read-only *view* they can toggle off. The URL is an entry point, never live-
// synced, so it is not rewritten on toggle.
const route = useRoute()
const reviewing = ref(route.query.review === '1')

const viewportEl = ref<HTMLElement | null>(null)
const { x, y, scale, interacted, worldStyle, zoomAt, fitTo, centerOn, toWorld, bindViewport } = usePanZoom()

const layout = computed(() => layoutForest(nodes.value))
// Workstream D: a quiet panel behind each parallel sibling set (≥2 children).
const bands = computed(() => siblingBands(layout.value.placed))
// Viewport pixel size, kept current for the minimap's framed "you are here" rect.
const viewSize = ref({ w: 0, h: 0 })
const total = computed(() => nodes.value.length)
const doneCount = computed(() => nodes.value.filter((n) => n.status === 'done').length)
const progressPct = computed(() => (total.value === 0 ? 0 : Math.round((doneCount.value / total.value) * 100)))

// id → position lookups for edge geometry.
const posById = computed(() => new Map(layout.value.placed.map((p) => [p.node.id, p])))

// C2: edges are neutral hairlines — node seals own status (One Seal rarity). A
// branch is "hot" when any node in its subtree was touched within 7 days; hot
// edges get a quiet emphasis (a lead's scan aid), deliberately a coarser signal
// than the card's exact "updated X ago" (collab Decisions). Mark recent nodes,
// then propagate hot up each ancestor chain in one pass.
const HOT_WINDOW_MS = 7 * 86_400_000
const hotIds = computed(() => {
  const now = Date.now()
  const parentOf = new Map(nodes.value.map((n) => [n.id, n.parentId]))
  const hot = new Set<string>()
  for (const n of nodes.value) {
    if (!n.updatedAt || now - new Date(n.updatedAt).getTime() > HOT_WINDOW_MS) continue
    let cur: string | null = n.id
    let guard = 0
    while (cur && !hot.has(cur) && guard++ <= nodes.value.length) {
      hot.add(cur)
      cur = parentOf.get(cur) ?? null
    }
  }
  return hot
})
function edgeHot(toId: string): boolean {
  return hotIds.value.has(toId)
}
function edgePath(fromId: string, toId: string): string {
  const a = posById.value.get(fromId)
  const b = posById.value.get(toId)
  if (!a || !b) return ''
  const sx = a.x + CARD_W
  const sy = a.y + CARD_H / 2
  const tx = b.x
  const ty = b.y + CARD_H / 2
  const dx = Math.max(24, (tx - sx) / 2)
  return `M ${sx} ${sy} C ${sx + dx} ${sy}, ${tx - dx} ${ty}, ${tx} ${ty}`
}

// A2 focus-refresh: `load()` (initial / retry, shows the skeleton) and a silent
// `refresh()` (never touches `loading`, never toasts — a failure just retries on
// the next focus). A monotonic seq id keeps latest-wins: a stale response is
// dropped, so a focus-refresh racing the initial load can't clobber fresh data or
// strand the skeleton. A node deleted elsewhere closes its own popover: cards are
// keyed by id, so it unmounts with the tree.
let fetchSeq = 0
async function fetchTree(silent: boolean) {
  const seq = ++fetchSeq
  if (!silent) {
    loading.value = true
    loadError.value = ''
  }
  let ok = false
  try {
    const fresh = await useServices().roadmap.list(props.projectId)
    if (seq !== fetchSeq) return
    nodes.value = fresh
    loadError.value = ''
    ok = true
  } catch {
    if (seq === fetchSeq && !silent) loadError.value = t('error.genericBody')
  } finally {
    // Only the latest request owns the skeleton; a stale one must not clear it.
    if (seq === fetchSeq) loading.value = false
  }
  if (!ok) return
  await nextTick()
  if (!interacted.value) refit()
}
function load() {
  return fetchTree(false)
}
function refresh() {
  return fetchTree(true)
}

// Converge when the tab comes back — the only sync mechanism (no polling).
// Covers both placements: embedded and the full-viewport route run this component.
function onVisibility() {
  if (document.visibilityState === 'visible') refresh()
}

// Reserve room for the floating toolbar band across the top so a fitted node
// never lands under it (its hover actions would be unreachable).
const TOOLBAR_INSET = { top: 96, right: 48, bottom: 48, left: 48 }
function refit() {
  const el = viewportEl.value
  if (!el || layout.value.placed.length === 0) return
  const rect = el.getBoundingClientRect()
  fitTo(layout.value.width, layout.value.height, rect.width, rect.height, TOOLBAR_INSET)
}

// Node mutations surface their failure here; recovery is "reload the tree".
function onNodeError(msg: string) {
  loadError.value = msg || t('error.genericBody')
}

function zoomStep(factor: number) {
  const el = viewportEl.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  zoomAt(factor, rect.width / 2, rect.height / 2)
}

// Workstream D: minimap click/drag recenters the world on the picked point.
function onMinimapCenter(wx: number, wy: number) {
  centerOn(wx, wy, viewSize.value.w, viewSize.value.h)
}

async function addGoal() {
  const title = newGoalTitle.value.trim()
  if (!title) return
  busy.value = true
  try {
    await useServices().roadmap.create(props.projectId, { title })
    newGoalTitle.value = ''
    addingGoal.value = false
    await load()
  } catch {
    // Keep the draft in the form so the user can retry; surface the failure.
    loadError.value = t('error.genericBody')
  } finally {
    busy.value = false
  }
}

// --- Drag to reparent -------------------------------------------------------
// Press a card body (not a button/input) and drag: onto another card to make it
// the parent, onto empty canvas to move to root. Cycle-guarded by isDescendant.
const dragNode = ref<LaidOutNode | null>(null)
const dragGhost = ref<{ x: number; y: number } | null>(null)
// C1: the same cursor point in world space, so the prospective edge (drawn in
// the SVG's coordinate system) can track it. `dragGhost` stays in viewport px.
const dragGhostWorld = ref<{ x: number; y: number } | null>(null)
const dropTargetId = ref<string | null>(null)
let dragStart: { cx: number; cy: number; node: LaidOutNode } | null = null
let dragMoved = false

// C1: the "this will attach here" hint — a dashed hairline from the prospective
// parent's right-center to the cursor (world space). Only while over a valid
// target; over empty canvas (drop → root) there is no anchor to draw from.
const prospectivePath = computed(() => {
  const target = dropTargetId.value
  const g = dragGhostWorld.value
  if (!target || !g) return ''
  const a = posById.value.get(target)
  if (!a) return ''
  const sx = a.x + CARD_W
  const sy = a.y + CARD_H / 2
  const dx = Math.max(24, Math.abs(g.x - sx) / 2)
  return `M ${sx} ${sy} C ${sx + dx} ${sy}, ${g.x - dx} ${g.y}, ${g.x} ${g.y}`
})

function onCardPointerDown(e: PointerEvent, p: LaidOutNode) {
  if (reviewing.value) return // B1: read-only — no reparent gesture
  if (e.button !== 0) return
  if ((e.target as HTMLElement).closest('button, input, a, form, textarea, select')) return
  dragStart = { cx: e.clientX, cy: e.clientY, node: p }
  dragMoved = false
  window.addEventListener('pointermove', onDragMove)
  window.addEventListener('pointerup', onDragUp)
  window.addEventListener('pointercancel', onDragCancel)
}

function onDragMove(e: PointerEvent) {
  const el = viewportEl.value
  if (!dragStart || !el) return
  if (!dragMoved && Math.hypot(e.clientX - dragStart.cx, e.clientY - dragStart.cy) < 5) return
  dragMoved = true
  dragNode.value = dragStart.node
  const rect = el.getBoundingClientRect()
  dragGhost.value = { x: e.clientX - rect.left, y: e.clientY - rect.top }
  const w = toWorld(e.clientX, e.clientY, rect)
  dragGhostWorld.value = { x: w.wx, y: w.wy }
  dropTargetId.value = hitTest(w.wx, w.wy, dragStart.node.node.id)
}

function hitTest(wx: number, wy: number, draggedId: string): string | null {
  for (const p of layout.value.placed) {
    if (p.node.id === draggedId) continue
    if (isDescendant(nodes.value, draggedId, p.node.id)) continue
    if (wx >= p.x && wx <= p.x + CARD_W && wy >= p.y && wy <= p.y + CARD_H) return p.node.id
  }
  return null
}

function endDrag() {
  window.removeEventListener('pointermove', onDragMove)
  window.removeEventListener('pointerup', onDragUp)
  window.removeEventListener('pointercancel', onDragCancel)
  dragStart = null
  dragMoved = false
  dragNode.value = null
  dragGhost.value = null
  dragGhostWorld.value = null
  dropTargetId.value = null
}

function onDragCancel() {
  // The OS cancelled the pointer (palm rejection, incoming call, system gesture):
  // drop the gesture entirely and do NOT apply a reparent.
  endDrag()
}

async function onDragUp() {
  const shouldMove = dragStart && dragMoved && dragNode.value
  const id = dragNode.value?.node.id
  const target = dropTargetId.value
  endDrag()
  if (shouldMove && id) {
    try {
      await useServices().roadmap.move(id, target, 0)
      await load()
    } catch {
      // Leave the tree as-is on a failed move.
    }
  }
}

let resizeObserver: ResizeObserver | null = null
onMounted(async () => {
  if (viewportEl.value) {
    bindViewport(viewportEl.value)
    const r = viewportEl.value.getBoundingClientRect()
    viewSize.value = { w: r.width, h: r.height }
    resizeObserver = new ResizeObserver(() => {
      const rr = viewportEl.value?.getBoundingClientRect()
      if (rr) viewSize.value = { w: rr.width, h: rr.height }
      if (!interacted.value) refit()
    })
    resizeObserver.observe(viewportEl.value)
  }
  document.addEventListener('visibilitychange', onVisibility)
  window.addEventListener('focus', refresh)
  await load()
})
onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  document.removeEventListener('visibilitychange', onVisibility)
  window.removeEventListener('focus', refresh)
  window.removeEventListener('pointermove', onDragMove)
  window.removeEventListener('pointerup', onDragUp)
  window.removeEventListener('pointercancel', onDragCancel)
})
</script>

<template>
  <div
    ref="viewportEl"
    class="relative h-full min-h-[340px] w-full touch-none overflow-hidden bg-surface-sunken"
    :class="{ 'rounded-card border border-stroke-subtle': embedded, 'cursor-grabbing': dragNode }"
    role="tree"
    tabindex="-1"
    :aria-label="t('roadmap.title')"
  >
    <!-- World: transformed container holding edges + cards. -->
    <div class="absolute left-0 top-0" :style="worldStyle">
      <svg
        v-if="layout.placed.length"
        class="absolute left-0 top-0 overflow-visible"
        :width="layout.width"
        :height="layout.height"
        aria-hidden="true"
      >
        <!-- Workstream D: quiet panels behind each parallel sibling set, drawn
             under the edges so the hairlines + cards stay on top. -->
        <rect
          v-for="b in bands"
          :key="`band-${b.parentId}`"
          class="rm-band"
          :x="b.x"
          :y="b.y"
          :width="b.w"
          :height="b.h"
          rx="16"
        />
        <path
          v-for="e in layout.edges"
          :key="`${e.fromId}->${e.toId}`"
          :d="edgePath(e.fromId, e.toId)"
          :class="edgeHot(e.toId) ? 'rm-edge rm-edge-hot' : 'rm-edge'"
          fill="none"
          stroke-linecap="round"
        />
        <!-- C1: prospective edge, drawn above the tree while dragging. -->
        <path v-if="prospectivePath" :d="prospectivePath" class="rm-edge-prospective" fill="none" stroke-linecap="round" />
      </svg>

      <div
        v-for="p in layout.placed"
        :key="p.node.id"
        class="rm-card absolute left-0 top-0"
        :style="{ transform: `translate(${p.x}px, ${p.y}px)`, width: `${CARD_W}px`, height: `${CARD_H}px` }"
        data-rm-node
        :data-rm-node-id="p.node.id"
        role="treeitem"
        :aria-expanded="p.node.children.length ? true : undefined"
        :class="{
          'opacity-40': dragNode && dragNode.node.id === p.node.id,
          'ring-2 ring-seal ring-offset-2 ring-offset-surface-sunken rounded-card': dropTargetId === p.node.id,
        }"
        @pointerdown="onCardPointerDown($event, p)"
      >
        <RoadmapCanvasNode
          :node="p.node"
          :project-id="projectId"
          :open-upward="p.y + CARD_H > layout.height / 2"
          :reviewing="reviewing"
          @refresh="load"
          @error="onNodeError"
        />
      </div>
    </div>

    <!-- Drag ghost: follows the cursor in viewport space, scale-independent. -->
    <div
      v-if="dragNode && dragGhost"
      class="pointer-events-none absolute z-30 flex items-center rounded-card border border-seal bg-surface-raised px-3 py-2 text-sm font-medium text-fg-primary shadow-popover"
      :style="{ left: `${dragGhost.x + 12}px`, top: `${dragGhost.y + 12}px`, maxWidth: `${CARD_W}px` }"
    >
      <span class="truncate">{{ dragNode.node.title }}</span>
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading" class="absolute inset-0 flex items-center justify-center gap-6 p-8">
      <div v-for="i in 3" :key="i" class="h-24 w-60 animate-pulse rounded-card bg-surface-raised" />
    </div>

    <!-- Error state (distinct from empty: a failed load must not look like "no roadmap") -->
    <div
      v-else-if="loadError"
      class="absolute inset-0 flex flex-col items-center justify-center gap-3 p-8 text-center"
    >
      <p class="max-w-sm text-sm font-medium text-fg-secondary">{{ t('error.genericTitle') }}</p>
      <p class="max-w-sm text-sm text-fg-muted">{{ loadError }}</p>
      <button
        type="button"
        class="mt-1 rounded-control border border-stroke-subtle bg-surface-raised px-3 py-1.5 text-sm font-medium text-fg-primary hover:bg-surface-sunken"
        @click="load"
      >
        {{ t('common.retry') }}
      </button>
    </div>

    <!-- Empty state -->
    <div
      v-else-if="!layout.placed.length"
      class="absolute inset-0 flex flex-col items-center justify-center gap-4 p-8 text-center"
    >
      <MapIcon class="h-8 w-8 text-fg-muted" />
      <p class="max-w-sm text-sm text-fg-muted">{{ t('roadmap.empty') }}</p>
      <form v-if="!reviewing" class="flex w-full max-w-sm gap-2" @submit.prevent="addGoal">
        <input
          v-model="newGoalTitle"
          type="text"
          :placeholder="t('roadmap.rootPlaceholder')"
          class="flex-1 rounded-control border border-stroke-subtle bg-surface-base px-3 py-2 text-sm text-fg-primary focus:border-stroke-focus focus:outline-none"
        />
        <button type="submit" :disabled="busy" class="inline-flex items-center rounded-control bg-btn px-3 py-2 text-sm font-medium text-btn-fg hover:opacity-90">
          <PlusIcon class="-ml-0.5 mr-1 h-4 w-4" />
          {{ t('roadmap.addRoot') }}
        </button>
      </form>
    </div>

    <!-- Toolbar (top-right, viewport space). Ink buttons, no new vermilion fills. -->
    <div class="absolute right-3 top-3 z-20 flex flex-col items-end gap-2">
      <div class="flex items-center gap-1 rounded-control border border-stroke-subtle bg-surface-raised/95 p-1 shadow-card backdrop-blur-sm">
        <button type="button" class="rounded p-1.5 text-fg-muted hover:bg-surface-sunken hover:text-fg-primary" :title="t('roadmap.canvas.zoomOut')" :aria-label="t('roadmap.canvas.zoomOut')" @click="zoomStep(1 / 1.2)">
          <MinusIcon class="h-4 w-4" />
        </button>
        <span class="w-11 text-center text-xs tabular-nums text-fg-muted">{{ Math.round(scale * 100) }}%</span>
        <button type="button" class="rounded p-1.5 text-fg-muted hover:bg-surface-sunken hover:text-fg-primary" :title="t('roadmap.canvas.zoomIn')" :aria-label="t('roadmap.canvas.zoomIn')" @click="zoomStep(1.2)">
          <PlusIcon class="h-4 w-4" />
        </button>
        <span class="mx-0.5 h-4 w-px bg-stroke-subtle" />
        <button type="button" class="rounded p-1.5 text-fg-muted hover:bg-surface-sunken hover:text-fg-primary" :title="t('roadmap.canvas.fitView')" :aria-label="t('roadmap.canvas.fitView')" @click="interacted = false; refit()">
          <ArrowsPointingOutIcon class="h-4 w-4" />
        </button>
        <span class="mx-0.5 h-4 w-px bg-stroke-subtle" />
        <!-- B1: review-mode toggle (a view state, not access control — D11).
             aria-pressed carries the on/off state; the icon swaps to match. -->
        <button
          type="button"
          class="rounded p-1.5 hover:bg-surface-sunken"
          :class="reviewing ? 'text-fg-primary' : 'text-fg-muted hover:text-fg-primary'"
          :title="t('roadmap.reviewToggle')"
          :aria-label="t('roadmap.reviewToggle')"
          :aria-pressed="reviewing"
          @click="reviewing = !reviewing"
        >
          <EyeSlashIcon v-if="reviewing" class="h-4 w-4" />
          <EyeIcon v-else class="h-4 w-4" />
        </button>
        <template v-if="embedded">
          <span class="mx-0.5 h-4 w-px bg-stroke-subtle" />
          <NuxtLink
            :to="`/projects/${projectId}/roadmap`"
            class="rounded p-1.5 text-fg-muted hover:bg-surface-sunken hover:text-fg-primary"
            :title="t('roadmap.canvas.openFull')"
            :aria-label="t('roadmap.canvas.openFull')"
          >
            <ArrowTopRightOnSquareIcon class="h-4 w-4" />
          </NuxtLink>
        </template>
      </div>

      <div class="flex items-center gap-2">
        <!-- B1: persistent read-only chip. Never vermilion (One Seal Rule) — a
             neutral sunken tag so it reads as a state, not a call to action. -->
        <div
          v-if="reviewing"
          class="flex items-center gap-1.5 rounded-control border border-stroke-subtle bg-surface-sunken px-2.5 py-1.5 shadow-card"
        >
          <EyeIcon class="h-3.5 w-3.5 text-fg-secondary" />
          <span class="text-xs font-medium text-fg-secondary">{{ t('roadmap.reviewing') }}</span>
        </div>
        <div v-if="total > 0" class="flex items-center gap-2 rounded-control border border-stroke-subtle bg-surface-raised/95 px-2.5 py-1.5 shadow-card backdrop-blur-sm">
          <div class="h-1.5 w-20 overflow-hidden rounded-pill bg-surface-sunken">
            <div class="h-full rounded-pill bg-seal transition-all" :style="{ width: progressPct + '%' }" />
          </div>
          <span class="text-xs tabular-nums text-fg-muted">{{ doneCount }}/{{ total }}</span>
        </div>
        <button
          v-if="!reviewing"
          type="button"
          class="inline-flex items-center rounded-control bg-btn px-2.5 py-1.5 text-xs font-medium text-btn-fg shadow-card hover:opacity-90"
          @click="addingGoal = !addingGoal"
        >
          <PlusIcon class="-ml-0.5 mr-1 h-3.5 w-3.5" />
          {{ t('roadmap.addRoot') }}
        </button>
      </div>

      <!-- Add-goal popover -->
      <form
        v-if="addingGoal && !reviewing"
        class="flex w-[260px] gap-2 rounded-card border border-stroke-subtle bg-surface-raised p-3 shadow-popover"
        @submit.prevent="addGoal"
      >
        <input
          v-model="newGoalTitle"
          type="text"
          :placeholder="t('roadmap.rootPlaceholder')"
          autofocus
          class="flex-1 rounded-control border border-stroke-subtle bg-surface-base px-2 py-1 text-sm text-fg-primary focus:border-stroke-focus focus:outline-none"
        />
        <button type="submit" :disabled="busy" class="rounded-control bg-btn px-2.5 py-1 text-xs font-medium text-btn-fg">{{ t('roadmap.add') }}</button>
      </form>
    </div>

    <!-- Workstream D: minimap / orientation aid — full route only (the embedded
         canvas is a glance; the full route is the territory to orient in). -->
    <RoadmapMinimap
      v-if="!embedded && !loading && layout.placed.length"
      class="absolute bottom-3 left-3 z-20"
      :placed="layout.placed"
      :content-w="layout.width"
      :content-h="layout.height"
      :view-w="viewSize.w"
      :view-h="viewSize.h"
      :pan-x="x"
      :pan-y="y"
      :scale="scale"
      @center="onMinimapCenter"
    />
  </div>
</template>

<style scoped>
/* C2: edges are neutral hairlines; a "hot" branch (touched within 7d) gets a
   quiet emphasis. In dark mode the stroke tokens barely differ (~13/channel),
   so hot edges also bump to 2.5px to stay legible (collab D3/C2). These classes
   override the paths' presentation attributes. */
.rm-edge {
  stroke: rgb(var(--color-stroke-subtle));
  stroke-width: 1.5;
}
.rm-edge-hot {
  stroke: rgb(var(--color-stroke-strong));
}
.dark .rm-edge-hot {
  stroke-width: 2.5;
}
/* C1: the prospective drag edge — dashed, so it reads as tentative. */
.rm-edge-prospective {
  stroke: rgb(var(--color-stroke-subtle));
  stroke-width: 1.5;
  stroke-dasharray: 4 4;
}
/* Workstream D: cards glide (don't teleport) when the layout re-flows after a
   data change. Wrappers are keyed by node id, so they keep their DOM node
   across re-renders and the transform transition fires only on genuine
   repositions — fresh mounts appear in place. Gated on the native media query:
   a CSS concern, unlike the motion-v entrance swaps that useReducedMotion
   gates. ponytail: rung-4 CSS over wiring the composable (CoachPanel precedent). */
.rm-card {
  transition: transform 260ms cubic-bezier(0.22, 1, 0.36, 1);
}
@media (prefers-reduced-motion: reduce) {
  .rm-card {
    transition: none;
  }
}
/* An inline transform makes every wrapper its own stacking context, so a
   card's add-child/edit popover (z-20) cannot rise above the sibling painted
   just below it — the form overflows under that sibling, its Add button is
   unclickable, and a press-slip there arms a reparent-drag (silent data
   mutation). Popovers autofocus and hold focus while open, so lifting the
   focused card above its z-auto siblings clears the overlap for add-child AND
   edit, both directions, with no JS. ponytail: tabbing fully out sinks it —
   fine, it is no longer the target; if that ever bites, key z-index off an
   emitted open state instead. */
.rm-card:focus-within {
  z-index: 1;
}
/* Workstream D: the parallel-track band — a whisper of a panel on the sunken
   ground. Neutral only, never vermilion (One Seal Rule). */
.rm-band {
  fill: rgb(var(--color-surface-raised) / 0.55);
  stroke: rgb(var(--color-stroke-strong) / 0.5);
  stroke-width: 1;
}
.dark .rm-band {
  fill: rgb(var(--color-surface-raised) / 0.4);
}
</style>
