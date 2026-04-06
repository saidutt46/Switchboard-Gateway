import { NavLink } from 'react-router-dom'
import { LayoutDashboard, Server, Route, Users, Puzzle, CircleDot, PanelLeftClose, PanelLeftOpen, LogOut } from 'lucide-react'
import { cn } from '../../lib/cn'
import { ThemeToggle } from '../shared/ThemeToggle'
import { useAdminHealth, useGatewayHealth } from '../../hooks/useHealth'
import { useSidebar } from '../../hooks/useSidebar'
import { useAuth } from '../../hooks/useAuth'

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard', color: 'text-slate-400' },
  { to: '/services', icon: Server, label: 'Services', color: 'text-blue-400' },
  { to: '/routes', icon: Route, label: 'Routes', color: 'text-teal-400' },
  { to: '/consumers', icon: Users, label: 'Consumers', color: 'text-amber-400' },
  { to: '/plugins', icon: Puzzle, label: 'Plugins', color: 'text-violet-400' },
]

function SwitchboardIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" className={className} stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      {/* Central hub */}
      <circle cx="12" cy="12" r="3" fill="currentColor" opacity="0.3" />
      <circle cx="12" cy="12" r="3" />
      {/* Nodes */}
      <circle cx="4" cy="6" r="1.5" />
      <circle cx="4" cy="18" r="1.5" />
      <circle cx="20" cy="6" r="1.5" />
      <circle cx="20" cy="18" r="1.5" />
      {/* Connections */}
      <line x1="5.5" y1="6" x2="9" y2="10.5" />
      <line x1="5.5" y1="18" x2="9" y2="13.5" />
      <line x1="15" y1="10.5" x2="18.5" y2="6" />
      <line x1="15" y1="13.5" x2="18.5" y2="18" />
    </svg>
  )
}

export function Sidebar() {
  const { collapsed, toggle } = useSidebar()
  const { user, logout } = useAuth()
  const { data: adminHealth } = useAdminHealth()
  const { data: gatewayHealth } = useGatewayHealth()
  const isHealthy = adminHealth?.status === 'healthy'
  const gatewayUp = gatewayHealth?.status === 'healthy'

  return (
    <aside className={cn(
      'flex h-screen flex-col bg-sidebar transition-all duration-200',
      collapsed ? 'w-16' : 'w-60'
    )}>
      {/* Logo */}
      <div className={cn('flex h-14 items-center border-b border-white/[0.06]', collapsed ? 'justify-center px-2' : 'gap-2.5 px-4')}>
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-emerald-500 to-teal-600">
          <SwitchboardIcon className="h-4 w-4 text-white" />
        </div>
        {!collapsed && <span className="text-sm font-semibold text-white tracking-tight">Switchboard</span>}
      </div>

      {/* Navigation */}
      <nav className={cn('flex-1 py-4 space-y-0.5', collapsed ? 'px-2' : 'px-3')}>
        {!collapsed && (
          <p className="px-3 pb-2 text-[10px] font-semibold uppercase tracking-[0.08em] text-sidebar-foreground/50">
            Navigation
          </p>
        )}
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/'}
            title={collapsed ? item.label : undefined}
            className={({ isActive }) =>
              cn(
                'flex items-center rounded-lg text-[13px] font-medium transition-all duration-150',
                collapsed ? 'justify-center p-2.5' : 'gap-3 px-3 py-2',
                isActive
                  ? 'bg-white/[0.1] text-white'
                  : 'text-sidebar-foreground hover:bg-white/[0.06] hover:text-white'
              )
            }
          >
            {({ isActive }) => (
              <>
                <item.icon className={cn('h-4 w-4 shrink-0', isActive ? item.color : 'text-sidebar-foreground')} />
                {!collapsed && item.label}
              </>
            )}
          </NavLink>
        ))}
      </nav>

      {/* Footer */}
      <div className={cn('border-t border-white/[0.06] py-2', collapsed ? 'px-2' : 'px-3')}>
        {/* Collapse toggle */}
        <button
          onClick={toggle}
          title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          className={cn(
            'flex items-center rounded-lg text-sidebar-foreground hover:text-white hover:bg-white/[0.06] transition-colors w-full',
            collapsed ? 'justify-center p-2.5' : 'gap-3 px-3 py-2'
          )}
        >
          {collapsed ? <PanelLeftOpen className="h-4 w-4" /> : <PanelLeftClose className="h-4 w-4" />}
          {!collapsed && <span className="text-[13px] font-medium">Collapse</span>}
        </button>

        {/* Theme toggle */}
        <ThemeToggle className={cn(
          'w-full rounded-lg text-sidebar-foreground hover:text-white hover:bg-white/[0.06]',
          collapsed ? 'justify-center p-2.5' : 'justify-start gap-3 px-3 py-2'
        )} />

        {/* Health */}
        <div className={cn('py-2', collapsed ? 'flex flex-col items-center gap-1.5' : 'flex flex-col gap-1 px-3')}>
          <div className={cn('flex items-center', collapsed ? 'justify-center' : 'gap-2')}>
            <CircleDot className={cn('h-3 w-3 shrink-0', gatewayUp ? 'text-emerald-400' : 'text-red-400')} />
            {!collapsed && (
              <span className="text-[11px] text-sidebar-foreground">
                Gateway {gatewayUp ? 'online' : 'offline'}
              </span>
            )}
          </div>
          <div className={cn('flex items-center', collapsed ? 'justify-center' : 'gap-2')}>
            <CircleDot className={cn('h-3 w-3 shrink-0', isHealthy ? 'text-emerald-400' : 'text-red-400')} />
            {!collapsed && (
              <span className="text-[11px] text-sidebar-foreground">
                API {isHealthy ? 'healthy' : 'unhealthy'}
              </span>
            )}
          </div>
        </div>

        {/* User + Logout */}
        <div className={cn('border-t border-white/[0.06] pt-2 mt-1', collapsed ? '' : 'px-1')}>
          {!collapsed && user && (
            <div className="px-2 py-1.5 mb-1">
              <p className="text-[12px] font-medium text-white truncate">{user.username}</p>
              <p className="text-[10px] text-sidebar-foreground">{user.role}</p>
            </div>
          )}
          <button
            onClick={logout}
            title="Sign out"
            className={cn(
              'flex items-center rounded-lg text-sidebar-foreground hover:text-red-400 hover:bg-red-500/10 transition-colors w-full',
              collapsed ? 'justify-center p-2.5' : 'gap-3 px-3 py-2'
            )}
          >
            <LogOut className="h-4 w-4 shrink-0" />
            {!collapsed && <span className="text-[13px] font-medium">Sign out</span>}
          </button>
        </div>
      </div>
    </aside>
  )
}
