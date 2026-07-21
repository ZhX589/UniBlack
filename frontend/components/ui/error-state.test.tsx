import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ErrorState } from '@/components/ui/error-state'

describe('ErrorState', () => {
  it('announces an error state', () => {
    render(<ErrorState message="Unable to load subjects" />)
    expect(screen.getByRole('alert')).toHaveTextContent('Unable to load subjects')
  })
})
