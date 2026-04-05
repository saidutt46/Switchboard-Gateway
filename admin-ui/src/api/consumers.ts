import api from './client'
import type { Consumer, ConsumerCreate, ConsumerUpdate, ApiKey, ApiKeyGenerated } from './types'

export const getConsumers = (params?: { skip?: number; limit?: number }) =>
  api.get('consumers', { searchParams: params as Record<string, string | number | boolean> }).json<Consumer[]>()

export const getConsumer = (id: string) =>
  api.get(`consumers/${id}`).json<Consumer>()

export const createConsumer = (data: ConsumerCreate) =>
  api.post('consumers', { json: data }).json<Consumer>()

export const updateConsumer = (id: string, data: ConsumerUpdate) =>
  api.put(`consumers/${id}`, { json: data }).json<Consumer>()

export const deleteConsumer = (id: string) =>
  api.delete(`consumers/${id}`)

// API Keys
export const getApiKeys = (consumerId: string) =>
  api.get(`consumers/${consumerId}/keys`).json<ApiKey[]>()

export const generateApiKey = (consumerId: string, name?: string) =>
  api.post(`consumers/${consumerId}/keys`, { searchParams: name ? { name } : {} }).json<ApiKeyGenerated>()

export const revokeApiKey = (consumerId: string, keyId: string) =>
  api.delete(`consumers/${consumerId}/keys/${keyId}`)

export const disableApiKey = (consumerId: string, keyId: string) =>
  api.patch(`consumers/${consumerId}/keys/${keyId}/disable`).json<{ message: string; key_id: string; enabled: boolean }>()

export const enableApiKey = (consumerId: string, keyId: string) =>
  api.patch(`consumers/${consumerId}/keys/${keyId}/enable`).json<{ message: string; key_id: string; enabled: boolean }>()
