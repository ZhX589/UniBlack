const tones = {
  neutral: 'glass bg-foreground/10 text-text-secondary',
  danger: 'glass bg-danger/10 text-danger',
  success: 'glass bg-success/10 text-success',
  warning: 'glass bg-warning/10 text-warning',
} as const

export function Badge({
  children,
  tone = 'neutral',
}: {
  children: React.ReactNode
  tone?: keyof typeof tones
}) {
  return <span className={`inline-flex rounded-full px-3 py-1 text-sm font-mono ${tones[tone]}`}>{children}</span>
}
