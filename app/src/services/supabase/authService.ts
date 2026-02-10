import type { IAuthService } from '@/types/services'
import type { UserProfile } from '@/types'
import { getSupabase } from '@/lib/supabase'

const ALLOWED_UPDATE_FIELDS: ReadonlyArray<keyof UserProfile> = [
  'username',
  'avatar_url',
  'bio',
  'keywords',
]

function toUserProfile(row: Record<string, unknown>): UserProfile {
  return {
    id: row.id as string,
    email: row.email as string,
    username: row.username as string,
    avatar_url: row.avatar_url as string | undefined,
    bio: row.bio as string | undefined,
    keywords: row.keywords as string[] | undefined,
    role: row.role as 'admin' | 'user',
    created_at: row.created_at as string,
  }
}

async function getAuthUser(): Promise<{ id: string; email: string }> {
  const supabase = getSupabase()
  const { data, error } = await supabase.auth.getUser()
  if (error || !data.user) {
    throw new Error(error?.message ?? '未登录')
  }
  return { id: data.user.id, email: data.user.email ?? '' }
}

export function createSupabaseAuthService(): IAuthService {
  return {
    async login(email: string): Promise<UserProfile> {
      const supabase = getSupabase()
      const { error } = await supabase.auth.signInWithOtp({ email })

      if (error) {
        throw new Error(error.message)
      }

      const { data: userData, error: userError } = await supabase.auth.getUser()
      if (userError || !userData.user) {
        throw new Error(userError?.message ?? 'OTP 登录失败')
      }

      const userId = userData.user.id
      const userEmail = userData.user.email ?? email

      const { data: profile, error: upsertError } = await supabase
        .schema('juanleme')
        .from('profiles')
        .upsert({
          id: userId,
          email: userEmail,
          username: userEmail.split('@')[0] ?? userEmail,
        })
        .eq('id', userId)
        .select()
        .single()

      if (upsertError || !profile) {
        throw new Error(upsertError?.message ?? '创建用户档案失败')
      }

      return toUserProfile(profile as Record<string, unknown>)
    },

    async getCurrentUser(): Promise<UserProfile> {
      const supabase = getSupabase()
      const authUser = await getAuthUser()

      const { data: profile, error } = await supabase
        .schema('juanleme')
        .from('profiles')
        .select()
        .eq('id', authUser.id)
        .maybeSingle()

      if (error) {
        throw new Error(error.message)
      }

      if (profile) {
        return toUserProfile(profile as Record<string, unknown>)
      }

      const { data: newProfile, error: insertError } = await supabase
        .schema('juanleme')
        .from('profiles')
        .insert({
          id: authUser.id,
          email: authUser.email,
          username: authUser.email.split('@')[0] ?? authUser.email,
        })
        .select()
        .single()

      if (insertError || !newProfile) {
        throw new Error(insertError?.message ?? '创建用户档案失败')
      }

      return toUserProfile(newProfile as Record<string, unknown>)
    },

    async updateProfile(data: Partial<UserProfile>): Promise<UserProfile> {
      const supabase = getSupabase()
      const authUser = await getAuthUser()

      const sanitized: Record<string, unknown> = {}
      for (const field of ALLOWED_UPDATE_FIELDS) {
        if (field in data) {
          sanitized[field] = data[field]
        }
      }

      const { data: updated, error } = await supabase
        .schema('juanleme')
        .from('profiles')
        .update(sanitized)
        .eq('id', authUser.id)
        .select()
        .single()

      if (error || !updated) {
        throw new Error(error?.message ?? '更新用户档案失败')
      }

      return toUserProfile(updated as Record<string, unknown>)
    },

    async logout(): Promise<void> {
      const supabase = getSupabase()
      const { error } = await supabase.auth.signOut()
      if (error) {
        throw new Error(error.message)
      }
    },
  }
}
