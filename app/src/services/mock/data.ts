import type { UserProfile, Workshop, RoadmapNode } from '@/types'

// 模拟当前用户
export const MOCK_USER: UserProfile = {
  id: 'user_001',
  email: 'frank@juanleme.com',
  username: '坏胖胖',
  avatar_url: 'https://api.dicebear.com/7.x/avataaars/svg?seed=Frank',
  bio: 'Vibe Coding 首席体验官',
  role: 'user',
  created_at: '2024-01-01T00:00:00Z'
}

// 模拟工作坊列表
export const MOCK_WORKSHOPS: Workshop[] = [
  {
    id: 'ws_001',
    title: '周三下午搓个垃圾出来',
    description: '别想太多，先动手做。不管是垃圾还是宝贝，做出来再说！适合所有想要尝试 Vibe Coding 的新手。',
    cover_url: 'https://images.unsplash.com/photo-1517048676732-d65bc937f952?w=800&q=80',
    code: '888888',
    creator_id: 'admin_001',
    status: 'active',
    created_at: '2024-03-20T10:00:00Z',
    member_count: 42,
    is_joined: true
  },
  {
    id: 'ws_002',
    title: 'AI 艺术创作工坊',
    description: '探索 Midjourney 和 Stable Diffusion 的无限可能。',
    cover_url: 'https://images.unsplash.com/photo-1547891654-e66ed7ebb968?w=800&q=80',
    code: '123456',
    creator_id: 'admin_002',
    status: 'active',
    created_at: '2024-03-22T14:00:00Z',
    member_count: 15,
    is_joined: false
  },
  {
    id: 'ws_003',
    title: '旧项目复盘大会',
    description: '把那些烂尾的项目拿出来晒晒，说不定能废物利用。',
    cover_url: 'https://images.unsplash.com/photo-1556761175-5973dc0f32e7?w=800&q=80',
    code: '654321',
    creator_id: 'user_001',
    status: 'completed',
    created_at: '2024-03-01T09:00:00Z',
    member_count: 8,
    is_joined: true
  }
]

// 模拟路线图节点
export const MOCK_NODES: RoadmapNode[] = [
  {
    id: 'node_1',
    workshop_id: 'ws_001',
    title: '👋 破冰环节：自我介绍',
    description: '用一句话介绍你自己，并说出你最想做的一个“垃圾”项目。',
    status: 'completed',
    order: 1,
    content: '大家好，我是 Trae。我想做一个自动给猫铲屎的机器人，但是是用乐高拼的。'
  },
  {
    id: 'node_2',
    workshop_id: 'ws_001',
    title: '🧠 头脑风暴：疯狂的点子',
    description: '不要管可行性，写下 3 个你觉得最离谱的想法。',
    status: 'in_progress',
    order: 2,
    content: ''
  },
  {
    id: 'node_3',
    workshop_id: 'ws_001',
    title: '🎨 原型设计：草图绘制',
    description: '拿出纸和笔，画出你的产品原型。不要在意美丑，关键是逻辑。',
    status: 'pending',
    order: 3
  },
  {
    id: 'node_4',
    workshop_id: 'ws_001',
    title: '💻 核心代码实现',
    description: '选择一个核心功能，用最脏的代码把它跑通。',
    status: 'locked',
    order: 4
  },
  {
    id: 'node_5',
    workshop_id: 'ws_001',
    title: '🎉 展示与庆祝',
    description: '向大家展示你的成果，哪怕它只是一个 Hello World。',
    status: 'locked',
    order: 5
  }
]
