import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Route as RouteIcon, Trash2, Pencil } from 'lucide-react'
import { Header } from '../components/layout/Header'
import { DataTable, type Column } from '../components/shared/DataTable'
import { StatusBadge } from '../components/shared/StatusBadge'
import { MethodBadge } from '../components/shared/MethodBadge'
import { SlidePanel } from '../components/shared/SlidePanel'
import { ConfirmDialog } from '../components/shared/ConfirmDialog'
import { RouteForm } from '../components/routes/RouteForm'
import { useRoutes, useCreateRoute, useUpdateRoute, useDeleteRoute } from '../hooks/useRoutes'
import { useToast } from '../components/shared/Toast'
import { cn } from '../lib/cn'
import type { Route, RouteCreate } from '../api/types'

export function RoutesPage() {
  const navigate = useNavigate()
  const { toast } = useToast()
  const { data: routes, isLoading } = useRoutes()
  const createMutation = useCreateRoute()
  const updateMutation = useUpdateRoute()
  const deleteMutation = useDeleteRoute()

  const [createOpen, setCreateOpen] = useState(false)
  const [editRoute, setEditRoute] = useState<Route | null>(null)
  const [deleteRoute, setDeleteRoute] = useState<Route | null>(null)

  const handleCreate = (data: RouteCreate) => {
    createMutation.mutate(data, {
      onSuccess: () => { setCreateOpen(false); toast('success', 'Route created') },
      onError: () => toast('error', 'Failed to create route'),
    })
  }

  const handleEdit = (data: RouteCreate) => {
    if (!editRoute) return
    updateMutation.mutate(
      { id: editRoute.id, data },
      {
        onSuccess: () => { setEditRoute(null); toast('success', 'Route updated') },
        onError: () => toast('error', 'Failed to update route'),
      }
    )
  }

  const handleToggle = (route: Route) => {
    updateMutation.mutate(
      { id: route.id, data: { enabled: !route.enabled } },
      {
        onSuccess: () => toast('success', `Route ${route.enabled ? 'disabled' : 'enabled'}`),
        onError: () => toast('error', 'Failed to update route'),
      }
    )
  }

  const handleDelete = () => {
    if (!deleteRoute) return
    deleteMutation.mutate(deleteRoute.id, {
      onSuccess: () => { setDeleteRoute(null); toast('success', 'Route deleted') },
      onError: () => toast('error', 'Failed to delete route'),
    })
  }

  const columns: Column<Route>[] = [
    {
      key: 'name',
      header: 'Name',
      render: (r) => <span className="font-medium text-foreground">{r.name || r.paths[0]}</span>,
    },
    {
      key: 'paths',
      header: 'Paths',
      render: (r) => (
        <div className="flex flex-wrap gap-1">
          {r.paths.slice(0, 3).map((p) => (
            <span key={p} className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs text-muted-foreground">{p}</span>
          ))}
          {r.paths.length > 3 && (
            <span className="text-xs text-muted-foreground">+{r.paths.length - 3}</span>
          )}
        </div>
      ),
    },
    {
      key: 'methods',
      header: 'Methods',
      render: (r) => (
        <div className="flex flex-wrap gap-1">
          {r.methods.map((m) => <MethodBadge key={m} method={m} />)}
        </div>
      ),
    },
    {
      key: 'enabled',
      header: 'Status',
      render: (r) => (
        <button onClick={(e) => { e.stopPropagation(); handleToggle(r) }}>
          <StatusBadge enabled={r.enabled} />
        </button>
      ),
    },
    {
      key: 'actions',
      header: '',
      className: 'w-20 text-right',
      render: (r) => (
        <div className="flex items-center justify-end gap-1">
          <button
            onClick={(e) => { e.stopPropagation(); setEditRoute(r) }}
            className="rounded p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
          >
            <Pencil className="h-3.5 w-3.5" />
          </button>
          <button
            onClick={(e) => { e.stopPropagation(); setDeleteRoute(r) }}
            className="rounded p-1.5 text-muted-foreground hover:text-destructive hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      ),
    },
  ]

  return (
    <div>
      <Header
        breadcrumbs={[{ label: 'Routes' }]}
        action={
          <button
            onClick={() => setCreateOpen(true)}
            className={cn(
              'inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium',
              'bg-primary text-primary-foreground hover:bg-primary/90 transition-colors'
            )}
          >
            <Plus className="h-3.5 w-3.5" />
            New Route
          </button>
        }
      />
      <div className="mx-auto max-w-5xl p-6">
        <DataTable
          columns={columns}
          data={routes || []}
          isLoading={isLoading}
          keyExtractor={(r) => r.id}
          onRowClick={(r) => navigate(`/routes/${r.id}`)}
          emptyState={{
            title: 'No routes yet',
            description: 'Create routes to map incoming requests to your services.',
            icon: <RouteIcon className="h-8 w-8" />,
            action: (
              <button onClick={() => setCreateOpen(true)} className="rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90">
                Create Route
              </button>
            ),
          }}
        />
      </div>

      <SlidePanel open={createOpen} onClose={() => setCreateOpen(false)} title="New Route">
        <RouteForm onSubmit={handleCreate} isLoading={createMutation.isPending} />
      </SlidePanel>

      <SlidePanel open={!!editRoute} onClose={() => setEditRoute(null)} title={`Edit ${editRoute?.name || 'Route'}`}>
        {editRoute && <RouteForm initial={editRoute} onSubmit={handleEdit} isLoading={updateMutation.isPending} />}
      </SlidePanel>

      <ConfirmDialog
        open={!!deleteRoute}
        onClose={() => setDeleteRoute(null)}
        onConfirm={handleDelete}
        title={`Delete ${deleteRoute?.name || 'route'}?`}
        description="This will also delete all plugins associated with this route."
        isLoading={deleteMutation.isPending}
      />
    </div>
  )
}
