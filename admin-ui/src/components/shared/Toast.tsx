import { createContext, useCallback, useContext, useState, useEffect, type ReactNode } from 'react'
import { CheckCircle, XCircle, AlertTriangle, Info, X } from 'lucide-react'
import { cn } from '../../lib/cn'
import { parseApiError } from '../../lib/errors'

type ToastType = 'success' | 'error' | 'warning' | 'info'

interface Toast {
  id: number
  type: ToastType
  title: string
  description?: string
  action?: { label: string; onClick: () => void }
  duration: number
}

interface ToastContextValue {
  toast: (type: ToastType, title: string, options?: { description?: string; action?: Toast['action']; duration?: number }) => void
  toastApiError: (error: unknown, fallbackTitle?: string) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

export function useToast() {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within ToastProvider')
  return ctx
}

const ICONS: Record<ToastType, React.ElementType> = {
  success: CheckCircle,
  error: XCircle,
  warning: AlertTriangle,
  info: Info,
}

const ICON_COLORS: Record<ToastType, string> = {
  success: 'text-emerald-500',
  error: 'text-red-500',
  warning: 'text-amber-500',
  info: 'text-blue-500',
}

const BORDER_COLORS: Record<ToastType, string> = {
  success: 'border-l-emerald-500',
  error: 'border-l-red-500',
  warning: 'border-l-amber-500',
  info: 'border-l-blue-500',
}

const DEFAULT_DURATIONS: Record<ToastType, number> = {
  success: 4000,
  error: 8000,
  warning: 6000,
  info: 5000,
}

let nextId = 0

function ToastItem({ toast: t, onDismiss }: { toast: Toast; onDismiss: () => void }) {
  const [visible, setVisible] = useState(false)
  const [exiting, setExiting] = useState(false)
  const Icon = ICONS[t.type]

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true))
    const timer = setTimeout(() => {
      setExiting(true)
      setTimeout(onDismiss, 200)
    }, t.duration)
    return () => clearTimeout(timer)
  }, [t.duration, onDismiss])

  const handleDismiss = () => {
    setExiting(true)
    setTimeout(onDismiss, 200)
  }

  return (
    <div
      className={cn(
        'flex gap-3 rounded-lg border border-l-[3px] bg-card px-4 py-3 shadow-lg backdrop-blur-sm',
        'min-w-[340px] max-w-[420px]',
        'transition-all duration-200',
        BORDER_COLORS[t.type],
        visible && !exiting ? 'translate-y-0 opacity-100' : '-translate-y-2 opacity-0',
      )}
    >
      <Icon className={cn('h-4.5 w-4.5 shrink-0 mt-0.5', ICON_COLORS[t.type])} />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-foreground">{t.title}</p>
        {t.description && (
          <p className="mt-0.5 text-xs text-muted-foreground leading-relaxed">{t.description}</p>
        )}
        {t.action && (
          <button
            onClick={() => { t.action!.onClick(); handleDismiss() }}
            className="mt-2 text-xs font-semibold text-primary hover:underline"
          >
            {t.action.label}
          </button>
        )}
      </div>
      <button
        onClick={handleDismiss}
        className="shrink-0 rounded-md p-0.5 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  )
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const toast = useCallback((
    type: ToastType,
    title: string,
    options?: { description?: string; action?: Toast['action']; duration?: number }
  ) => {
    const id = nextId++
    setToasts((prev) => [
      ...prev,
      {
        id,
        type,
        title,
        description: options?.description,
        action: options?.action,
        duration: options?.duration || DEFAULT_DURATIONS[type],
      },
    ])
  }, [])

  const toastApiError = useCallback(async (error: unknown, fallbackTitle?: string) => {
    const message = await parseApiError(error)
    toast('error', fallbackTitle || 'Request failed', { description: message })
  }, [toast])

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  return (
    <ToastContext.Provider value={{ toast, toastApiError }}>
      {children}
      <div className="fixed top-16 right-6 z-[100] flex flex-col gap-2">
        {toasts.map((t) => (
          <ToastItem key={t.id} toast={t} onDismiss={() => dismiss(t.id)} />
        ))}
      </div>
    </ToastContext.Provider>
  )
}
