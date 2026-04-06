import { METHOD_COLORS } from '../../lib/constants'
import { cn } from '../../lib/cn'

export function MethodBadge({ method }: { method: string }) {
  const color = METHOD_COLORS[method.toUpperCase()] || METHOD_COLORS.HEAD
  return (
    <span className={cn('inline-flex rounded px-1.5 py-0.5 text-xs font-mono font-medium', color)}>
      {method}
    </span>
  )
}
