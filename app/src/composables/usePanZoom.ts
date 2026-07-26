import { computed, onBeforeUnmount, ref } from 'vue'

/**
 * Pan/zoom state machine for the roadmap canvas viewport. The world container
 * is transformed by `translate(x, y) scale(scale)`; all interaction math lives
 * here so the component stays declarative.
 *
 * Conventions (map-style, so an embedded canvas never fights page scroll):
 * - drag the background to pan (drags starting on a node card are the canvas's
 *   reparent gesture and are ignored here),
 * - wheel pans (trackpad two-finger scroll),
 * - ctrl/cmd + wheel (or pinch, which browsers report as ctrl+wheel) zooms at
 *   the cursor.
 */
export function usePanZoom(opts: { min?: number; max?: number } = {}) {
  const MIN = opts.min ?? 0.25
  const MAX = opts.max ?? 2.5

  const x = ref(0)
  const y = ref(0)
  const scale = ref(1)
  /** True once the user takes manual control; auto-fit stops touching the view. */
  const interacted = ref(false)
  const panning = ref(false)

  const clampScale = (s: number) => Math.min(MAX, Math.max(MIN, s))

  function panBy(dx: number, dy: number) {
    x.value += dx
    y.value += dy
    interacted.value = true
  }

  /** Zoom by `factor`, keeping the viewport point (cx, cy) visually fixed. */
  function zoomAt(factor: number, cx: number, cy: number) {
    const next = clampScale(scale.value * factor)
    const k = next / scale.value
    x.value = cx - (cx - x.value) * k
    y.value = cy - (cy - y.value) * k
    scale.value = next
    interacted.value = true
  }

  /**
   * Center the whole tree inside the padded box; never zooms past 100% on fit.
   * Per-side insets let callers reserve room for UI that floats over the
   * viewport (e.g. the canvas toolbar band across the top), so a fitted node
   * never lands underneath it.
   */
  function fitTo(
    worldW: number,
    worldH: number,
    viewW: number,
    viewH: number,
    pad: { top: number; right: number; bottom: number; left: number } = { top: 48, right: 48, bottom: 48, left: 48 },
  ) {
    if (!worldW || !worldH || !viewW || !viewH) return
    const availW = viewW - pad.left - pad.right
    const availH = viewH - pad.top - pad.bottom
    if (availW <= 0 || availH <= 0) return
    const s = clampScale(Math.min(availW / worldW, availH / worldH, 1))
    scale.value = s
    x.value = pad.left + (availW - worldW * s) / 2
    y.value = pad.top + (availH - worldH * s) / 2
  }

  /** Pan (no zoom) so world point (wx, wy) sits at the viewport center. Used by
   *  the minimap's click/drag-to-navigate; marks the view user-controlled so the
   *  auto-fit stops fighting it. */
  function centerOn(wx: number, wy: number, viewW: number, viewH: number) {
    if (!viewW || !viewH) return
    x.value = viewW / 2 - wx * scale.value
    y.value = viewH / 2 - wy * scale.value
    interacted.value = true
  }

  /** Convert a client (screen) point to world coordinates. */
  function toWorld(clientX: number, clientY: number, viewportRect: DOMRect) {
    return {
      wx: (clientX - viewportRect.left - x.value) / scale.value,
      wy: (clientY - viewportRect.top - y.value) / scale.value,
    }
  }

  const worldStyle = computed(() => ({
    transform: `translate(${x.value}px, ${y.value}px) scale(${scale.value})`,
    transformOrigin: '0 0',
  }))

  // --- DOM wiring -----------------------------------------------------------

  let panSession: { sx: number; sy: number; ox: number; oy: number } | null = null

  function onPointerMove(e: PointerEvent) {
    if (!panSession) return
    panning.value = true
    x.value = panSession.ox + (e.clientX - panSession.sx)
    y.value = panSession.oy + (e.clientY - panSession.sy)
  }

  function endPan() {
    if (panSession) interacted.value = true
    panSession = null
    panning.value = false
    window.removeEventListener('pointermove', onPointerMove)
    window.removeEventListener('pointerup', endPan)
    window.removeEventListener('pointercancel', endPan)
  }

  /** Attach pan/zoom listeners to the viewport element. Call from onMounted. */
  function bindViewport(el: HTMLElement) {
    const onPointerDown = (e: PointerEvent) => {
      if (e.button !== 0) return
      // Node cards opt out — their drags mean "reparent", not "pan".
      if ((e.target as HTMLElement).closest('[data-rm-node]')) return
      panSession = { sx: e.clientX, sy: e.clientY, ox: x.value, oy: y.value }
      window.addEventListener('pointermove', onPointerMove)
      window.addEventListener('pointerup', endPan)
      window.addEventListener('pointercancel', endPan)
    }
    const onWheel = (e: WheelEvent) => {
      e.preventDefault()
      const rect = el.getBoundingClientRect()
      if (e.ctrlKey || e.metaKey) {
        zoomAt(Math.exp(-e.deltaY * 0.0016), e.clientX - rect.left, e.clientY - rect.top)
      } else {
        panBy(-e.deltaX, -e.deltaY)
      }
    }
    el.addEventListener('pointerdown', onPointerDown)
    el.addEventListener('wheel', onWheel, { passive: false })
    onBeforeUnmount(() => {
      el.removeEventListener('pointerdown', onPointerDown)
      el.removeEventListener('wheel', onWheel)
      endPan()
    })
  }

  return { x, y, scale, interacted, panning, panBy, zoomAt, fitTo, centerOn, toWorld, worldStyle, bindViewport }
}
