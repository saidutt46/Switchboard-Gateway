import api from './client'
import type { AuthStatus, AuthTokens, AdminUser, AdminUserCreate, AdminUserUpdate } from './types'

export const getAuthStatus = () =>
  api.get('auth/status').json<AuthStatus>()

export const setup = (data: { username: string; password: string; email?: string }) =>
  api.post('auth/setup', { json: data }).json<AdminUser>()

export const login = (data: { username: string; password: string }) =>
  api.post('auth/login', { json: data }).json<AuthTokens>()

export const refreshTokens = (refresh_token: string) =>
  api.post('auth/refresh', { json: { refresh_token } }).json<AuthTokens>()

export const getMe = () =>
  api.get('auth/me').json<AdminUser>()

export const updateMe = (data: AdminUserUpdate) =>
  api.put('auth/me', { json: data }).json<AdminUser>()

export const getUsers = () =>
  api.get('auth/users').json<AdminUser[]>()

export const createUser = (data: AdminUserCreate) =>
  api.post('auth/users', { json: data }).json<AdminUser>()

export const updateUser = (id: string, data: AdminUserUpdate) =>
  api.put(`auth/users/${id}`, { json: data }).json<AdminUser>()

export const deleteUser = (id: string) =>
  api.delete(`auth/users/${id}`)
