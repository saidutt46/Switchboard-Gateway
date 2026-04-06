import { useFormContext } from 'react-hook-form'

const inputClass = 'w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background'
const labelClass = 'block text-sm font-medium text-foreground mb-1.5'

export function CorsConfig() {
  const { register } = useFormContext()

  return (
    <div className="space-y-4">
      <div>
        <label className={labelClass}>Allowed Origins <span className="text-muted-foreground font-normal">(comma-separated)</span></label>
        <input {...register('config.allowed_origins_text')} defaultValue="*" placeholder="*,https://example.com" className={inputClass} />
      </div>
      <div>
        <label className={labelClass}>Allowed Methods <span className="text-muted-foreground font-normal">(comma-separated)</span></label>
        <input {...register('config.allowed_methods_text')} defaultValue="GET,POST,PUT,DELETE,PATCH,OPTIONS" className={inputClass} />
      </div>
      <div>
        <label className={labelClass}>Allowed Headers <span className="text-muted-foreground font-normal">(comma-separated)</span></label>
        <input {...register('config.allowed_headers_text')} defaultValue="Content-Type,Authorization,X-API-Key" className={inputClass} />
      </div>
      <div>
        <label className={labelClass}>Exposed Headers <span className="text-muted-foreground font-normal">(comma-separated, optional)</span></label>
        <input {...register('config.exposed_headers_text')} placeholder="X-Request-ID" className={inputClass} />
      </div>
      <div>
        <label className={labelClass}>Max Age (seconds)</label>
        <input type="number" {...register('config.max_age', { valueAsNumber: true })} defaultValue={86400} className={inputClass} />
      </div>
      <div className="flex items-center gap-2">
        <input type="checkbox" {...register('config.allow_credentials')} id="cors-creds" className="rounded border-input" />
        <label htmlFor="cors-creds" className="text-sm text-foreground">Allow credentials</label>
      </div>
    </div>
  )
}
