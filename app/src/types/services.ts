import type {
  UserProfile,
  Workspace,
  WorkspaceMember,
  Project,
  ProjectDetail,
  ProjectUpdateKind,
  Idea,
  Prd,
  PrdRevision,
  Conversation,
  CoachFact,
  CoachMessage,
  ExportFormat,
  RoadmapNode,
  RoadmapStatus,
} from '@/types'

export interface IAuthService {
  register(email: string, password: string, username: string): Promise<UserProfile>
  login(email: string, password: string): Promise<UserProfile>
  getCurrentUser(): Promise<UserProfile>
  updateProfile(data: Partial<Pick<UserProfile, 'username' | 'bio' | 'keywords' | 'avatarUrl'>>): Promise<UserProfile>
  logout(): Promise<void>
}

export interface IWorkspaceService {
  list(): Promise<Workspace[]>
  get(id: string): Promise<Workspace>
  create(title: string, description: string): Promise<Workspace>
  join(code: string): Promise<Workspace>
  update(id: string, data: Partial<Pick<Workspace, 'title' | 'description' | 'coverUrl'>>): Promise<Workspace>
  remove(id: string): Promise<void>
  listMembers(id: string): Promise<WorkspaceMember[]>
  updateMemberRole(id: string, userId: string, role: 'admin' | 'member'): Promise<void>
  removeMember(id: string, userId: string): Promise<void>
}

export interface IProjectService {
  list(workspaceId: string): Promise<Project[]>
  get(id: string): Promise<ProjectDetail>
  create(workspaceId: string, title: string, description: string): Promise<Project>
  update(
    id: string,
    data: { title: string; description: string; status: Project['status']; ownerId?: string | null },
  ): Promise<Project>
  remove(id: string): Promise<void>
  addUpdate(projectId: string, kind: ProjectUpdateKind, content: string): Promise<void>
  editUpdate(updateId: string, content: string): Promise<void>
  removeUpdate(updateId: string): Promise<void>
  setRepo(id: string, repoUrl: string): Promise<Project>
  syncGithub(id: string): Promise<{ added: number; repoUrl: string }>
}

export interface IIdeaService {
  list(workspaceId: string): Promise<Idea[]>
  get(id: string): Promise<Idea>
  create(workspaceId: string, title: string, content: string): Promise<Idea>
  update(id: string, title: string, content: string): Promise<Idea>
  drop(id: string): Promise<void>
  convert(id: string): Promise<{ prdId: string; conversationId: string }>
}

export interface IPrdService {
  get(id: string): Promise<Prd>
  /** title is optional: pass null on a section-only save so the stored title
   *  is left untouched (a section save must never clobber a concurrent title
   *  edit). Pass a string only to deliberately rename the PRD. */
  updateSections(
    id: string,
    title: string | null,
    sections: Partial<Prd['sections']>,
  ): Promise<Prd>
  listRevisions(id: string): Promise<PrdRevision[]>
  createRevision(id: string, note?: string): Promise<PrdRevision>
  updateStatus(id: string, status: Prd['status']): Promise<Prd>
  /** Turns the PRD into a project with an AI-generated branched roadmap.
   *  If the live roadmap is non-empty, the call 409s with the live count;
   *  confirm by retrying with that expectedCount. */
  build(id: string, expectedCount?: number): Promise<{ projectId: string; nodeCount: number }>
}

export interface ICoachService {
  getConversation(id: string): Promise<Conversation>
  /** Resolves to null if this PRD has no coach conversation yet. */
  getConversationForPrd(prdId: string): Promise<Conversation | null>
  listMessages(id: string): Promise<CoachMessage[]>
  /**
   * Sends a message and streams the reply over SSE. Callbacks fire as
   * events arrive; the returned promise resolves once the stream ends.
   * See docs/plans/2026-07-20-go-rewrite/03-agents.md §4 for the event
   * contract this mirrors.
   */
  send(
    conversationId: string,
    content: string,
    handlers: {
      onDelta?: (text: string) => void
      onPrdUpdated?: (section: string) => void
      onStageChanged?: (stage: string) => void
      onFactRecorded?: (fact: CoachFact) => void
      onDone?: () => void
    },
  ): Promise<void>
}

export interface IExportService {
  download(prdId: string, format: ExportFormat): Promise<{ content: string; contentType: string }>
}

export interface IRoadmapService {
  list(projectId: string): Promise<RoadmapNode[]>
  create(projectId: string, data: { parentId?: string | null; title: string; description?: string }): Promise<RoadmapNode>
  /** Partial update — only the keys passed are written; the rest are untouched. */
  update(nodeId: string, data: Partial<{ title: string; description: string; status: RoadmapStatus }>): Promise<RoadmapNode>
  move(nodeId: string, parentId: string | null, position: number): Promise<RoadmapNode>
  remove(nodeId: string): Promise<void>
  /** AI: break a node into subtasks (appended as its children). */
  expand(nodeId: string): Promise<RoadmapNode[]>
}
