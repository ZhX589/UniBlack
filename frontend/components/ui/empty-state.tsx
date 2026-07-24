export function EmptyState({ message }: { message: string }) {
  return (
    <div className="rounded border border-dashed border-foreground/10 bg-foreground/[0.02] p-8 text-center text-text-secondary glass">
      {message}
    </div>
  )
}
