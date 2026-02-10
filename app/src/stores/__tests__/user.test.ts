import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useUserStore } from '@/stores/user'

describe('useUserStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  describe('login', () => {
    it('登录后 isAuthenticated 应为 true 且 user 被填充', async () => {
      const store = useUserStore()

      await store.login('test@test.com', 'any')

      expect(store.isAuthenticated).toBe(true)
      expect(store.user).not.toBeNull()
      expect(store.user!.email).toBe('test@test.com')
      expect(store.token).toBeTruthy()
    })

    it('登录后 username 从 email 的 @ 前部分派生', async () => {
      const store = useUserStore()

      await store.login('hello.world@example.com', 'password123')

      expect(store.user!.username).toBe('hello.world')
    })

    it('登录后用户信息和 token 持久化到 localStorage', async () => {
      const store = useUserStore()

      await store.login('test@test.com', 'pass')

      expect(localStorage.getItem('juanleme-token')).toBeTruthy()
      const savedUser = JSON.parse(localStorage.getItem('juanleme-user')!)
      expect(savedUser.email).toBe('test@test.com')
      expect(savedUser.username).toBe('test')
    })
  })

  describe('logout', () => {
    it('登出后 isAuthenticated 应为 false，user 应为 null', async () => {
      const store = useUserStore()
      await store.login('test@test.com', 'any')

      await store.logout()

      expect(store.isAuthenticated).toBe(false)
      expect(store.user).toBeNull()
      expect(store.token).toBeNull()
    })

    it('登出后 localStorage 中的 token 和 user 被清除', async () => {
      const store = useUserStore()
      await store.login('test@test.com', 'any')

      await store.logout()

      expect(localStorage.getItem('juanleme-token')).toBeNull()
      expect(localStorage.getItem('juanleme-user')).toBeNull()
    })
  })

  describe('initFromStorage', () => {
    it('当 localStorage 有 token 和 user 时恢复会话', () => {
      const mockUser = {
        id: 'user_123',
        email: 'saved@test.com',
        username: 'saved',
        avatar_url: '',
        bio: '',
        role: 'user' as const,
        created_at: '2026-01-01T00:00:00.000Z',
      }
      localStorage.setItem('juanleme-token', 'saved_token')
      localStorage.setItem('juanleme-user', JSON.stringify(mockUser))

      const store = useUserStore()
      store.initFromStorage()

      expect(store.isAuthenticated).toBe(true)
      expect(store.user!.email).toBe('saved@test.com')
      expect(store.token).toBe('saved_token')
    })

    it('当 localStorage 没有 token 时不做任何事', () => {
      const store = useUserStore()
      store.initFromStorage()

      expect(store.isAuthenticated).toBe(false)
      expect(store.user).toBeNull()
      expect(store.token).toBeNull()
    })

    it('当 localStorage 中 user JSON 损坏时调用 logout 清理', () => {
      localStorage.setItem('juanleme-token', 'bad_token')
      localStorage.setItem('juanleme-user', 'not-valid-json')

      const store = useUserStore()
      store.initFromStorage()

      expect(store.isAuthenticated).toBe(false)
      expect(store.user).toBeNull()
      expect(localStorage.getItem('juanleme-token')).toBeNull()
    })
  })
})
