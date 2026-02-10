import type { IWorkshopService } from '@/types/services'
import type { Workshop, RoadmapNode, UserSubmission } from '@/types'
import { MOCK_WORKSHOPS, MOCK_NODES } from './data'

const delay = (ms = 500) =>
  new Promise(resolve => setTimeout(resolve, ms + Math.random() * 500))

const SUBMISSION_PREFIX = 'submission:'

export function createMockWorkshopService(): IWorkshopService {
  return {
    async getWorkshops(): Promise<Workshop[]> {
      await delay()
      return MOCK_WORKSHOPS
    },

    async getWorkshop(id: string): Promise<Workshop | undefined> {
      await delay()
      return MOCK_WORKSHOPS.find(w => w.id === id)
    },

    async getRoadmap(_workshopId: string): Promise<RoadmapNode[]> {
      await delay()
      return MOCK_NODES.filter(node => node.workshop_id === 'ws_001')
    },

    async joinWorkshop(code: string): Promise<boolean> {
      await delay()
      if (code.length !== 6) throw new Error('Invalid invitation code')
      return true
    },

    async createWorkshop(data: Partial<Workshop>): Promise<Workshop> {
      await delay()
      return {
        ...MOCK_WORKSHOPS[0],
        id: `ws_${Math.random()}`,
        title: data.title || 'New Workshop',
        description: data.description || 'No description',
        code: data.code || '000000',
        creator_id: data.creator_id || 'user_001',
        status: data.status || 'draft',
        member_count: 1,
        created_at: new Date().toISOString(),
      }
    },

    async updateNodeStatus(
      nodeId: string,
      status: RoadmapNode['status'],
    ): Promise<RoadmapNode> {
      await delay(300)
      const node = MOCK_NODES.find(n => n.id === nodeId)
      if (!node) throw new Error(`Node not found: ${nodeId}`)
      node.status = status
      return { ...node }
    },

    async getSubmission(nodeId: string): Promise<UserSubmission | undefined> {
      await delay(200)
      const raw = localStorage.getItem(`${SUBMISSION_PREFIX}${nodeId}`)
      return raw ? (JSON.parse(raw) as UserSubmission) : undefined
    },

    async saveSubmission(submission: UserSubmission): Promise<UserSubmission> {
      await delay(300)
      localStorage.setItem(
        `${SUBMISSION_PREFIX}${submission.nodeId}`,
        JSON.stringify(submission),
      )
      return submission
    },
  }
}
