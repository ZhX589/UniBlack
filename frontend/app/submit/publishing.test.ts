import { describe, expect, it } from 'vitest'
import {
  buildPublishPayload,
  getPublishSectionCompletion,
  listPublishFiles,
  mapPublishErrorToSection,
  validatePublishDraft,
  type PublishDraft,
  type PublishRequirements,
} from './publishing'

// Both verification requirements enabled (matches an authenticated submitter
// who has not yet completed the email + captcha steps).
const reqBoth: PublishRequirements = { emailVerification: true, captcha: true }
// No verification required (e.g. captcha disabled in settings + email already
// verified-at-session-creation). Sections 1-3 still apply.
const reqNone: PublishRequirements = { emailVerification: false, captcha: false }

const okDraft: PublishDraft = {
  displayName: '',
  accounts: [{ platform: 'qq', username: 'alice', account_id: '' }],
  events: [
    { title: 'dishonest trade', details: 'never delivered', severity: 2, text_evidence: '', link_url: '', link_title: '', files: [] },
  ],
  verificationCode: '123456',
  captchaToken: 'demo-token',
}

function makeFile(name: string, size = 8): File {
  // Enough bytes to satisfy "content required" checks; fields length is not
  // validated client-side for emptiness, only size + extension.
  const content = new Uint8Array(size)
  return new File([content], name)
}

describe('validatePublishDraft / empty draft', () => {
  it('flags all three sections incomplete when both requirements enabled', () => {
    const draft: PublishDraft = {
      displayName: '',
      accounts: [],
      events: [],
      verificationCode: '',
      captchaToken: '',
    }
    const issues = validatePublishDraft(draft, reqBoth)
    // At least one issue targets each section.
    expect(issues.some((i) => i.section === 'accounts')).toBe(true)
    expect(issues.some((i) => i.section === 'facts')).toBe(true)
    expect(issues.some((i) => i.section === 'evidence-confirmation')).toBe(true)
    const completion = getPublishSectionCompletion(draft, reqBoth)
    expect(completion).toEqual({
      accounts: false,
      facts: false,
      'evidence-confirmation': false,
    })
  })
})

describe('validatePublishDraft / accounts section', () => {
  it('completes accounts when platform + username provided (display name falls back to username)', () => {
    const draft: PublishDraft = {
      ...okDraft,
      displayName: '',
      accounts: [{ platform: 'QQ', username: 'alice', account_id: '' }],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'accounts')
    expect(issues).toEqual([])
    expect(getPublishSectionCompletion(draft, reqBoth).accounts).toBe(true)
  })

  it('requires an explicit display name when only account_id (no username) is present', () => {
    const draft: PublishDraft = {
      ...okDraft,
      displayName: '',
      accounts: [{ platform: 'qq', username: '', account_id: '12345' }],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'accounts')
    expect(issues.some((i) => i.field === 'display_name')).toBe(true)
    expect(getPublishSectionCompletion(draft, reqBoth).accounts).toBe(false)
  })

  it('accepts an account_id-only row when display name is supplied', () => {
    const draft: PublishDraft = {
      ...okDraft,
      displayName: 'Scammer X',
      accounts: [{ platform: 'qq', username: '', account_id: '12345' }],
    }
    expect(validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'accounts')).toEqual([])
  })

  it('flags every account row that is missing both username and account_id', () => {
    const draft: PublishDraft = {
      ...okDraft,
      displayName: 'Named',
      accounts: [
        { platform: 'qq', username: 'alice', account_id: '' },
        { platform: 'custom', username: '', account_id: '' },
      ],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'accounts')
    // One issue for the empty row (platform present, both identifiers blank).
    expect(issues.length).toBe(1)
    expect(issues.some((i) => i.field === 'accounts.1')).toBe(true)
  })

  it('rejects normalized duplicate platform/value pairs (account_id dedup)', () => {
    const draft: PublishDraft = {
      ...okDraft,
      accounts: [
        { platform: 'QQ', username: 'Alice', account_id: '' },
        { platform: 'qq', username: 'alice', account_id: '' },
      ],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'accounts')
    // UI messages are in Chinese (Task 1 convention); 'duplicate' keyword is
    // for mapPublishErrorToSection which classifies raw server errors.
    expect(issues.some((i) => i.message.includes('重复'))).toBe(true)
  })

  it('detects duplicates by account_id when both rows specify one', () => {
    const draft: PublishDraft = {
      ...okDraft,
      displayName: 'Named',
      accounts: [
        { platform: 'qq', username: 'alice', account_id: '42' },
        { platform: 'qq', username: 'bob', account_id: '42' },
      ],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'accounts')
    expect(issues.some((i) => i.message.includes('重复'))).toBe(true)
  })

  it('flags an account row that is missing platform', () => {
    const draft: PublishDraft = {
      ...okDraft,
      displayName: 'Named',
      accounts: [{ platform: '  ', username: 'alice', account_id: '' }],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'accounts')
    expect(issues.length).toBe(1)
    expect(issues.some((i) => i.field === 'accounts.0.platform')).toBe(true)
  })
})

describe('validatePublishDraft / facts section', () => {
  it('flags every event missing trimmed title or trimmed details', () => {
    const draft: PublishDraft = {
      ...okDraft,
      events: [
        { title: '   ', details: 'ok details', severity: 1, text_evidence: '', link_url: '', link_title: '', files: [] },
        { title: 'ok title', details: '   ', severity: 1, text_evidence: '', link_url: '', link_title: '', files: [] },
      ],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'facts')
    expect(issues.some((i) => i.field === 'events.0.title')).toBe(true)
    expect(issues.some((i) => i.field === 'events.1.details')).toBe(true)
  })

  it('flags severity outside 1-5', () => {
    const low: PublishDraft = {
      ...okDraft,
      events: [{ title: 't', details: 'd', severity: 0, text_evidence: '', link_url: '', link_title: '', files: [] }],
    }
    expect(
      validatePublishDraft(low, reqBoth).some((i) => i.section === 'facts' && i.field === 'events.0.severity'),
    ).toBe(true)
    const high: PublishDraft = {
      ...okDraft,
      events: [{ title: 't', details: 'd', severity: 6, text_evidence: '', link_url: '', link_title: '', files: [] }],
    }
    expect(
      validatePublishDraft(high, reqBoth).some((i) => i.section === 'facts' && i.field === 'events.0.severity'),
    ).toBe(true)
  })

  it('accepts severity boundary values 1 and 5', () => {
    const draft: PublishDraft = {
      ...okDraft,
      events: [
        { title: 'a', details: 'b', severity: 1, text_evidence: '', link_url: '', link_title: '', files: [] },
        { title: 'c', details: 'e', severity: 5, text_evidence: '', link_url: '', link_title: '', files: [] },
      ],
    }
    expect(validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'facts')).toEqual([])
  })
})

describe('validatePublishDraft / evidence is recommended, not required', () => {
  it('completes evidence-confirmation with no evidence when verification requirements satisfied', () => {
    const draft: PublishDraft = {
      ...okDraft,
      events: [
        { title: 't', details: 'd', severity: 1, text_evidence: '', link_url: '', link_title: '', files: [] },
      ],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'evidence-confirmation')
    expect(issues).toEqual([])
    expect(getPublishSectionCompletion(draft, reqBoth)['evidence-confirmation']).toBe(true)
  })
})

describe('validatePublishDraft / link evidence', () => {
  it('flags partially filled link rows (both URL and title required when either is present)', () => {
    const draft: PublishDraft = {
      ...okDraft,
      events: [
        {
          title: 't',
          details: 'd',
          severity: 1,
          text_evidence: '',
          link_url: 'https://example.test/a',
          link_title: '   ',
          files: [],
        },
      ],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'evidence-confirmation')
    expect(issues.some((i) => i.field === 'events.0.link_title')).toBe(true)
  })

  it('flags non-http(s) absolute URLs', () => {
    const draft: PublishDraft = {
      ...okDraft,
      events: [
        {
          title: 't',
          details: 'd',
          severity: 1,
          text_evidence: '',
          link_url: 'ftp://example.test/a',
          link_title: 'report',
          files: [],
        },
      ],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'evidence-confirmation')
    expect(issues.some((i) => i.field === 'events.0.link_url')).toBe(true)
  })

  it('accepts an absolute https URL with a host and matching title', () => {
    const draft: PublishDraft = {
      ...okDraft,
      events: [
        {
          title: 't',
          details: 'd',
          severity: 1,
          text_evidence: '',
          link_url: 'https://example.test/report',
          link_title: 'report',
          files: [],
        },
      ],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'evidence-confirmation')
    expect(issues).toEqual([])
  })
})

describe('validatePublishDraft / text evidence byte length', () => {
  it('accepts exactly 200*1024 UTF-8 bytes', () => {
    // Compose a string that is exactly 200*1024 bytes after UTF-8 encoding.
    // ASCII fills exactly one byte per char.
    const limit = 200 * 1024
    const text = 'x'.repeat(limit)
    const draft: PublishDraft = {
      ...okDraft,
      events: [
        { title: 't', details: 'd', severity: 1, text_evidence: text, link_url: '', link_title: '', files: [] },
      ],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'evidence-confirmation')
    expect(issues).toEqual([])
  })

  it('rejects 200*1024 + 1 UTF-8 bytes', () => {
    const limit = 200 * 1024
    const text = 'x'.repeat(limit + 1)
    const draft: PublishDraft = {
      ...okDraft,
      events: [
        { title: 't', details: 'd', severity: 1, text_evidence: text, link_url: '', link_title: '', files: [] },
      ],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'evidence-confirmation')
    expect(issues.some((i) => i.field === 'events.0.text_evidence')).toBe(true)
  })

  it('measures UTF-8 byte length, not string length (multi-byte Chinese fills the budget faster)', () => {
    // Each Chinese char is 3 bytes when UTF-8 encoded.
    // Build text using Chinese chars so that JS string length is < 200*1024
    // but byte length exceeds the limit by a clear margin.
    const chars = Math.floor((200 * 1024) / 3) + 1 // ~68267 chars, bytes ~204801
    const text = '中'.repeat(chars)
    expect(new TextEncoder().encode(text).byteLength).toBeGreaterThan(200 * 1024)
    const draft: PublishDraft = {
      ...okDraft,
      events: [
        { title: 't', details: 'd', severity: 1, text_evidence: text, link_url: '', link_title: '', files: [] },
      ],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'evidence-confirmation')
    expect(issues.some((i) => i.field === 'events.0.text_evidence')).toBe(true)
  })

  it('accepts text evidence containing multi-byte Chinese well under the byte limit', () => {
    const text = '中'.repeat(1000) // 3000 bytes — well within budget
    const draft: PublishDraft = {
      ...okDraft,
      events: [
        { title: 't', details: 'd', severity: 1, text_evidence: text, link_url: '', link_title: '', files: [] },
      ],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'evidence-confirmation')
    expect(issues).toEqual([])
  })
})

describe('validatePublishDraft / file evidence', () => {
  it('rejects files larger than 20 MiB', () => {
    const draft: PublishDraft = {
      ...okDraft,
      events: [
        {
          title: 't',
          details: 'd',
          severity: 1,
          text_evidence: '',
          link_url: '',
          link_title: '',
          files: [makeFile('big.png', (20 << 20) + 1)],
        },
      ],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'evidence-confirmation')
    expect(issues.some((i) => i.field === 'events.0.files' && /2[01] ?MiB|20 MiB|too large/i.test(i.message) || i.message.includes('MiB'))).toBe(true)
  })

  it('accepts a file exactly at 20 MiB with an allowed extension', () => {
    const draft: PublishDraft = {
      ...okDraft,
      events: [
        {
          title: 't',
          details: 'd',
          severity: 1,
          text_evidence: '',
          link_url: '',
          link_title: '',
          files: [makeFile('boundary.png', 20 << 20)],
        },
      ],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'evidence-confirmation')
    expect(issues).toEqual([])
  })

  it('rejects unsupported extensions', () => {
    const draft: PublishDraft = {
      ...okDraft,
      events: [
        {
          title: 't',
          details: 'd',
          severity: 1,
          text_evidence: '',
          link_url: '',
          link_title: '',
          files: [makeFile('payload.exe', 8)],
        },
      ],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'evidence-confirmation')
    expect(issues.some((i) => i.field === 'events.0.files')).toBe(true)
  })

  it('accepts every documented allowed extension', () => {
    const exts = ['.png', '.jpg', '.jpeg', '.gif', '.webp', '.pdf', '.txt', '.bin', '.zip', '.webm', '.mp4', '.md']
    for (const ext of exts) {
      const draft: PublishDraft = {
        ...okDraft,
        events: [
          {
            title: 't',
            details: 'd',
            severity: 1,
            text_evidence: '',
            link_url: '',
            link_title: '',
            files: [makeFile(`f${ext}`, 8)],
          },
        ],
      }
      const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'evidence-confirmation')
      expect(issues).toEqual([])
    }
  })

  it('rejects uppercase .EXE through lower-casing the extension before lookup', () => {
    const draft: PublishDraft = {
      ...okDraft,
      events: [
        {
          title: 't',
          details: 'd',
          severity: 1,
          text_evidence: '',
          link_url: '',
          link_title: '',
          files: [makeFile('payload.EXE', 8)],
        },
      ],
    }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'evidence-confirmation')
    expect(issues.some((i) => i.field === 'events.0.files')).toBe(true)
  })
})

describe('validatePublishDraft / verification requirements', () => {
  it('flags missing verification code when email verification required', () => {
    const draft: PublishDraft = { ...okDraft, verificationCode: '' }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'evidence-confirmation')
    expect(issues.some((i) => i.field === 'verification_code')).toBe(true)
    expect(getPublishSectionCompletion(draft, reqBoth)['evidence-confirmation']).toBe(false)
  })

  it('flags missing captcha token when captcha required', () => {
    const draft: PublishDraft = { ...okDraft, captchaToken: '' }
    const issues = validatePublishDraft(draft, reqBoth).filter((i) => i.section === 'evidence-confirmation')
    expect(issues.some((i) => i.field === 'captcha_token')).toBe(true)
  })

  it('does not require verification fields when requirements are disabled', () => {
    const draft: PublishDraft = { ...okDraft, verificationCode: '', captchaToken: '' }
    const issues = validatePublishDraft(draft, reqNone).filter((i) => i.section === 'evidence-confirmation')
    expect(issues).toEqual([])
  })
})

describe('mapPublishErrorToSection', () => {
  it('maps account keywords to accounts', () => {
    expect(mapPublishErrorToSection(new Error('invalid account'))).toBe('accounts')
    expect(mapPublishErrorToSection(new Error('duplicate account'))).toBe('accounts')
    expect(mapPublishErrorToSection(new Error('display_name required'))).toBe('accounts')
  })

  it('maps event/title/details/severity to facts', () => {
    expect(mapPublishErrorToSection(new Error('event title and details required'))).toBe('facts')
    expect(mapPublishErrorToSection(new Error('title required'))).toBe('facts')
    expect(mapPublishErrorToSection(new Error('details required'))).toBe('facts')
    expect(mapPublishErrorToSection(new Error('severity out of range'))).toBe('facts')
  })

  it('maps evidence/file/verification/captcha/submission to evidence-confirmation', () => {
    expect(mapPublishErrorToSection(new Error('evidence event index invalid'))).toBe('evidence-confirmation')
    expect(mapPublishErrorToSection(new Error('file extension not allowed: .exe'))).toBe('evidence-confirmation')
    expect(mapPublishErrorToSection(new Error('verification code expired'))).toBe('evidence-confirmation')
    expect(mapPublishErrorToSection(new Error('captcha verification failed'))).toBe('evidence-confirmation')
    expect(mapPublishErrorToSection(new Error('submission restricted'))).toBe('evidence-confirmation')
  })

  it('defaults unknown and network errors to evidence-confirmation', () => {
    expect(mapPublishErrorToSection(new Error('something unexpected'))).toBe('evidence-confirmation')
    expect(mapPublishErrorToSection(new TypeError('network failed'))).toBe('evidence-confirmation')
    expect(mapPublishErrorToSection(null)).toBe('evidence-confirmation')
    expect(mapPublishErrorToSection(undefined)).toBe('evidence-confirmation')
    expect(mapPublishErrorToSection({ status: 500, message: 'oops' })).toBe('evidence-confirmation')
  })

  it('uses lower-cased message for matching when available, avoids crash on non-Error values', () => {
    // An object with a `.message` should still be classified by message content.
    const obj = { message: 'account not found' } as unknown
    expect(mapPublishErrorToSection(obj)).toBe('accounts')
  })
})

describe('buildPublishPayload', () => {
  it('preserves display_name unchanged (server resolves defaults)', () => {
    const draft: PublishDraft = {
      ...okDraft,
      displayName: '   Trim Me   ',
      accounts: [{ platform: '  QQ  ', username: '  Alice  ', account_id: '  42  ' }],
    }
    const payload = buildPublishPayload(draft)
    // display_name is preserved verbatim; the server trims it.
    expect(payload.display_name).toBe('   Trim Me   ')
  })

  it('preserves event order and severity as-is', () => {
    const draft: PublishDraft = {
      ...okDraft,
      displayName: 'X',
      events: [
        { title: 'a', details: 'b', severity: 1, text_evidence: '', link_url: '', link_title: '', files: [] },
        { title: 'c', details: 'e', severity: 5, text_evidence: '', link_url: '', link_title: '', files: [] },
      ],
    }
    const payload = buildPublishPayload(draft)
    expect(payload.events.map((e) => e.title)).toEqual(['a', 'c'])
    expect(payload.events.map((e) => e.severity)).toEqual([1, 5])
  })

  it('trims and filters empty text evidence; preserves event_index ordering', () => {
    const draft: PublishDraft = {
      ...okDraft,
      displayName: 'X',
      events: [
        { title: 'a', details: 'b', severity: 1, text_evidence: '  hello  ', link_url: '', link_title: '', files: [] },
        { title: 'c', details: 'd', severity: 2, text_evidence: '   ', link_url: '', link_title: '', files: [] },
      ],
    }
    const payload = buildPublishPayload(draft)
    expect(payload.text_evidence).toEqual([{ event_index: 0, title: '提交文本证据', text: 'hello' }])
  })

  it('trims and filters empty link evidence, includes only rows with non-empty url', () => {
    const draft: PublishDraft = {
      ...okDraft,
      displayName: 'X',
      events: [
        {
          title: 'a',
          details: 'b',
          severity: 1,
          text_evidence: '',
          link_url: '  https://example.test/a  ',
          link_title: '  report  ',
          files: [],
        },
        { title: 'c', details: 'd', severity: 2, text_evidence: '', link_url: '  ', link_title: 'x', files: [] },
      ],
    }
    const payload = buildPublishPayload(draft)
    expect(payload.link_evidence).toEqual([
      { event_index: 0, title: 'report', description: '', url: 'https://example.test/a' },
    ])
  })

  it('uses "链接证据" as the link title when link_title is empty but url is present', () => {
    const draft: PublishDraft = {
      ...okDraft,
      displayName: 'X',
      events: [
        {
          title: 'a',
          details: 'b',
          severity: 1,
          text_evidence: '',
          link_url: 'https://example.test/a',
          link_title: '   ',
          files: [],
        },
      ],
    }
    const payload = buildPublishPayload(draft)
    expect(payload.link_evidence).toEqual([
      { event_index: 0, title: '链接证据', description: '', url: 'https://example.test/a' },
    ])
  })

  it('emits file evidence metadata for each file with field file_{eventIndex}_{fileIndex}', () => {
    const draft: PublishDraft = {
      ...okDraft,
      displayName: 'X',
      events: [
        {
          title: 'a',
          details: 'b',
          severity: 1,
          text_evidence: '',
          link_url: '',
          link_title: '',
          files: [makeFile('one.png'), makeFile('two.txt')],
        },
        {
          title: 'c',
          details: 'd',
          severity: 2,
          text_evidence: '',
          link_url: '',
          link_title: '',
          files: [makeFile('three.pdf')],
        },
      ],
    }
    const payload = buildPublishPayload(draft)
    expect(payload.file_evidence).toEqual([
      { event_index: 0, title: 'one.png', filename: 'one.png', field: 'file_0_0' },
      { event_index: 0, title: 'two.txt', filename: 'two.txt', field: 'file_0_1' },
      { event_index: 1, title: 'three.pdf', filename: 'three.pdf', field: 'file_1_0' },
    ])
  })

  it('excludes File objects from the JSON payload (they go via FormData)', () => {
    const draft: PublishDraft = {
      ...okDraft,
      displayName: 'X',
      events: [
        {
          title: 'a',
          details: 'b',
          severity: 1,
          text_evidence: '',
          link_url: '',
          link_title: '',
          files: [makeFile('one.png')],
        },
      ],
    }
    const payload = buildPublishPayload(draft)
    expect(JSON.stringify(payload)).not.toContain('[object File]')
    // JSON stringification converts File objects to JSON implicitly; ensure they
    // do not appear anywhere in the serialized payload.
    expect(JSON.parse(JSON.stringify(payload)).file_evidence[0]).not.toHaveProperty('content')
  })

  it('propagates verification_code and captcha_token to the payload', () => {
    const draft: PublishDraft = { ...okDraft, verificationCode: '12ab', captchaToken: 'tok-9' }
    const payload = buildPublishPayload(draft)
    expect(payload.verification_code).toBe('12ab')
    expect(payload.captcha_token).toBe('tok-9')
  })
})

describe('listPublishFiles', () => {
  it('emits { field: file_{eventIndex}_{fileIndex}, file } for all files in order', () => {
    const draft: PublishDraft = {
      ...okDraft,
      displayName: 'X',
      events: [
        {
          title: 'a',
          details: 'b',
          severity: 1,
          text_evidence: '',
          link_url: '',
          link_title: '',
          files: [makeFile('a.png'), makeFile('b.png')],
        },
        {
          title: 'c',
          details: 'd',
          severity: 2,
          text_evidence: '',
          link_url: '',
          link_title: '',
          files: [makeFile('c.png')],
        },
      ],
    }
    expect(listPublishFiles(draft)).toEqual([
      { field: 'file_0_0', file: expect.any(File) },
      { field: 'file_0_1', file: expect.any(File) },
      { field: 'file_1_0', file: expect.any(File) },
    ])
  })

  it('returns an empty list when no files are present', () => {
    const draft: PublishDraft = { ...okDraft }
    expect(listPublishFiles(draft)).toEqual([])
  })
})
