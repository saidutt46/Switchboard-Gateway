import ky from 'ky'

const api = ky.create({
  prefixUrl: import.meta.env.VITE_API_URL || '/api',
  timeout: 10000,
  hooks: {
    beforeRequest: [
      (request) => {
        const token = localStorage.getItem('switchboard_access_token')
        if (token) {
          request.headers.set('Authorization', `Bearer ${token}`)
        }
      },
    ],
    afterResponse: [
      (_request, _options, response) => {
        // On 401, clear tokens and redirect to login
        if (response.status === 401) {
          const isAuthEndpoint = response.url.includes('/auth/login') ||
            response.url.includes('/auth/setup') ||
            response.url.includes('/auth/status')
          if (!isAuthEndpoint) {
            localStorage.removeItem('switchboard_access_token')
            localStorage.removeItem('switchboard_refresh_token')
            localStorage.removeItem('switchboard_token_expires')
            if (window.location.pathname !== '/login') {
              window.location.href = '/login'
            }
          }
        }
      },
    ],
  },
})

export default api
