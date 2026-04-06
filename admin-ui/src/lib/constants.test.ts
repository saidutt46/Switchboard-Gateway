import { describe, it, expect } from 'vitest'
import { HTTP_METHODS, METHOD_COLORS, PLUGIN_SCOPES, PROTOCOLS } from './constants'

describe('constants', () => {
  it('has all standard HTTP methods', () => {
    expect(HTTP_METHODS).toContain('GET')
    expect(HTTP_METHODS).toContain('POST')
    expect(HTTP_METHODS).toContain('PUT')
    expect(HTTP_METHODS).toContain('DELETE')
    expect(HTTP_METHODS).toContain('PATCH')
  })

  it('has color for each method', () => {
    for (const method of HTTP_METHODS) {
      expect(METHOD_COLORS[method]).toBeDefined()
    }
  })

  it('has all plugin scopes', () => {
    expect(PLUGIN_SCOPES).toEqual(['global', 'service', 'route', 'consumer'])
  })

  it('has all protocols', () => {
    expect(PROTOCOLS).toEqual(['http', 'https', 'grpc'])
  })
})
