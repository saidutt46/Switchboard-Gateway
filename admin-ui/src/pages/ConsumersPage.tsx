import { useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Users, Trash2, Pencil } from 'lucide-react'
import { SearchInput } from '../components/shared/SearchInput'
import { Header } from '../components/layout/Header'
import { DataTable, type Column } from '../components/shared/DataTable'
import { SlidePanel } from '../components/shared/SlidePanel'
import { ConfirmDialog } from '../components/shared/ConfirmDialog'
import { ConsumerForm } from '../components/consumers/ConsumerForm'
import { useConsumers, useCreateConsumer, useUpdateConsumer, useDeleteConsumer } from '../hooks/useConsumers'
import { useToast } from '../components/shared/Toast'
import { formatDate } from '../lib/formatters'
import { cn } from '../lib/cn'
import type { Consumer, ConsumerCreate } from '../api/types'

export function ConsumersPage() {
  const navigate = useNavigate()
  const { toast, toastApiError } = useToast()
  const { data: consumers, isLoading } = useConsumers()
  const createMutation = useCreateConsumer()
  const updateMutation = useUpdateConsumer()
  const deleteMutation = useDeleteConsumer()

  const [search, setSearch] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [editConsumer, setEditConsumer] = useState<Consumer | null>(null)
  const [deleteConsumer, setDeleteConsumer] = useState<Consumer | null>(null)

  const filteredConsumers = useMemo(() => {
    if (!search) return consumers || []
    return (consumers || []).filter((c) =>
      c.username.toLowerCase().includes(search.toLowerCase()) ||
      (c.email || '').toLowerCase().includes(search.toLowerCase())
    )
  }, [consumers, search])

  const handleCreate = (data: ConsumerCreate) => {
    createMutation.mutate(data, {
      onSuccess: () => { setCreateOpen(false); toast('success', 'Consumer created') },
      onError: (err) => toastApiError(err, 'Failed to create consumer'),
    })
  }

  const handleEdit = (data: ConsumerCreate) => {
    if (!editConsumer) return
    updateMutation.mutate({ id: editConsumer.id, data }, {
      onSuccess: () => { setEditConsumer(null); toast('success', 'Consumer updated') },
      onError: (err) => toastApiError(err, 'Failed to update consumer'),
    })
  }

  const handleDelete = () => {
    if (!deleteConsumer) return
    deleteMutation.mutate(deleteConsumer.id, {
      onSuccess: () => { setDeleteConsumer(null); toast('success', 'Consumer deleted') },
      onError: (err) => toastApiError(err, 'Failed to delete consumer'),
    })
  }

  const columns: Column<Consumer>[] = [
    { key: 'username', header: 'Username', sortable: true, render: (c) => <span className="font-medium text-foreground">{c.username}</span> },
    { key: 'email', header: 'Email', render: (c) => <span className="text-muted-foreground">{c.email || '—'}</span> },
    { key: 'custom_id', header: 'Custom ID', render: (c) => <span className="text-muted-foreground">{c.custom_id || '—'}</span> },
    { key: 'created_at', header: 'Created', render: (c) => <span className="text-muted-foreground">{formatDate(c.created_at)}</span> },
    {
      key: 'actions', header: '', className: 'w-20 text-right',
      render: (c) => (
        <div className="flex items-center justify-end gap-1">
          <button onClick={(e) => { e.stopPropagation(); setEditConsumer(c) }} className="rounded p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors">
            <Pencil className="h-3.5 w-3.5" />
          </button>
          <button onClick={(e) => { e.stopPropagation(); setDeleteConsumer(c) }} className="rounded p-1.5 text-muted-foreground hover:text-destructive hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors">
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      ),
    },
  ]

  return (
    <div>
      <Header breadcrumbs={[{ label: 'Consumers' }]} action={
        <button onClick={() => setCreateOpen(true)} className={cn('inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-colors')}>
          <Plus className="h-3.5 w-3.5" /> New Consumer
        </button>
      } />
      <div className="mx-auto max-w-5xl p-6 space-y-4">
        <SearchInput value={search} onChange={setSearch} placeholder="Search consumers..." className="max-w-xs" />
        <DataTable columns={columns} data={filteredConsumers} isLoading={isLoading} keyExtractor={(c) => c.id} onRowClick={(c) => navigate(`/consumers/${c.id}`)}
          emptyState={{ title: 'No consumers yet', description: 'Create consumers to manage API access.', icon: <Users className="h-8 w-8" />,
            action: <button onClick={() => setCreateOpen(true)} className="rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90">Create Consumer</button> }}
        />
      </div>
      <SlidePanel open={createOpen} onClose={() => setCreateOpen(false)} title="New Consumer">
        <ConsumerForm onSubmit={handleCreate} isLoading={createMutation.isPending} />
      </SlidePanel>
      <SlidePanel open={!!editConsumer} onClose={() => setEditConsumer(null)} title={`Edit ${editConsumer?.username || ''}`}>
        {editConsumer && <ConsumerForm initial={editConsumer} onSubmit={handleEdit} isLoading={updateMutation.isPending} />}
      </SlidePanel>
      <ConfirmDialog open={!!deleteConsumer} onClose={() => setDeleteConsumer(null)} onConfirm={handleDelete}
        title={`Delete ${deleteConsumer?.username || ''}?`} description="This will also delete all API keys and plugins associated with this consumer." isLoading={deleteMutation.isPending} />
    </div>
  )
}
