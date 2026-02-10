import type {
  IAuthService,
  IWorkshopService,
  IEditorService,
  IAiService,
  IExportService,
} from '@/types/services'
import { createMockAuthService } from './mock/authService'
import { createMockWorkshopService } from './mock/workshopService'
import { createMockEditorService } from './mock/editorService'
import { createMockAiService } from './mock/aiService'
import { createMockExportService } from './mock/exportService'

interface ServiceRegistry {
  auth: IAuthService
  workshop: IWorkshopService
  editor: IEditorService
  ai: IAiService
  export: IExportService
}

let _services: ServiceRegistry | null = null

function createMockServices(): ServiceRegistry {
  return {
    auth: createMockAuthService(),
    workshop: createMockWorkshopService(),
    editor: createMockEditorService(),
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
      throw new Error(`Unsupported API mode: ${mode}`)
    }
  }
  return _services
}

export const services = getServices()
