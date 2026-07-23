import { buildServices, _setProvidedServices } from '@/services/registry'

// Built once per request (server) / once per app load (client). No env
// switching anymore — the Go API is the only backend.
export default defineNuxtPlugin(() => {
  const services = buildServices()
  _setProvidedServices(services) // backs the sync accessor requireServices()
  return { provide: { services } } // $services for useServices()
})
