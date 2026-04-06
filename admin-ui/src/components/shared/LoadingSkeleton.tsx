import { cn } from '../../lib/cn'

export function LoadingSkeleton({ rows = 5, cols = 4 }: { rows?: number; cols?: number }) {
  return (
    <div className="animate-pulse space-y-3">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex gap-4">
          {Array.from({ length: cols }).map((_, j) => (
            <div
              key={j}
              className={cn('h-4 rounded bg-muted', j === 0 ? 'w-32' : 'w-24')}
            />
          ))}
        </div>
      ))}
    </div>
  )
}
