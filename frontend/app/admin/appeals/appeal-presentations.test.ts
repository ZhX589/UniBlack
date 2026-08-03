import { describe, expect, it } from 'vitest'
import { appealTarget } from './presentation'

describe('appeal workbench presentation', () => {
  it('prefers Event targets over legacy Case targets', () => {
    expect(appealTarget({ event_id: 'event-1', case_id: 'case-1' })).toEqual({
      kind: 'event',
      id: 'event-1',
      href: '/events/event-1',
    })
  })

  it('keeps legacy Case targets visible but unlinked', () => {
    expect(appealTarget({ event_id: null, case_id: 'case-1' })).toEqual({
      kind: 'legacy-case',
      id: 'case-1',
      href: null,
    })
  })

  it('returns missing when neither event nor case', () => {
    expect(appealTarget({ event_id: null, case_id: null })).toEqual({
      kind: 'missing',
      id: null,
      href: null,
    })
  })

  it('treats undefined fields the same as null', () => {
    expect(appealTarget({})).toEqual({ kind: 'missing', id: null, href: null })
  })
})
