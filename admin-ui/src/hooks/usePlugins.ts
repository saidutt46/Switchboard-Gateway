import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as api from '../api/plugins'
import type { PluginCreate, PluginUpdate } from '../api/types'

export const usePlugins = (params?: {
  skip?: number; limit?: number; scope?: string; name?: string;
  service_id?: string; route_id?: string; consumer_id?: string; enabled_only?: boolean
}) => useQuery({ queryKey: ['plugins', params], queryFn: () => api.getPlugins(params) })

export const usePlugin = (id: string) =>
  useQuery({ queryKey: ['plugins', id], queryFn: () => api.getPlugin(id), enabled: !!id })

export const useAvailablePlugins = () =>
  useQuery({ queryKey: ['plugins', 'available'], queryFn: api.getAvailablePlugins, staleTime: Infinity })

export const useCreatePlugin = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: PluginCreate) => api.createPlugin(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['plugins'] }),
  })
}

export const useUpdatePlugin = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: PluginUpdate }) => api.updatePlugin(id, data),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['plugins'] })
      qc.invalidateQueries({ queryKey: ['plugins', vars.id] })
    },
  })
}

export const useDeletePlugin = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.deletePlugin(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['plugins'] }),
  })
}
