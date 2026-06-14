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

// 由调用方（plugins/services.ts）从 useRuntimeConfig().public 读取后传入，
// 不再在本模块内读取 import.meta.env，保证 registry 在 SSR / 测试中可纯导入。
export interface BundleConfig {
  apiMode?: string
  bundleAuth?: string
  bundleData?: string
  bundleAiExport?: string
}

export interface ServiceDeps {
  // 由插件注入的 Supabase 客户端（仅在选择 supabase bundle 时才会构建）。
  supabase?: import('@supabase/supabase-js').SupabaseClient
}

export function resolveBundleModes(cfg: BundleConfig): BundleModes {
  const globalMode = (cfg.apiMode || 'mock') as AdapterMode
  return {
    auth: (cfg.bundleAuth as AdapterMode) || globalMode,
    data: (cfg.bundleData as AdapterMode) || globalMode,
    aiExport: (cfg.bundleAiExport as AdapterMode) || globalMode,
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

async function createBundledServices(
  modes: BundleModes,
  deps: ServiceDeps,
): Promise<ServiceRegistry> {
  const registry: Partial<ServiceRegistry> = {}

  if (modes.auth === 'supabase') {
    const mod = await import('./supabase/authService')
    registry.auth = mod.createSupabaseAuthService(deps.supabase!)
  } else {
    registry.auth = createMockAuthService()
  }

  if (modes.data === 'supabase') {
    const [workshopMod, editorMod] = await Promise.all([
      import('./supabase/workshopService'),
      import('./supabase/editorService'),
    ])
    registry.workshop = workshopMod.createSupabaseWorkshopService(deps.supabase!)
    registry.editor = editorMod.createSupabaseEditorService(deps.supabase!)
  } else {
    registry.workshop = createMockWorkshopService()
    registry.editor = createMockEditorService()
  }

  if (modes.aiExport === 'supabase') {
    const [aiMod, exportMod] = await Promise.all([
      import('./supabase/aiService'),
      import('./supabase/exportService'),
    ])
    registry.ai = aiMod.createSupabaseAiService(deps.supabase!)
    registry.export = exportMod.createSupabaseExportService(deps.supabase!)
  } else {
    registry.ai = createMockAiService()
    registry.export = createMockExportService()
  }

  return registry as ServiceRegistry
}

// 插件使用的纯异步构建器。此处不做单例缓存（由 Nuxt 按 app/请求提供）。
export async function buildServices(
  modes: BundleModes,
  deps: ServiceDeps,
): Promise<ServiceRegistry> {
  validateBundleModes(modes)
  if (isAllMock(modes)) return createMockServices()
  return createBundledServices(modes, deps)
}

// 兼容旧名：getServicesAsync 现在指向新的 buildServices(modes, deps)。
export const getServicesAsync = buildServices

// 供无法 await 的 composable/component 使用的同步访问器。
// 背后由 Nuxt 插件通过 _setProvidedServices 写入（见 useServices()）。未就绪时抛错。
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
