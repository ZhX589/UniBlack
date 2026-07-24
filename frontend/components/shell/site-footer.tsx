'use client'

import { useSite } from '@/app/providers'

export function SiteFooter() {
  const { name, description } = useSite()
  return (
    <footer className="mt-8 border-t border-black/5 bg-white/65 py-4 text-text-secondary glass">
      <div className="container mx-auto px-4 text-center text-sm">
        <p className="font-medium text-text-body">{name}</p>
        {description ? <p className="mt-1">{description}</p> : <p className="mt-1">可复用的社区云黑名单系统</p>}
      </div>
    </footer>
  )
}
