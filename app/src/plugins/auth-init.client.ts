import { useUserStore } from '@/stores/user'

// 客户端启动时从 localStorage 恢复会话，需在路由守卫之前完成。
export default defineNuxtPlugin(() => {
  const userStore = useUserStore()
  userStore.initFromStorage()
})
