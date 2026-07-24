'use client'

import { useSite } from '@/app/providers'
import { useScrollDirection } from '@/lib/use-scroll-direction'

export function SiteFooter() {
  const { name, description } = useSite()
  const dir = useScrollDirection()

  return (
    <footer
      className={`glass shell-slide border-t border-black/5 bg-white/72 text-foreground ${
        dir === 'down' ? 'translate-y-full' : 'translate-y-0'
      }`}
    >
      <div className="container mx-auto px-4 py-4 text-center text-sm text-muted">
        <p className="font-medium text-foreground">{name}</p>
        {description ? <p className="mt-1">{description}</p> : <p className="mt-1">可复用的社区云黑名单系统</p>}
      </div>
    </footer>
  )
}
