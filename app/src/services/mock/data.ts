import type { UserProfile, Workshop, RoadmapNode } from '@/types'

// AI 回复模板类型
export interface AiTemplate {
  keywords: string[]
  response: string
}

// AI 回复模板（关键词匹配）
export const AI_TEMPLATES: AiTemplate[] = [
  {
    keywords: ['灵感', '想法', 'inspiration'],
    response:
      '💡 很棒的方向！试试这个思路：先把你最疯狂的想法写下来，不要在意可行性。' +
      '然后从中挑一个最让你兴奋的，用一句话描述它能给别人带来什么价值。' +
      '记住，最好的灵感往往来自生活中的小痛点！',
  },
  {
    keywords: ['产品', '功能', 'feature'],
    response:
      '🚀 关于产品功能，建议你先问自己三个问题：\n' +
      '1. 用户最核心的痛点是什么？\n' +
      '2. 解决这个痛点的最简方案是什么？\n' +
      '3. 做出来之后，你会不会自己天天用？\n' +
      '先做 MVP，快速验证，别想着一步到位！',
  },
  {
    keywords: ['设计', 'UI', '界面'],
    response:
      '🎨 设计建议：先别打开 Figma！拿张纸画个草图。\n' +
      '核心原则：少即是多。先用黑白灰把信息层级搞清楚，再加颜色。' +
      '推荐参考 Dribbble 上的作品找找感觉，但别照抄——你的用户场景是独特的。',
  },
  {
    keywords: ['技术', '代码', 'code'],
    response:
      '💻 技术选型建议：选你最熟悉的技术栈，别在工具上纠结。\n' +
      '先跑通核心链路，代码丑不要紧，能跑就行。' +
      '重构的事情以后再说——先让它 work，再让它 right，最后让它 fast。',
  },
  {
    keywords: ['文案', 'copy', '写'],
    response:
      '✍️ 写文案的秘诀：假装你在跟朋友聊天，用最自然的语气说出来。\n' +
      '好文案的公式：痛点 + 解决方案 + 行动号召。' +
      '写完之后大声读一遍，如果听起来像广告，就重写。',
  },
  {
    keywords: ['团队', '协作', 'team'],
    response:
      '🤝 团队协作小贴士：明确分工很重要，但别搞成流水线。\n' +
      '建议每天花 15 分钟同步进度，遇到问题马上沟通，别憋着。' +
      '记住：信任是协作的基础，多给队友正反馈！',
  },
]

// 默认 AI 回复（无关键词匹配时使用）
export const AI_DEFAULT_RESPONSE =
  '🌟 加油！你正在创造一些很酷的东西。\n' +
  '不管现在进展如何，坚持做下去就对了。' +
  '有什么具体的问题，随时可以问我——关于灵感、产品、设计、技术或文案，我都可以帮忙！'

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
