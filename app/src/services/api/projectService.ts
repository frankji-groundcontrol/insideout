import type { IProjectService } from '@/types/services'
import { apiFetch } from './http'

export function createApiProjectService(): IProjectService {
  return {
    list(workspaceId) {
      return apiFetch(`/workspaces/${workspaceId}/projects`)
    },
    get(id) {
      return apiFetch(`/projects/${id}`)
    },
    create(workspaceId, title, description) {
      return apiFetch(`/workspaces/${workspaceId}/projects`, { method: 'POST', body: { title, description } })
    },
    update(id, data) {
      return apiFetch(`/projects/${id}`, { method: 'PATCH', body: data })
    },
    async remove(id) {
      await apiFetch(`/projects/${id}`, { method: 'DELETE' })
    },
    async addUpdate(projectId, kind, content) {
      await apiFetch(`/projects/${projectId}/updates`, { method: 'POST', body: { kind, content } })
    },
    async editUpdate(updateId, content) {
      await apiFetch(`/updates/${updateId}`, { method: 'PATCH', body: { content } })
    },
    async removeUpdate(updateId) {
      await apiFetch(`/updates/${updateId}`, { method: 'DELETE' })
    },
    setRepo(id, repoUrl) {
      return apiFetch(`/projects/${id}/repo`, { method: 'PUT', body: { repoUrl } })
    },
    syncGithub(id) {
      return apiFetch(`/projects/${id}/sync-github`, { method: 'POST' })
    },
  }
}
