import type { ServiceRegistry } from '@/services/registry'

export function useServices(): ServiceRegistry {
  return useNuxtApp().$services as ServiceRegistry
}
