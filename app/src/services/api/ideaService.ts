import type { IIdeaService } from '@/types/services'
import { apiFetch } from './http'

export function createApiIdeaService(): IIdeaService {
  return {
    list(workspaceId) {
      return apiFetch(`/workspaces/${workspaceId}/ideas`)
    },
    get(id) {
      return apiFetch(`/ideas/${id}`)
    },
    create(workspaceId, title, content) {
      return apiFetch(`/workspaces/${workspaceId}/ideas`, { method: 'POST', body: { title, content } })
    },
    update(id, title, content) {
      return apiFetch(`/ideas/${id}`, { method: 'PATCH', body: { title, content } })
    },
    async drop(id) {
      await apiFetch(`/ideas/${id}`, { method: 'DELETE' })
    },
    convert(id) {
      return apiFetch(`/ideas/${id}/convert`, { method: 'POST' })
    },
  }
}
