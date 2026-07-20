/** Helpers for public/admin system settings payloads. */

export function parseSettingValue(raw: unknown): unknown {
  if (raw === null || raw === undefined) return raw
  if (typeof raw !== 'string') return raw
  try {
    return JSON.parse(raw)
  } catch {
    return raw
  }
}

export function settingBool(map: Record<string, unknown>, key: string, fallback = false): boolean {
  const v = map[key]
  if (typeof v === 'boolean') return v
  if (typeof v === 'string') return v === 'true' || v === '"true"'
  return fallback
}

export function settingString(map: Record<string, unknown>, key: string, fallback = ''): string {
  const v = map[key]
  if (typeof v === 'string') return v
  if (v == null) return fallback
  return String(v)
}

/** Flatten admin settings list: { settings: [{key,value}] } or array or map. */
export function flattenAdminSettings(data: unknown): Array<{ key: string; value: string; description?: string }> {
  if (!data) return []
  if (Array.isArray(data)) {
    return data.map((row: any) => ({
      key: row.key,
      value: typeof row.value === 'string' ? row.value : JSON.stringify(row.value),
      description: row.description,
    }))
  }
  if (typeof data === 'object' && data !== null && Array.isArray((data as any).settings)) {
    return flattenAdminSettings((data as any).settings)
  }
  // grouped map { site: [...], security: [...] }
  if (typeof data === 'object' && data !== null) {
    const out: Array<{ key: string; value: string; description?: string }> = []
    for (const group of Object.values(data as Record<string, unknown>)) {
      if (Array.isArray(group)) {
        out.push(...flattenAdminSettings(group))
      }
    }
    return out
  }
  return []
}

export function adminSettingsToMap(rows: Array<{ key: string; value: string }>): Record<string, unknown> {
  const map: Record<string, unknown> = {}
  for (const row of rows) {
    map[row.key] = parseSettingValue(row.value)
  }
  return map
}
