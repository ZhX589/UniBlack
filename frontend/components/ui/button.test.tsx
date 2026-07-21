import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Button } from '@/components/ui/button'

describe('Button', () => {
  it('keeps the primary button keyboard-focusable and at least 44px tall', () => {
    render(<Button>Save changes</Button>)
    const button = screen.getByRole('button', { name: 'Save changes' })
    expect(button).toHaveClass('min-h-touch')
    expect(button).not.toBeDisabled()
  })

  it('supports danger and disabled variants', () => {
    render(
      <Button variant="danger" disabled>
        Delete
      </Button>,
    )
    expect(screen.getByRole('button', { name: 'Delete' })).toBeDisabled()
  })
})
