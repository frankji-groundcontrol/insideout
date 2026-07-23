import type { IAuthService } from '@/types/services'
import { apiFetch } from './http'

export function createApiAuthService(): IAuthService {
  return {
    register(email, password, username) {
      return apiFetch('/auth/register', { method: 'POST', body: { email, password, username } })
    },
    login(email, password) {
      return apiFetch('/auth/login', { method: 'POST', body: { email, password } })
    },
    getCurrentUser() {
      return apiFetch('/me')
    },
    updateProfile(data) {
      return apiFetch('/me', { method: 'PATCH', body: data })
    },
    async logout() {
      await apiFetch('/auth/logout', { method: 'POST' })
    },
  }
}
