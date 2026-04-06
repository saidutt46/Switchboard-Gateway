import { Moon, Sun } from 'lucide-react'
import { useTheme } from '../../hooks/useTheme'
import { cn } from '../../lib/cn'

export function ThemeToggle({ className }: { className?: string }) {
  const { theme, toggleTheme } = useTheme()

  return (
    <button
      onClick={toggleTheme}
      className={cn(
        'inline-flex items-center justify-center rounded-md p-2',
        'text-muted-foreground hover:text-foreground hover:bg-accent',
        'transition-colors',
        className
      )}
      aria-label={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
    >
      {theme === 'light' ? <Moon className="h-4 w-4" /> : <Sun className="h-4 w-4" />}
    </button>
  )
}
