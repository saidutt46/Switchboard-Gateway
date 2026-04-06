import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Header } from './Header'

describe('Header', () => {
  it('renders breadcrumbs', () => {
    render(<Header breadcrumbs={[{ label: 'Services' }, { label: 'my-service' }]} />)
    expect(screen.getByText('Services')).toBeInTheDocument()
    expect(screen.getByText('my-service')).toBeInTheDocument()
  })

  it('renders single breadcrumb', () => {
    render(<Header breadcrumbs={[{ label: 'Dashboard' }]} />)
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
  })

  it('renders action when provided', () => {
    render(
      <Header
        breadcrumbs={[{ label: 'Test' }]}
        action={<button>New Item</button>}
      />
    )
    expect(screen.getByText('New Item')).toBeInTheDocument()
  })

  it('last breadcrumb has font-medium styling', () => {
    render(<Header breadcrumbs={[{ label: 'Parent' }, { label: 'Current' }]} />)
    const current = screen.getByText('Current')
    expect(current.className).toContain('font-semibold')
  })
})
