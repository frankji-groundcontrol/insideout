import { useUserStore } from '@/stores/user'

// Runs on both server and client now that sessions live in httpOnly
// cookies (previously skipped the server entirely because the session
// was localStorage-only — see docs/plans/2026-07-20-go-rewrite/04-frontend.md §2).
export default defineNuxtRouteMiddleware(async (to) => {
  const userStore = useUserStore()
  if (!userStore.hydrated) {
    await userStore.hydrate()
  }

  if (to.path === '/login' && userStore.isAuthenticated) {
    return navigateTo('/dashboard')
  }

  if (!to.meta.public && !userStore.isAuthenticated) {
    return navigateTo('/login')
  }
})
