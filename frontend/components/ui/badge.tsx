const tones = {
  neutral: 'bg-background text-muted',
  danger: 'bg-red-100 text-danger',
  success: 'bg-green-100 text-success',
  warning: 'bg-yellow-100 text-warning',
} as const

export function Badge({
  children,
  tone = 'neutral',
}: {
  children: React.ReactNode
  tone?: keyof typeof tones
}) {
  return <span className={`inline-flex rounded-full px-3 py-1 text-sm ${tones[tone]}`}>{children}</span>
}
