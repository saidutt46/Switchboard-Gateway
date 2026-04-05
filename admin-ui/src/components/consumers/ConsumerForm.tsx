import { useForm } from 'react-hook-form'
import { cn } from '../../lib/cn'
import type { Consumer, ConsumerCreate } from '../../api/types'

interface ConsumerFormProps {
  initial?: Consumer
  onSubmit: (data: ConsumerCreate) => void
  isLoading?: boolean
}

export function ConsumerForm({ initial, onSubmit, isLoading }: ConsumerFormProps) {
  const { register, handleSubmit, formState: { errors } } = useForm<ConsumerCreate>({
    defaultValues: initial
      ? { username: initial.username, email: initial.email || '', custom_id: initial.custom_id || '' }
      : { username: '', email: '', custom_id: '' },
  })

  const inputClass = cn(
    'w-full rounded-md border border-input bg-background px-3 py-2 text-sm',
    'placeholder:text-muted-foreground',
    'focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background'
  )
  const labelClass = 'block text-sm font-medium text-foreground mb-1.5'

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div>
        <label className={labelClass}>Username</label>
        <input
          {...register('username', { required: 'Username is required' })}
          placeholder="mobile-app-ios"
          className={inputClass}
        />
        {errors.username && <p className="text-xs text-destructive mt-1">{errors.username.message}</p>}
      </div>
      <div>
        <label className={labelClass}>Email <span className="text-muted-foreground font-normal">(optional)</span></label>
        <input {...register('email')} placeholder="team@example.com" type="email" className={inputClass} />
      </div>
      <div>
        <label className={labelClass}>Custom ID <span className="text-muted-foreground font-normal">(optional)</span></label>
        <input {...register('custom_id')} placeholder="ext-12345" className={inputClass} />
      </div>
      <button
        type="submit"
        disabled={isLoading}
        className={cn(
          'w-full rounded-md px-4 py-2 text-sm font-medium',
          'bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors'
        )}
      >
        {isLoading ? 'Saving...' : initial ? 'Update Consumer' : 'Create Consumer'}
      </button>
    </form>
  )
}
