import { useQuery } from '@tanstack/react-query'
import { getAdminHealth, getGatewayHealth } from '../api/health'

export const useAdminHealth = () =>
  useQuery({
    queryKey: ['health', 'admin'],
    queryFn: getAdminHealth,
    refetchInterval: 30000,
  })

export const useGatewayHealth = () =>
  useQuery({
    queryKey: ['health', 'gateway'],
    queryFn: getGatewayHealth,
    refetchInterval: 30000,
    retry: 1,
  })
