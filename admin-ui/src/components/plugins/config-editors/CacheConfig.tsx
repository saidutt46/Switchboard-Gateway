import { useFormContext } from 'react-hook-form'

const inputClass = 'w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background'
const labelClass = 'block text-sm font-medium text-foreground mb-1.5'

export function CacheConfig() {
  const { register } = useFormContext()

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className={labelClass}>TTL (seconds)</label>
          <input type="number" {...register('config.ttl_seconds', { valueAsNumber: true })} defaultValue={60} className={inputClass} />
        </div>
        <div>
          <label className={labelClass}>Max TTL (seconds)</label>
          <input type="number" {...register('config.max_ttl_seconds', { valueAsNumber: true })} defaultValue={3600} className={inputClass} />
        </div>
      </div>
      <div>
        <label className={labelClass}>Cacheable Methods <span className="text-muted-foreground font-normal">(comma-separated)</span></label>
        <input {...register('config.cache_methods_text')} defaultValue="GET,HEAD" placeholder="GET,HEAD" className={inputClass} />
      </div>
      <div>
        <label className={labelClass}>Cacheable Status Codes <span className="text-muted-foreground font-normal">(comma-separated)</span></label>
        <input {...register('config.cache_status_codes_text')} defaultValue="200,301,302" placeholder="200,301,302" className={inputClass} />
      </div>
      <div>
        <label className={labelClass}>Max Body Size (bytes)</label>
        <input type="number" {...register('config.max_body_size_bytes', { valueAsNumber: true })} defaultValue={1048576} className={inputClass} />
      </div>
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <input type="checkbox" {...register('config.vary_by_consumer')} id="vary-consumer" className="rounded border-input" />
          <label htmlFor="vary-consumer" className="text-sm text-foreground">Vary by consumer</label>
        </div>
        <div className="flex items-center gap-2">
          <input type="checkbox" {...register('config.vary_by_query')} defaultChecked id="vary-query" className="rounded border-input" />
          <label htmlFor="vary-query" className="text-sm text-foreground">Vary by query params</label>
        </div>
        <div className="flex items-center gap-2">
          <input type="checkbox" {...register('config.respect_cache_control')} defaultChecked id="respect-cc" className="rounded border-input" />
          <label htmlFor="respect-cc" className="text-sm text-foreground">Respect Cache-Control header</label>
        </div>
      </div>
      <div>
        <label className={labelClass}>Redis URL</label>
        <input {...register('config.redis_url')} defaultValue="redis://redis:6379/0" placeholder="redis://redis:6379/0" className={inputClass} />
        <p className="text-xs text-muted-foreground mt-1">Use <code className="font-mono">redis://redis:6379/0</code> for Docker, <code className="font-mono">redis://localhost:6379/0</code> for local</p>
      </div>
    </div>
  )
}
