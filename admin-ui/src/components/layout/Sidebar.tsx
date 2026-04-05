import { NavLink } from 'react-router-dom'
import { LayoutDashboard, Server, Route, Users, Puzzle, Zap } from 'lucide-react'
import { cn } from '../../lib/cn'
import { ThemeToggle } from '../shared/ThemeToggle'
import { useAdminHealth } from '../../hooks/useHealth'

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/services', icon: Server, label: 'Services' },
  { to: '/routes', icon: Route, label: 'Routes' },
  { to: '/consumers', icon: Users, label: 'Consumers' },
  { to: '/plugins', icon: Puzzle, label: 'Plugins' },
]

export function Sidebar() {
  const { data: health } = useAdminHealth()
  const isHealthy = health?.status === 'healthy'

  return (
    <aside className="flex h-screen w-60 flex-col border-r bg-background">
      {/* Logo */}
      <div className="flex h-14 items-center gap-2 border-b px-4">
        <Zap className="h-5 w-5 text-foreground" />
        <span className="text-sm font-semibold text-foreground">Switchboard</span>
      </div>

      {/* Navigation */}
      <nav className="flex-1 space-y-1 p-3">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/'}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-accent text-accent-foreground'
                  : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
              )
            }
          >
            <item.icon className="h-4 w-4" />
            {item.label}
          </NavLink>
        ))}
      </nav>

      {/* Footer */}
      <div className="border-t p-3 space-y-2">
        <ThemeToggle className="w-full justify-start gap-3 px-3 py-2" />
        <div className="flex items-center gap-2 px-3 py-1">
          <div
            className={cn(
              'h-2 w-2 rounded-full',
              isHealthy ? 'bg-emerald-500' : 'bg-red-500'
            )}
          />
          <span className="text-xs text-muted-foreground">
            {isHealthy ? 'Healthy' : 'Unhealthy'}
          </span>
        </div>
      </div>
    </aside>
  )
}
