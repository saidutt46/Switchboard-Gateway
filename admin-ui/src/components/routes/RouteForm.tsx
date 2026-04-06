import { useForm } from 'react-hook-form'
import { cn } from '../../lib/cn'
import { HTTP_METHODS } from '../../lib/constants'
import { useServices } from '../../hooks/useServices'
import type { Route, RouteCreate } from '../../api/types'

interface RouteFormProps {
  initial?: Route
  defaultServiceId?: string
  onSubmit: (data: RouteCreate) => void
  isLoading?: boolean
}

export function RouteForm({ initial, defaultServiceId, onSubmit, isLoading }: RouteFormProps) {
  const { data: services } = useServices()

  const { register, handleSubmit, watch, setValue, formState: { errors } } = useForm<RouteCreate & { pathsText: string; methodsList: string[] }>({
    defaultValues: initial
      ? {
          service_id: initial.service_id,
          name: initial.name || '',
          paths: initial.paths,
          pathsText: initial.paths.join('\n'),
          methods: initial.methods,
          methodsList: initial.methods,
          strip_path: initial.strip_path,
          preserve_host: initial.preserve_host,
          enabled: initial.enabled,
        }
      : {
          service_id: defaultServiceId || '',
          name: '',
          pathsText: '',
          methodsList: ['GET', 'POST', 'PUT', 'DELETE', 'PATCH'],
          strip_path: false,
          preserve_host: false,
          enabled: true,
        },
  })

  const selectedMethods = watch('methodsList') || []

  const toggleMethod = (method: string) => {
    const current = selectedMethods
    const updated = current.includes(method)
      ? current.filter((m) => m !== method)
      : [...current, method]
    setValue('methodsList', updated)
  }

  const onFormSubmit = handleSubmit((data) => {
    const paths = (data.pathsText || '')
      .split('\n')
      .map((p) => p.trim())
      .filter(Boolean)
    if (paths.length === 0) return
    onSubmit({
      service_id: data.service_id,
      name: data.name || undefined,
      paths,
      methods: data.methodsList,
      strip_path: data.strip_path,
      preserve_host: data.preserve_host,
      enabled: data.enabled,
    })
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
    <form onSubmit={onFormSubmit} className="space-y-4">
      {/* Service */}
      <div>
        <label className={labelClass}>Service</label>
        <select
          {...register('service_id', { required: 'Service is required' })}
          className={inputClass}
        >
          <option value="">Select a service...</option>
          {services?.map((s) => (
            <option key={s.id} value={s.id}>{s.name}</option>
          ))}
        </select>
        {errors.service_id && <p className={errorClass}>{errors.service_id.message}</p>}
      </div>

      {/* Name */}
      <div>
        <label className={labelClass}>Name <span className="text-muted-foreground font-normal">(optional)</span></label>
        <input {...register('name')} placeholder="my-api-routes" className={inputClass} />
      </div>

      {/* Paths */}
      <div>
        <label className={labelClass}>Paths <span className="text-muted-foreground font-normal">(one per line)</span></label>
        <textarea
          {...register('pathsText', { required: 'At least one path is required' })}
          placeholder={'/api/users\n/api/users/:id'}
          rows={4}
          className={cn(inputClass, 'font-mono text-xs')}
        />
        {errors.pathsText && <p className={errorClass}>{errors.pathsText.message}</p>}
      </div>

      {/* Methods */}
      <div>
        <label className={labelClass}>Methods</label>
        <div className="flex flex-wrap gap-2">
          {HTTP_METHODS.map((method) => (
            <button
              key={method}
              type="button"
              onClick={() => toggleMethod(method)}
              className={cn(
                'rounded px-2.5 py-1 text-xs font-mono font-medium border transition-colors',
                selectedMethods.includes(method)
                  ? 'bg-primary text-primary-foreground border-primary'
                  : 'bg-background text-muted-foreground border-input hover:bg-accent'
              )}
            >
              {method}
            </button>
          ))}
        </div>
      </div>

      {/* Options */}
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <input type="checkbox" {...register('strip_path')} id="strip-path" className="rounded border-input" />
          <label htmlFor="strip-path" className="text-sm text-foreground">Strip path prefix</label>
        </div>
        <div className="flex items-center gap-2">
          <input type="checkbox" {...register('preserve_host')} id="preserve-host" className="rounded border-input" />
          <label htmlFor="preserve-host" className="text-sm text-foreground">Preserve host header</label>
        </div>
        <div className="flex items-center gap-2">
          <input type="checkbox" {...register('enabled')} id="route-enabled" className="rounded border-input" />
          <label htmlFor="route-enabled" className="text-sm text-foreground">Enabled</label>
        </div>
      </div>

      <button
        type="submit"
        disabled={isLoading}
        className={cn(
          'w-full rounded-md px-4 py-2 text-sm font-medium',
          'bg-primary text-primary-foreground hover:bg-primary/90',
          'disabled:opacity-50 transition-colors'
        )}
      >
        {isLoading ? 'Saving...' : initial ? 'Update Route' : 'Create Route'}
      </button>
    </form>
  )
}
