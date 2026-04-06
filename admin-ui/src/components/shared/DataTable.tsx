import { useState, type ReactNode } from 'react'
import { ArrowUpDown } from 'lucide-react'
import { cn } from '../../lib/cn'
import { LoadingSkeleton } from './LoadingSkeleton'
import { EmptyState } from './EmptyState'

export interface Column<T> {
  key: string
  header: string
  render: (item: T) => ReactNode
  sortable?: boolean
  className?: string
}

interface DataTableProps<T> {
  columns: Column<T>[]
  data: T[]
  isLoading?: boolean
  onRowClick?: (item: T) => void
  emptyState?: { title: string; description: string; icon?: ReactNode; action?: ReactNode }
  keyExtractor: (item: T) => string
}

export function DataTable<T>({
  columns,
  data,
  isLoading,
  onRowClick,
  emptyState,
  keyExtractor,
}: DataTableProps<T>) {
  const [sortKey, setSortKey] = useState<string | null>(null)
  const [_sortDir, setSortDir] = useState<'asc' | 'desc'>('asc')
  void _sortDir

  const handleSort = (key: string) => {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir('asc')
    }
  }

  if (isLoading) {
    return (
      <div className="rounded-xl border bg-card p-4">
        <LoadingSkeleton rows={5} cols={columns.length} />
      </div>
    )
  }

  if (data.length === 0 && emptyState) {
    return (
      <div className="rounded-xl border bg-card">
        <EmptyState {...emptyState} />
      </div>
    )
  }

  return (
    <div className="rounded-xl border bg-card">
      <table className="w-full">
        <thead>
          <tr className="border-b bg-muted/50">
            {columns.map((col) => (
              <th
                key={col.key}
                className={cn(
                  'px-4 py-3 text-left text-xs font-medium text-muted-foreground',
                  col.sortable && 'cursor-pointer select-none hover:text-foreground',
                  col.className
                )}
                onClick={col.sortable ? () => handleSort(col.key) : undefined}
              >
                <span className="flex items-center gap-1">
                  {col.header}
                  {col.sortable && <ArrowUpDown className="h-3 w-3" />}
                </span>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.map((item) => (
            <tr
              key={keyExtractor(item)}
              className={cn(
                'border-b last:border-0 transition-colors',
                onRowClick && 'cursor-pointer hover:bg-muted/50'
              )}
              onClick={onRowClick ? () => onRowClick(item) : undefined}
            >
              {columns.map((col) => (
                <td key={col.key} className={cn('px-4 py-3 text-sm', col.className)}>
                  {col.render(item)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
