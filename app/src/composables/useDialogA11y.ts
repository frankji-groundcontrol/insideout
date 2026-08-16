import { watch, nextTick, onBeforeUnmount, type Ref } from 'vue'

/**
 * Reusable dialog a11y behavior — extracted verbatim from
 * components/common/BaseModal.vue (the a11y-reviewed modal) so other dialog
 * shells (e.g. auth/AuthDoor.vue) can reuse it. Owns the focus trap
 * (Escape → close, Tab wrap), body scroll lock, focus save/restore, and the
 * keydown listener lifecycle. Behavior is intentionally identical to the
 * original, including the import.meta.client SSR guards.
 */
export function useDialogA11y(
  panelRef: Ref<HTMLElement | null>,
  isOpen: Ref<boolean>,
  onClose: () => void,
) {
  let previouslyFocused: HTMLElement | null = null

  const FOCUSABLE =
    'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'

  function visibleFocusables(): HTMLElement[] {
    if (!panelRef.value) return []
    return Array.from(panelRef.value.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
      (el) => el.offsetParent !== null,
    )
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.stopPropagation()
      onClose()
      return
    }
    if (e.key !== 'Tab') return
    const focusables = visibleFocusables()
    if (!focusables.length) return
    const first = focusables[0]!
    const last = focusables[focusables.length - 1]!
    const active = document.activeElement as HTMLElement | null
    if (e.shiftKey && active === first) {
      e.preventDefault()
      last.focus()
    } else if (!e.shiftKey && active === last) {
      e.preventDefault()
      first.focus()
    }
  }

  watch(
    isOpen,
    async (open) => {
      if (!import.meta.client) return
      if (open) {
        previouslyFocused = document.activeElement as HTMLElement | null
        document.body.style.overflow = 'hidden'
        document.addEventListener('keydown', onKeydown, true)
        await nextTick()
        const target = visibleFocusables()[0] ?? panelRef.value
        target?.focus()
      } else {
        document.body.style.overflow = ''
        document.removeEventListener('keydown', onKeydown, true)
        previouslyFocused?.focus?.()
        previouslyFocused = null
      }
    },
  )

  onBeforeUnmount(() => {
    if (!import.meta.client) return
    document.body.style.overflow = ''
    document.removeEventListener('keydown', onKeydown, true)
  })
}
