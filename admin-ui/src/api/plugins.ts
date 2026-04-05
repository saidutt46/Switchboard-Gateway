import api from './client'
import type { Plugin, PluginCreate, PluginUpdate, AvailablePlugins } from './types'

export const getPlugins = (params?: {
  skip?: number
  limit?: number
  scope?: string
  name?: string
  service_id?: string
  route_id?: string
  consumer_id?: string
  enabled_only?: boolean
}) => api.get('plugins', { searchParams: params as Record<string, string | number | boolean> }).json<Plugin[]>()

export const getPlugin = (id: string) =>
  api.get(`plugins/${id}`).json<Plugin>()

export const getAvailablePlugins = () =>
  api.get('plugins/available').json<AvailablePlugins>()

export const createPlugin = (data: PluginCreate) =>
  api.post('plugins', { json: data }).json<Plugin>()

export const updatePlugin = (id: string, data: PluginUpdate) =>
  api.put(`plugins/${id}`, { json: data }).json<Plugin>()

export const deletePlugin = (id: string) =>
  api.delete(`plugins/${id}`)
