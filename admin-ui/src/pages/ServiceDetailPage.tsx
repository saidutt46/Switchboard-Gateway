import { useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { Pencil, Trash2, ExternalLink } from 'lucide-react'
import { Header } from '../components/layout/Header'
import { StatusBadge } from '../components/shared/StatusBadge'
import { MethodBadge } from '../components/shared/MethodBadge'
import { SlidePanel } from '../components/shared/SlidePanel'
import { ConfirmDialog } from '../components/shared/ConfirmDialog'
import { ServiceForm } from '../components/services/ServiceForm'
import { LoadingSkeleton } from '../components/shared/LoadingSkeleton'
import { useService, useServiceStats, useUpdateService, useDeleteService } from '../hooks/useServices'
import { useRoutes } from '../hooks/useRoutes'
import { usePlugins } from '../hooks/usePlugins'
import { useToast } from '../components/shared/Toast'
import { formatDate, shortId } from '../lib/formatters'
import { cn } from '../lib/cn'
import type { ServiceCreate } from '../api/types'

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

  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

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
            <button onClick={() => setEditOpen(true)} className={cn('inline-flex items-center gap-2 rounded-md border px-3 py-1.5 text-sm font-medium text-foreground hover:bg-accent transition-colors')}>
              <Pencil className="h-3.5 w-3.5" /> Edit
            </button>
            <button onClick={() => setDeleteOpen(true)} className={cn('inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium bg-destructive text-destructive-foreground hover:bg-destructive/90 transition-colors')}>
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
        </div>

        {/* Stats */}
        {stats && (
          <div className="grid grid-cols-3 gap-3">
            {[
              { label: 'Routes', value: stats.routes_count },
              { label: 'Targets', value: stats.targets_count },
              { label: 'Plugins', value: stats.plugins_count },
            ].map((s) => (
              <div key={s.label} className="rounded-md border p-4 text-center">
                <p className="text-2xl font-semibold text-foreground">{s.value}</p>
                <p className="text-xs text-muted-foreground mt-1">{s.label}</p>
              </div>
            ))}
          </div>
        )}

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          {/* Main — 2 cols */}
          <div className="lg:col-span-2 space-y-6">
            {/* Routes */}
            <div className="rounded-md border">
              <div className="border-b px-4 py-2.5">
                <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  Routes ({serviceRoutes?.length || 0})
                </h3>
              </div>
              {serviceRoutes && serviceRoutes.length > 0 ? (
                <div className="divide-y">
                  {serviceRoutes.map((r) => (
                    <Link key={r.id} to={`/routes/${r.id}`} className="flex items-center justify-between px-4 py-3 hover:bg-muted/50 transition-colors">
                      <div className="space-y-1">
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
                <div className="px-4 py-6 text-center text-sm text-muted-foreground">No routes configured</div>
              )}
            </div>

            {/* Plugins */}
            {servicePlugins && servicePlugins.length > 0 && (
              <div className="rounded-md border">
                <div className="border-b px-4 py-2.5">
                  <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Plugins ({servicePlugins.length})</h3>
                </div>
                <div className="divide-y">
                  {servicePlugins.map((p) => (
                    <Link key={p.id} to={`/plugins/${p.id}`} className="flex items-center justify-between px-4 py-3 hover:bg-muted/50 transition-colors">
                      <div className="flex items-center gap-3">
                        <span className="text-sm font-medium text-foreground">{p.name}</span>
                        <StatusBadge enabled={p.enabled} />
                      </div>
                      <span className="text-xs text-muted-foreground">Priority {p.priority}</span>
                    </Link>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Sidebar — 1 col */}
          <div className="space-y-6">
            <div className="rounded-md border">
              <div className="border-b px-4 py-2.5">
                <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Configuration</h3>
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
                  { label: 'ID', value: shortId(service.id), mono: true },
                ].map((item) => (
                  <div key={item.label} className="flex items-center justify-between px-4 py-2.5 text-sm">
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

      <ConfirmDialog open={deleteOpen} onClose={() => setDeleteOpen(false)} onConfirm={handleDelete}
        title={`Delete ${service.name}?`} description="This will delete all routes and plugins associated with this service. This action cannot be undone." isLoading={deleteMutation.isPending} />
    </div>
  )
}
