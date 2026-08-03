// Page-scoped pure presentation helper for the `/admin/appeals` workbench.
//
// Sibling to `page.tsx` (same pattern as `/app/submit/publishing.ts`) so the
// `appealTarget` mapping is unit-testable without importing the Next page and
// pulling in `'use client'` side effects. Kept deliberately side-effect free.

import type { Appeal } from '@/lib/types'

export type AppealTarget =
  | { kind: 'event'; id: string; href: string }
  | { kind: 'legacy-case'; id: string; href: null }
  | { kind: 'missing'; id: null; href: null }

/**
 * Resolve the display target for an appeal row.
 *
 * Event targets (the canonical publish model, Phase 13) always win over legacy
 * Case targets when both are present — the workbench surfaces the Event link
 * while still keeping the audit trail connected. Legacy Case targets stay
 * visible but are never linkified (the Case routes soft-redirect to Events and
 * the workbench must not imply Case is the primary surface). When neither is
 * present, `missing` is returned so the row can render "未关联事件/事项".
 */
export function appealTarget(appeal: Pick<Appeal, 'event_id' | 'case_id'>): AppealTarget {
  if (appeal.event_id && appeal.event_id.trim() !== '') {
    return { kind: 'event', id: appeal.event_id, href: `/events/${appeal.event_id}` }
  }
  if (appeal.case_id && appeal.case_id.trim() !== '') {
    return { kind: 'legacy-case', id: appeal.case_id, href: null }
  }
  return { kind: 'missing', id: null, href: null }
}
