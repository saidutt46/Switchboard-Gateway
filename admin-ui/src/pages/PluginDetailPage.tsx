import { useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { Pencil, Trash2, ExternalLink } from 'lucide-react'
import { Header } from '../components/layout/Header'
import { StatusBadge } from '../components/shared/StatusBadge'
import { SlidePanel } from '../components/shared/SlidePanel'
import { ConfirmDialog } from '../components/shared/ConfirmDialog'
import { PluginForm } from '../components/plugins/PluginForm'
import { LoadingSkeleton } from '../components/shared/LoadingSkeleton'
import { usePlugin, useUpdatePlugin, useDeletePlugin } from '../hooks/usePlugins'
import { useService } from '../hooks/useServices'
import { useRoute } from '../hooks/useRoutes'
import { useConsumer } from '../hooks/useConsumers'
import { useToast } from '../components/shared/Toast'
import { formatDate, shortId } from '../lib/formatters'
import { cn } from '../lib/cn'
import type { PluginCreate } from '../api/types'

const PLUGIN_LABELS: Record<string, { label: string; category: string }> = {
  'api-key-auth': { label: 'API Key Authentication', category: 'Authentication' },
  'jwt-auth': { label: 'JWT Authentication', category: 'Authentication' },
  'rate-limit': { label: 'Rate Limiting', category: 'Traffic Control' },
  'cache': { label: 'Response Cache', category: 'Performance' },
  'cors': { label: 'CORS Headers', category: 'Transformation' },
  'request-logger': { label: 'Request Logger', category: 'Observability' },
}

export function PluginDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { toast, toastApiError } = useToast()
  const { data: plugin, isLoading } = usePlugin(id!)
  const { data: linkedService } = useService(plugin?.service_id || '')
  const { data: linkedRoute } = useRoute(plugin?.route_id || '')
  const { data: linkedConsumer } = useConsumer(plugin?.consumer_id || '')
  const updateMutation = useUpdatePlugin()
  const deleteMutation = useDeletePlugin()

  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  const handleEdit = (data: PluginCreate) => {
    updateMutation.mutate({ id: id!, data }, {
      onSuccess: () => { setEditOpen(false); toast('success', 'Plugin updated') },
      onError: (err) => toastApiError(err, 'Failed to update plugin'),
    })
  }

  const handleDelete = () => {
    deleteMutation.mutate(id!, {
      onSuccess: () => { toast('success', 'Plugin deleted'); navigate('/plugins') },
      onError: (err) => toastApiError(err, 'Failed to delete plugin'),
    })
  }

  const handleToggle = () => {
    if (!plugin) return
    updateMutation.mutate(
      { id: id!, data: { enabled: !plugin.enabled } },
      { onSuccess: () => toast('success', `Plugin ${plugin.enabled ? 'disabled' : 'enabled'}`) }
    )
  }

  if (isLoading) {
    return (
      <div>
        <Header breadcrumbs={[{ label: 'Plugins' }, { label: 'Loading...' }]} />
        <div className="mx-auto max-w-5xl p-6"><LoadingSkeleton /></div>
      </div>
    )
  }

  if (!plugin) {
    return (
      <div>
        <Header breadcrumbs={[{ label: 'Plugins' }, { label: 'Not Found' }]} />
        <div className="mx-auto max-w-5xl p-6"><p className="text-sm text-muted-foreground">Plugin not found.</p></div>
      </div>
    )
  }

  const info = PLUGIN_LABELS[plugin.name] || { label: plugin.name, category: 'Custom' }
  const configEntries = Object.entries(plugin.config)

  return (
    <div>
      <Header
        breadcrumbs={[{ label: 'Plugins' }, { label: plugin.name }]}
        action={
          <div className="flex items-center gap-2">
            <button onClick={() => setEditOpen(true)} className={cn('inline-flex items-center gap-2 rounded-xl border bg-card px-3 py-1.5 text-sm font-medium text-foreground hover:bg-accent transition-colors')}>
              <Pencil className="h-3.5 w-3.5" /> Edit
            </button>
            <button onClick={() => setDeleteOpen(true)} className={cn('inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium bg-destructive text-destructive-foreground hover:bg-destructive/90 transition-colors')}>
              <Trash2 className="h-3.5 w-3.5" /> Delete
            </button>
          </div>
        }
      />

      <div className="mx-auto max-w-5xl p-6 space-y-6">
        {/* Title bar */}
        <div className="flex items-center gap-3 flex-wrap">
          <button onClick={handleToggle}><StatusBadge enabled={plugin.enabled} /></button>
          <span className="rounded-full bg-muted px-2.5 py-0.5 text-xs font-medium text-muted-foreground">{info.category}</span>
          <span className="text-sm text-muted-foreground font-mono">{shortId(plugin.id)}</span>
        </div>

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          {/* Main — 2 cols */}
          <div className="lg:col-span-2 space-y-6">
            {/* Config rendered as key-value */}
            <div className="rounded-xl border bg-card">
              <div className="border-b px-4 py-2.5">
                <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Configuration</h3>
              </div>
              {configEntries.length > 0 ? (
                <div className="divide-y">
                  {configEntries.map(([key, value]) => (
                    <div key={key} className="flex items-start justify-between px-4 py-3 text-sm gap-4">
                      <span className="text-muted-foreground font-mono text-xs shrink-0">{key}</span>
                      <span className="text-foreground text-right break-all">
                        {typeof value === 'boolean' ? (
                          <span className={cn('rounded-full px-2 py-0.5 text-xs font-medium', value ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400' : 'bg-muted text-muted-foreground')}>
                            {value ? 'true' : 'false'}
                          </span>
                        ) : Array.isArray(value) ? (
                          <div className="flex flex-wrap gap-1 justify-end">
                            {value.map((v, i) => (
                              <span key={i} className="rounded border bg-muted/50 px-1.5 py-0.5 font-mono text-xs">{String(v)}</span>
                            ))}
                          </div>
                        ) : typeof value === 'object' && value !== null ? (
                          <pre className="font-mono text-xs text-left bg-muted rounded p-2">{JSON.stringify(value, null, 2)}</pre>
                        ) : (
                          <span className="font-mono text-xs">{String(value)}</span>
                        )}
                      </span>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="px-4 py-6 text-center text-sm text-muted-foreground">No configuration</div>
              )}
            </div>

            {/* Raw JSON */}
            <div className="rounded-xl border bg-card">
              <div className="border-b px-4 py-2.5">
                <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Raw JSON</h3>
              </div>
              <div className="p-4">
                <pre className="text-xs font-mono text-foreground bg-muted rounded-md p-4 overflow-x-auto leading-relaxed">
                  {JSON.stringify(plugin.config, null, 2)}
                </pre>
              </div>
            </div>
          </div>

          {/* Sidebar — 1 col */}
          <div className="space-y-6">
            {/* Details */}
            <div className="rounded-xl border bg-card">
              <div className="border-b px-4 py-2.5">
                <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Details</h3>
              </div>
              <div className="divide-y">
                {[
                  { label: 'Type', value: info.label },
                  { label: 'Scope', value: plugin.scope.charAt(0).toUpperCase() + plugin.scope.slice(1) },
                  { label: 'Priority', value: String(plugin.priority) },
                  { label: 'Created', value: formatDate(plugin.created_at, 'absolute') },
                  { label: 'Updated', value: formatDate(plugin.updated_at, 'absolute') },
                ].map((item) => (
                  <div key={item.label} className="flex items-center justify-between px-4 py-2.5 text-sm">
                    <span className="text-muted-foreground">{item.label}</span>
                    <span className="text-foreground">{item.value}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Linked Entity */}
            {plugin.scope !== 'global' && (
              <div className="rounded-xl border bg-card">
                <div className="border-b px-4 py-2.5">
                  <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Attached {plugin.scope}
                  </h3>
                </div>
                {linkedService && (
                  <Link to={`/services/${linkedService.id}`} className="flex items-center justify-between px-4 py-3 hover:bg-muted/50 transition-colors">
                    <div>
                      <p className="text-sm font-medium text-foreground">{linkedService.name}</p>
                      <p className="text-xs font-mono text-muted-foreground">{linkedService.host}:{linkedService.port}</p>
                    </div>
                    <ExternalLink className="h-3.5 w-3.5 text-muted-foreground" />
                  </Link>
                )}
                {linkedRoute && (
                  <Link to={`/routes/${linkedRoute.id}`} className="flex items-center justify-between px-4 py-3 hover:bg-muted/50 transition-colors">
                    <div>
                      <p className="text-sm font-medium text-foreground">{linkedRoute.name || linkedRoute.paths[0]}</p>
                      <p className="text-xs font-mono text-muted-foreground">{linkedRoute.paths.join(', ')}</p>
                    </div>
                    <ExternalLink className="h-3.5 w-3.5 text-muted-foreground" />
                  </Link>
                )}
                {linkedConsumer && (
                  <Link to={`/consumers/${linkedConsumer.id}`} className="flex items-center justify-between px-4 py-3 hover:bg-muted/50 transition-colors">
                    <div>
                      <p className="text-sm font-medium text-foreground">{linkedConsumer.username}</p>
                      <p className="text-xs text-muted-foreground">{linkedConsumer.email || 'No email'}</p>
                    </div>
                    <ExternalLink className="h-3.5 w-3.5 text-muted-foreground" />
                  </Link>
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      <SlidePanel open={editOpen} onClose={() => setEditOpen(false)} title={`Edit ${plugin.name}`}>
        <PluginForm initial={plugin} onSubmit={handleEdit} isLoading={updateMutation.isPending} />
      </SlidePanel>

      <ConfirmDialog open={deleteOpen} onClose={() => setDeleteOpen(false)} onConfirm={handleDelete}
        title={`Delete ${plugin.name}?`} description="This plugin will be removed and the gateway will reload." isLoading={deleteMutation.isPending} />
    </div>
  )
}
