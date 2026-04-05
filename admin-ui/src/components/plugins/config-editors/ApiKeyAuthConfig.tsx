import { useFormContext } from 'react-hook-form'

const inputClass = 'w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background'
const labelClass = 'block text-sm font-medium text-foreground mb-1.5'

export function ApiKeyAuthConfig() {
  const { register } = useFormContext()

  return (
    <div className="space-y-4">
      <div>
        <label className={labelClass}>Key Header Name</label>
        <input {...register('config.key_names.0')} defaultValue="X-API-Key" placeholder="X-API-Key" className={inputClass} />
      </div>
      <div className="flex items-center gap-2">
        <input type="checkbox" {...register('config.key_in_header')} defaultChecked id="key-in-header" className="rounded border-input" />
        <label htmlFor="key-in-header" className="text-sm text-foreground">Accept key in header</label>
      </div>
      <div className="flex items-center gap-2">
        <input type="checkbox" {...register('config.key_in_query')} defaultChecked id="key-in-query" className="rounded border-input" />
        <label htmlFor="key-in-query" className="text-sm text-foreground">Accept key in query param</label>
      </div>
      <div className="flex items-center gap-2">
        <input type="checkbox" {...register('config.hide_credentials')} defaultChecked id="hide-creds" className="rounded border-input" />
        <label htmlFor="hide-creds" className="text-sm text-foreground">Hide credentials before proxying</label>
      </div>
      <div>
        <label className={labelClass}>Cache TTL (seconds)</label>
        <input type="number" {...register('config.cache_ttl_seconds', { valueAsNumber: true })} defaultValue={300} className={inputClass} />
      </div>
      <div>
        <label className={labelClass}>Bypass Paths <span className="text-muted-foreground font-normal">(comma-separated)</span></label>
        <input {...register('config.bypass_paths_text')} defaultValue="/health,/ready,/metrics" placeholder="/health,/ready" className={inputClass} />
      </div>
    </div>
  )
}
