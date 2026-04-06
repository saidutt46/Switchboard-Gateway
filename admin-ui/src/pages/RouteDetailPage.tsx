import { useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { Pencil, Trash2, ExternalLink, Plus } from 'lucide-react'
import { Header } from '../components/layout/Header'
import { StatusBadge } from '../components/shared/StatusBadge'
import { MethodBadge } from '../components/shared/MethodBadge'
import { SlidePanel } from '../components/shared/SlidePanel'
import { ConfirmDialog } from '../components/shared/ConfirmDialog'
import { RouteForm } from '../components/routes/RouteForm'
import { PluginForm } from '../components/plugins/PluginForm'
import { DetailSkeleton } from '../components/shared/DetailSkeleton'
import { useRoute, useUpdateRoute, useDeleteRoute } from '../hooks/useRoutes'
import { useService } from '../hooks/useServices'
import { usePlugins, useCreatePlugin } from '../hooks/usePlugins'
import { useToast } from '../components/shared/Toast'
import { formatDate, shortId } from '../lib/formatters'
import { cn } from '../lib/cn'
import type { RouteCreate, PluginCreate } from '../api/types'

export function RouteDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { toast, toastApiError } = useToast()
  const { data: route, isLoading } = useRoute(id!)
  const { data: parentService } = useService(route?.service_id || '')
  const { data: routePlugins } = usePlugins(route ? { route_id: route.id } : undefined)
  const updateMutation = useUpdateRoute()
  const deleteMutation = useDeleteRoute()
  const createPluginMutation = useCreatePlugin()

  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [addPluginOpen, setAddPluginOpen] = useState(false)

  const handleEdit = (data: RouteCreate) => {
    updateMutation.mutate({ id: id!, data }, {
      onSuccess: () => { setEditOpen(false); toast('success', 'Route updated') },
      onError: (err) => toastApiError(err, 'Failed to update route'),
    })
  }

  const handleDelete = () => {
    deleteMutation.mutate(id!, {
      onSuccess: () => { toast('success', 'Route deleted'); navigate('/routes') },
      onError: (err) => toastApiError(err, 'Failed to delete route'),
    })
  }

  const handleToggle = () => {
    if (!route) return
    updateMutation.mutate(
      { id: id!, data: { enabled: !route.enabled } },
      { onSuccess: () => toast('success', `Route ${route.enabled ? 'disabled' : 'enabled'}`) }
    )
  }

  const handleAddPlugin = (data: PluginCreate) => {
    createPluginMutation.mutate({ ...data, scope: 'route', route_id: route!.id }, {
      onSuccess: () => { setAddPluginOpen(false); toast('success', 'Plugin added') },
      onError: (err) => toastApiError(err, 'Failed to add plugin'),
    })
  }

  if (isLoading) {
    return (
      <div>
        <Header breadcrumbs={[{ label: 'Routes' }, { label: 'Loading...' }]} />
        <div className="mx-auto max-w-5xl p-6"><DetailSkeleton /></div>
      </div>
    )
  }

  if (!route) {
    return (
      <div>
        <Header breadcrumbs={[{ label: 'Routes' }, { label: 'Not Found' }]} />
        <div className="mx-auto max-w-5xl p-6"><p className="text-sm text-muted-foreground">Route not found.</p></div>
      </div>
    )
  }

  return (
    <div>
      <Header
        breadcrumbs={[{ label: 'Routes' }, { label: route.name || route.paths[0] }]}
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
          <button onClick={handleToggle}><StatusBadge enabled={route.enabled} /></button>
          {route.name && <span className="text-sm text-muted-foreground font-mono">{shortId(route.id)}</span>}
        </div>

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          {/* Main — 2 cols */}
          <div className="lg:col-span-2 space-y-6">
            {/* Paths */}
            <div className="rounded-xl border bg-card overflow-hidden">
              <div className="border-b px-5 py-3.5">
                <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Paths</h3>
              </div>
              <div className="p-5">
                <div className="flex flex-wrap gap-2">
                  {route.paths.map((p) => (
                    <span key={p} className="rounded-lg border bg-muted/50 px-3 py-1.5 font-mono text-sm text-foreground">{p}</span>
                  ))}
                </div>
              </div>
            </div>

            {/* Methods */}
            <div className="rounded-xl border bg-card overflow-hidden">
              <div className="border-b px-5 py-3.5">
                <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Methods</h3>
              </div>
              <div className="p-5">
                <div className="flex flex-wrap gap-2">
                  {route.methods.map((m) => <MethodBadge key={m} method={m} />)}
                </div>
              </div>
            </div>

            {/* Plugins */}
            <div className="rounded-xl border bg-card overflow-hidden">
              <div className="flex items-center justify-between border-b px-5 py-3.5">
                <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                  Plugins ({routePlugins?.length || 0})
                </h3>
                <button
                  onClick={() => setAddPluginOpen(true)}
                  className="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1 text-xs font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
                >
                  <Plus className="h-3 w-3" /> Add Plugin
                </button>
              </div>
              {routePlugins && routePlugins.length > 0 ? (
                <div className="divide-y">
                  {routePlugins.map((p) => (
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
                <div className="px-5 py-8 text-center">
                  <p className="text-sm text-muted-foreground">No plugins on this route</p>
                  <button
                    onClick={() => setAddPluginOpen(true)}
                    className="mt-2 text-xs font-medium text-primary hover:underline"
                  >
                    Add your first plugin
                  </button>
                </div>
              )}
            </div>
          </div>

          {/* Sidebar — 1 col */}
          <div className="space-y-6">
            {/* Parent Service */}
            {parentService && (
              <div className="rounded-xl border bg-card overflow-hidden">
                <div className="border-b px-5 py-3.5">
                  <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Service</h3>
                </div>
                <Link to={`/services/${parentService.id}`} className="flex items-center justify-between px-5 py-3.5 hover:bg-muted/50 transition-colors">
                  <div>
                    <p className="text-sm font-medium text-foreground">{parentService.name}</p>
                    <p className="text-xs font-mono text-muted-foreground">{parentService.protocol}://{parentService.host}:{parentService.port}</p>
                  </div>
                  <ExternalLink className="h-3.5 w-3.5 text-muted-foreground" />
                </Link>
              </div>
            )}

            {/* Config */}
            <div className="rounded-xl border bg-card overflow-hidden">
              <div className="border-b px-5 py-3.5">
                <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Configuration</h3>
              </div>
              <div className="divide-y">
                {[
                  { label: 'Strip Path', value: route.strip_path ? 'Yes' : 'No' },
                  { label: 'Preserve Host', value: route.preserve_host ? 'Yes' : 'No' },
                  { label: 'Created', value: formatDate(route.created_at, 'absolute') },
                  { label: 'Updated', value: formatDate(route.updated_at, 'absolute') },
                ].map((item) => (
                  <div key={item.label} className="flex items-center justify-between px-5 py-2.5 text-sm">
                    <span className="text-muted-foreground">{item.label}</span>
                    <span className="text-foreground">{item.value}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>

      <SlidePanel open={editOpen} onClose={() => setEditOpen(false)} title={`Edit ${route.name || 'Route'}`}>
        <RouteForm initial={route} onSubmit={handleEdit} isLoading={updateMutation.isPending} />
      </SlidePanel>

      <SlidePanel open={addPluginOpen} onClose={() => setAddPluginOpen(false)} title="Add Plugin to Route">
        <PluginForm onSubmit={handleAddPlugin} isLoading={createPluginMutation.isPending} />
      </SlidePanel>

      <ConfirmDialog open={deleteOpen} onClose={() => setDeleteOpen(false)} onConfirm={handleDelete}
        title={`Delete ${route.name || 'route'}?`} description="This will also delete all plugins associated with this route." isLoading={deleteMutation.isPending} />
    </div>
  )
}
