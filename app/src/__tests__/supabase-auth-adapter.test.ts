import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { SupabaseClient } from '@supabase/supabase-js'
import type { UserProfile } from '@/types'
import type { IAuthService } from '@/types/services'
import { createSupabaseAuthService } from '@/services/supabase/authService'

const mockSignInWithOtp = vi.fn()
const mockGetUser = vi.fn()
const mockSignOut = vi.fn()
const mockFrom = vi.fn()

function createQueryBuilder(resolvedValue: unknown) {
  const builder = {
    select: vi.fn().mockReturnThis(),
    insert: vi.fn().mockReturnThis(),
    update: vi.fn().mockReturnThis(),
    upsert: vi.fn().mockReturnThis(),
    eq: vi.fn().mockReturnThis(),
    single: vi.fn().mockResolvedValue(resolvedValue),
    maybeSingle: vi.fn().mockResolvedValue(resolvedValue),
  }
  return builder
}

// 注入式 stub 客户端：适配器工厂现在接收 SupabaseClient 参数（请求作用域），
// 不再从 @/lib/supabase 取模块单例。
function createStubClient(): SupabaseClient {
  return {
    auth: {
      signInWithOtp: mockSignInWithOtp,
      getUser: mockGetUser,
      signOut: mockSignOut,
    },
    schema: () => ({ from: mockFrom }),
    from: mockFrom,
  } as unknown as SupabaseClient
}

const FAKE_USER_ID = '550e8400-e29b-41d4-a716-446655440000'
const FAKE_EMAIL = 'test@juanleme.com'
const FAKE_PROFILE: UserProfile = {
  id: FAKE_USER_ID,
  email: FAKE_EMAIL,
  username: 'test',
  avatar_url: 'https://example.com/avatar.png',
  bio: '测试用户',
  keywords: ['vue', 'supabase'],
  role: 'user',
  created_at: '2024-01-01T00:00:00Z',
}

describe('Supabase Auth 适配器', () => {
  let authService: IAuthService

  beforeEach(() => {
    vi.clearAllMocks()

    authService = createSupabaseAuthService(createStubClient())
  })

  describe('login', () => {
    it('调用 supabase.auth.signInWithOtp 并返回 UserProfile', async () => {
      mockSignInWithOtp.mockResolvedValue({
        data: { user: null, session: null },
        error: null,
      })
      mockGetUser.mockResolvedValue({
        data: { user: { id: FAKE_USER_ID, email: FAKE_EMAIL } },
        error: null,
      })

      const queryBuilder = createQueryBuilder({
        data: FAKE_PROFILE,
        error: null,
      })
      mockFrom.mockReturnValue(queryBuilder)

      const result = await authService.login(FAKE_EMAIL)

      expect(mockSignInWithOtp).toHaveBeenCalledWith({ email: FAKE_EMAIL })
      expect(result).toHaveProperty('id')
      expect(result).toHaveProperty('email')
      expect(result).toHaveProperty('username')
      expect(result).toHaveProperty('role')
      expect(result).toHaveProperty('created_at')
    })

    it('登录失败时抛出错误', async () => {
      mockSignInWithOtp.mockResolvedValue({
        data: { user: null, session: null },
        error: { message: 'Invalid email' },
      })

      await expect(authService.login('bad@test.com')).rejects.toThrow()
    })
  })

  describe('getCurrentUser', () => {
    it('获取当前认证用户并从 profiles 表返回 UserProfile', async () => {
      mockGetUser.mockResolvedValue({
        data: { user: { id: FAKE_USER_ID, email: FAKE_EMAIL } },
        error: null,
      })

      const queryBuilder = createQueryBuilder({
        data: FAKE_PROFILE,
        error: null,
      })
      mockFrom.mockReturnValue(queryBuilder)

      const result = await authService.getCurrentUser()

      expect(mockGetUser).toHaveBeenCalled()
      expect(result).toHaveProperty('id')
      expect(result).toHaveProperty('email')
      expect(result).toHaveProperty('username')
      expect(result).toHaveProperty('role')
      expect(result).toHaveProperty('created_at')
    })

    it('未登录时抛出错误', async () => {
      mockGetUser.mockResolvedValue({
        data: { user: null },
        error: { message: 'Not authenticated' },
      })

      await expect(authService.getCurrentUser()).rejects.toThrow()
    })
  })

  describe('updateProfile', () => {
    it('更新 profiles 表中允许的字段并返回更新后的 UserProfile', async () => {
      mockGetUser.mockResolvedValue({
        data: { user: { id: FAKE_USER_ID, email: FAKE_EMAIL } },
        error: null,
      })

      const updatedProfile = {
        ...FAKE_PROFILE,
        username: '新名字',
        bio: '更新后的简介',
      }

      const queryBuilder = createQueryBuilder({
        data: updatedProfile,
        error: null,
      })
      mockFrom.mockReturnValue(queryBuilder)

      const result = await authService.updateProfile({
        username: '新名字',
        bio: '更新后的简介',
      })

      expect(mockFrom).toHaveBeenCalledWith('profiles')
      expect(result).toHaveProperty('id')
      expect(result).toHaveProperty('email')
      expect(result).toHaveProperty('username')
      expect(result).toHaveProperty('role')
    })

    it('不允许通过 updateProfile 修改 role 字段', async () => {
      mockGetUser.mockResolvedValue({
        data: { user: { id: FAKE_USER_ID, email: FAKE_EMAIL } },
        error: null,
      })

      const queryBuilder = createQueryBuilder({
        data: FAKE_PROFILE,
        error: null,
      })
      mockFrom.mockReturnValue(queryBuilder)

      await authService.updateProfile({
        username: '新名字',
        role: 'admin',
      } as Partial<UserProfile>)

      const updateCall = queryBuilder.update.mock.calls[0]
      if (updateCall) {
        const updateData = updateCall[0] as Record<string, unknown>
        expect(updateData).not.toHaveProperty('role')
      }
    })

    it('不允许通过 updateProfile 修改 email 字段', async () => {
      mockGetUser.mockResolvedValue({
        data: { user: { id: FAKE_USER_ID, email: FAKE_EMAIL } },
        error: null,
      })

      const queryBuilder = createQueryBuilder({
        data: FAKE_PROFILE,
        error: null,
      })
      mockFrom.mockReturnValue(queryBuilder)

      await authService.updateProfile({
        email: 'hacked@evil.com',
      } as Partial<UserProfile>)

      const updateCall = queryBuilder.update.mock.calls[0]
      if (updateCall) {
        const updateData = updateCall[0] as Record<string, unknown>
        expect(updateData).not.toHaveProperty('email')
      }
    })
  })

  describe('logout', () => {
    it('调用 supabase.auth.signOut', async () => {
      mockSignOut.mockResolvedValue({ error: null })

      await authService.logout()

      expect(mockSignOut).toHaveBeenCalled()
    })
  })

  describe('契约一致性', () => {
    it('实现了 IAuthService 的所有方法', () => {
      expect(typeof authService.login).toBe('function')
      expect(typeof authService.getCurrentUser).toBe('function')
      expect(typeof authService.updateProfile).toBe('function')
      expect(typeof authService.logout).toBe('function')
    })
  })
})
