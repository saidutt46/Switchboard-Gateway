import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate } from 'react-router-dom'
import { Eye, EyeOff, ArrowRight, AlertCircle } from 'lucide-react'
import { useAuth } from '../hooks/useAuth'
import { cn } from '../lib/cn'
import { parseApiError } from '../lib/errors'

interface LoginForm {
  username: string
  password: string
}

function NetworkPattern() {
  return (
    <svg className="absolute inset-0 w-full h-full opacity-[0.03]" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <pattern id="net" width="60" height="60" patternUnits="userSpaceOnUse">
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
      <rect width="100%" height="100%" fill="url(#net)" />
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

export function LoginPage() {
  const navigate = useNavigate()
  const { login } = useAuth()
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const { register, handleSubmit, formState: { errors } } = useForm<LoginForm>()

  const onSubmit = async (data: LoginForm) => {
    setError(null)
    setIsSubmitting(true)
    try {
      await login(data.username, data.password)
      navigate('/')
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

      {/* Ambient glow */}
      <div className="absolute top-1/4 left-1/2 -translate-x-1/2 w-[600px] h-[400px] bg-emerald-500/[0.04] rounded-full blur-[120px] pointer-events-none" />

      <div className="relative w-full max-w-sm mx-4">
        {/* Logo */}
        <div className="flex justify-center mb-8">
          <SwitchboardLogo />
        </div>

        {/* Card */}
        <div className="rounded-2xl border bg-card p-8 shadow-xl shadow-black/[0.04] dark:shadow-black/[0.2]">
          <div className="mb-6">
            <h2 className="text-base font-semibold text-foreground">Sign in</h2>
            <p className="mt-1 text-sm text-muted-foreground">Enter your credentials to access the dashboard</p>
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
                {...register('username', { required: 'Username is required' })}
                placeholder="admin"
                autoComplete="username"
                autoFocus
                className={inputClass}
              />
              {errors.username && <p className="mt-1 text-xs text-red-500">{errors.username.message}</p>}
            </div>

            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Password</label>
              <div className="relative">
                <input
                  {...register('password', { required: 'Password is required' })}
                  type={showPassword ? 'text' : 'password'}
                  placeholder="Enter password"
                  autoComplete="current-password"
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
                  Sign in
                  <ArrowRight className="h-4 w-4" />
                </>
              )}
            </button>
          </form>
        </div>

        {/* Footer */}
        <p className="mt-6 text-center text-[11px] text-muted-foreground/50 font-mono">
          Switchboard Gateway
        </p>
      </div>
    </div>
  )
}
