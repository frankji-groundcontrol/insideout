import { onBeforeUnmount, onMounted, ref } from 'vue'

export function useWorkshopViewport(mediaQueryText = '(max-width: 767px)') {
  const isMobile = ref(false)
  let mediaQuery: MediaQueryList | null = null

  function updateViewportState(event: MediaQueryList | MediaQueryListEvent) {
    isMobile.value = event.matches
  }

  onMounted(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
      return
    }

    mediaQuery = window.matchMedia(mediaQueryText)
    updateViewportState(mediaQuery)
    mediaQuery.addEventListener('change', updateViewportState)
  })

  onBeforeUnmount(() => {
    if (!mediaQuery) {
      return
    }

    mediaQuery.removeEventListener('change', updateViewportState)
    mediaQuery = null
  })

  return {
    isMobile,
  }
}
