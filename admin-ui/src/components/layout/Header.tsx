import { ChevronRight } from 'lucide-react'
import { cn } from '../../lib/cn'

interface Breadcrumb {
  label: string
  href?: string
}

interface HeaderProps {
  breadcrumbs: Breadcrumb[]
  action?: React.ReactNode
}

export function Header({ breadcrumbs, action }: HeaderProps) {
  return (
    <header className="flex h-14 items-center justify-between border-b px-6">
      <nav className="flex items-center gap-1.5">
        {breadcrumbs.map((crumb, i) => (
          <span key={i} className="flex items-center gap-1.5">
            {i > 0 && <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />}
            <span
              className={cn(
                'text-sm',
                i === breadcrumbs.length - 1
                  ? 'font-medium text-foreground'
                  : 'text-muted-foreground'
              )}
            >
              {crumb.label}
            </span>
          </span>
        ))}
      </nav>
      {action && <div>{action}</div>}
    </header>
  )
}
