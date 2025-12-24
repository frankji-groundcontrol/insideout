import { MOCK_USER, MOCK_WORKSHOPS, MOCK_NODES } from './data'
import type { UserProfile, Workshop, RoadmapNode } from '@/types'

// 模拟网络延迟 (500ms - 1000ms)
const delay = (ms = 500) =>
  new Promise(resolve => setTimeout(resolve, ms + Math.random() * 500))

export const mockApi = {
  // 用户相关
  auth: {
    login: async (email: string): Promise<UserProfile> => {
      await delay()
      if (email === 'error@test.com') throw new Error('Invalid credentials')
      return MOCK_USER
    },
    getCurrentUser: async (): Promise<UserProfile> => {
      await delay(200)
      return MOCK_USER
    },
    updateProfile: async (data: Partial<UserProfile>): Promise<UserProfile> => {
      await delay()
      return { ...MOCK_USER, ...data }
    }
  },

  // 工作坊相关
  workshop: {
    list: async (): Promise<Workshop[]> => {
      await delay()
      return MOCK_WORKSHOPS
    },
    getById: async (id: string): Promise<Workshop | undefined> => {
      await delay()
      return MOCK_WORKSHOPS.find(w => w.id === id)
    },
    join: async (code: string): Promise<boolean> => {
      await delay()
      // 模拟：只要邀请码是 6 位数就算成功
      if (code.length !== 6) throw new Error('Invalid invitation code')
      return true
    },
    create: async (data: Partial<Workshop>): Promise<Workshop> => {
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
        created_at: new Date().toISOString()
      }
    },
    getRoadmap: async (workshopId: string): Promise<RoadmapNode[]> => {
      await delay()
      return MOCK_NODES.filter(node => node.workshop_id === 'ws_001') // 暂时只返回 ws_001 的数据用于演示
    }
  }
}
