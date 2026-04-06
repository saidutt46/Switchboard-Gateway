import ky from 'ky'

const api = ky.create({
  prefixUrl: import.meta.env.VITE_API_URL || '/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

export default api
