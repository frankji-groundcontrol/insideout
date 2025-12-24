export interface UserProfile {
  id: string
  email: string
  username: string
  avatar_url?: string
  bio?: string
  role: 'admin' | 'user'
  created_at: string
}

export interface Workshop {
  id: string
  title: string
  description: string
  cover_url?: string
  code: string // 6位邀请码
  creator_id: string
  status: 'draft' | 'active' | 'completed'
  created_at: string
  member_count: number
  is_joined?: boolean // 当前用户是否已加入
}

export interface RoadmapNode {
  id: string
  workshop_id: string
  title: string
  description: string
  status: 'locked' | 'pending' | 'in_progress' | 'completed'
  order: number
  content?: string // 用户填写的内容
}
