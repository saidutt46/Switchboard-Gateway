import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as api from '../api/routes'
import type { RouteCreate, RouteUpdate } from '../api/types'

export const useRoutes = (params?: { skip?: number; limit?: number; service_id?: string; enabled_only?: boolean }) =>
  useQuery({ queryKey: ['routes', params], queryFn: () => api.getRoutes(params) })

export const useRoute = (id: string) =>
  useQuery({ queryKey: ['routes', id], queryFn: () => api.getRoute(id), enabled: !!id })

export const useRouteDetails = (id: string) =>
  useQuery({ queryKey: ['routes', id, 'details'], queryFn: () => api.getRouteDetails(id), enabled: !!id })

export const useCreateRoute = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: RouteCreate) => api.createRoute(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['routes'] })
      qc.invalidateQueries({ queryKey: ['services'] })
    },
  })
}

export const useUpdateRoute = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: RouteUpdate }) => api.updateRoute(id, data),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['routes'] })
      qc.invalidateQueries({ queryKey: ['routes', vars.id] })
    },
  })
}

export const useDeleteRoute = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.deleteRoute(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['routes'] })
      qc.invalidateQueries({ queryKey: ['services'] })
    },
  })
}
