import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { UserProfile } from '@/types'

// localStorage 键名
const TOKEN_KEY = 'juanleme-token'
const USER_KEY = 'juanleme-user'

export const useUserStore = defineStore('user', () => {
  // 状态
  const user = ref<UserProfile | null>(null)
  const token = ref<string | null>(null)

  // 计算属性
  const isAuthenticated = computed(() => !!token.value && !!user.value)

  // 登录 — mock 模式接受任何凭证
  async function login(email: string, _password: string) {
    // 模拟网络延迟
    await new Promise((resolve) => setTimeout(resolve, 500))

    const username = email.split('@')[0] ?? email
    const mockUser: UserProfile = {
      id: `user_${Date.now()}`,
      email,
      username,
      avatar_url: `https://api.dicebear.com/7.x/avataaars/svg?seed=${username}`,
      bio: '',
      role: 'user',
      created_at: new Date().toISOString(),
    }

    const mockToken = `mock_token_${Date.now()}`

    user.value = mockUser
    token.value = mockToken

    // 持久化到 localStorage
    localStorage.setItem(TOKEN_KEY, mockToken)
    localStorage.setItem(USER_KEY, JSON.stringify(mockUser))
  }

  // 登出
  function logout() {
    user.value = null
    token.value = null
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
  }

  // 从 localStorage 恢复会话
  function initFromStorage() {
    const savedToken = localStorage.getItem(TOKEN_KEY)
    const savedUser = localStorage.getItem(USER_KEY)
    if (savedToken && savedUser) {
      try {
        token.value = savedToken
        user.value = JSON.parse(savedUser) as UserProfile
      } catch {
        logout()
      }
    }
  }

  return { user, token, isAuthenticated, login, logout, initFromStorage }
})
