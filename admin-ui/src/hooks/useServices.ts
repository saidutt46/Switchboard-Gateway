import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as api from '../api/services'
import type { ServiceCreate, ServiceUpdate } from '../api/types'

export const useServices = (params?: { skip?: number; limit?: number; enabled_only?: boolean }) =>
  useQuery({ queryKey: ['services', params], queryFn: () => api.getServices(params) })

export const useService = (id: string) =>
  useQuery({ queryKey: ['services', id], queryFn: () => api.getService(id), enabled: !!id })

export const useServiceStats = (id: string) =>
  useQuery({ queryKey: ['services', id, 'stats'], queryFn: () => api.getServiceStats(id), enabled: !!id })

export const useCreateService = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: ServiceCreate) => api.createService(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['services'] }),
  })
}

export const useUpdateService = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: ServiceUpdate }) => api.updateService(id, data),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['services'] })
      qc.invalidateQueries({ queryKey: ['services', vars.id] })
    },
  })
}

export const useDeleteService = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.deleteService(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['services'] }),
  })
}
