import type { IRoadmapService } from '@/types/services'
import { apiFetch } from './http'

export function createApiRoadmapService(): IRoadmapService {
  return {
    list(projectId) {
      return apiFetch(`/projects/${projectId}/roadmap`)
    },
    create(projectId, data) {
      return apiFetch(`/projects/${projectId}/roadmap`, { method: 'POST', body: data })
    },
    update(nodeId, data) {
      return apiFetch(`/roadmap/${nodeId}`, { method: 'PATCH', body: data })
    },
    move(nodeId, parentId, position) {
      return apiFetch(`/roadmap/${nodeId}/move`, { method: 'POST', body: { parentId, position } })
    },
    async remove(nodeId) {
      await apiFetch(`/roadmap/${nodeId}`, { method: 'DELETE' })
    },
    expand(nodeId) {
      return apiFetch(`/roadmap/${nodeId}/expand`, { method: 'POST' })
    },
  }
}
