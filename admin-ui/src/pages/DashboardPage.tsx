import { useMemo } from 'react'
import { Header } from '../components/layout/Header'
import { useAdminHealth, useGatewayHealth } from '../hooks/useHealth'
import { useServices } from '../hooks/useServices'
import { useRoutes } from '../hooks/useRoutes'
import { useConsumers } from '../hooks/useConsumers'
import { usePlugins } from '../hooks/usePlugins'
import { cn } from '../lib/cn'
import { Server, Route, Users, Puzzle, Activity, Database, HardDrive, RefreshCw } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { PieChart, Pie, Cell, BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts'

const CHART_COLORS = ['#737373', '#a3a3a3', '#525252', '#d4d4d4', '#404040', '#e5e5e5']

function HealthCard({ label, status, icon: Icon, detail }: { label: string; status: string; icon: React.ElementType; detail?: string }) {
  const isHealthy = status === 'healthy' || status === 'pass' || status === 'operational'
  return (
    <div className="rounded-md border p-4">
      <div className="flex items-center gap-3">
        <div className={cn('flex h-9 w-9 items-center justify-center rounded-md', isHealthy ? 'bg-emerald-100 dark:bg-emerald-900/30' : 'bg-red-100 dark:bg-red-900/30')}>
          <Icon className={cn('h-4 w-4', isHealthy ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400')} />
        </div>
        <div>
          <p className="text-sm font-medium text-foreground">{label}</p>
          <p className={cn('text-xs', isHealthy ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400')}>{status}</p>
        </div>
      </div>
      {detail && <p className="mt-2 text-xs text-muted-foreground">{detail}</p>}
    </div>
  )
}

function CountCard({ label, count, icon: Icon, href }: { label: string; count: number | undefined; icon: React.ElementType; href: string }) {
  const navigate = useNavigate()
  return (
    <button onClick={() => navigate(href)} className="rounded-md border p-4 text-left transition-colors hover:bg-muted/50 w-full">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs text-muted-foreground">{label}</p>
          <p className="mt-1 text-2xl font-semibold text-foreground">{count !== undefined ? count : '—'}</p>
        </div>
        <Icon className="h-5 w-5 text-muted-foreground" />
      </div>
    </button>
  )
}

function ChartCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-md border">
      <div className="border-b px-4 py-2.5">
        <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{title}</h3>
      </div>
      <div className="p-4">{children}</div>
    </div>
  )
}

export function DashboardPage() {
  const queryClient = useQueryClient()
  const { data: adminHealth } = useAdminHealth()
  const { data: gatewayHealth } = useGatewayHealth()
  const { data: services } = useServices()
  const { data: routes } = useRoutes()
  const { data: consumers } = useConsumers()
  const { data: plugins } = usePlugins()

  const pluginsByScope = useMemo(() => {
    if (!plugins) return []
    const counts: Record<string, number> = {}
    plugins.forEach((p) => { counts[p.scope] = (counts[p.scope] || 0) + 1 })
    return Object.entries(counts).map(([name, value]) => ({ name, value }))
  }, [plugins])

  const pluginsByType = useMemo(() => {
    if (!plugins) return []
    const counts: Record<string, number> = {}
    plugins.forEach((p) => { counts[p.name] = (counts[p.name] || 0) + 1 })
    return Object.entries(counts).map(([name, value]) => ({ name, value })).sort((a, b) => b.value - a.value)
  }, [plugins])

  const methodDistribution = useMemo(() => {
    if (!routes) return []
    const counts: Record<string, number> = {}
    routes.forEach((r) => r.methods.forEach((m) => { counts[m] = (counts[m] || 0) + 1 }))
    return Object.entries(counts).map(([name, value]) => ({ name, value })).sort((a, b) => b.value - a.value)
  }, [routes])

  const enabledBreakdown = useMemo(() => {
    if (!services || !routes) return []
    const sEnabled = services.filter((s) => s.enabled).length
    const sDisabled = services.length - sEnabled
    const rEnabled = routes.filter((r) => r.enabled).length
    const rDisabled = routes.length - rEnabled
    return [
      { name: 'Services', enabled: sEnabled, disabled: sDisabled },
      { name: 'Routes', enabled: rEnabled, disabled: rDisabled },
    ]
  }, [services, routes])

  const protocolBreakdown = useMemo(() => {
    if (!services) return []
    const counts: Record<string, number> = {}
    services.forEach((s) => { counts[s.protocol] = (counts[s.protocol] || 0) + 1 })
    return Object.entries(counts).map(([name, value]) => ({ name, value }))
  }, [services])

  const handleRefresh = () => {
    queryClient.invalidateQueries()
  }

  const hasData = (services?.length || 0) + (routes?.length || 0) + (plugins?.length || 0) > 0

  return (
    <div>
      <Header breadcrumbs={[{ label: 'Dashboard' }]} action={
        <button onClick={handleRefresh} className={cn('inline-flex items-center gap-2 rounded-md border px-3 py-1.5 text-sm font-medium text-foreground hover:bg-accent transition-colors')}>
          <RefreshCw className="h-3.5 w-3.5" /> Refresh
        </button>
      } />

      <div className="mx-auto max-w-5xl p-6 space-y-6">
        {/* Health */}
        <div>
          <h2 className="text-sm font-medium text-foreground mb-3">System Health</h2>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <HealthCard label="Gateway" status={gatewayHealth?.status || 'unknown'} icon={Activity}
              detail={gatewayHealth?.uptime ? `Uptime: ${gatewayHealth.uptime}` : undefined} />
            <HealthCard label="Database" status={adminHealth?.database || 'unknown'} icon={Database} />
            <HealthCard label="Redis" status={adminHealth?.redis || 'unknown'} icon={HardDrive} />
          </div>
        </div>

        {/* Counts */}
        <div>
          <h2 className="text-sm font-medium text-foreground mb-3">Overview</h2>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <CountCard label="Services" count={services?.length} icon={Server} href="/services" />
            <CountCard label="Routes" count={routes?.length} icon={Route} href="/routes" />
            <CountCard label="Consumers" count={consumers?.length} icon={Users} href="/consumers" />
            <CountCard label="Plugins" count={plugins?.length} icon={Puzzle} href="/plugins" />
          </div>
        </div>

        {/* Charts */}
        {hasData && (
          <div>
            <h2 className="text-sm font-medium text-foreground mb-3">Analytics</h2>
            <div className="grid grid-cols-1 gap-3 md:grid-cols-2">

              {/* Plugin Distribution by Type */}
              {pluginsByType.length > 0 && (
                <ChartCard title="Plugins by Type">
                  <ResponsiveContainer width="100%" height={200}>
                    <BarChart data={pluginsByType} layout="vertical" margin={{ left: 0, right: 16 }}>
                      <XAxis type="number" allowDecimals={false} tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
                      <YAxis type="category" dataKey="name" tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" width={100} />
                      <Tooltip contentStyle={{ backgroundColor: 'hsl(var(--card))', border: '1px solid hsl(var(--border))', borderRadius: 6, fontSize: 12 }} />
                      <Bar dataKey="value" fill="hsl(var(--foreground))" radius={[0, 4, 4, 0]} barSize={20} />
                    </BarChart>
                  </ResponsiveContainer>
                </ChartCard>
              )}

              {/* Plugin Scope Distribution */}
              {pluginsByScope.length > 0 && (
                <ChartCard title="Plugins by Scope">
                  <div className="flex items-center justify-center">
                    <ResponsiveContainer width="100%" height={200}>
                      <PieChart>
                        <Pie data={pluginsByScope} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={50} outerRadius={80}
                          stroke="hsl(var(--background))" strokeWidth={2} label={({ name, value }) => `${name} (${value})`} labelLine={false}>
                          {pluginsByScope.map((_entry, i) => (
                            <Cell key={i} fill={CHART_COLORS[i % CHART_COLORS.length]} />
                          ))}
                        </Pie>
                        <Tooltip contentStyle={{ backgroundColor: 'hsl(var(--card))', border: '1px solid hsl(var(--border))', borderRadius: 6, fontSize: 12 }} />
                      </PieChart>
                    </ResponsiveContainer>
                  </div>
                </ChartCard>
              )}

              {/* HTTP Methods */}
              {methodDistribution.length > 0 && (
                <ChartCard title="Route Methods Distribution">
                  <ResponsiveContainer width="100%" height={200}>
                    <BarChart data={methodDistribution} margin={{ left: 0, right: 16 }}>
                      <XAxis dataKey="name" tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
                      <YAxis allowDecimals={false} tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
                      <Tooltip contentStyle={{ backgroundColor: 'hsl(var(--card))', border: '1px solid hsl(var(--border))', borderRadius: 6, fontSize: 12 }} />
                      <Bar dataKey="value" fill="hsl(var(--foreground))" radius={[4, 4, 0, 0]} barSize={32} />
                    </BarChart>
                  </ResponsiveContainer>
                </ChartCard>
              )}

              {/* Enabled/Disabled Breakdown */}
              {enabledBreakdown.length > 0 && (
                <ChartCard title="Enabled vs Disabled">
                  <ResponsiveContainer width="100%" height={200}>
                    <BarChart data={enabledBreakdown} margin={{ left: 0, right: 16 }}>
                      <XAxis dataKey="name" tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
                      <YAxis allowDecimals={false} tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
                      <Tooltip contentStyle={{ backgroundColor: 'hsl(var(--card))', border: '1px solid hsl(var(--border))', borderRadius: 6, fontSize: 12 }} />
                      <Bar dataKey="enabled" stackId="a" fill="#737373" name="Enabled" radius={[0, 0, 0, 0]} barSize={40} />
                      <Bar dataKey="disabled" stackId="a" fill="#d4d4d4" name="Disabled" radius={[4, 4, 0, 0]} barSize={40} />
                    </BarChart>
                  </ResponsiveContainer>
                </ChartCard>
              )}

              {/* Protocol Breakdown */}
              {protocolBreakdown.length > 0 && (
                <ChartCard title="Service Protocols">
                  <div className="space-y-2">
                    {protocolBreakdown.map((p) => (
                      <div key={p.name} className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <span className="font-mono text-sm text-foreground">{p.name}</span>
                        </div>
                        <div className="flex items-center gap-2">
                          <div className="h-2 rounded-full bg-foreground" style={{ width: `${Math.max(24, (p.value / (services?.length || 1)) * 120)}px` }} />
                          <span className="text-sm font-medium text-foreground">{p.value}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </ChartCard>
              )}
            </div>
          </div>
        )}

        {/* Version */}
        {adminHealth?.version && (
          <div className="rounded-md border p-4">
            <p className="text-xs text-muted-foreground">Admin API v{adminHealth.version}</p>
          </div>
        )}
      </div>
    </div>
  )
}
