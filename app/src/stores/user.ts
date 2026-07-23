import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { UserProfile } from '@/types'
import { useServices } from '@/composables/useServices'

// Session lives entirely in httpOnly cookies set by the Go API — nothing
// is stored in localStorage. `hydrate()` calls GET /me to discover
// whether the incoming request/browser is authenticated; it's the only
// source of truth. See docs/plans/2026-07-20-go-rewrite/04-frontend.md §2.
export const useUserStore = defineStore('user', () => {
  const user = ref<UserProfile | null>(null)
  const hydrated = ref(false)

  const isAuthenticated = computed(() => !!user.value)

  async function hydrate() {
    const { auth } = useServices()
    try {
      user.value = await auth.getCurrentUser()
    } catch {
      user.value = null
    } finally {
      hydrated.value = true
    }
  }

  async function register(email: string, password: string, username: string) {
    const { auth } = useServices()
    user.value = await auth.register(email, password, username)
  }

  async function login(email: string, password: string) {
    const { auth } = useServices()
    user.value = await auth.login(email, password)
  }

  async function logout() {
    const { auth } = useServices()
    await auth.logout()
    user.value = null
  }

  async function updateProfile(payload: Partial<Pick<UserProfile, 'username' | 'bio' | 'keywords' | 'avatarUrl'>>) {
    const { auth } = useServices()
    user.value = await auth.updateProfile(payload)
  }

  return { user, hydrated, isAuthenticated, hydrate, register, login, logout, updateProfile }
})
