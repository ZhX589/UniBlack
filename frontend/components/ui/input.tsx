import type { InputHTMLAttributes, SelectHTMLAttributes, TextareaHTMLAttributes } from 'react'

const inputBase =
  'w-full rounded min-h-touch border border-black/10 bg-white/60 px-3 py-2 text-sm text-text-body glass placeholder:text-text-secondary transition disabled:opacity-50 focus-visible:outline-2 focus-visible:outline-focus'

export function Input(props: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={inputBase} {...props} />
}

export function Select(props: SelectHTMLAttributes<HTMLSelectElement>) {
  return <select className={inputBase} {...props} />
}

export function Textarea(props: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={inputBase} {...props} />
}
