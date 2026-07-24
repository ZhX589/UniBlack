import type { ButtonHTMLAttributes } from 'react'

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger'

const variants: Record<Variant, string> = {
  primary: 'bg-primary text-primary-foreground hover:opacity-90 font-display tracking-wide',
  secondary: 'glass border border-black/10 bg-white/65 text-body hover:bg-white/80 font-display tracking-wide',
  ghost: 'text-body hover:bg-white/40 glass',
  danger: 'bg-danger text-white hover:opacity-90 font-display tracking-wide',
}

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: Variant
}

export function Button({ variant = 'primary', className = '', type = 'button', ...props }: ButtonProps) {
  return (
    <button
      type={type}
      className={`inline-flex min-h-touch items-center justify-center rounded px-4 py-2 text-sm font-medium duration-150 transition disabled:cursor-not-allowed disabled:opacity-50 ${variants[variant]} ${className}`}
      {...props}
    />
  )
}
