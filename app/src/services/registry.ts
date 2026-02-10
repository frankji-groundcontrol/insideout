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

interface ServiceRegistry {
  auth: IAuthService
  workshop: IWorkshopService
  editor: IEditorService
  ai: IAiService
  export: IExportService
}

let _services: ServiceRegistry | null = null
let _servicesPromise: Promise<ServiceRegistry> | null = null

function createMockServices(): ServiceRegistry {
  return {
    auth: createMockAuthService(),
    workshop: createMockWorkshopService(),
    editor: createMockEditorService(),
    ai: createMockAiService(),
    export: createMockExportService(),
  }
}

async function createSupabaseServices(): Promise<ServiceRegistry> {
  const [authMod, workshopMod, editorMod] = await Promise.all([
    import('./supabase/authService'),
    import('./supabase/workshopService'),
    import('./supabase/editorService'),
  ])
  return {
    auth: authMod.createSupabaseAuthService(),
    workshop: workshopMod.createSupabaseWorkshopService(),
    editor: editorMod.createSupabaseEditorService(),
    ai: createMockAiService(),
    export: createMockExportService(),
  }
}

export function getServices(): ServiceRegistry {
  if (!_services) {
    const mode = import.meta.env.VITE_API_MODE || 'mock'
    if (mode === 'mock') {
      _services = createMockServices()
    } else {
      throw new Error(
        `同步模式不支持 "${mode}"，请使用 getServicesAsync()`,
      )
    }
  }
  return _services
}

export async function getServicesAsync(): Promise<ServiceRegistry> {
  if (_services) return _services
  const mode = import.meta.env.VITE_API_MODE || 'mock'
  if (mode === 'mock') {
    _services = createMockServices()
    return _services
  }
  if (mode === 'supabase') {
    if (!_servicesPromise) {
      _servicesPromise = createSupabaseServices().then((s) => {
        _services = s
        return s
      })
    }
    return _servicesPromise
  }
  throw new Error(`Unsupported API mode: ${mode}`)
}

export const services = getServices()
