import { afterEach, describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ApiError } from '@/lib/api-error'

// Mock @/lib/api and capture the call shape so we don't depend on a live server.
const apiRequestMock = vi.fn()
vi.mock('@/lib/api', () => ({
  apiRequest: (...args: unknown[]) => apiRequestMock(...args),
}))

import { DemoCaptcha } from './demo-captcha'

// vitest's config does not enable `globals: true`, so the RTL auto-cleanup hook
// (which relies on a global `afterEach`) is not registered. We mount fresh per
// test, so we must explicitly tear down the container between cases to avoid
// `getByRole` ambiguity from previously rendered buttons.
afterEach(() => {
  cleanup()
})

describe('DemoCaptcha', () => {
  beforeEach(() => {
    apiRequestMock.mockReset()
  })

  it('starts unverified (aria-pressed=false)', () => {
    render(<DemoCaptcha value="" onChange={() => {}} />)
    expect(screen.getByRole('button')).toHaveAttribute('aria-pressed', 'false')
  })

  it('shows verified state (aria-pressed=true) when a token is present', () => {
    render(<DemoCaptcha value="some-token" onChange={() => {}} />)
    expect(screen.getByRole('button')).toHaveAttribute('aria-pressed', 'true')
  })

  it('posts to /api/verification/demo/submission with auth=true and empty json body for the submission purpose', async () => {
    const user = userEvent.setup()
    apiRequestMock.mockResolvedValueOnce({ token: 'demo-token' })

    render(<DemoCaptcha value="" onChange={() => {}} purpose="submission" />)
    await user.click(screen.getByRole('button'))

    expect(apiRequestMock).toHaveBeenCalledTimes(1)
    expect(apiRequestMock).toHaveBeenCalledWith('/api/verification/demo/submission', {
      method: 'POST',
      auth: true,
      json: {},
    })
  })

  it('posts to /api/verification/demo/register with auth=false for the register purpose', async () => {
    const user = userEvent.setup()
    apiRequestMock.mockResolvedValueOnce({ token: 'reg-tok' })

    render(<DemoCaptcha value="" onChange={() => {}} purpose="register" />)
    await user.click(screen.getByRole('button'))

    expect(apiRequestMock).toHaveBeenCalledWith('/api/verification/demo/register', {
      method: 'POST',
      auth: false,
      json: {},
    })
  })

  it('forwards the server token to onChange on success', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    apiRequestMock.mockResolvedValueOnce({ token: 'demo-token' })

    render(<DemoCaptcha value="" onChange={onChange} purpose="submission" />)
    await user.click(screen.getByRole('button'))

    await vi.waitFor(() => expect(onChange).toHaveBeenCalledWith('demo-token'))
  })

  it('clears the token when re-clicking an already-verified button without any API call', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()

    render(<DemoCaptcha value="existing-token" onChange={onChange} purpose="submission" />)
    await user.click(screen.getByRole('button'))

    expect(onChange).toHaveBeenCalledWith('')
    expect(apiRequestMock).not.toHaveBeenCalled()
  })

  it('disables the button and shows 验证中... while a request is in flight', async () => {
    const user = userEvent.setup()
    // Never resolve the request so the loading state persists.
    apiRequestMock.mockReturnValueOnce(new Promise(() => {}))

    render(<DemoCaptcha value="" onChange={() => {}} purpose="submission" />)
    await user.click(screen.getByRole('button'))

    const button = screen.getByRole('button')
    expect(button).toBeDisabled()
    expect(button).toHaveTextContent('验证中...')
  })

  it('calls onError with ApiError.message and does NOT call onChange on a known API failure', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    const onError = vi.fn()
    const apiErr = new ApiError(410, 'captcha expired', { message: 'captcha expired' })
    apiRequestMock.mockRejectedValueOnce(apiErr)

    render(
      <DemoCaptcha
        value=""
        onChange={onChange}
        onError={onError}
        purpose="submission"
      />,
    )
    await user.click(screen.getByRole('button'))

    await vi.waitFor(() => expect(onError).toHaveBeenCalled())
    expect(onError).toHaveBeenCalledWith('captcha expired')
    expect(onChange).not.toHaveBeenCalled()
  })

  it('calls onError with a generic 人机验证失败，请重试 message on unknown errors and does NOT call onChange', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    const onError = vi.fn()
    apiRequestMock.mockRejectedValueOnce(new TypeError('network down'))

    render(
      <DemoCaptcha
        value=""
        onChange={onChange}
        onError={onError}
        purpose="submission"
      />,
    )
    await user.click(screen.getByRole('button'))

    await vi.waitFor(() => expect(onError).toHaveBeenCalled())
    expect(onError).toHaveBeenCalledWith('人机验证失败，请重试')
    expect(onChange).not.toHaveBeenCalled()
  })

  it('does NOT call onChange or onError when onError is unset and the request rejects (preserves current value)', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    apiRequestMock.mockRejectedValueOnce(new Error('boom'))

    render(<DemoCaptcha value="" onChange={onChange} purpose="submission" />)
    await user.click(screen.getByRole('button'))

    // Wait for the catch to run without an unhandled rejection hanging.
    await vi.waitFor(() => expect(apiRequestMock).toHaveBeenCalledTimes(1))
    expect(onChange).not.toHaveBeenCalled()
  })

  it('re-enables the button after a failed attempt (so the user can retry)', async () => {
    const user = userEvent.setup()
    const onError = vi.fn()
    apiRequestMock.mockRejectedValueOnce(new ApiError(410, 'captcha expired'))

    render(<DemoCaptcha value="" onChange={() => {}} onError={onError} purpose="submission" />)
    await user.click(screen.getByRole('button'))

    await vi.waitFor(() => expect(onError).toHaveBeenCalled())
    expect(screen.getByRole('button')).not.toBeDisabled()
  })
})
