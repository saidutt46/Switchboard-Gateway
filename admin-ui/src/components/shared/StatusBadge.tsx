import { cn } from '../../lib/cn'

export function StatusBadge({ enabled }: { enabled: boolean }) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
        enabled
          ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
          : 'bg-muted text-muted-foreground'
      )}
    >
      {enabled ? 'Enabled' : 'Disabled'}
    </span>
  )
}
