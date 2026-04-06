import api from './client'
import type { Route, RouteCreate, RouteUpdate, RouteDetails } from './types'

export const getRoutes = (params?: { skip?: number; limit?: number; service_id?: string; enabled_only?: boolean }) =>
  api.get('routes', { searchParams: params as Record<string, string | number | boolean> }).json<Route[]>()

export const getRoute = (id: string) =>
  api.get(`routes/${id}`).json<Route>()

export const getRouteDetails = (id: string) =>
  api.get(`routes/${id}/details`).json<RouteDetails>()

export const createRoute = (data: RouteCreate) =>
  api.post('routes', { json: data }).json<Route>()

export const updateRoute = (id: string, data: RouteUpdate) =>
  api.put(`routes/${id}`, { json: data }).json<Route>()

export const deleteRoute = (id: string) =>
  api.delete(`routes/${id}`)
