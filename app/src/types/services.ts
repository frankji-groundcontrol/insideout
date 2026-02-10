import type {
  UserProfile,
  Workshop,
  RoadmapNode,
  EditorDraft,
  AiMessage,
  AiConversation,
  ExportConfig,
  UserSubmission,
} from '@/types'

export interface IAuthService {
  login(email: string): Promise<UserProfile>
  getCurrentUser(): Promise<UserProfile>
  updateProfile(data: Partial<UserProfile>): Promise<UserProfile>
  logout(): Promise<void>
}

export interface IWorkshopService {
  getWorkshops(): Promise<Workshop[]>
  getWorkshop(id: string): Promise<Workshop | undefined>
  getRoadmap(workshopId: string): Promise<RoadmapNode[]>
  joinWorkshop(code: string): Promise<boolean>
  createWorkshop(data: Partial<Workshop>): Promise<Workshop>
  updateNodeStatus(
    nodeId: string,
    status: RoadmapNode['status'],
  ): Promise<RoadmapNode>
  getSubmission(nodeId: string): Promise<UserSubmission | undefined>
  saveSubmission(submission: UserSubmission): Promise<UserSubmission>
}

export interface IEditorService {
  loadDraft(key: EditorDraft['key']): Promise<EditorDraft | undefined>
  saveDraft(draft: EditorDraft): Promise<EditorDraft>
  deleteDraft(key: EditorDraft['key']): Promise<void>
}

export interface IAiService {
  reply(nodeId: string, message: string): Promise<AiMessage>
  getConversation(nodeId: string): Promise<AiConversation | undefined>
  clearConversation(nodeId: string): Promise<void>
}

export interface IExportService {
  generateMarkdown(config: ExportConfig): Promise<string>
  generatePrintHTML(config: ExportConfig): Promise<string>
}
