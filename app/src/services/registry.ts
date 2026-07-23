import type {
  IAuthService,
  IWorkspaceService,
  IProjectService,
  IIdeaService,
  IPrdService,
  ICoachService,
  IExportService,
  IRoadmapService,
} from '@/types/services'
import { createApiAuthService } from './api/authService'
import { createApiWorkspaceService } from './api/workspaceService'
import { createApiProjectService } from './api/projectService'
import { createApiIdeaService } from './api/ideaService'
import { createApiPrdService } from './api/prdService'
import { createApiCoachService } from './api/coachService'
import { createApiExportService } from './api/exportService'
import { createApiRoadmapService } from './api/roadmapService'

export interface ServiceRegistry {
  auth: IAuthService
  workspace: IWorkspaceService
  project: IProjectService
  idea: IIdeaService
  prd: IPrdService
  coach: ICoachService
  export: IExportService
  roadmap: IRoadmapService
}

// A single real bundle backed by the Go API — no mock/supabase modes.
// Kept as a factory function (not top-level singletons) so SSR builds a
// fresh set per request and tests can construct their own registry.
export function buildServices(): ServiceRegistry {
  return {
    auth: createApiAuthService(),
    workspace: createApiWorkspaceService(),
    project: createApiProjectService(),
    idea: createApiIdeaService(),
    prd: createApiPrdService(),
    coach: createApiCoachService(),
    export: createApiExportService(),
    roadmap: createApiRoadmapService(),
  }
}

// Sync accessor for composables/components that can't await — populated
// once per request by plugins/services.ts via _setProvidedServices().
let _provided: ServiceRegistry | null = null
export function _setProvidedServices(s: ServiceRegistry): void {
  _provided = s
}
export function requireServices(): ServiceRegistry {
  if (!_provided) {
    throw new Error('services not initialized: ensure plugins/services.ts has run')
  }
  return _provided
}
