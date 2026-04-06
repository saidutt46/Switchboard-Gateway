import { useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { Pencil, Trash2, ExternalLink, Plus, Route as RouteIcon, Puzzle } from 'lucide-react'
import { Header } from '../components/layout/Header'
import { StatusBadge } from '../components/shared/StatusBadge'
import { MethodBadge } from '../components/shared/MethodBadge'
import { SlidePanel } from '../components/shared/SlidePanel'
import { ConfirmDialog } from '../components/shared/ConfirmDialog'
import { ServiceForm } from '../components/services/ServiceForm'
import { RouteForm } from '../components/routes/RouteForm'
import { LoadingSkeleton } from '../components/shared/LoadingSkeleton'
import { useService, useServiceStats, useUpdateService, useDeleteService } from '../hooks/useServices'
import { useRoutes, useCreateRoute } from '../hooks/useRoutes'
import { usePlugins } from '../hooks/usePlugins'
import { useToast } from '../components/shared/Toast'
import { formatDate, shortId } from '../lib/formatters'
import { cn } from '../lib/cn'
import type { ServiceCreate, RouteCreate } from '../api/types'

const STAT_CONFIG = [
  { key: 'routes_count', label: 'Routes', color: '#14b8a6', icon: RouteIcon },
  { key: 'targets_count', label: 'Targets', color: '#3b82f6', icon: ExternalLink },
  { key: 'plugins_count', label: 'Plugins', color: '#8b5cf6', icon: Puzzle },
] as const

export function ServiceDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { toast } = useToast()
  const { data: service, isLoading } = useService(id!)
  const { data: stats } = useServiceStats(id!)
  const { data: serviceRoutes } = useRoutes({ service_id: id! })
  const { data: servicePlugins } = usePlugins({ service_id: id! })
  const updateMutation = useUpdateService()
  const deleteMutation = useDeleteService()
  const createRouteMutation = useCreateRoute()

  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [addRouteOpen, setAddRouteOpen] = useState(false)

  const handleEdit = (data: ServiceCreate) => {
    updateMutation.mutate({ id: id!, data }, {
      onSuccess: () => { setEditOpen(false); toast('success', 'Service updated') },
      onError: () => toast('error', 'Failed to update service'),
    })
  }

  const handleDelete = () => {
    deleteMutation.mutate(id!, {
      onSuccess: () => { toast('success', 'Service deleted'); navigate('/services') },
      onError: () => toast('error', 'Failed to delete service'),
    })
  }

  const handleToggle = () => {
    if (!service) return
    updateMutation.mutate(
      { id: id!, data: { enabled: !service.enabled } },
      { onSuccess: () => toast('success', `Service ${service.enabled ? 'disabled' : 'enabled'}`) }
    )
  }

  const handleAddRoute = (data: RouteCreate) => {
    createRouteMutation.mutate(data, {
      onSuccess: () => { setAddRouteOpen(false); toast('success', 'Route created') },
      onError: () => toast('error', 'Failed to create route'),
    })
  }

  if (isLoading) {
    return (
      <div>
        <Header breadcrumbs={[{ label: 'Services' }, { label: 'Loading...' }]} />
        <div className="mx-auto max-w-5xl p-6"><LoadingSkeleton /></div>
      </div>
    )
  }

  if (!service) {
    return (
      <div>
        <Header breadcrumbs={[{ label: 'Services' }, { label: 'Not Found' }]} />
        <div className="mx-auto max-w-5xl p-6"><p className="text-sm text-muted-foreground">Service not found.</p></div>
      </div>
    )
  }

  return (
    <div>
      <Header
        breadcrumbs={[{ label: 'Services' }, { label: service.name }]}
        action={
          <div className="flex items-center gap-2">
            <button onClick={() => setEditOpen(true)} className={cn('inline-flex items-center gap-2 rounded-lg border px-3 py-1.5 text-sm font-medium text-foreground hover:bg-accent transition-colors')}>
              <Pencil className="h-3.5 w-3.5" /> Edit
            </button>
            <button onClick={() => setDeleteOpen(true)} className={cn('inline-flex items-center gap-2 rounded-lg px-3 py-1.5 text-sm font-medium bg-destructive text-destructive-foreground hover:bg-destructive/90 transition-colors')}>
              <Trash2 className="h-3.5 w-3.5" /> Delete
            </button>
          </div>
        }
      />

      <div className="mx-auto max-w-5xl p-6 space-y-6">
        {/* Status bar */}
        <div className="flex items-center gap-3">
          <button onClick={handleToggle}><StatusBadge enabled={service.enabled} /></button>
          <span className="font-mono text-sm text-muted-foreground">
            {service.protocol}://{service.host}:{service.port}{service.path || ''}
          </span>
          <span className="text-xs text-muted-foreground font-mono ml-auto">{shortId(service.id)}</span>
        </div>

        {/* Stats */}
        {stats && (
          <div className="grid grid-cols-3 gap-4">
            {STAT_CONFIG.map((s) => {
              const value = stats[s.key]
              return (
                <div key={s.key} className="rounded-xl border bg-card p-4 relative overflow-hidden">
                  <div className="absolute top-0 right-0 w-16 h-16 rounded-full opacity-[0.06] -translate-y-6 translate-x-6" style={{ backgroundColor: s.color }} />
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-xs text-muted-foreground font-medium">{s.label}</p>
                      <p className="text-2xl font-bold text-foreground mt-1">{value}</p>
                    </div>
                    <div className="h-9 w-9 rounded-lg flex items-center justify-center" style={{ backgroundColor: `${s.color}15` }}>
                      <s.icon className="h-4 w-4" style={{ color: s.color }} />
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          {/* Main — 2 cols */}
          <div className="lg:col-span-2 space-y-6">
            {/* Routes */}
            <div className="rounded-xl border bg-card overflow-hidden">
              <div className="flex items-center justify-between border-b px-5 py-3.5">
                <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Routes ({serviceRoutes?.length || 0})
                </h3>
                <button
                  onClick={() => setAddRouteOpen(true)}
                  className="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1 text-xs font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
                >
                  <Plus className="h-3 w-3" /> Add Route
                </button>
              </div>
              {serviceRoutes && serviceRoutes.length > 0 ? (
                <div className="divide-y">
                  {serviceRoutes.map((r) => (
                    <Link key={r.id} to={`/routes/${r.id}`} className="flex items-center justify-between px-5 py-3.5 hover:bg-muted/50 transition-colors">
                      <div className="space-y-1.5">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium text-foreground">{r.name || r.paths[0]}</span>
                          <StatusBadge enabled={r.enabled} />
                        </div>
                        <div className="flex items-center gap-2">
                          <div className="flex gap-1">
                            {r.methods.slice(0, 4).map((m) => <MethodBadge key={m} method={m} />)}
                            {r.methods.length > 4 && <span className="text-xs text-muted-foreground">+{r.methods.length - 4}</span>}
                          </div>
                          <span className="text-xs font-mono text-muted-foreground">{r.paths.join(', ')}</span>
                        </div>
                      </div>
                      <ExternalLink className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                    </Link>
                  ))}
                </div>
              ) : (
                <div className="px-5 py-8 text-center">
                  <p className="text-sm text-muted-foreground">No routes yet</p>
                  <button
                    onClick={() => setAddRouteOpen(true)}
                    className="mt-2 text-xs font-medium text-primary hover:underline"
                  >
                    Create your first route
                  </button>
                </div>
              )}
            </div>

            {/* Plugins */}
            <div className="rounded-xl border bg-card overflow-hidden">
              <div className="flex items-center justify-between border-b px-5 py-3.5">
                <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Plugins ({servicePlugins?.length || 0})
                </h3>
              </div>
              {servicePlugins && servicePlugins.length > 0 ? (
                <div className="divide-y">
                  {servicePlugins.map((p) => (
                    <Link key={p.id} to={`/plugins/${p.id}`} className="flex items-center justify-between px-5 py-3.5 hover:bg-muted/50 transition-colors">
                      <div className="flex items-center gap-3">
                        <span className="text-sm font-medium text-foreground">{p.name}</span>
                        <StatusBadge enabled={p.enabled} />
                      </div>
                      <span className="text-xs text-muted-foreground">Priority {p.priority}</span>
                    </Link>
                  ))}
                </div>
              ) : (
                <div className="px-5 py-8 text-center text-sm text-muted-foreground">No plugins configured</div>
              )}
            </div>
          </div>

          {/* Sidebar — 1 col */}
          <div className="space-y-6">
            <div className="rounded-xl border bg-card overflow-hidden">
              <div className="border-b px-5 py-3.5">
                <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Configuration</h3>
              </div>
              <div className="divide-y">
                {[
                  { label: 'Protocol', value: service.protocol },
                  { label: 'Host', value: service.host, mono: true },
                  { label: 'Port', value: String(service.port) },
                  { label: 'Path', value: service.path || '—' },
                  { label: 'Connect Timeout', value: `${service.connect_timeout_ms}ms` },
                  { label: 'Read Timeout', value: `${service.read_timeout_ms}ms` },
                  { label: 'Write Timeout', value: `${service.write_timeout_ms}ms` },
                  { label: 'Retries', value: String(service.retries) },
                  { label: 'Load Balancer', value: service.load_balancer_type },
                  { label: 'Created', value: formatDate(service.created_at, 'absolute') },
                  { label: 'Updated', value: formatDate(service.updated_at, 'absolute') },
                ].map((item) => (
                  <div key={item.label} className="flex items-center justify-between px-5 py-2.5 text-sm">
                    <span className="text-muted-foreground">{item.label}</span>
                    <span className={cn('text-foreground', item.mono && 'font-mono text-xs')}>{item.value}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>

      <SlidePanel open={editOpen} onClose={() => setEditOpen(false)} title={`Edit ${service.name}`}>
        <ServiceForm initial={service} onSubmit={handleEdit} isLoading={updateMutation.isPending} />
      </SlidePanel>

      <SlidePanel open={addRouteOpen} onClose={() => setAddRouteOpen(false)} title="Add Route">
        <RouteForm defaultServiceId={service.id} onSubmit={handleAddRoute} isLoading={createRouteMutation.isPending} />
      </SlidePanel>

      <ConfirmDialog open={deleteOpen} onClose={() => setDeleteOpen(false)} onConfirm={handleDelete}
        title={`Delete ${service.name}?`} description="This will delete all routes and plugins associated with this service." isLoading={deleteMutation.isPending} />
    </div>
  )
}
