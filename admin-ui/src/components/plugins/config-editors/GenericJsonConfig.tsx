import { useFormContext } from 'react-hook-form'
import { cn } from '../../../lib/cn'

export function GenericJsonConfig() {
  const { register, formState: { errors } } = useFormContext()

  return (
    <div>
      <label className="block text-sm font-medium text-foreground mb-1.5">Configuration (JSON)</label>
      <textarea
        {...register('config.raw_json', {
          validate: (value: string) => {
            if (!value || value.trim() === '') return true
            try { JSON.parse(value); return true } catch { return 'Invalid JSON' }
          },
        })}
        defaultValue="{}"
        rows={10}
        className={cn(
          'w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono',
          'placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background'
        )}
      />
      {errors.config && typeof errors.config === 'object' && 'raw_json' in errors.config && (
        <p className="text-xs text-destructive mt-1">{String((errors.config as Record<string, { message?: string }>).raw_json?.message)}</p>
      )}
    </div>
  )
}
