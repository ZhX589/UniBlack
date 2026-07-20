'use client'

export function DemoCaptcha({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  const verified = value !== ''
  return (
    <button
      type="button"
      aria-pressed={verified}
      onClick={() => onChange(verified ? '' : 'demo-ok')}
      className={`w-full rounded-lg border p-4 text-left ${verified ? 'border-green-600 bg-green-50 text-green-800' : 'border-gray-300 bg-gray-50 text-gray-700'}`}
    >
      <strong>{verified ? '验证已完成' : '我不是自动程序'}</strong>
      <span className="ml-2 text-sm">这是 UniBlack 内置演示验证，不会连接第三方服务。</span>
    </button>
  )
}
