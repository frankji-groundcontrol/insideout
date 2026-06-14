import { buildServices, resolveBundleModes, _setProvidedServices, type ServiceDeps } from '@/services/registry'
import { createSupabaseClient } from '@/lib/supabase'

export default defineNuxtPlugin(async () => {
  const cfg = useRuntimeConfig().public
  const modes = resolveBundleModes({
    apiMode: cfg.apiMode as string,
    bundleAuth: cfg.bundleAuth as string,
    bundleData: cfg.bundleData as string,
    bundleAiExport: cfg.bundleAiExport as string,
  })
  const needsSupabase = modes.auth === 'supabase' || modes.data === 'supabase' || modes.aiExport === 'supabase'
  const deps: ServiceDeps = needsSupabase
    ? { supabase: createSupabaseClient(cfg.supabaseUrl as string, cfg.supabaseAnonKey as string) }
    : {}
  const services = await buildServices(modes, deps) // 按请求执行：插件在服务端每个请求都会运行
  _setProvidedServices(services) // 支撑同步访问器 requireServices()
  return { provide: { services } } // $services 供 useServices() 使用
})
