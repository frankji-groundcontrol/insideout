import { useI18n } from 'vue-i18n'

/** Days elapsed since an ISO timestamp (0 for today/invalid). Shared bucketing. */
function daysSince(iso: string | undefined | null): number {
  if (!iso) return NaN
  const diffMs = Date.now() - new Date(iso).getTime()
  return Math.floor(diffMs / 86_400_000)
}

/**
 * Minimal English relative-time formatter — no dependency for a one-line need.
 * Kept for non-setup call sites; components should prefer the localized
 * `useTimeAgo()` so the string follows the active locale (collab B2).
 */
export function timeAgo(iso: string | undefined | null): string {
  const days = daysSince(iso)
  if (Number.isNaN(days)) return ''
  if (days <= 0) return 'today'
  if (days === 1) return 'yesterday'
  if (days < 30) return `${days}d ago`
  const months = Math.floor(days / 30)
  if (months < 12) return `${months}mo ago`
  return `${Math.floor(months / 12)}y ago`
}

/**
 * i18n-aware relative time. Returns a formatter that renders today / yesterday /
 * Nd / Nmo / Ny in the active locale via the `time.*` keys.
 */
export function useTimeAgo() {
  const { t } = useI18n()
  return (iso: string | undefined | null): string => {
    const days = daysSince(iso)
    if (Number.isNaN(days)) return ''
    if (days <= 0) return t('time.today')
    if (days === 1) return t('time.yesterday')
    if (days < 30) return t('time.days', { n: days })
    const months = Math.floor(days / 30)
    if (months < 12) return t('time.months', { n: months })
    return t('time.years', { n: Math.floor(months / 12) })
  }
}
