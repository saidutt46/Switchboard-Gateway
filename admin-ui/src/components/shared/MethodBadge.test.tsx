import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MethodBadge } from './MethodBadge'

describe('MethodBadge', () => {
  it('renders the method text', () => {
    render(<MethodBadge method="GET" />)
    expect(screen.getByText('GET')).toBeInTheDocument()
  })

  it('applies green color for GET', () => {
    render(<MethodBadge method="GET" />)
    expect(screen.getByText('GET').className).toContain('emerald')
  })

  it('applies blue color for POST', () => {
    render(<MethodBadge method="POST" />)
    expect(screen.getByText('POST').className).toContain('blue')
  })

  it('applies red color for DELETE', () => {
    render(<MethodBadge method="DELETE" />)
    expect(screen.getByText('DELETE').className).toContain('red')
  })

  it('handles lowercase methods', () => {
    render(<MethodBadge method="get" />)
    expect(screen.getByText('get')).toBeInTheDocument()
  })
})
