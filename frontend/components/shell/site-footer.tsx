'use client'

import { useSite } from '@/app/providers'

export function SiteFooter() {
  const { name, description } = useSite()
  return (
    <footer className="mt-8 border-t border-border bg-surface p-4 text-foreground">
      <div className="container mx-auto text-center text-sm text-muted">
        <p className="font-medium text-foreground">{name}</p>
        {description ? <p className="mt-1">{description}</p> : <p className="mt-1">可复用的社区云黑名单系统</p>}
      </div>
    </footer>
  )
}
