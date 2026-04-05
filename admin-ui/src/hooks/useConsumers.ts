import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as api from '../api/consumers'
import type { ConsumerCreate, ConsumerUpdate } from '../api/types'

export const useConsumers = (params?: { skip?: number; limit?: number }) =>
  useQuery({ queryKey: ['consumers', params], queryFn: () => api.getConsumers(params) })

export const useConsumer = (id: string) =>
  useQuery({ queryKey: ['consumers', id], queryFn: () => api.getConsumer(id), enabled: !!id })

export const useCreateConsumer = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: ConsumerCreate) => api.createConsumer(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['consumers'] }),
  })
}

export const useUpdateConsumer = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: ConsumerUpdate }) => api.updateConsumer(id, data),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['consumers'] })
      qc.invalidateQueries({ queryKey: ['consumers', vars.id] })
    },
  })
}

export const useDeleteConsumer = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.deleteConsumer(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['consumers'] }),
  })
}

// API Keys
export const useApiKeys = (consumerId: string) =>
  useQuery({ queryKey: ['consumers', consumerId, 'keys'], queryFn: () => api.getApiKeys(consumerId), enabled: !!consumerId })

export const useGenerateApiKey = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ consumerId, name }: { consumerId: string; name?: string }) => api.generateApiKey(consumerId, name),
    onSuccess: (_data, vars) => qc.invalidateQueries({ queryKey: ['consumers', vars.consumerId, 'keys'] }),
  })
}

export const useRevokeApiKey = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ consumerId, keyId }: { consumerId: string; keyId: string }) => api.revokeApiKey(consumerId, keyId),
    onSuccess: (_data, vars) => qc.invalidateQueries({ queryKey: ['consumers', vars.consumerId, 'keys'] }),
  })
}

export const useToggleApiKey = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ consumerId, keyId, enable }: { consumerId: string; keyId: string; enable: boolean }) =>
      enable ? api.enableApiKey(consumerId, keyId) : api.disableApiKey(consumerId, keyId),
    onSuccess: (_data, vars) => qc.invalidateQueries({ queryKey: ['consumers', vars.consumerId, 'keys'] }),
  })
}
