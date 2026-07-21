export function LoadingState({ message = '加载中...' }: { message?: string }) {
  return <p className="py-12 text-center text-muted">{message}</p>
}
