import { Fragment, useState } from 'react'
import { Dialog, DialogPanel, DialogTitle, Transition, TransitionChild } from '@headlessui/react'
import { Key, Copy, Check, AlertTriangle } from 'lucide-react'
import { cn } from '../../lib/cn'
import type { ApiKeyGenerated } from '../../api/types'

interface ApiKeyGenerateDialogProps {
  open: boolean
  onClose: () => void
  onGenerate: (name: string) => void
  generatedKey: ApiKeyGenerated | null
  isLoading?: boolean
}

export function ApiKeyGenerateDialog({ open, onClose, onGenerate, generatedKey, isLoading }: ApiKeyGenerateDialogProps) {
  const [name, setName] = useState('')
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    if (!generatedKey) return
    await navigator.clipboard.writeText(generatedKey.key)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleClose = () => {
    setName('')
    setCopied(false)
    onClose()
  }

  const inputClass = cn(
    'w-full rounded-md border border-input bg-background px-3 py-2 text-sm',
    'placeholder:text-muted-foreground',
    'focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background'
  )

  return (
    <Transition show={open} as={Fragment}>
      <Dialog onClose={handleClose} className="relative z-50">
        <TransitionChild as={Fragment} enter="ease-out duration-200" enterFrom="opacity-0" enterTo="opacity-100" leave="ease-in duration-150" leaveFrom="opacity-100" leaveTo="opacity-0">
          <div className="fixed inset-0 bg-black/30" />
        </TransitionChild>
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <TransitionChild as={Fragment} enter="ease-out duration-200" enterFrom="opacity-0 scale-95" enterTo="opacity-100 scale-100" leave="ease-in duration-150" leaveFrom="opacity-100 scale-100" leaveTo="opacity-0 scale-95">
            <DialogPanel className="w-full max-w-md rounded-lg border bg-background p-6 shadow-lg">
              {!generatedKey ? (
                <>
                  <div className="flex items-center gap-3 mb-4">
                    <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
                      <Key className="h-5 w-5 text-foreground" />
                    </div>
                    <DialogTitle className="text-sm font-semibold text-foreground">Generate API Key</DialogTitle>
                  </div>
                  <div className="mb-4">
                    <label className="block text-sm font-medium text-foreground mb-1.5">
                      Key Name <span className="text-muted-foreground font-normal">(optional)</span>
                    </label>
                    <input
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      placeholder="production-key"
                      className={inputClass}
                    />
                  </div>
                  <div className="flex justify-end gap-3">
                    <button onClick={handleClose} className="rounded-md border px-3 py-1.5 text-sm font-medium text-foreground hover:bg-accent transition-colors">
                      Cancel
                    </button>
                    <button
                      onClick={() => onGenerate(name)}
                      disabled={isLoading}
                      className="rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors"
                    >
                      {isLoading ? 'Generating...' : 'Generate'}
                    </button>
                  </div>
                </>
              ) : (
                <>
                  <div className="flex items-center gap-3 mb-4">
                    <div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-900/30">
                      <AlertTriangle className="h-5 w-5 text-amber-600 dark:text-amber-400" />
                    </div>
                    <div>
                      <DialogTitle className="text-sm font-semibold text-foreground">API Key Generated</DialogTitle>
                      <p className="text-xs text-destructive font-medium">Save this key now. It won't be shown again.</p>
                    </div>
                  </div>

                  <div className="rounded-md border bg-muted p-3 mb-4">
                    <div className="flex items-center justify-between gap-2">
                      <code className="text-xs font-mono text-foreground break-all">{generatedKey.key}</code>
                      <button
                        onClick={handleCopy}
                        className="shrink-0 rounded p-1.5 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
                      >
                        {copied ? <Check className="h-4 w-4 text-emerald-500" /> : <Copy className="h-4 w-4" />}
                      </button>
                    </div>
                  </div>

                  <button
                    onClick={handleClose}
                    className="w-full rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
                  >
                    Done — I've saved the key
                  </button>
                </>
              )}
            </DialogPanel>
          </TransitionChild>
        </div>
      </Dialog>
    </Transition>
  )
}
