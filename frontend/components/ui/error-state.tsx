export function ErrorState({ message }: { message: string }) {
  return (
    <div role="alert" className="rounded border border-danger/30 bg-red-50 p-6 text-center text-danger">
      {message}
    </div>
  )
}
