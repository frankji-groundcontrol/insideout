import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { UserProfile } from '@/types'
import { useServices } from '@/composables/useServices'

// localStorage 键名
const TOKEN_KEY = 'juanleme-token'
const USER_KEY = 'juanleme-user'

export const useUserStore = defineStore('user', () => {
  // 状态
  const user = ref<UserProfile | null>(null)
  const token = ref<string | null>(null)

  // 计算属性
  const isAuthenticated = computed(() => !!token.value && !!user.value)

  // 持久化用户会话到 localStorage（localStorage 仅在客户端可用，SSR 期间跳过写入）
  function persistSession(profile: UserProfile, sessionToken: string) {
    user.value = profile
    token.value = sessionToken
    if (import.meta.client) {
      localStorage.setItem(TOKEN_KEY, sessionToken)
      localStorage.setItem(USER_KEY, JSON.stringify(profile))
    }
  }

  // 清除本地会话状态（localStorage 仅在客户端可用，SSR 期间跳过清理）
  function clearSession() {
    user.value = null
    token.value = null
    if (import.meta.client) {
      localStorage.removeItem(TOKEN_KEY)
      localStorage.removeItem(USER_KEY)
    }
  }

  // 登录 — 委托给 auth 服务
  async function login(email: string, _password: string) {
    const { auth } = useServices()
    const profile = await auth.login(email)
    const sessionToken = `session_${Date.now()}`
    persistSession(profile, sessionToken)
  }

  // 登出 — 委托给 auth 服务并清理本地状态
  async function logout() {
    const { auth } = useServices()
    await auth.logout()
    clearSession()
  }

  // 从 localStorage 恢复会话（会话仅存于 localStorage，SSR 期间没有可恢复的状态，直接跳过）
  function initFromStorage() {
    if (!import.meta.client) {
      return
    }

    const savedToken = localStorage.getItem(TOKEN_KEY)
    const savedUser = localStorage.getItem(USER_KEY)
    if (savedToken && savedUser) {
      try {
        token.value = savedToken
        user.value = JSON.parse(savedUser) as UserProfile
      } catch {
        clearSession()
      }
    }
  }

  // 更新用户资料 — 委托给 auth 服务
  async function updateProfile(payload: Partial<UserProfile>) {
    if (!user.value) {
      return
    }

    const { auth } = useServices()
    const updated = await auth.updateProfile(payload)
    user.value = updated
    if (import.meta.client) {
      localStorage.setItem(USER_KEY, JSON.stringify(updated))
    }
  }

  return { user, token, isAuthenticated, login, logout, initFromStorage, updateProfile }
})
