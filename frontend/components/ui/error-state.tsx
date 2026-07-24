export function ErrorState({ message }: { message: string }) {
  return (
    <div role="alert" className="rounded border border-danger/20 bg-danger/5 p-6 text-center text-danger glass">
      {message}
    </div>
  )
}
