import { useForm } from 'react-hook-form'
import { cn } from '../../lib/cn'
import { PROTOCOLS } from '../../lib/constants'
import type { Service, ServiceCreate } from '../../api/types'

interface ServiceFormProps {
  initial?: Service
  onSubmit: (data: ServiceCreate) => void
  isLoading?: boolean
}

export function ServiceForm({ initial, onSubmit, isLoading }: ServiceFormProps) {
  const { register, handleSubmit, formState: { errors } } = useForm<ServiceCreate>({
    defaultValues: initial
      ? {
          name: initial.name,
          protocol: initial.protocol,
          host: initial.host,
          port: initial.port,
          path: initial.path || '',
          connect_timeout_ms: initial.connect_timeout_ms,
          read_timeout_ms: initial.read_timeout_ms,
          write_timeout_ms: initial.write_timeout_ms,
          retries: initial.retries,
          load_balancer_type: initial.load_balancer_type,
          enabled: initial.enabled,
        }
      : {
          protocol: 'http',
          port: 80,
          connect_timeout_ms: 5000,
          read_timeout_ms: 60000,
          write_timeout_ms: 60000,
          retries: 0,
          load_balancer_type: 'round-robin',
          enabled: true,
        },
  })

  const inputClass = cn(
    'w-full rounded-md border border-input bg-background px-3 py-2 text-sm',
    'placeholder:text-muted-foreground',
    'focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background',
    'disabled:opacity-50'
  )

  const labelClass = 'block text-sm font-medium text-foreground mb-1.5'
  const errorClass = 'text-xs text-destructive mt-1'

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      {/* Name */}
      <div>
        <label className={labelClass}>Name</label>
        <input
          {...register('name', { required: 'Name is required', minLength: { value: 1, message: 'Name is required' } })}
          placeholder="my-backend-service"
          className={inputClass}
        />
        {errors.name && <p className={errorClass}>{errors.name.message}</p>}
      </div>

      {/* Protocol + Host + Port row */}
      <div className="grid grid-cols-3 gap-3">
        <div>
          <label className={labelClass}>Protocol</label>
          <select {...register('protocol')} className={inputClass}>
            {PROTOCOLS.map((p) => (
              <option key={p} value={p}>{p}</option>
            ))}
          </select>
        </div>
        <div>
          <label className={labelClass}>Host</label>
          <input
            {...register('host', { required: 'Host is required' })}
            placeholder="api.example.com"
            className={inputClass}
          />
          {errors.host && <p className={errorClass}>{errors.host.message}</p>}
        </div>
        <div>
          <label className={labelClass}>Port</label>
          <input
            type="number"
            {...register('port', { valueAsNumber: true, min: { value: 1, message: 'Min 1' }, max: { value: 65535, message: 'Max 65535' } })}
            className={inputClass}
          />
          {errors.port && <p className={errorClass}>{errors.port.message}</p>}
        </div>
      </div>

      {/* Path */}
      <div>
        <label className={labelClass}>Base Path <span className="text-muted-foreground font-normal">(optional)</span></label>
        <input {...register('path')} placeholder="/v1" className={inputClass} />
      </div>

      {/* Timeouts */}
      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-2 uppercase tracking-wider">Timeouts (ms)</label>
        <div className="grid grid-cols-3 gap-3">
          <div>
            <label className={labelClass}>Connect</label>
            <input type="number" {...register('connect_timeout_ms', { valueAsNumber: true, min: 100 })} className={inputClass} />
          </div>
          <div>
            <label className={labelClass}>Read</label>
            <input type="number" {...register('read_timeout_ms', { valueAsNumber: true, min: 100 })} className={inputClass} />
          </div>
          <div>
            <label className={labelClass}>Write</label>
            <input type="number" {...register('write_timeout_ms', { valueAsNumber: true, min: 100 })} className={inputClass} />
          </div>
        </div>
      </div>

      {/* Retries + Load Balancer */}
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className={labelClass}>Retries</label>
          <input type="number" {...register('retries', { valueAsNumber: true, min: 0, max: 10 })} className={inputClass} />
        </div>
        <div>
          <label className={labelClass}>Load Balancer</label>
          <select {...register('load_balancer_type')} className={inputClass}>
            <option value="round-robin">Round Robin</option>
            <option value="least-connections">Least Connections</option>
            <option value="weighted">Weighted</option>
            <option value="ip-hash">IP Hash</option>
          </select>
        </div>
      </div>

      {/* Enabled */}
      <div className="flex items-center gap-2">
        <input type="checkbox" {...register('enabled')} id="service-enabled" className="rounded border-input" />
        <label htmlFor="service-enabled" className="text-sm text-foreground">Enabled</label>
      </div>

      {/* Submit */}
      <button
        type="submit"
        disabled={isLoading}
        className={cn(
          'w-full rounded-md px-4 py-2 text-sm font-medium',
          'bg-primary text-primary-foreground hover:bg-primary/90',
          'disabled:opacity-50 transition-colors'
        )}
      >
        {isLoading ? 'Saving...' : initial ? 'Update Service' : 'Create Service'}
      </button>
    </form>
  )
}
