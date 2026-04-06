import { useEffect, useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { getAuthStatus } from '../../api/auth'
import { useAuth } from '../../hooks/useAuth'

/**
 * AuthGuard — top-level routing logic:
 * 1. If no users exist → redirect to /setup
 * 2. If not authenticated → redirect to /login
 * 3. If authenticated → show children (the app)
 */
export function AuthGuard({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading: authLoading } = useAuth()
  const [setupRequired, setSetupRequired] = useState<boolean | null>(null)
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    const checkSetup = async () => {
      try {
        const status = await getAuthStatus()
        setSetupRequired(status.setup_required)
      } catch {
        // If we can't reach the API, proceed — the CRUD calls will fail with proper errors
        setSetupRequired(false)
      }
    }
    checkSetup()
  }, [])

  useEffect(() => {
    if (setupRequired === null || authLoading) return

    const isAuthPage = location.pathname === '/login' || location.pathname === '/setup'

    if (setupRequired && location.pathname !== '/setup') {
      navigate('/setup', { replace: true })
    } else if (!setupRequired && !isAuthenticated && location.pathname !== '/login') {
      navigate('/login', { replace: true })
    } else if (!setupRequired && isAuthenticated && isAuthPage) {
      navigate('/', { replace: true })
    }
  }, [setupRequired, isAuthenticated, authLoading, location.pathname, navigate])

  // Loading state
  if (setupRequired === null || authLoading) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <div className="flex flex-col items-center gap-3">
          <div className="h-6 w-6 border-2 border-muted-foreground/20 border-t-foreground rounded-full animate-spin" />
          <p className="text-sm text-muted-foreground">Connecting...</p>
        </div>
      </div>
    )
  }

  return <>{children}</>
}
