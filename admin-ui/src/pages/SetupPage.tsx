import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Eye, EyeOff, ShieldCheck, AlertCircle, CheckCircle2 } from 'lucide-react'
import { setup as apiSetup } from '../api/auth'
import { cn } from '../lib/cn'
import { parseApiError } from '../lib/errors'

interface SetupForm {
  username: string
  email: string
  password: string
  confirmPassword: string
}

function NetworkPattern() {
  return (
    <svg className="absolute inset-0 w-full h-full opacity-[0.03]" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <pattern id="net-setup" width="60" height="60" patternUnits="userSpaceOnUse">
          <circle cx="30" cy="30" r="1.5" fill="currentColor" />
          <circle cx="0" cy="0" r="1" fill="currentColor" />
          <circle cx="60" cy="0" r="1" fill="currentColor" />
          <circle cx="0" cy="60" r="1" fill="currentColor" />
          <circle cx="60" cy="60" r="1" fill="currentColor" />
          <line x1="0" y1="0" x2="30" y2="30" stroke="currentColor" strokeWidth="0.5" />
          <line x1="60" y1="0" x2="30" y2="30" stroke="currentColor" strokeWidth="0.5" />
          <line x1="0" y1="60" x2="30" y2="30" stroke="currentColor" strokeWidth="0.5" />
          <line x1="60" y1="60" x2="30" y2="30" stroke="currentColor" strokeWidth="0.5" />
        </pattern>
      </defs>
      <rect width="100%" height="100%" fill="url(#net-setup)" />
    </svg>
  )
}

function SwitchboardLogo() {
  return (
    <div className="flex items-center gap-3">
      <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-emerald-500 to-teal-600 shadow-lg shadow-emerald-500/20">
        <svg viewBox="0 0 24 24" fill="none" className="h-5 w-5 text-white" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <circle cx="12" cy="12" r="3" fill="currentColor" opacity="0.3" />
          <circle cx="12" cy="12" r="3" />
          <circle cx="4" cy="6" r="1.5" />
          <circle cx="4" cy="18" r="1.5" />
          <circle cx="20" cy="6" r="1.5" />
          <circle cx="20" cy="18" r="1.5" />
          <line x1="5.5" y1="6" x2="9" y2="10.5" />
          <line x1="5.5" y1="18" x2="9" y2="13.5" />
          <line x1="15" y1="10.5" x2="18.5" y2="6" />
          <line x1="15" y1="13.5" x2="18.5" y2="18" />
        </svg>
      </div>
      <div>
        <h1 className="text-lg font-bold text-foreground tracking-tight">Switchboard</h1>
        <p className="text-[11px] text-muted-foreground font-medium tracking-wide uppercase">API Gateway</p>
      </div>
    </div>
  )
}

const PASSWORD_RULES = [
  { test: (p: string) => p.length >= 8, label: 'At least 8 characters' },
  { test: (p: string) => /[a-zA-Z]/.test(p), label: 'Contains a letter' },
  { test: (p: string) => /[0-9]/.test(p), label: 'Contains a number' },
]

export function SetupPage() {
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [success, setSuccess] = useState(false)

  const { register, handleSubmit, watch, formState: { errors } } = useForm<SetupForm>()
  const watchPassword = watch('password', '')

  const onSubmit = async (data: SetupForm) => {
    if (data.password !== data.confirmPassword) return
    setError(null)
    setIsSubmitting(true)
    try {
      await apiSetup({
        username: data.username,
        password: data.password,
        email: data.email || undefined,
      })
      setSuccess(true)
      // Full reload so AuthGuard re-checks /auth/status
      setTimeout(() => { window.location.href = '/login' }, 2000)
    } catch (err) {
      const message = await parseApiError(err)
      setError(message)
    } finally {
      setIsSubmitting(false)
    }
  }

  const inputClass = cn(
    'w-full h-11 rounded-lg border border-input bg-background px-4 text-sm text-foreground',
    'placeholder:text-muted-foreground/60',
    'focus:outline-none focus:ring-2 focus:ring-emerald-500/30 focus:border-emerald-500/50',
    'transition-all duration-150'
  )

  return (
    <div className="flex min-h-screen items-center justify-center bg-background relative overflow-hidden">
      <NetworkPattern />

      <div className="absolute top-1/3 left-1/2 -translate-x-1/2 w-[600px] h-[400px] bg-emerald-500/[0.04] rounded-full blur-[120px] pointer-events-none" />

      <div className="relative w-full max-w-sm mx-4">
        {/* Logo */}
        <div className="flex justify-center mb-8">
          <SwitchboardLogo />
        </div>

        {/* Card */}
        <div className="rounded-2xl border bg-card p-8 shadow-xl shadow-black/[0.04] dark:shadow-black/[0.2]">
          {/* Success state */}
          {success ? (
            <div className="flex flex-col items-center py-4 text-center">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-emerald-100 dark:bg-emerald-900/30 mb-4">
                <CheckCircle2 className="h-6 w-6 text-emerald-600 dark:text-emerald-400" />
              </div>
              <h2 className="text-base font-semibold text-foreground">Account Created</h2>
              <p className="mt-1.5 text-sm text-muted-foreground">Redirecting to login...</p>
              <div className="mt-4 h-1 w-24 rounded-full bg-muted overflow-hidden">
                <div className="h-full bg-emerald-500 rounded-full animate-[fill_2s_ease-in-out_forwards]" style={{ animation: 'fill 2s ease-in-out forwards' }} />
              </div>
            </div>
          ) : (
          <>
          {/* Header with shield icon */}
          <div className="mb-6">
            <div className="flex items-center gap-2 mb-2">
              <ShieldCheck className="h-4.5 w-4.5 text-emerald-500" />
              <h2 className="text-base font-semibold text-foreground">Initial Setup</h2>
            </div>
            <p className="text-sm text-muted-foreground leading-relaxed">
              Create the first admin account. This user will have full access to manage the gateway.
            </p>
          </div>

          {/* Error */}
          {error && (
            <div className="mb-4 flex items-start gap-2.5 rounded-lg border border-red-200 dark:border-red-900/50 bg-red-50 dark:bg-red-950/30 px-3.5 py-3">
              <AlertCircle className="h-4 w-4 shrink-0 text-red-500 mt-0.5" />
              <p className="text-sm text-red-700 dark:text-red-400">{error}</p>
            </div>
          )}

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Username</label>
              <input
                {...register('username', {
                  required: 'Username is required',
                  minLength: { value: 3, message: 'At least 3 characters' },
                })}
                placeholder="admin"
                autoComplete="username"
                autoFocus
                className={inputClass}
              />
              {errors.username && <p className="mt-1 text-xs text-red-500">{errors.username.message}</p>}
            </div>

            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">
                Email <span className="text-muted-foreground font-normal">(optional)</span>
              </label>
              <input
                {...register('email')}
                type="email"
                placeholder="admin@example.com"
                autoComplete="email"
                className={inputClass}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Password</label>
              <div className="relative">
                <input
                  {...register('password', {
                    required: 'Password is required',
                    minLength: { value: 8, message: 'At least 8 characters' },
                    validate: {
                      hasLetter: (v) => /[a-zA-Z]/.test(v) || 'Must contain a letter',
                      hasNumber: (v) => /[0-9]/.test(v) || 'Must contain a number',
                    },
                  })}
                  type={showPassword ? 'text' : 'password'}
                  placeholder="Create a strong password"
                  autoComplete="new-password"
                  className={cn(inputClass, 'pr-10')}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground/50 hover:text-foreground transition-colors"
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              {errors.password && <p className="mt-1 text-xs text-red-500">{errors.password.message}</p>}

              {/* Password strength indicators */}
              {watchPassword && (
                <div className="mt-2 space-y-1">
                  {PASSWORD_RULES.map((rule) => {
                    const passed = rule.test(watchPassword)
                    return (
                      <div key={rule.label} className="flex items-center gap-1.5">
                        <CheckCircle2 className={cn('h-3 w-3', passed ? 'text-emerald-500' : 'text-muted-foreground/30')} />
                        <span className={cn('text-[11px]', passed ? 'text-emerald-600 dark:text-emerald-400' : 'text-muted-foreground/50')}>
                          {rule.label}
                        </span>
                      </div>
                    )
                  })}
                </div>
              )}
            </div>

            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Confirm Password</label>
              <input
                {...register('confirmPassword', {
                  required: 'Please confirm your password',
                  validate: (v) => v === watchPassword || 'Passwords do not match',
                })}
                type={showPassword ? 'text' : 'password'}
                placeholder="Confirm password"
                autoComplete="new-password"
                className={inputClass}
              />
              {errors.confirmPassword && <p className="mt-1 text-xs text-red-500">{errors.confirmPassword.message}</p>}
            </div>

            <button
              type="submit"
              disabled={isSubmitting}
              className={cn(
                'w-full h-11 rounded-lg text-sm font-semibold',
                'bg-gradient-to-r from-emerald-600 to-teal-600 text-white',
                'hover:from-emerald-500 hover:to-teal-500',
                'focus:outline-none focus:ring-2 focus:ring-emerald-500/50 focus:ring-offset-2 focus:ring-offset-background',
                'disabled:opacity-50 disabled:cursor-not-allowed',
                'transition-all duration-150',
                'flex items-center justify-center gap-2',
                'shadow-lg shadow-emerald-600/20'
              )}
            >
              {isSubmitting ? (
                <div className="h-4 w-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
              ) : (
                <>
                  <ShieldCheck className="h-4 w-4" />
                  Create Admin Account
                </>
              )}
            </button>
          </form>
          </>
          )}
        </div>

        <p className="mt-6 text-center text-[11px] text-muted-foreground/50 font-mono">
          Switchboard Gateway
        </p>
      </div>
    </div>
  )
}
