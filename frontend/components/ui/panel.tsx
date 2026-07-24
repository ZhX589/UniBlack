import type { HTMLAttributes } from 'react'

export function Panel({ className = '', ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={`rounded border border-black/5 bg-white/65 p-6 shadow-sm glass ${className}`} {...props} />
  )
}
