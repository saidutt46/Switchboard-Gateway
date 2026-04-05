import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StatusBadge } from './StatusBadge'

describe('StatusBadge', () => {
  it('renders "Enabled" when enabled is true', () => {
    render(<StatusBadge enabled={true} />)
    expect(screen.getByText('Enabled')).toBeInTheDocument()
  })

  it('renders "Disabled" when enabled is false', () => {
    render(<StatusBadge enabled={false} />)
    expect(screen.getByText('Disabled')).toBeInTheDocument()
  })

  it('has green styling when enabled', () => {
    render(<StatusBadge enabled={true} />)
    const badge = screen.getByText('Enabled')
    expect(badge.className).toContain('emerald')
  })

  it('has muted styling when disabled', () => {
    render(<StatusBadge enabled={false} />)
    const badge = screen.getByText('Disabled')
    expect(badge.className).toContain('muted')
  })
})
