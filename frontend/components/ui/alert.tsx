export function Alert({
  children,
  tone = 'danger',
}: {
  children: React.ReactNode
  tone?: 'danger' | 'success' | 'warning'
}) {
  const classes =
    tone === 'success'
      ? 'border-success/20 bg-success/5 text-success glass'
      : tone === 'warning'
        ? 'border-warning/20 bg-warning/5 text-warning glass'
        : 'border-danger/20 bg-danger/5 text-danger glass'
  return (
    <div role="alert" className={`rounded border p-3 text-sm ${classes}`}>
      {children}
    </div>
  )
}
