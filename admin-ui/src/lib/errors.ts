import { HTTPError } from 'ky'

interface ApiErrorBody {
  detail?: string | { msg: string }[]
  message?: string
  error?: string
}

/**
 * Extract a human-readable error message from a ky HTTPError or generic Error.
 * Handles FastAPI's error response formats (422 validation, 409 conflict, etc.)
 */
export async function parseApiError(error: unknown): Promise<string> {
  if (error instanceof HTTPError) {
    const status = error.response.status

    try {
      const body: ApiErrorBody = await error.response.json()

      // FastAPI validation error (422)
      if (Array.isArray(body.detail)) {
        return body.detail.map((d) => d.msg).join('. ')
      }

      // Standard error detail (409, 404, 400, etc.)
      if (typeof body.detail === 'string') {
        return body.detail
      }

      if (body.message) return body.message
      if (body.error) return body.error
    } catch {
      // Response wasn't JSON
    }

    // Fallback to status text
    const statusMessages: Record<number, string> = {
      400: 'Invalid request. Please check your input.',
      401: 'Unauthorized. Authentication required.',
      403: 'Forbidden. You do not have permission.',
      404: 'Not found. The resource may have been deleted.',
      409: 'Conflict. A resource with this name already exists.',
      422: 'Validation error. Please check your input.',
      429: 'Too many requests. Please try again later.',
      500: 'Server error. Please try again.',
      502: 'Bad gateway. The server is unreachable.',
      503: 'Service unavailable. Please try again later.',
    }

    return statusMessages[status] || `Request failed (${status})`
  }

  if (error instanceof Error) {
    if (error.message.includes('fetch')) return 'Network error. Please check your connection.'
    return error.message
  }

  return 'An unexpected error occurred'
}
