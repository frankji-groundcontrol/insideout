import { useUserStore } from '@/stores/user'

// 导航守卫：未认证用户只能访问公开页面
export default defineNuxtRouteMiddleware((to) => {
  // 会话仅存于 localStorage，SSR 阶段恒为未认证；
  // 跳过服务端以避免把所有受保护路由都重定向到 /login。
  if (import.meta.server) return

  const userStore = useUserStore()

  if (to.path === '/login' && userStore.isAuthenticated) {
    return navigateTo('/dashboard')
  }

  if (!to.meta.public && !userStore.isAuthenticated) {
    return navigateTo('/login')
  }
})
