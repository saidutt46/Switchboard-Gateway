import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as api from '../api/auth'
import type { AdminUserCreate, AdminUserUpdate } from '../api/types'

export const useUsers = () =>
  useQuery({ queryKey: ['users'], queryFn: api.getUsers })

export const useCreateUser = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: AdminUserCreate) => api.createUser(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['users'] }),
  })
}

export const useUpdateUser = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: AdminUserUpdate }) => api.updateUser(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['users'] }),
  })
}

export const useDeleteUser = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.deleteUser(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['users'] }),
  })
}

export const useUpdateProfile = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: AdminUserUpdate) => api.updateMe(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['users'] }),
  })
}
