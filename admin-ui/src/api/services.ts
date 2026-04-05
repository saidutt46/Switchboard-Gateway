import api from './client'
import type { Service, ServiceCreate, ServiceUpdate, ServiceStats } from './types'

export const getServices = (params?: { skip?: number; limit?: number; enabled_only?: boolean }) =>
  api.get('services', { searchParams: params as Record<string, string | number | boolean> }).json<Service[]>()

export const getService = (id: string) =>
  api.get(`services/${id}`).json<Service>()

export const getServiceStats = (id: string) =>
  api.get(`services/${id}/stats`).json<ServiceStats>()

export const createService = (data: ServiceCreate) =>
  api.post('services', { json: data }).json<Service>()

export const updateService = (id: string, data: ServiceUpdate) =>
  api.put(`services/${id}`, { json: data }).json<Service>()

export const deleteService = (id: string) =>
  api.delete(`services/${id}`)
