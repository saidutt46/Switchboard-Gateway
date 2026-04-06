import { useMemo } from 'react'
import { Header } from '../components/layout/Header'
import { useAdminHealth, useGatewayHealth } from '../hooks/useHealth'
import { useServices } from '../hooks/useServices'
import { useRoutes } from '../hooks/useRoutes'
import { useConsumers } from '../hooks/useConsumers'
import { usePlugins } from '../hooks/usePlugins'
import { cn } from '../lib/cn'
import {
  Server, Route, Users, Puzzle, Activity, Database, HardDrive,
  RefreshCw, ArrowUpRight, ShieldCheck, Gauge, Layers,
} from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import {
  PieChart, Pie, Cell, BarChart, Bar, XAxis, YAxis, Tooltip,
  ResponsiveContainer, RadialBarChart, RadialBar, Legend,
  CartesianGrid,
} from 'recharts'

/* ─── Color palette for charts ─── */
const COLORS = {
  blue: '#3b82f6',
  teal: '#14b8a6',
  amber: '#f59e0b',
  violet: '#8b5cf6',
  rose: '#f43f5e',
  emerald: '#10b981',
  indigo: '#6366f1',
  cyan: '#06b6d4',
  orange: '#f97316',
  slate: '#64748b',
}

const SCOPE_COLORS: Record<string, string> = {
  global: COLORS.blue,
  service: COLORS.teal,
  route: COLORS.amber,
  consumer: COLORS.violet,
}

const METHOD_CHART_COLORS: Record<string, string> = {
  GET: COLORS.emerald,
  POST: COLORS.blue,
  PUT: COLORS.amber,
  DELETE: COLORS.rose,
  PATCH: COLORS.violet,
  HEAD: COLORS.slate,
  OPTIONS: COLORS.cyan,
}

const PLUGIN_TYPE_COLORS: Record<string, string> = {
  'api-key-auth': COLORS.blue,
  'jwt-auth': COLORS.indigo,
  'rate-limit': COLORS.amber,
  'cache': COLORS.teal,
  'cors': COLORS.violet,
  'request-logger': COLORS.cyan,
}

/* ─── Tooltip component ─── */
const tooltipStyle = {
  backgroundColor: 'hsl(222 40% 9%)',
  border: '1px solid hsl(223 30% 20%)',
  borderRadius: 8,
  fontSize: 12,
  color: '#e2e8f0',
  boxShadow: '0 8px 24px rgba(0,0,0,0.3)',
}

/* ─── Health Card ─── */
function HealthCard({
  label, status, icon: Icon, detail, accentColor,
}: {
  label: string; status: string; icon: React.ElementType; detail?: string; accentColor: string
}) {
  const isHealthy = status === 'healthy' || status === 'pass' || status === 'operational'
  return (
    <div className="rounded-xl border bg-card p-4 relative overflow-hidden group hover:shadow-md transition-shadow duration-200">
      <div className="absolute top-0 right-0 w-24 h-24 rounded-full opacity-[0.04] -translate-y-8 translate-x-8" style={{ backgroundColor: accentColor }} />
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl" style={{ backgroundColor: `${accentColor}15` }}>
          <Icon className="h-4.5 w-4.5" style={{ color: accentColor }} />
        </div>
        <div>
          <p className="text-sm font-semibold text-foreground">{label}</p>
          <div className="flex items-center gap-1.5 mt-0.5">
            <div className={cn('h-1.5 w-1.5 rounded-full', isHealthy ? 'bg-emerald-500' : 'bg-red-500')} />
            <p className={cn('text-xs font-medium', isHealthy ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400')}>
              {status}
            </p>
          </div>
        </div>
      </div>
      {detail && <p className="mt-3 text-[11px] text-muted-foreground font-mono">{detail}</p>}
    </div>
  )
}

/* ─── Count Card ─── */
function CountCard({
  label, count, icon: Icon, href, accentColor, description,
}: {
  label: string; count: number | undefined; icon: React.ElementType; href: string; accentColor: string; description: string
}) {
  const navigate = useNavigate()
  return (
    <button
      onClick={() => navigate(href)}
      className="rounded-xl border bg-card p-5 text-left transition-all duration-200 hover:shadow-md hover:-translate-y-0.5 w-full group relative overflow-hidden"
    >
      <div className="absolute bottom-0 left-0 right-0 h-[2px] opacity-0 group-hover:opacity-100 transition-opacity duration-200" style={{ backgroundColor: accentColor }} />
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">{label}</p>
          <p className="mt-2 text-3xl font-bold text-foreground tracking-tight">
            {count !== undefined ? count : '—'}
          </p>
          <p className="mt-1 text-[11px] text-muted-foreground">{description}</p>
        </div>
        <div className="flex h-9 w-9 items-center justify-center rounded-lg" style={{ backgroundColor: `${accentColor}12` }}>
          <Icon className="h-4 w-4" style={{ color: accentColor }} />
        </div>
      </div>
      <div className="mt-3 flex items-center gap-1 text-[11px] font-medium" style={{ color: accentColor }}>
        <span>View all</span>
        <ArrowUpRight className="h-3 w-3" />
      </div>
    </button>
  )
}

/* ─── Chart Card ─── */
function ChartCard({ title, icon: Icon, children, className }: {
  title: string; icon?: React.ElementType; children: React.ReactNode; className?: string
}) {
  return (
    <div className={cn('rounded-xl border bg-card overflow-hidden', className)}>
      <div className="flex items-center gap-2 border-b px-5 py-3.5">
        {Icon && <Icon className="h-3.5 w-3.5 text-muted-foreground" />}
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">{title}</h3>
      </div>
      <div className="p-5">{children}</div>
    </div>
  )
}

/* ─── Dashboard Page ─── */
export function DashboardPage() {
  const queryClient = useQueryClient()
  const { data: adminHealth } = useAdminHealth()
  const { data: gatewayHealth } = useGatewayHealth()
  const { data: services } = useServices()
  const { data: routes } = useRoutes()
  const { data: consumers } = useConsumers()
  const { data: plugins } = usePlugins()

  /* Plugin by scope for donut chart */
  const pluginsByScope = useMemo(() => {
    if (!plugins) return []
    const counts: Record<string, number> = {}
    plugins.forEach((p) => { counts[p.scope] = (counts[p.scope] || 0) + 1 })
    return Object.entries(counts).map(([name, value]) => ({ name, value, fill: SCOPE_COLORS[name] || COLORS.slate }))
  }, [plugins])

  /* Plugin by type for bar chart */
  const pluginsByType = useMemo(() => {
    if (!plugins) return []
    const counts: Record<string, number> = {}
    plugins.forEach((p) => { counts[p.name] = (counts[p.name] || 0) + 1 })
    return Object.entries(counts)
      .map(([name, value]) => ({ name, value, fill: PLUGIN_TYPE_COLORS[name] || COLORS.slate }))
      .sort((a, b) => b.value - a.value)
  }, [plugins])

  /* HTTP method distribution */
  const methodDistribution = useMemo(() => {
    if (!routes) return []
    const counts: Record<string, number> = {}
    routes.forEach((r) => r.methods.forEach((m) => { counts[m] = (counts[m] || 0) + 1 }))
    return Object.entries(counts)
      .map(([name, value]) => ({ name, value, fill: METHOD_CHART_COLORS[name] || COLORS.slate }))
      .sort((a, b) => b.value - a.value)
  }, [routes])

  /* Enabled/disabled radial gauge */
  const healthGauge = useMemo(() => {
    if (!services || !routes || !plugins) return []
    const total = services.length + routes.length + plugins.length
    const enabled = services.filter(s => s.enabled).length + routes.filter(r => r.enabled).length + plugins.filter(p => p.enabled).length
    const pct = total > 0 ? Math.round((enabled / total) * 100) : 0
    return [
      { name: 'Active', value: pct, fill: COLORS.emerald },
    ]
  }, [services, routes, plugins])

  /* Gateway config summary for area chart placeholder */
  const configTimeline = useMemo(() => {
    const counts = [
      { name: 'Services', value: services?.length || 0, fill: COLORS.blue },
      { name: 'Routes', value: routes?.length || 0, fill: COLORS.teal },
      { name: 'Consumers', value: consumers?.length || 0, fill: COLORS.amber },
      { name: 'Plugins', value: plugins?.length || 0, fill: COLORS.violet },
    ]
    return counts
  }, [services, routes, consumers, plugins])

  const handleRefresh = () => { queryClient.invalidateQueries() }

  const hasData = (services?.length || 0) + (routes?.length || 0) + (plugins?.length || 0) > 0
  const totalEntities = (services?.length || 0) + (routes?.length || 0) + (consumers?.length || 0) + (plugins?.length || 0)

  return (
    <div>
      <Header breadcrumbs={[{ label: 'Dashboard' }]} action={
        <button onClick={handleRefresh} className={cn(
          'inline-flex items-center gap-2 rounded-lg border px-3.5 py-1.5 text-sm font-medium',
          'text-foreground hover:bg-accent transition-all duration-150'
        )}>
          <RefreshCw className="h-3.5 w-3.5" /> Refresh
        </button>
      } />

      <div className="mx-auto max-w-6xl p-6 space-y-8">

        {/* ─── System Health ─── */}
        <section>
          <div className="flex items-center gap-2 mb-4">
            <Activity className="h-4 w-4 text-muted-foreground" />
            <h2 className="text-sm font-semibold text-foreground">System Health</h2>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <HealthCard label="Gateway" status={gatewayHealth?.status || 'unknown'} icon={Activity}
              detail={gatewayHealth?.uptime ? `Uptime: ${gatewayHealth.uptime}` : undefined}
              accentColor={COLORS.blue} />
            <HealthCard label="Database" status={adminHealth?.database || 'unknown'} icon={Database}
              accentColor={COLORS.teal} />
            <HealthCard label="Redis" status={adminHealth?.redis || 'unknown'} icon={HardDrive}
              accentColor={COLORS.violet} />
          </div>
        </section>

        {/* ─── Entity Overview ─── */}
        <section>
          <div className="flex items-center gap-2 mb-4">
            <Layers className="h-4 w-4 text-muted-foreground" />
            <h2 className="text-sm font-semibold text-foreground">Gateway Overview</h2>
            <span className="ml-auto text-xs text-muted-foreground font-mono">{totalEntities} total entities</span>
          </div>
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <CountCard label="Services" count={services?.length} icon={Server} href="/services"
              accentColor={COLORS.blue} description="Backend targets" />
            <CountCard label="Routes" count={routes?.length} icon={Route} href="/routes"
              accentColor={COLORS.teal} description="Path mappings" />
            <CountCard label="Consumers" count={consumers?.length} icon={Users} href="/consumers"
              accentColor={COLORS.amber} description="API clients" />
            <CountCard label="Plugins" count={plugins?.length} icon={Puzzle} href="/plugins"
              accentColor={COLORS.violet} description="Active middleware" />
          </div>
        </section>

        {/* ─── Analytics Charts ─── */}
        {hasData && (
          <section>
            <div className="flex items-center gap-2 mb-4">
              <Gauge className="h-4 w-4 text-muted-foreground" />
              <h2 className="text-sm font-semibold text-foreground">Analytics</h2>
            </div>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">

              {/* HTTP Methods — colorful bar chart */}
              {methodDistribution.length > 0 && (
                <ChartCard title="HTTP Methods" icon={Route}>
                  <ResponsiveContainer width="100%" height={220}>
                    <BarChart data={methodDistribution} margin={{ left: -10, right: 8, top: 8, bottom: 0 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" vertical={false} />
                      <XAxis dataKey="name" tick={{ fontSize: 11, fontFamily: 'JetBrains Mono' }} stroke="hsl(var(--muted-foreground))" tickLine={false} axisLine={false} />
                      <YAxis allowDecimals={false} tick={{ fontSize: 10 }} stroke="hsl(var(--muted-foreground))" tickLine={false} axisLine={false} />
                      <Tooltip cursor={{ fill: 'hsl(var(--muted))', radius: 4 }} contentStyle={tooltipStyle} />
                      <Bar dataKey="value" radius={[6, 6, 0, 0]} barSize={36}>
                        {methodDistribution.map((e, j) => (
                          <Cell key={j} fill={e.fill} />
                        ))}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                </ChartCard>
              )}

              {/* Plugin Scope — donut chart */}
              {pluginsByScope.length > 0 && (
                <ChartCard title="Plugin Scopes" icon={ShieldCheck}>
                  <ResponsiveContainer width="100%" height={220}>
                    <PieChart>
                      <Pie
                        data={pluginsByScope}
                        dataKey="value"
                        nameKey="name"
                        cx="50%"
                        cy="50%"
                        innerRadius={55}
                        outerRadius={85}
                        paddingAngle={3}
                        stroke="none"
                      >
                        {pluginsByScope.map((entry, i) => (
                          <Cell key={i} fill={entry.fill} />
                        ))}
                      </Pie>
                      <Tooltip contentStyle={tooltipStyle} />
                      <Legend
                        verticalAlign="bottom"
                        iconType="circle"
                        iconSize={8}
                        wrapperStyle={{ fontSize: 11, paddingTop: 12 }}
                      />
                    </PieChart>
                  </ResponsiveContainer>
                </ChartCard>
              )}

              {/* System Health Gauge */}
              {healthGauge.length > 0 && healthGauge[0].value > 0 && (
                <ChartCard title="Active Rate" icon={Activity}>
                  <ResponsiveContainer width="100%" height={220}>
                    <RadialBarChart
                      cx="50%"
                      cy="50%"
                      innerRadius="60%"
                      outerRadius="90%"
                      data={healthGauge}
                      startAngle={180}
                      endAngle={0}
                    >
                      <RadialBar
                        dataKey="value"
                        cornerRadius={12}
                        background={{ fill: 'hsl(var(--muted))' }}
                      />
                    </RadialBarChart>
                  </ResponsiveContainer>
                  <div className="text-center -mt-16 pb-2">
                    <p className="text-3xl font-bold text-foreground">{healthGauge[0].value}%</p>
                    <p className="text-xs text-muted-foreground mt-0.5">entities enabled</p>
                  </div>
                </ChartCard>
              )}

              {/* Plugins by Type */}
              {pluginsByType.length > 0 && (
                <ChartCard title="Plugin Types" icon={Puzzle} className="md:col-span-2 lg:col-span-2">
                  <ResponsiveContainer width="100%" height={200}>
                    <BarChart data={pluginsByType} layout="vertical" margin={{ left: 0, right: 24, top: 4, bottom: 4 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" horizontal={false} />
                      <XAxis type="number" allowDecimals={false} tick={{ fontSize: 10 }} stroke="hsl(var(--muted-foreground))" tickLine={false} axisLine={false} />
                      <YAxis type="category" dataKey="name" tick={{ fontSize: 11, fontFamily: 'JetBrains Mono' }} stroke="hsl(var(--muted-foreground))" width={110} tickLine={false} axisLine={false} />
                      <Tooltip cursor={{ fill: 'hsl(var(--muted))', radius: 4 }} contentStyle={tooltipStyle} />
                      <Bar dataKey="value" radius={[0, 6, 6, 0]} barSize={22}>
                        {pluginsByType.map((entry, i) => (
                          <Cell key={i} fill={entry.fill} />
                        ))}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                </ChartCard>
              )}

              {/* Entity Composition — stacked area */}
              {configTimeline.length > 0 && (
                <ChartCard title="Configuration Summary" icon={Layers}>
                  <div className="space-y-3">
                    {configTimeline.map((item) => (
                      <div key={item.name} className="flex items-center gap-3">
                        <div className="w-20 text-xs text-muted-foreground font-medium">{item.name}</div>
                        <div className="flex-1 h-6 bg-muted/50 rounded-md overflow-hidden relative">
                          <div
                            className="h-full rounded-md transition-all duration-500"
                            style={{
                              width: `${Math.max(8, (item.value / Math.max(totalEntities, 1)) * 100)}%`,
                              backgroundColor: item.fill,
                              opacity: 0.8,
                            }}
                          />
                        </div>
                        <span className="w-6 text-right text-sm font-bold text-foreground">{item.value}</span>
                      </div>
                    ))}
                  </div>
                </ChartCard>
              )}
            </div>
          </section>
        )}

        {/* ─── Footer ─── */}
        {adminHealth?.version && (
          <div className="flex items-center justify-between rounded-xl border bg-card px-5 py-3">
            <p className="text-xs text-muted-foreground">
              Admin API <span className="font-mono font-medium text-foreground">v{adminHealth.version}</span>
            </p>
            <p className="text-[10px] text-muted-foreground/50 font-mono">
              Switchboard Gateway
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
