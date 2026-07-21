import type { HTMLAttributes } from 'react'

export function Panel({ className = '', ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={`rounded border border-border bg-surface p-6 shadow-sm ${className}`} {...props} />
}
