import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Server, Trash2, Pencil } from 'lucide-react'
import { SearchInput } from '../components/shared/SearchInput'
import { Header } from '../components/layout/Header'
import { DataTable, type Column } from '../components/shared/DataTable'
import { StatusBadge } from '../components/shared/StatusBadge'
import { SlidePanel } from '../components/shared/SlidePanel'
import { ConfirmDialog } from '../components/shared/ConfirmDialog'
import { ServiceForm } from '../components/services/ServiceForm'
import { useServices, useCreateService, useUpdateService, useDeleteService } from '../hooks/useServices'
import { useToast } from '../components/shared/Toast'
import { cn } from '../lib/cn'
import type { Service, ServiceCreate } from '../api/types'

export function ServicesPage() {
  const navigate = useNavigate()
  const { toast } = useToast()
  const { data: services, isLoading } = useServices()
  const createMutation = useCreateService()
  const updateMutation = useUpdateService()
  const deleteMutation = useDeleteService()

  const [search, setSearch] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [editService, setEditService] = useState<Service | null>(null)
  const [deleteService, setDeleteService] = useState<Service | null>(null)

  const handleCreate = (data: ServiceCreate) => {
    createMutation.mutate(data, {
      onSuccess: () => {
        setCreateOpen(false)
        toast('success', 'Service created')
      },
      onError: () => toast('error', 'Failed to create service'),
    })
  }

  const handleEdit = (data: ServiceCreate) => {
    if (!editService) return
    updateMutation.mutate(
      { id: editService.id, data },
      {
        onSuccess: () => {
          setEditService(null)
          toast('success', 'Service updated')
        },
        onError: () => toast('error', 'Failed to update service'),
      }
    )
  }

  const handleToggle = (service: Service) => {
    updateMutation.mutate(
      { id: service.id, data: { enabled: !service.enabled } },
      {
        onSuccess: () => toast('success', `Service ${service.enabled ? 'disabled' : 'enabled'}`),
        onError: () => toast('error', 'Failed to update service'),
      }
    )
  }

  const handleDelete = () => {
    if (!deleteService) return
    deleteMutation.mutate(deleteService.id, {
      onSuccess: () => {
        setDeleteService(null)
        toast('success', 'Service deleted')
      },
      onError: () => toast('error', 'Failed to delete service'),
    })
  }

  const columns: Column<Service>[] = [
    {
      key: 'name',
      header: 'Name',
      sortable: true,
      render: (s) => <span className="font-medium text-foreground">{s.name}</span>,
    },
    {
      key: 'target',
      header: 'Target',
      render: (s) => (
        <span className="font-mono text-xs text-muted-foreground">
          {s.protocol}://{s.host}:{s.port}{s.path || ''}
        </span>
      ),
    },
    {
      key: 'enabled',
      header: 'Status',
      render: (s) => (
        <button onClick={(e) => { e.stopPropagation(); handleToggle(s) }}>
          <StatusBadge enabled={s.enabled} />
        </button>
      ),
    },
    {
      key: 'actions',
      header: '',
      className: 'w-20 text-right',
      render: (s) => (
        <div className="flex items-center justify-end gap-1">
          <button
            onClick={(e) => { e.stopPropagation(); setEditService(s) }}
            className="rounded p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
          >
            <Pencil className="h-3.5 w-3.5" />
          </button>
          <button
            onClick={(e) => { e.stopPropagation(); setDeleteService(s) }}
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
        breadcrumbs={[{ label: 'Services' }]}
        action={
          <button
            onClick={() => setCreateOpen(true)}
            className={cn(
              'inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium',
              'bg-primary text-primary-foreground hover:bg-primary/90 transition-colors'
            )}
          >
            <Plus className="h-3.5 w-3.5" />
            New Service
          </button>
        }
      />
      <div className="mx-auto max-w-5xl p-6 space-y-4">
        <SearchInput value={search} onChange={setSearch} placeholder="Search services..." className="max-w-xs" />
        <DataTable
          columns={columns}
          data={(services || []).filter((s) => !search || s.name.toLowerCase().includes(search.toLowerCase()) || s.host.toLowerCase().includes(search.toLowerCase()))}
          isLoading={isLoading}
          keyExtractor={(s) => s.id}
          onRowClick={(s) => navigate(`/services/${s.id}`)}
          emptyState={{
            title: 'No services yet',
            description: 'Create your first backend service to get started.',
            icon: <Server className="h-8 w-8" />,
            action: (
              <button
                onClick={() => setCreateOpen(true)}
                className="rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
              >
                Create Service
              </button>
            ),
          }}
        />
      </div>

      {/* Create Panel */}
      <SlidePanel open={createOpen} onClose={() => setCreateOpen(false)} title="New Service">
        <ServiceForm onSubmit={handleCreate} isLoading={createMutation.isPending} />
      </SlidePanel>

      {/* Edit Panel */}
      <SlidePanel
        open={!!editService}
        onClose={() => setEditService(null)}
        title={`Edit ${editService?.name || ''}`}
      >
        {editService && (
          <ServiceForm initial={editService} onSubmit={handleEdit} isLoading={updateMutation.isPending} />
        )}
      </SlidePanel>

      {/* Delete Dialog */}
      <ConfirmDialog
        open={!!deleteService}
        onClose={() => setDeleteService(null)}
        onConfirm={handleDelete}
        title={`Delete ${deleteService?.name || ''}?`}
        description="This will also delete all routes and plugins associated with this service. This action cannot be undone."
        isLoading={deleteMutation.isPending}
      />
    </div>
  )
}
