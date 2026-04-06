import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AppShell } from './components/layout/AppShell'
import { ToastProvider } from './components/shared/Toast'
import { AuthProvider } from './hooks/useAuth'
import { AuthGuard } from './components/auth/AuthGuard'
import { ProtectedRoute } from './components/auth/ProtectedRoute'
import { LoginPage } from './pages/LoginPage'
import { SetupPage } from './pages/SetupPage'
import { DashboardPage } from './pages/DashboardPage'
import { ServicesPage } from './pages/ServicesPage'
import { ServiceDetailPage } from './pages/ServiceDetailPage'
import { RoutesPage } from './pages/RoutesPage'
import { RouteDetailPage } from './pages/RouteDetailPage'
import { ConsumersPage } from './pages/ConsumersPage'
import { ConsumerDetailPage } from './pages/ConsumerDetailPage'
import { PluginsPage } from './pages/PluginsPage'
import { PluginDetailPage } from './pages/PluginDetailPage'
import { NotFoundPage } from './pages/NotFoundPage'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30000,
      retry: 1,
    },
  },
})

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <AuthProvider>
          <BrowserRouter>
            <AuthGuard>
              <Routes>
                {/* Public auth pages */}
                <Route path="/login" element={<LoginPage />} />
                <Route path="/setup" element={<SetupPage />} />

                {/* Protected app — requires authentication */}
                <Route element={
                  <ProtectedRoute>
                    <AppShell />
                  </ProtectedRoute>
                }>
                  <Route path="/" element={<DashboardPage />} />
                  <Route path="/services" element={<ServicesPage />} />
                  <Route path="/services/:id" element={<ServiceDetailPage />} />
                  <Route path="/routes" element={<RoutesPage />} />
                  <Route path="/routes/:id" element={<RouteDetailPage />} />
                  <Route path="/consumers" element={<ConsumersPage />} />
                  <Route path="/consumers/:id" element={<ConsumerDetailPage />} />
                  <Route path="/plugins" element={<PluginsPage />} />
                  <Route path="/plugins/:id" element={<PluginDetailPage />} />
                  <Route path="*" element={<NotFoundPage />} />
                </Route>
              </Routes>
            </AuthGuard>
          </BrowserRouter>
        </AuthProvider>
      </ToastProvider>
    </QueryClientProvider>
  )
}
