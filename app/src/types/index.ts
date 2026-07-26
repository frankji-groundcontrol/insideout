// Domain types matching the Go API's camelCase JSON contract — see
// docs/plans/2026-07-20-go-rewrite/02-backend-go.md §3.

export interface UserProfile {
  id: string
  email: string
  username: string
  avatarUrl?: string | null
  bio: string
  keywords: string[]
}

export interface Workspace {
  id: string
  title: string
  description: string
  coverUrl?: string | null
  code: string // 6位邀请码
  status: 'draft' | 'active' | 'completed'
  memberCount: number
  myRole: 'admin' | 'member'
  createdAt: string
}

export interface WorkspaceMember {
  userId: string
  username: string
  email: string
  role: 'admin' | 'member'
}

export type ProjectStatus = 'planning' | 'active' | 'paused' | 'done' | 'archived'
export type ProjectUpdateKind = 'progress' | 'blocker' | 'note'

export interface Project {
  id: string
  workspaceId: string
  title: string
  description: string
  ownerId?: string | null
  status: ProjectStatus
  repoUrl?: string
  createdAt: string
  latestUpdateKind?: ProjectUpdateKind
  latestUpdateContent?: string
  latestUpdateAt?: string
}

export interface ProjectDetail extends Project {
  updates: ProjectUpdate[]
}

export interface ProjectUpdate {
  id: string
  projectId: string
  authorId: string
  kind: ProjectUpdateKind
  content: string
  createdAt: string
}

export type RoadmapStatus = 'locked' | 'pending' | 'in_progress' | 'done'

/** One node in a project's branched roadmap tree (flat form, as the API returns it). */
export interface RoadmapNode {
  id: string
  projectId: string
  parentId: string | null
  title: string
  description: string
  status: RoadmapStatus
  position: number
  createdAt: string
  updatedAt: string
  /** Display name of the creator / last editor (B3 attribution). Absent for
   *  pre-migration rows or a removed author — the card then shows "unknown". */
  creatorName?: string | null
  editorName?: string | null
}

/** A roadmap node with its children assembled — the recursive tree the UI renders. */
export interface RoadmapTreeNode extends RoadmapNode {
  children: RoadmapTreeNode[]
}

export type IdeaStatus = 'inbox' | 'refining' | 'converted' | 'dropped'

export interface Idea {
  id: string
  workspaceId: string
  authorId: string
  title: string
  content: string
  status: IdeaStatus
  prdId?: string | null
  createdAt: string
  updatedAt: string
}

// The 8 fixed PRD sections, in display order — see
// docs/plans/2026-07-20-go-rewrite/03-agents.md §2.
export const PRD_SECTION_KEYS = [
  'background',
  'users',
  'goals',
  'nonGoals',
  'stories',
  'requirements',
  'constraints',
  'risks',
] as const
export type PrdSectionKey = (typeof PRD_SECTION_KEYS)[number]

export type PrdStatus = 'draft' | 'reviewing' | 'approved' | 'rejected'

export interface Prd {
  id: string
  workspaceId: string
  ideaId?: string | null
  projectId?: string | null
  authorId: string
  title: string
  sections: Record<PrdSectionKey, string>
  status: PrdStatus
  currentRevision: number
  updatedAt: string
}

export interface PrdRevision {
  id: string
  revision: number
  sections: Record<PrdSectionKey, string>
  createdBy: string
  note?: string | null
  createdAt: string
}

export type CoachStage = 'clarify' | 'draft' | 'critique' | 'finalize'

export interface Conversation {
  id: string
  workspaceId: string
  prdId: string
  stage: CoachStage
  status: 'active' | 'completed' | 'abandoned'
  updatedAt: string
}

export interface CoachMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  createdAt: string
}

/** One entry in the coach's evidence ledger — see fact_recorded in the
 * SSE contract (docs/plans/2026-07-21-prd-agent-harness/plan.md §4.1). */
export interface CoachFact {
  id: string
  kind: string
  text: string
  status: 'attested' | 'assumed' | 'needs-validation'
}

export class AiRateLimitError extends Error {
  retryAfterSeconds: number
  currentCount?: number
  maxRequests?: number

  constructor(
    message: string,
    retryAfterSeconds: number,
    currentCount?: number,
    maxRequests?: number,
  ) {
    super(message)
    this.name = 'AiRateLimitError'
    this.retryAfterSeconds = retryAfterSeconds
    this.currentCount = currentCount
    this.maxRequests = maxRequests
  }
}

export class AiServiceUnavailableError extends Error {
  retryAfterSeconds: number
  circuitState?: string

  constructor(message: string, retryAfterSeconds: number, circuitState?: string) {
    super(message)
    this.name = 'AiServiceUnavailableError'
    this.retryAfterSeconds = retryAfterSeconds
    this.circuitState = circuitState
  }
}

/** 409 from build-from-PRD when the live roadmap has nodes the user hasn't confirmed replacing. */
export class RoadmapReplaceConflictError extends Error {
  liveCount: number

  constructor(message: string, liveCount: number) {
    super(message)
    this.name = 'RoadmapReplaceConflictError'
    this.liveCount = liveCount
  }
}

export type ExportFormat = 'markdown' | 'print'
