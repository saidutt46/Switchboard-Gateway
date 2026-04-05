import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Pencil, Trash2, Plus } from 'lucide-react'
import { Header } from '../components/layout/Header'
import { SlidePanel } from '../components/shared/SlidePanel'
import { ConfirmDialog } from '../components/shared/ConfirmDialog'
import { LoadingSkeleton } from '../components/shared/LoadingSkeleton'
import { ConsumerForm } from '../components/consumers/ConsumerForm'
import { ApiKeyList } from '../components/consumers/ApiKeyList'
import { ApiKeyGenerateDialog } from '../components/consumers/ApiKeyGenerateDialog'
import { useConsumer, useUpdateConsumer, useDeleteConsumer, useApiKeys, useGenerateApiKey, useToggleApiKey, useRevokeApiKey } from '../hooks/useConsumers'
import { useToast } from '../components/shared/Toast'
import { formatDate } from '../lib/formatters'
import { cn } from '../lib/cn'
import type { ConsumerCreate, ApiKeyGenerated } from '../api/types'

export function ConsumerDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { toast } = useToast()
  const { data: consumer, isLoading } = useConsumer(id!)
  const { data: keys } = useApiKeys(id!)
  const updateMutation = useUpdateConsumer()
  const deleteMutation = useDeleteConsumer()
  const generateMutation = useGenerateApiKey()
  const toggleKeyMutation = useToggleApiKey()
  const revokeKeyMutation = useRevokeApiKey()

  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [generateOpen, setGenerateOpen] = useState(false)
  const [generatedKey, setGeneratedKey] = useState<ApiKeyGenerated | null>(null)
  const [revokeKeyId, setRevokeKeyId] = useState<string | null>(null)

  const handleEdit = (data: ConsumerCreate) => {
    updateMutation.mutate({ id: id!, data }, {
      onSuccess: () => { setEditOpen(false); toast('success', 'Consumer updated') },
      onError: () => toast('error', 'Failed to update consumer'),
    })
  }

  const handleDelete = () => {
    deleteMutation.mutate(id!, {
      onSuccess: () => { toast('success', 'Consumer deleted'); navigate('/consumers') },
      onError: () => toast('error', 'Failed to delete consumer'),
    })
  }

  const handleGenerate = (name: string) => {
    generateMutation.mutate({ consumerId: id!, name: name || undefined }, {
      onSuccess: (data) => setGeneratedKey(data),
      onError: () => toast('error', 'Failed to generate key'),
    })
  }

  const handleToggleKey = (keyId: string, enable: boolean) => {
    toggleKeyMutation.mutate({ consumerId: id!, keyId, enable }, {
      onSuccess: () => toast('success', `Key ${enable ? 'enabled' : 'disabled'}`),
      onError: () => toast('error', 'Failed to update key'),
    })
  }

  const handleRevokeKey = () => {
    if (!revokeKeyId) return
    revokeKeyMutation.mutate({ consumerId: id!, keyId: revokeKeyId }, {
      onSuccess: () => { setRevokeKeyId(null); toast('success', 'Key revoked') },
      onError: () => toast('error', 'Failed to revoke key'),
    })
  }

  if (isLoading) {
    return (
      <div>
        <Header breadcrumbs={[{ label: 'Consumers' }, { label: 'Loading...' }]} />
        <div className="mx-auto max-w-5xl p-6"><LoadingSkeleton /></div>
      </div>
    )
  }

  if (!consumer) {
    return (
      <div>
        <Header breadcrumbs={[{ label: 'Consumers' }, { label: 'Not Found' }]} />
        <div className="mx-auto max-w-5xl p-6"><p className="text-sm text-muted-foreground">Consumer not found.</p></div>
      </div>
    )
  }

  return (
    <div>
      <Header breadcrumbs={[{ label: 'Consumers' }, { label: consumer.username }]} action={
        <div className="flex items-center gap-2">
          <button onClick={() => setEditOpen(true)} className={cn('inline-flex items-center gap-2 rounded-md border px-3 py-1.5 text-sm font-medium text-foreground hover:bg-accent transition-colors')}>
            <Pencil className="h-3.5 w-3.5" /> Edit
          </button>
          <button onClick={() => setDeleteOpen(true)} className={cn('inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium bg-destructive text-destructive-foreground hover:bg-destructive/90 transition-colors')}>
            <Trash2 className="h-3.5 w-3.5" /> Delete
          </button>
        </div>
      } />

      <div className="mx-auto max-w-5xl p-6 space-y-6">
        {/* Info */}
        <div className="rounded-md border">
          <div className="border-b px-4 py-2">
            <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Details</h3>
          </div>
          <div className="divide-y">
            {[
              { label: 'Username', value: consumer.username },
              { label: 'Email', value: consumer.email || '—' },
              { label: 'Custom ID', value: consumer.custom_id || '—' },
              { label: 'Created', value: formatDate(consumer.created_at, 'absolute') },
              { label: 'Updated', value: formatDate(consumer.updated_at, 'absolute') },
            ].map((item) => (
              <div key={item.label} className="flex items-center justify-between px-4 py-2.5 text-sm">
                <span className="text-muted-foreground">{item.label}</span>
                <span className="text-foreground">{item.value}</span>
              </div>
            ))}
          </div>
        </div>

        {/* API Keys */}
        <div className="rounded-md border">
          <div className="flex items-center justify-between border-b px-4 py-2">
            <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">API Keys</h3>
            <button
              onClick={() => { setGeneratedKey(null); setGenerateOpen(true) }}
              className="inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              <Plus className="h-3 w-3" /> Generate Key
            </button>
          </div>
          <div className="px-4">
            <ApiKeyList
              keys={keys || []}
              onToggle={handleToggleKey}
              onRevoke={(keyId) => setRevokeKeyId(keyId)}
            />
          </div>
        </div>
      </div>

      <SlidePanel open={editOpen} onClose={() => setEditOpen(false)} title={`Edit ${consumer.username}`}>
        <ConsumerForm initial={consumer} onSubmit={handleEdit} isLoading={updateMutation.isPending} />
      </SlidePanel>

      <ConfirmDialog open={deleteOpen} onClose={() => setDeleteOpen(false)} onConfirm={handleDelete}
        title={`Delete ${consumer.username}?`} description="This will delete all API keys and plugins associated with this consumer." isLoading={deleteMutation.isPending} />

      <ApiKeyGenerateDialog
        open={generateOpen}
        onClose={() => { setGenerateOpen(false); setGeneratedKey(null) }}
        onGenerate={handleGenerate}
        generatedKey={generatedKey}
        isLoading={generateMutation.isPending}
      />

      <ConfirmDialog open={!!revokeKeyId} onClose={() => setRevokeKeyId(null)} onConfirm={handleRevokeKey}
        title="Revoke API key?" description="This key will be permanently deleted and can no longer be used for authentication."
        confirmLabel="Revoke" isLoading={revokeKeyMutation.isPending} />
    </div>
  )
}
