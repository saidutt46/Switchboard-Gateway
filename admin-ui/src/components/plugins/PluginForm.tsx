import { useForm, FormProvider } from 'react-hook-form'
import { cn } from '../../lib/cn'
import { PLUGIN_SCOPES } from '../../lib/constants'
import { useServices } from '../../hooks/useServices'
import { useRoutes } from '../../hooks/useRoutes'
import { useConsumers } from '../../hooks/useConsumers'
import { ApiKeyAuthConfig } from './config-editors/ApiKeyAuthConfig'
import { JwtAuthConfig } from './config-editors/JwtAuthConfig'
import { RateLimitConfig } from './config-editors/RateLimitConfig'
import { CacheConfig } from './config-editors/CacheConfig'
import { CorsConfig } from './config-editors/CorsConfig'
import { RequestLoggerConfig } from './config-editors/RequestLoggerConfig'
import { GenericJsonConfig } from './config-editors/GenericJsonConfig'
import type { Plugin, PluginCreate } from '../../api/types'

const PLUGIN_TYPES = [
  { value: 'api-key-auth', label: 'API Key Auth', category: 'Authentication' },
  { value: 'jwt-auth', label: 'JWT Auth', category: 'Authentication' },
  { value: 'rate-limit', label: 'Rate Limit', category: 'Traffic Control' },
  { value: 'cache', label: 'Cache', category: 'Performance' },
  { value: 'cors', label: 'CORS', category: 'Transformation' },
  { value: 'request-logger', label: 'Request Logger', category: 'Observability' },
]

const CONFIG_EDITORS: Record<string, React.ComponentType> = {
  'api-key-auth': ApiKeyAuthConfig,
  'jwt-auth': JwtAuthConfig,
  'rate-limit': RateLimitConfig,
  'cache': CacheConfig,
  'cors': CorsConfig,
  'request-logger': RequestLoggerConfig,
}

interface PluginFormProps {
  initial?: Plugin
  onSubmit: (data: PluginCreate) => void
  isLoading?: boolean
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function buildConfig(pluginName: string, formConfig: any): Record<string, unknown> {
  if (formConfig?.raw_json) {
    try { return JSON.parse(formConfig.raw_json) } catch { return {} }
  }

  const config: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(formConfig || {})) {
    if (key.endsWith('_text') && typeof value === 'string') {
      const realKey = key.replace(/_text$/, '')
      config[realKey] = value.split(',').map((s: string) => s.trim()).filter(Boolean)
    } else if (key === 'key_names' && typeof value === 'object') {
      config.key_names = Object.values(value as Record<string, string>).filter(Boolean)
    } else {
      config[key] = value
    }
  }
  void pluginName
  return config
}

export function PluginForm({ initial, onSubmit, isLoading }: PluginFormProps) {
  const methods = useForm({
    defaultValues: initial
      ? { name: initial.name, scope: initial.scope, service_id: initial.service_id || '', route_id: initial.route_id || '', consumer_id: initial.consumer_id || '', priority: initial.priority, enabled: initial.enabled, config: initial.config }
      : { name: '', scope: 'global', service_id: '', route_id: '', consumer_id: '', priority: 100, enabled: true, config: {} },
  })

  const { register, handleSubmit, watch, formState: { errors } } = methods
  const selectedName = watch('name')
  const selectedScope = watch('scope')

  const { data: services } = useServices()
  const { data: routes } = useRoutes()
  const { data: consumers } = useConsumers()

  const ConfigEditor = selectedName ? (CONFIG_EDITORS[selectedName] || GenericJsonConfig) : null

  const onFormSubmit = handleSubmit((data) => {
    const config = buildConfig(data.name, data.config)
    onSubmit({
      name: data.name,
      scope: data.scope,
      service_id: data.scope === 'service' ? data.service_id || undefined : undefined,
      route_id: data.scope === 'route' ? data.route_id || undefined : undefined,
      consumer_id: data.scope === 'consumer' ? data.consumer_id || undefined : undefined,
      config,
      enabled: data.enabled,
      priority: Number(data.priority),
    })
  })

  const inputClass = cn(
    'w-full rounded-md border border-input bg-background px-3 py-2 text-sm',
    'placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background'
  )
  const labelClass = 'block text-sm font-medium text-foreground mb-1.5'

  return (
    <FormProvider {...methods}>
      <form onSubmit={onFormSubmit} className="space-y-4">
        {/* Plugin Type */}
        <div>
          <label className={labelClass}>Plugin Type</label>
          <select {...register('name', { required: 'Plugin type is required' })} className={inputClass} disabled={!!initial}>
            <option value="">Select a plugin...</option>
            {PLUGIN_TYPES.map((pt) => (
              <option key={pt.value} value={pt.value}>{pt.label} — {pt.category}</option>
            ))}
          </select>
          {errors.name && <p className="text-xs text-destructive mt-1">{errors.name.message}</p>}
        </div>

        {/* Scope */}
        <div>
          <label className={labelClass}>Scope</label>
          <select {...register('scope')} className={inputClass}>
            {PLUGIN_SCOPES.map((s) => (
              <option key={s} value={s}>{s.charAt(0).toUpperCase() + s.slice(1)}</option>
            ))}
          </select>
        </div>

        {/* Entity Selector */}
        {selectedScope === 'service' && (
          <div>
            <label className={labelClass}>Service</label>
            <select {...register('service_id', { required: 'Service is required' })} className={inputClass}>
              <option value="">Select a service...</option>
              {services?.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </select>
          </div>
        )}
        {selectedScope === 'route' && (
          <div>
            <label className={labelClass}>Route</label>
            <select {...register('route_id', { required: 'Route is required' })} className={inputClass}>
              <option value="">Select a route...</option>
              {routes?.map((r) => <option key={r.id} value={r.id}>{r.name || r.paths[0]}</option>)}
            </select>
          </div>
        )}
        {selectedScope === 'consumer' && (
          <div>
            <label className={labelClass}>Consumer</label>
            <select {...register('consumer_id', { required: 'Consumer is required' })} className={inputClass}>
              <option value="">Select a consumer...</option>
              {consumers?.map((c) => <option key={c.id} value={c.id}>{c.username}</option>)}
            </select>
          </div>
        )}

        {/* Priority */}
        <div>
          <label className={labelClass}>Priority <span className="text-muted-foreground font-normal">(lower = runs first)</span></label>
          <input type="number" {...register('priority', { valueAsNumber: true, min: 1, max: 1000 })} className={inputClass} />
        </div>

        {/* Config Editor */}
        {ConfigEditor && (
          <div>
            <label className="block text-xs font-medium text-muted-foreground mb-2 uppercase tracking-wider">Configuration</label>
            <div className="rounded-md border p-4 space-y-4 bg-muted/30">
              <ConfigEditor />
            </div>
          </div>
        )}

        {/* Enabled */}
        <div className="flex items-center gap-2">
          <input type="checkbox" {...register('enabled')} id="plugin-enabled" className="rounded border-input" />
          <label htmlFor="plugin-enabled" className="text-sm text-foreground">Enabled</label>
        </div>

        <button
          type="submit"
          disabled={isLoading}
          className={cn(
            'w-full rounded-md px-4 py-2 text-sm font-medium',
            'bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors'
          )}
        >
          {isLoading ? 'Saving...' : initial ? 'Update Plugin' : 'Create Plugin'}
        </button>
      </form>
    </FormProvider>
  )
}
