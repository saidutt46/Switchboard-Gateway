import { Trash2 } from 'lucide-react'
import { StatusBadge } from '../shared/StatusBadge'
import { formatDate } from '../../lib/formatters'
import { cn } from '../../lib/cn'
import type { ApiKey } from '../../api/types'

interface ApiKeyListProps {
  keys: ApiKey[]
  onToggle: (keyId: string, enable: boolean) => void
  onRevoke: (keyId: string) => void
}

export function ApiKeyList({ keys, onToggle, onRevoke }: ApiKeyListProps) {
  if (keys.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-4 text-center">
        No API keys. Generate one to get started.
      </p>
    )
  }

  return (
    <div className="divide-y">
      {keys.map((key) => (
        <div key={key.id} className="flex items-center justify-between py-3 px-1">
          <div className="space-y-0.5">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-foreground">{key.name || 'Unnamed key'}</span>
              <button onClick={() => onToggle(key.id, !key.enabled)}>
                <StatusBadge enabled={key.enabled} />
              </button>
            </div>
            <div className="flex items-center gap-3 text-xs text-muted-foreground">
              <span className="font-mono">{key.key_preview}...</span>
              <span>Created {formatDate(key.created_at)}</span>
              {key.last_used_at && <span>Last used {formatDate(key.last_used_at)}</span>}
            </div>
          </div>
          <button
            onClick={() => onRevoke(key.id)}
            className={cn(
              'rounded p-1.5 text-muted-foreground',
              'hover:text-destructive hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors'
            )}
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      ))}
    </div>
  )
}
