import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Header } from '../components/layout/Header'
import { useAuth } from '../hooks/useAuth'
import { useUpdateProfile } from '../hooks/useUsers'
import { useToast } from '../components/shared/Toast'
import { formatDate } from '../lib/formatters'
import { cn } from '../lib/cn'

interface PasswordForm {
  password: string
  confirmPassword: string
}

interface EmailForm {
  email: string
}

export function ProfilePage() {
  const { user } = useAuth()
  const { toast, toastApiError } = useToast()
  const updateMutation = useUpdateProfile()
  const [showPassword, setShowPassword] = useState(false)

  const passwordForm = useForm<PasswordForm>()
  const emailForm = useForm<EmailForm>({ defaultValues: { email: user?.email || '' } })

  const handlePasswordChange = (data: PasswordForm) => {
    if (data.password !== data.confirmPassword) return
    updateMutation.mutate({ password: data.password }, {
      onSuccess: () => {
        toast('success', 'Password updated')
        passwordForm.reset()
      },
      onError: (err) => toastApiError(err, 'Failed to update password'),
    })
  }

  const handleEmailChange = (data: EmailForm) => {
    updateMutation.mutate({ email: data.email || undefined }, {
      onSuccess: () => toast('success', 'Email updated'),
      onError: (err) => toastApiError(err, 'Failed to update email'),
    })
  }

  const inputClass = cn(
    'w-full h-10 rounded-lg border border-input bg-background px-3 text-sm',
    'placeholder:text-muted-foreground/60',
    'focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background'
  )

  if (!user) return null

  return (
    <div>
      <Header breadcrumbs={[{ label: 'Settings' }, { label: 'Profile' }]} />
      <div className="mx-auto max-w-2xl p-6 space-y-6">
        {/* Profile info */}
        <div className="rounded-xl border bg-card overflow-hidden">
          <div className="border-b px-5 py-3.5">
            <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Account</h3>
          </div>
          <div className="divide-y">
            {[
              { label: 'Username', value: user.username },
              { label: 'Role', value: user.role },
              { label: 'Created', value: formatDate(user.created_at, 'absolute') },
            ].map((item) => (
              <div key={item.label} className="flex items-center justify-between px-5 py-3 text-sm">
                <span className="text-muted-foreground">{item.label}</span>
                <span className="text-foreground font-medium">{item.value}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Update email */}
        <div className="rounded-xl border bg-card overflow-hidden">
          <div className="border-b px-5 py-3.5">
            <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Email</h3>
          </div>
          <form onSubmit={emailForm.handleSubmit(handleEmailChange)} className="p-5 space-y-3">
            <input {...emailForm.register('email')} type="email" placeholder="your@email.com" className={inputClass} />
            <button type="submit" disabled={updateMutation.isPending} className={cn(
              'rounded-lg px-4 py-2 text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors'
            )}>
              Update Email
            </button>
          </form>
        </div>

        {/* Change password */}
        <div className="rounded-xl border bg-card overflow-hidden">
          <div className="border-b px-5 py-3.5">
            <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Change Password</h3>
          </div>
          <form onSubmit={passwordForm.handleSubmit(handlePasswordChange)} className="p-5 space-y-3">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">New Password</label>
              <input
                {...passwordForm.register('password', {
                  required: 'Required',
                  minLength: { value: 8, message: 'Min 8 characters' },
                  validate: {
                    hasLetter: (v) => /[a-zA-Z]/.test(v) || 'Must contain a letter',
                    hasNumber: (v) => /[0-9]/.test(v) || 'Must contain a number',
                  },
                })}
                type={showPassword ? 'text' : 'password'}
                placeholder="New password"
                className={inputClass}
              />
              {passwordForm.formState.errors.password && (
                <p className="mt-1 text-xs text-red-500">{passwordForm.formState.errors.password.message}</p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Confirm Password</label>
              <input
                {...passwordForm.register('confirmPassword', {
                  validate: (v) => v === passwordForm.watch('password') || 'Passwords do not match',
                })}
                type={showPassword ? 'text' : 'password'}
                placeholder="Confirm new password"
                className={inputClass}
              />
              {passwordForm.formState.errors.confirmPassword && (
                <p className="mt-1 text-xs text-red-500">{passwordForm.formState.errors.confirmPassword.message}</p>
              )}
            </div>
            <div className="flex items-center gap-3">
              <button type="submit" disabled={updateMutation.isPending} className={cn(
                'rounded-lg px-4 py-2 text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors'
              )}>
                Change Password
              </button>
              <button type="button" onClick={() => setShowPassword(!showPassword)} className="text-xs text-muted-foreground hover:text-foreground">
                {showPassword ? 'Hide' : 'Show'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
