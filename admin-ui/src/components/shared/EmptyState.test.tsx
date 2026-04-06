import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { EmptyState } from './EmptyState'

describe('EmptyState', () => {
  it('renders title and description', () => {
    render(<EmptyState title="No items" description="Create your first item." />)
    expect(screen.getByText('No items')).toBeInTheDocument()
    expect(screen.getByText('Create your first item.')).toBeInTheDocument()
  })

  it('renders action button when provided', () => {
    render(
      <EmptyState
        title="No items"
        description="Create one."
        action={<button>Create</button>}
      />
    )
    expect(screen.getByText('Create')).toBeInTheDocument()
  })

  it('renders icon when provided', () => {
    render(
      <EmptyState
        title="No items"
        description="Create one."
        icon={<span data-testid="icon">icon</span>}
      />
    )
    expect(screen.getByTestId('icon')).toBeInTheDocument()
  })
})
