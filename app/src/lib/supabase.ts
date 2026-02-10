import { createClient, type SupabaseClient } from '@supabase/supabase-js'

let _client: SupabaseClient | null = null

export function getSupabase(): SupabaseClient {
  if (!_client) {
    const url = import.meta.env.VITE_SUPABASE_URL as string | undefined
    const key = import.meta.env.VITE_SUPABASE_ANON_KEY as string | undefined
    if (!url || !key) {
      throw new Error(
        'VITE_SUPABASE_URL and VITE_SUPABASE_ANON_KEY are required when using Supabase mode',
      )
    }
    _client = createClient(url, key)
  }
  return _client
}
