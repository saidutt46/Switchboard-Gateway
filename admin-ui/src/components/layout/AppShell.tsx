import { Outlet } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { SidebarContext, useSidebarState } from '../../hooks/useSidebar'

export function AppShell() {
  const sidebarState = useSidebarState()

  return (
    <SidebarContext.Provider value={sidebarState}>
      <div className="flex h-screen overflow-hidden">
        <Sidebar />
        <main className="flex-1 overflow-y-auto bg-background">
          <Outlet />
        </main>
      </div>
    </SidebarContext.Provider>
  )
}
