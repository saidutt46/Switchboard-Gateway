import { createContext, useContext, useState, useCallback } from 'react'

interface SidebarContextValue {
  collapsed: boolean
  toggle: () => void
}

export const SidebarContext = createContext<SidebarContextValue>({ collapsed: false, toggle: () => {} })

export function useSidebar() {
  return useContext(SidebarContext)
}

export function useSidebarState() {
  const [collapsed, setCollapsed] = useState(() => {
    if (typeof window === 'undefined') return false
    return localStorage.getItem('sidebar-collapsed') === 'true'
  })

  const toggle = useCallback(() => {
    setCollapsed((prev) => {
      const next = !prev
      localStorage.setItem('sidebar-collapsed', String(next))
      return next
    })
  }, [])

  return { collapsed, toggle }
}
