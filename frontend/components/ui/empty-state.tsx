export function EmptyState({ message }: { message: string }) {
  return <div className="rounded border border-dashed border-border bg-background p-8 text-center text-muted">{message}</div>
}
