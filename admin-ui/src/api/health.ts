import api from './client'
import type { HealthResponse, GatewayHealth } from './types'

export const getAdminHealth = () =>
  api.get('health').json<HealthResponse>()

export const getGatewayHealth = async (): Promise<GatewayHealth> => {
  const response = await fetch('/gateway/health')
  return response.json()
}
