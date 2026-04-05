import { useFormContext } from 'react-hook-form'

const inputClass = 'w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background'
const labelClass = 'block text-sm font-medium text-foreground mb-1.5'

export function RequestLoggerConfig() {
  const { register } = useFormContext()

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <input type="checkbox" {...register('config.log_headers')} defaultChecked id="log-headers" className="rounded border-input" />
        <label htmlFor="log-headers" className="text-sm text-foreground">Log request/response headers</label>
      </div>
      <div className="flex items-center gap-2">
        <input type="checkbox" {...register('config.log_query_params')} defaultChecked id="log-query" className="rounded border-input" />
        <label htmlFor="log-query" className="text-sm text-foreground">Log query parameters</label>
      </div>
      <div>
        <label className={labelClass}>Excluded Paths <span className="text-muted-foreground font-normal">(comma-separated)</span></label>
        <input {...register('config.excluded_paths_text')} defaultValue="/health,/ready,/metrics" placeholder="/health,/ready" className={inputClass} />
      </div>
      <div>
        <label className={labelClass}>Max Body Log Size (bytes, 0 = none)</label>
        <input type="number" {...register('config.max_body_log_size', { valueAsNumber: true })} defaultValue={0} className={inputClass} />
      </div>
    </div>
  )
}
