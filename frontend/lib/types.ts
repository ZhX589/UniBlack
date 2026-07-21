export type AuthUser = {
  user_id: string
  username: string
  email: string
  roles: string[]
}

export type PublicSettings = Record<string, unknown>

export type Account = {
  id?: string
  platform: string
  username?: string | null
  account_id?: string | null
  account_type?: string
  is_primary?: boolean
  custom_attributes?: Record<string, unknown>
}

export type Evidence = {
  id?: string
  type: string
  title?: string | null
  description?: string | null
  url?: string | null
  sha256?: string | null
}

export type EventItem = {
  id: string
  title: string
  details?: string
  status: string
  severity?: number
  evidence?: Evidence[]
  created_at?: string
}

export type Subject = {
  id: string
  public_id?: string
  display_name: string
  status: string
  risk_level?: number
  accounts?: Account[]
  identifiers?: Array<{ platform: string; value: string; account_type?: string }>
  events?: EventItem[]
}

export type Statistics = {
  subjects: number
  events: number
  case_count?: number
}

export type Sanction = {
  id: string
  user_id: string
  type: string
  reason: string
  starts_at: string
  ends_at?: string | null
  revoked_at?: string | null
}

export type Appeal = {
  id: string
  event_id?: string | null
  case_id?: string | null
  reason: string
  status: string
  outcome?: string | null
  resolution_reason?: string | null
  submitted_by?: string | null
  created_at?: string
}

export type AdminUser = {
  id: string
  username: string
  email: string
  is_active: boolean
  roles?: string[]
}

export type AccessListEntry = {
  id: string
  type: string
  target: string
  value: string
  reason?: string
}

export type Pagination = {
  page: number
  page_size: number
  total: number
}
