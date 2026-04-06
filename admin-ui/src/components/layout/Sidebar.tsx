import { NavLink } from 'react-router-dom'
import { LayoutDashboard, Server, Route, Users, Puzzle, Zap, CircleDot } from 'lucide-react'
import { cn } from '../../lib/cn'
import { ThemeToggle } from '../shared/ThemeToggle'
import { useAdminHealth, useGatewayHealth } from '../../hooks/useHealth'

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard', color: 'text-slate-400' },
  { to: '/services', icon: Server, label: 'Services', color: 'text-blue-400' },
  { to: '/routes', icon: Route, label: 'Routes', color: 'text-teal-400' },
  { to: '/consumers', icon: Users, label: 'Consumers', color: 'text-amber-400' },
  { to: '/plugins', icon: Puzzle, label: 'Plugins', color: 'text-violet-400' },
]

export function Sidebar() {
  const { data: adminHealth } = useAdminHealth()
  const { data: gatewayHealth } = useGatewayHealth()
  const isHealthy = adminHealth?.status === 'healthy'
  const gatewayUp = gatewayHealth?.status === 'healthy'

  return (
    <aside className="flex h-screen w-60 flex-col bg-sidebar">
      {/* Logo */}
      <div className="flex h-14 items-center gap-2.5 px-5 border-b border-white/[0.08]">
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-gradient-to-br from-blue-500 to-violet-600">
          <Zap className="h-3.5 w-3.5 text-white" />
        </div>
        <span className="text-sm font-semibold text-white tracking-tight">Switchboard</span>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-3 py-4 space-y-0.5">
        <p className="px-3 pb-2 text-[10px] font-semibold uppercase tracking-[0.08em] text-sidebar-foreground/50">
          Navigation
        </p>
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/'}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 rounded-lg px-3 py-2 text-[13px] font-medium transition-all duration-150',
                isActive
                  ? 'bg-white/[0.08] text-white shadow-sm'
                  : 'text-sidebar-foreground hover:bg-white/[0.05] hover:text-white'
              )
            }
          >
            {({ isActive }) => (
              <>
                <item.icon className={cn('h-4 w-4', isActive ? item.color : 'text-sidebar-foreground')} />
                {item.label}
              </>
            )}
          </NavLink>
        ))}
      </nav>

      {/* Footer */}
      <div className="border-t border-white/[0.08] p-3 space-y-1">
        <ThemeToggle className="w-full justify-start gap-3 px-3 py-2 text-sidebar-foreground hover:text-white hover:bg-white/[0.05] rounded-lg" />
        <div className="flex flex-col gap-1.5 px-3 py-2">
          <div className="flex items-center gap-2">
            <CircleDot className={cn('h-3 w-3', gatewayUp ? 'text-emerald-400' : 'text-red-400')} />
            <span className="text-[11px] text-sidebar-foreground">
              Gateway {gatewayUp ? 'online' : 'offline'}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <CircleDot className={cn('h-3 w-3', isHealthy ? 'text-emerald-400' : 'text-red-400')} />
            <span className="text-[11px] text-sidebar-foreground">
              API {isHealthy ? 'healthy' : 'unhealthy'}
            </span>
          </div>
        </div>
      </div>
    </aside>
  )
}
