import { useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Puzzle, Trash2, Pencil } from 'lucide-react'
import { SearchInput } from '../components/shared/SearchInput'
import { Header } from '../components/layout/Header'
import { DataTable, type Column } from '../components/shared/DataTable'
import { StatusBadge } from '../components/shared/StatusBadge'
import { SlidePanel } from '../components/shared/SlidePanel'
import { ConfirmDialog } from '../components/shared/ConfirmDialog'
import { PluginForm } from '../components/plugins/PluginForm'
import { usePlugins, useCreatePlugin, useUpdatePlugin, useDeletePlugin } from '../hooks/usePlugins'
import { useToast } from '../components/shared/Toast'
import { shortId } from '../lib/formatters'
import { cn } from '../lib/cn'
import type { Plugin, PluginCreate } from '../api/types'

export function PluginsPage() {
  const navigate = useNavigate()
  const { toast, toastApiError } = useToast()
  const { data: plugins, isLoading } = usePlugins()
  const createMutation = useCreatePlugin()
  const updateMutation = useUpdatePlugin()
  const deleteMutation = useDeletePlugin()

  const [search, setSearch] = useState('')
  const [scopeFilter, setScopeFilter] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [editPlugin, setEditPlugin] = useState<Plugin | null>(null)
  const [deletePlugin, setDeletePlugin] = useState<Plugin | null>(null)

  const filteredPlugins = useMemo(() => {
    let result = plugins || []
    if (scopeFilter) result = result.filter((p) => p.scope === scopeFilter)
    if (search) result = result.filter((p) => p.name.toLowerCase().includes(search.toLowerCase()))
    return result
  }, [plugins, search, scopeFilter])

  const handleCreate = (data: PluginCreate) => {
    createMutation.mutate(data, {
      onSuccess: () => { setCreateOpen(false); toast('success', 'Plugin created') },
      onError: (err) => toastApiError(err, 'Failed to create plugin'),
    })
  }

  const handleEdit = (data: PluginCreate) => {
    if (!editPlugin) return
    updateMutation.mutate({ id: editPlugin.id, data }, {
      onSuccess: () => { setEditPlugin(null); toast('success', 'Plugin updated') },
      onError: (err) => toastApiError(err, 'Failed to update plugin'),
    })
  }

  const handleToggle = (plugin: Plugin) => {
    updateMutation.mutate(
      { id: plugin.id, data: { enabled: !plugin.enabled } },
      {
        onSuccess: () => toast('success', `Plugin ${plugin.enabled ? 'disabled' : 'enabled'}`),
        onError: (err) => toastApiError(err, 'Failed to update plugin'),
      }
    )
  }

  const handleDelete = () => {
    if (!deletePlugin) return
    deleteMutation.mutate(deletePlugin.id, {
      onSuccess: () => { setDeletePlugin(null); toast('success', 'Plugin deleted') },
      onError: (err) => toastApiError(err, 'Failed to delete plugin'),
    })
  }

  const columns: Column<Plugin>[] = [
    { key: 'name', header: 'Plugin', sortable: true, render: (p) => <span className="font-medium text-foreground">{p.name}</span> },
    {
      key: 'scope', header: 'Scope',
      render: (p) => <span className="inline-flex rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">{p.scope}</span>,
    },
    {
      key: 'target', header: 'Target',
      render: (p) => {
        const targetId = p.service_id || p.route_id || p.consumer_id
        return <span className="font-mono text-xs text-muted-foreground">{targetId ? shortId(targetId) : '—'}</span>
      },
    },
    { key: 'priority', header: 'Priority', sortable: true, render: (p) => <span className="text-muted-foreground">{p.priority}</span> },
    {
      key: 'enabled', header: 'Status',
      render: (p) => <button onClick={(e) => { e.stopPropagation(); handleToggle(p) }}><StatusBadge enabled={p.enabled} /></button>,
    },
    {
      key: 'actions', header: '', className: 'w-20 text-right',
      render: (p) => (
        <div className="flex items-center justify-end gap-1">
          <button onClick={(e) => { e.stopPropagation(); setEditPlugin(p) }} className="rounded p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors">
            <Pencil className="h-3.5 w-3.5" />
          </button>
          <button onClick={(e) => { e.stopPropagation(); setDeletePlugin(p) }} className="rounded p-1.5 text-muted-foreground hover:text-destructive hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors">
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      ),
    },
  ]

  return (
    <div>
      <Header breadcrumbs={[{ label: 'Plugins' }]} action={
        <button onClick={() => setCreateOpen(true)} className={cn('inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-colors')}>
          <Plus className="h-3.5 w-3.5" /> New Plugin
        </button>
      } />
      <div className="mx-auto max-w-5xl p-6 space-y-4">
        <div className="flex items-center gap-3">
          <SearchInput value={search} onChange={setSearch} placeholder="Search plugins..." className="max-w-xs" />
          <select
            value={scopeFilter}
            onChange={(e) => setScopeFilter(e.target.value)}
            className="h-9 rounded-md border border-input bg-background px-3 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          >
            <option value="">All scopes</option>
            <option value="global">Global</option>
            <option value="service">Service</option>
            <option value="route">Route</option>
            <option value="consumer">Consumer</option>
          </select>
        </div>
        <DataTable columns={columns} data={filteredPlugins} isLoading={isLoading} keyExtractor={(p) => p.id} onRowClick={(p) => navigate(`/plugins/${p.id}`)}
          emptyState={{ title: 'No plugins configured', description: 'Add plugins to enable authentication, rate limiting, caching, and more.', icon: <Puzzle className="h-8 w-8" />,
            action: <button onClick={() => setCreateOpen(true)} className="rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90">Add Plugin</button> }}
        />
      </div>
      <SlidePanel open={createOpen} onClose={() => setCreateOpen(false)} title="New Plugin">
        <PluginForm onSubmit={handleCreate} isLoading={createMutation.isPending} />
      </SlidePanel>
      <SlidePanel open={!!editPlugin} onClose={() => setEditPlugin(null)} title={`Edit ${editPlugin?.name || 'Plugin'}`}>
        {editPlugin && <PluginForm initial={editPlugin} onSubmit={handleEdit} isLoading={updateMutation.isPending} />}
      </SlidePanel>
      <ConfirmDialog open={!!deletePlugin} onClose={() => setDeletePlugin(null)} onConfirm={handleDelete}
        title={`Delete ${deletePlugin?.name || 'plugin'}?`} description="This plugin will be removed and the gateway will reload its configuration." isLoading={deleteMutation.isPending} />
    </div>
  )
}
