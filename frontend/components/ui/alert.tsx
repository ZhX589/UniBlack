export function Alert({
  children,
  tone = 'danger',
}: {
  children: React.ReactNode
  tone?: 'danger' | 'success' | 'warning'
}) {
  const classes =
    tone === 'success'
      ? 'border-success/30 bg-green-50 text-success'
      : tone === 'warning'
        ? 'border-warning/30 bg-yellow-50 text-warning'
        : 'border-danger/30 bg-red-50 text-danger'
  return (
    <div role="alert" className={`rounded border p-3 text-sm ${classes}`}>
      {children}
    </div>
  )
}
