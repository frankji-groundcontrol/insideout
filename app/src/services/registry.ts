import type {
  IAiService,
  IAuthService,
  IEditorService,
  IExportService,
  IWorkshopService,
} from '@/types/services'
import { createMockAiService } from './mock/aiService'
import { createMockAuthService } from './mock/authService'
import { createMockEditorService } from './mock/editorService'
import { createMockExportService } from './mock/exportService'
import { createMockWorkshopService } from './mock/workshopService'

export interface ServiceRegistry {
  auth: IAuthService
  workshop: IWorkshopService
  editor: IEditorService
  ai: IAiService
  export: IExportService
}

type AdapterMode = 'mock' | 'supabase'

export type BundleModes = {
  auth: AdapterMode
  data: AdapterMode
  aiExport: AdapterMode
}

let _services: ServiceRegistry | null = null
let _servicesPromise: Promise<ServiceRegistry> | null = null

export function resolveBundleModes(): BundleModes {
  const globalMode = (import.meta.env.VITE_API_MODE || 'mock') as AdapterMode
  return {
    auth: (import.meta.env.VITE_BUNDLE_AUTH as AdapterMode) || globalMode,
    data: (import.meta.env.VITE_BUNDLE_DATA as AdapterMode) || globalMode,
    aiExport: (import.meta.env.VITE_BUNDLE_AI_EXPORT as AdapterMode) || globalMode,
  }
}

export function validateBundleModes(modes: BundleModes): void {
  const validValues: ReadonlyArray<AdapterMode> = ['mock', 'supabase']

  for (const [key, value] of Object.entries(modes)) {
    if (!validValues.includes(value as AdapterMode)) {
      throw new Error(`Bundle "${key}" 配置值无效："${value}"，仅支持 mock | supabase`)
    }
  }

  if (modes.data === 'supabase' && modes.auth !== 'supabase') {
    throw new Error(
      'Bundle 配置冲突：data=supabase 要求 auth=supabase（工作坊/编辑器依赖 Supabase 认证会话）',
    )
  }

  if (modes.aiExport === 'supabase' && modes.auth !== 'supabase') {
    throw new Error(
      'Bundle 配置冲突：aiExport=supabase 要求 auth=supabase（AI/导出依赖 Supabase 认证会话）',
    )
  }
}

function isAllMock(modes: BundleModes): boolean {
  return modes.auth === 'mock' && modes.data === 'mock' && modes.aiExport === 'mock'
}

function createMockServices(): ServiceRegistry {
  return {
    auth: createMockAuthService(),
    workshop: createMockWorkshopService(),
    editor: createMockEditorService(),
    ai: createMockAiService(),
    export: createMockExportService(),
  }
}

async function createBundledServices(modes: BundleModes): Promise<ServiceRegistry> {
  const registry: Partial<ServiceRegistry> = {}

  if (modes.auth === 'supabase') {
    const mod = await import('./supabase/authService')
    registry.auth = mod.createSupabaseAuthService()
  } else {
    registry.auth = createMockAuthService()
  }

  if (modes.data === 'supabase') {
    const [workshopMod, editorMod] = await Promise.all([
      import('./supabase/workshopService'),
      import('./supabase/editorService'),
    ])
    registry.workshop = workshopMod.createSupabaseWorkshopService()
    registry.editor = editorMod.createSupabaseEditorService()
  } else {
    registry.workshop = createMockWorkshopService()
    registry.editor = createMockEditorService()
  }

  if (modes.aiExport === 'supabase') {
    const [aiMod, exportMod] = await Promise.all([
      import('./supabase/aiService'),
      import('./supabase/exportService'),
    ])
    registry.ai = aiMod.createSupabaseAiService()
    registry.export = exportMod.createSupabaseExportService()
  } else {
    registry.ai = createMockAiService()
    registry.export = createMockExportService()
  }

  return registry as ServiceRegistry
}

export function getServices(): ServiceRegistry {
  if (!_services) {
    const modes = resolveBundleModes()
    validateBundleModes(modes)
    if (isAllMock(modes)) {
      _services = createMockServices()
    } else {
      throw new Error(
        '同步模式仅支持全 mock 配置，请使用 getServicesAsync() 初始化含 supabase 的 bundle',
      )
    }
  }
  return _services
}

export async function getServicesAsync(): Promise<ServiceRegistry> {
  if (_services) return _services
  const modes = resolveBundleModes()
  validateBundleModes(modes)

  if (isAllMock(modes)) {
    _services = createMockServices()
    return _services
  }

  if (!_servicesPromise) {
    _servicesPromise = createBundledServices(modes).then((s) => {
      _services = s
      return s
    })
  }
  return _servicesPromise
}

export function resetRegistry(): void {
  _services = null
  _servicesPromise = null
}

export const services = getServices()
