import type { IAuthService } from '@/types/services'
import type { UserProfile } from '@/types'
import { MOCK_USER } from './data'
import { safeStorage } from '@/lib/safeStorage'

const delay = (ms = 500) =>
  new Promise(resolve => setTimeout(resolve, ms + Math.random() * 500))

export function createMockAuthService(): IAuthService {
  return {
    async login(email: string): Promise<UserProfile> {
      await delay()
      if (email === 'error@test.com') throw new Error('Invalid credentials')
      return { ...MOCK_USER, email, username: email.split('@')[0] || email }
    },

    async getCurrentUser(): Promise<UserProfile> {
      await delay(200)
      return MOCK_USER
    },

    async updateProfile(data: Partial<UserProfile>): Promise<UserProfile> {
      await delay()
      return { ...MOCK_USER, ...data }
    },

    async logout(): Promise<void> {
      await delay(100)
      safeStorage.removeItem('juanleme-token')
      safeStorage.removeItem('juanleme-user')
    },
  }
}
