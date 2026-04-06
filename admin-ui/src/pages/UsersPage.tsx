import { useState } from 'react'
import { Plus, UserPlus, Trash2, Shield, Eye } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { Header } from '../components/layout/Header'
import { DataTable, type Column } from '../components/shared/DataTable'
import { SlidePanel } from '../components/shared/SlidePanel'
import { ConfirmDialog } from '../components/shared/ConfirmDialog'
import { useUsers, useCreateUser, useDeleteUser } from '../hooks/useUsers'
import { useAuth } from '../hooks/useAuth'
import { useToast } from '../components/shared/Toast'
import { formatDate } from '../lib/formatters'
import { cn } from '../lib/cn'
import type { AdminUser, AdminUserCreate } from '../api/types'

function CreateUserForm({ onSubmit, isLoading }: { onSubmit: (data: AdminUserCreate) => void; isLoading?: boolean }) {
  const { register, handleSubmit, watch, formState: { errors } } = useForm<AdminUserCreate & { confirmPassword: string }>({
    defaultValues: { role: 'admin' },
  })
  const [showPassword, setShowPassword] = useState(false)
  const watchPassword = watch('password', '')

  const inputClass = cn(
    'w-full h-10 rounded-lg border border-input bg-background px-3 text-sm',
    'placeholder:text-muted-foreground/60',
    'focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background'
  )

  return (
    <form onSubmit={handleSubmit((data) => onSubmit({ username: data.username, password: data.password, email: data.email || undefined, role: data.role }))} className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-foreground mb-1.5">Username</label>
        <input {...register('username', { required: 'Required', minLength: { value: 3, message: 'Min 3 characters' } })} placeholder="devops-engineer" className={inputClass} />
        {errors.username && <p className="mt-1 text-xs text-red-500">{errors.username.message}</p>}
      </div>
      <div>
        <label className="block text-sm font-medium text-foreground mb-1.5">Email <span className="text-muted-foreground font-normal">(optional)</span></label>
        <input {...register('email')} type="email" placeholder="user@example.com" className={inputClass} />
      </div>
      <div>
        <label className="block text-sm font-medium text-foreground mb-1.5">Password</label>
        <input
          {...register('password', {
            required: 'Required',
            minLength: { value: 8, message: 'Min 8 characters' },
            validate: {
              hasLetter: (v) => /[a-zA-Z]/.test(v) || 'Must contain a letter',
              hasNumber: (v) => /[0-9]/.test(v) || 'Must contain a number',
            },
          })}
          type={showPassword ? 'text' : 'password'}
          placeholder="Min 8 chars, letter + number"
          className={inputClass}
        />
        {errors.password && <p className="mt-1 text-xs text-red-500">{errors.password.message}</p>}
        <button type="button" onClick={() => setShowPassword(!showPassword)} className="mt-1 text-[11px] text-muted-foreground hover:text-foreground">
          {showPassword ? 'Hide' : 'Show'} password
        </button>
      </div>
      <div>
        <label className="block text-sm font-medium text-foreground mb-1.5">Confirm Password</label>
        <input
          {...register('confirmPassword', { validate: (v) => v === watchPassword || 'Passwords do not match' })}
          type={showPassword ? 'text' : 'password'}
          placeholder="Confirm password"
          className={inputClass}
        />
        {errors.confirmPassword && <p className="mt-1 text-xs text-red-500">{errors.confirmPassword.message}</p>}
      </div>
      <div>
        <label className="block text-sm font-medium text-foreground mb-1.5">Role</label>
        <select {...register('role')} className={inputClass}>
          <option value="admin">Admin — full access</option>
          <option value="viewer">Viewer — read-only</option>
        </select>
      </div>
      <button type="submit" disabled={isLoading} className={cn(
        'w-full h-10 rounded-lg text-sm font-medium',
        'bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors'
      )}>
        {isLoading ? 'Creating...' : 'Create User'}
      </button>
    </form>
  )
}

export function UsersPage() {
  const { user: currentUser } = useAuth()
  const { toast, toastApiError } = useToast()
  const { data: users, isLoading } = useUsers()
  const createMutation = useCreateUser()
  const deleteMutation = useDeleteUser()

  const [createOpen, setCreateOpen] = useState(false)
  const [deleteUser, setDeleteUser] = useState<AdminUser | null>(null)

  const handleCreate = (data: AdminUserCreate) => {
    createMutation.mutate(data, {
      onSuccess: () => { setCreateOpen(false); toast('success', 'User created') },
      onError: (err) => toastApiError(err, 'Failed to create user'),
    })
  }

  const handleDelete = () => {
    if (!deleteUser) return
    deleteMutation.mutate(deleteUser.id, {
      onSuccess: () => { setDeleteUser(null); toast('success', 'User deleted') },
      onError: (err) => toastApiError(err, 'Failed to delete user'),
    })
  }

  const columns: Column<AdminUser>[] = [
    {
      key: 'username', header: 'Username', sortable: true,
      render: (u) => (
        <div className="flex items-center gap-2">
          <span className="font-medium text-foreground">{u.username}</span>
          {u.id === currentUser?.id && (
            <span className="rounded-full bg-emerald-100 dark:bg-emerald-900/30 px-1.5 py-0.5 text-[10px] font-medium text-emerald-700 dark:text-emerald-400">you</span>
          )}
        </div>
      ),
    },
    { key: 'email', header: 'Email', render: (u) => <span className="text-muted-foreground">{u.email || '—'}</span> },
    {
      key: 'role', header: 'Role',
      render: (u) => (
        <span className={cn(
          'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium',
          u.role === 'admin' ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' : 'bg-muted text-muted-foreground'
        )}>
          {u.role === 'admin' ? <Shield className="h-3 w-3" /> : <Eye className="h-3 w-3" />}
          {u.role}
        </span>
      ),
    },
    { key: 'created_at', header: 'Created', render: (u) => <span className="text-muted-foreground">{formatDate(u.created_at)}</span> },
    {
      key: 'actions', header: '', className: 'w-12 text-right',
      render: (u) => u.id !== currentUser?.id ? (
        <button onClick={(e) => { e.stopPropagation(); setDeleteUser(u) }} className="rounded p-1.5 text-muted-foreground hover:text-destructive hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors">
          <Trash2 className="h-3.5 w-3.5" />
        </button>
      ) : null,
    },
  ]

  return (
    <div>
      <Header breadcrumbs={[{ label: 'Settings' }, { label: 'Users' }]} action={
        <button onClick={() => setCreateOpen(true)} className={cn('inline-flex items-center gap-2 rounded-lg px-3 py-1.5 text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-colors')}>
          <Plus className="h-3.5 w-3.5" /> New User
        </button>
      } />
      <div className="mx-auto max-w-5xl p-6">
        <DataTable columns={columns} data={users || []} isLoading={isLoading} keyExtractor={(u) => u.id}
          emptyState={{ title: 'No users', description: 'Create admin users to manage the gateway.', icon: <UserPlus className="h-8 w-8" />,
            action: <button onClick={() => setCreateOpen(true)} className="rounded-lg bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90">Create User</button> }}
        />
      </div>
      <SlidePanel open={createOpen} onClose={() => setCreateOpen(false)} title="New User">
        <CreateUserForm onSubmit={handleCreate} isLoading={createMutation.isPending} />
      </SlidePanel>
      <ConfirmDialog open={!!deleteUser} onClose={() => setDeleteUser(null)} onConfirm={handleDelete}
        title={`Delete ${deleteUser?.username}?`} description="This user will lose all access to the admin dashboard." isLoading={deleteMutation.isPending} />
    </div>
  )
}
