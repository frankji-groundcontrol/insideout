import { onBeforeUnmount, onMounted, ref } from 'vue'

/**
 * Tracks the user's `prefers-reduced-motion` setting. False during SSR, then
 * read synchronously on the client so the gate is correct on the very first
 * render — motion-v resolves each component's `initial` once at construction,
 * so a value that only arrives in `onMounted` is too late to suppress the
 * entrance. Use it to swap motion-v entrance animations for an instantly-
 * visible resting state so motion-sensitive visitors aren't animated at. The
 * AssemblyDiagram's CSS click-in handles the same via a media query.
 */
export function useReducedMotion() {
  const prefersReduced = ref(
    import.meta.client ? window.matchMedia('(prefers-reduced-motion: reduce)').matches : false
  )
  let mq: MediaQueryList | null = null
  const update = () => {
    prefersReduced.value = mq?.matches ?? false
  }
  onMounted(() => {
    mq = window.matchMedia('(prefers-reduced-motion: reduce)')
    update()
    mq.addEventListener('change', update)
  })
  onBeforeUnmount(() => mq?.removeEventListener('change', update))
  return prefersReduced
}
