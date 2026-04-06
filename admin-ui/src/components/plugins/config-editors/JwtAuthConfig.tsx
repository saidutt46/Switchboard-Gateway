import { useFormContext } from 'react-hook-form'

const inputClass = 'w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background'
const labelClass = 'block text-sm font-medium text-foreground mb-1.5'

export function JwtAuthConfig() {
  const { register } = useFormContext()

  return (
    <div className="space-y-4">
      <div>
        <label className={labelClass}>Secret Key</label>
        <input type="password" {...register('config.secret_key', { required: 'Secret key is required' })} placeholder="your-secret-key" className={inputClass} />
      </div>
      <div>
        <label className={labelClass}>Algorithm</label>
        <select {...register('config.algorithm')} defaultValue="HS256" className={inputClass}>
          <option value="HS256">HS256</option>
        </select>
      </div>
      <div>
        <label className={labelClass}>Header Name</label>
        <input {...register('config.header_name')} defaultValue="Authorization" className={inputClass} />
      </div>
      <div>
        <label className={labelClass}>Cookie Name <span className="text-muted-foreground font-normal">(optional)</span></label>
        <input {...register('config.cookie_name')} placeholder="jwt_token" className={inputClass} />
      </div>
      <div>
        <label className={labelClass}>Issuer <span className="text-muted-foreground font-normal">(optional)</span></label>
        <input {...register('config.issuer')} placeholder="your-issuer" className={inputClass} />
      </div>
      <div>
        <label className={labelClass}>Audience <span className="text-muted-foreground font-normal">(optional)</span></label>
        <input {...register('config.audience')} placeholder="your-audience" className={inputClass} />
      </div>
      <div>
        <label className={labelClass}>Cache TTL (seconds)</label>
        <input type="number" {...register('config.cache_ttl_seconds', { valueAsNumber: true })} defaultValue={300} className={inputClass} />
      </div>
    </div>
  )
}
