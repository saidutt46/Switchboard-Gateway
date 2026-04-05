import { createContext, useCallback, useContext, useState, type ReactNode } from 'react'
import { Transition } from '@headlessui/react'
import { CheckCircle, XCircle, X } from 'lucide-react'
import { cn } from '../../lib/cn'

interface Toast {
  id: number
  type: 'success' | 'error'
  message: string
}

interface ToastContextValue {
  toast: (type: Toast['type'], message: string) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

export function useToast() {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within ToastProvider')
  return ctx
}

let nextId = 0

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const toast = useCallback((type: Toast['type'], message: string) => {
    const id = nextId++
    setToasts((prev) => [...prev, { id, type, message }])
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id))
    }, 4000)
  }, [])

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      <div className="fixed bottom-4 right-4 z-50 space-y-2">
        {toasts.map((t) => (
          <Transition
            key={t.id}
            appear
            show
            enter="transform transition ease-out duration-200"
            enterFrom="translate-y-2 opacity-0"
            enterTo="translate-y-0 opacity-100"
          >
            <div
              className={cn(
                'flex items-center gap-3 rounded-md border px-4 py-3 shadow-lg',
                'bg-background text-foreground min-w-[300px]'
              )}
            >
              {t.type === 'success' ? (
                <CheckCircle className="h-4 w-4 shrink-0 text-emerald-500" />
              ) : (
                <XCircle className="h-4 w-4 shrink-0 text-destructive" />
              )}
              <span className="flex-1 text-sm">{t.message}</span>
              <button
                onClick={() => dismiss(t.id)}
                className="shrink-0 text-muted-foreground hover:text-foreground"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </div>
          </Transition>
        ))}
      </div>
    </ToastContext.Provider>
  )
}
