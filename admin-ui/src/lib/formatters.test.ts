import { describe, it, expect } from 'vitest'
import { shortId, formatDate } from './formatters'

describe('shortId', () => {
  it('returns first 8 characters of a UUID', () => {
    expect(shortId('550e8400-e29b-41d4-a716-446655440001')).toBe('550e8400')
  })

  it('handles short strings', () => {
    expect(shortId('abc')).toBe('abc')
  })
})

describe('formatDate', () => {
  it('returns "just now" for very recent dates', () => {
    const now = new Date().toISOString()
    expect(formatDate(now)).toBe('just now')
  })

  it('returns minutes ago for recent dates', () => {
    const fiveMinAgo = new Date(Date.now() - 5 * 60 * 1000).toISOString()
    expect(formatDate(fiveMinAgo)).toBe('5m ago')
  })

  it('returns hours ago', () => {
    const threeHoursAgo = new Date(Date.now() - 3 * 3600 * 1000).toISOString()
    expect(formatDate(threeHoursAgo)).toBe('3h ago')
  })

  it('returns days ago', () => {
    const twoDaysAgo = new Date(Date.now() - 2 * 86400 * 1000).toISOString()
    expect(formatDate(twoDaysAgo)).toBe('2d ago')
  })

  it('returns absolute date when mode is absolute', () => {
    const result = formatDate('2024-06-15T10:30:00Z', 'absolute')
    expect(result).toContain('Jun')
    expect(result).toContain('15')
    expect(result).toContain('2024')
  })
})
