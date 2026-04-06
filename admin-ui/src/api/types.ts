// Types mirroring admin-api/schemas.py Pydantic models

// ============================================================================
// Service
// ============================================================================

export interface Service {
  id: string
  name: string
  protocol: 'http' | 'https' | 'grpc'
  host: string
  port: number
  path: string | null
  connect_timeout_ms: number
  read_timeout_ms: number
  write_timeout_ms: number
  retries: number
  load_balancer_type: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface ServiceCreate {
  name: string
  protocol?: string
  host: string
  port?: number
  path?: string | null
  connect_timeout_ms?: number
  read_timeout_ms?: number
  write_timeout_ms?: number
  retries?: number
  load_balancer_type?: string
  enabled?: boolean
}

export type ServiceUpdate = Partial<ServiceCreate>

export interface ServiceStats {
  service_id: string
  service_name: string
  routes_count: number
  targets_count: number
  plugins_count: number
  enabled: boolean
}

// ============================================================================
// Route
// ============================================================================

export interface Route {
  id: string
  service_id: string
  name: string | null
  hosts: string[] | null
  paths: string[]
  methods: string[]
  strip_path: boolean
  preserve_host: boolean
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface RouteCreate {
  service_id: string
  name?: string | null
  hosts?: string[] | null
  paths: string[]
  methods?: string[]
  strip_path?: boolean
  preserve_host?: boolean
  enabled?: boolean
}

export type RouteUpdate = Partial<RouteCreate>

export interface RouteDetails {
  route: Route
  service: Service
  plugins_count: number
}

// ============================================================================
// Consumer
// ============================================================================

export interface Consumer {
  id: string
  username: string
  email: string | null
  custom_id: string | null
  custom_metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface ConsumerCreate {
  username: string
  email?: string | null
  custom_id?: string | null
  custom_metadata?: Record<string, unknown>
}

export type ConsumerUpdate = Partial<ConsumerCreate>

// ============================================================================
// API Key
// ============================================================================

export interface ApiKey {
  id: string
  name: string | null
  enabled: boolean
  created_at: string
  last_used_at: string | null
  expires_at: string | null
  key_preview: string
}

export interface ApiKeyGenerated {
  id: string
  key: string
  name: string | null
  consumer_id: string
  consumer_username: string
  enabled: boolean
  created_at: string
  warning: string
}

// ============================================================================
// Plugin
// ============================================================================

export interface Plugin {
  id: string
  name: string
  scope: 'global' | 'service' | 'route' | 'consumer'
  service_id: string | null
  route_id: string | null
  consumer_id: string | null
  config: Record<string, unknown>
  enabled: boolean
  priority: number
  created_at: string
  updated_at: string
}

export interface PluginCreate {
  name: string
  scope: string
  service_id?: string | null
  route_id?: string | null
  consumer_id?: string | null
  config?: Record<string, unknown>
  enabled?: boolean
  priority?: number
}

export type PluginUpdate = Partial<PluginCreate>

export interface AvailablePlugin {
  name: string
  description: string
  config_schema: Record<string, unknown>
}

export interface AvailablePlugins {
  authentication: AvailablePlugin[]
  traffic_control: AvailablePlugin[]
  performance: AvailablePlugin[]
  resilience: AvailablePlugin[]
  transformation: AvailablePlugin[]
}

// ============================================================================
// Auth
// ============================================================================

export interface AuthStatus {
  setup_required: boolean
}

export interface AuthTokens {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
}

export interface AdminUser {
  id: string
  username: string
  email: string | null
  role: 'admin' | 'viewer'
  created_at: string
  updated_at: string
}

export interface AdminUserCreate {
  username: string
  password: string
  email?: string | null
  role?: string
}

export type AdminUserUpdate = Partial<Omit<AdminUserCreate, 'username'>>

// ============================================================================
// Health
// ============================================================================

export interface HealthResponse {
  status: string
  version: string
  database: string
  redis: string
}

export interface GatewayHealth {
  status: string
  uptime: string
  database: {
    status: string
    open_connections: number
  }
  checks: {
    database: {
      status: string
      message: string
    }
  }
}
