import type { IWorkspaceService } from '@/types/services'
import { apiFetch } from './http'

export function createApiWorkspaceService(): IWorkspaceService {
  return {
    list() {
      return apiFetch('/workspaces')
    },
    get(id) {
      return apiFetch(`/workspaces/${id}`)
    },
    create(title, description) {
      return apiFetch('/workspaces', { method: 'POST', body: { title, description } })
    },
    join(code) {
      return apiFetch('/workspaces/join', { method: 'POST', body: { code } })
    },
    update(id, data) {
      return apiFetch(`/workspaces/${id}`, { method: 'PATCH', body: data })
    },
    async remove(id) {
      await apiFetch(`/workspaces/${id}`, { method: 'DELETE' })
    },
    listMembers(id) {
      return apiFetch(`/workspaces/${id}/members`)
    },
    async updateMemberRole(id, userId, role) {
      await apiFetch(`/workspaces/${id}/members/${userId}`, { method: 'PATCH', body: { role } })
    },
    async removeMember(id, userId) {
      await apiFetch(`/workspaces/${id}/members/${userId}`, { method: 'DELETE' })
    },
  }
}
