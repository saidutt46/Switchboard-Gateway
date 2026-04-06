import { useNavigate } from 'react-router-dom'
import { Header } from '../components/layout/Header'

export function NotFoundPage() {
  const navigate = useNavigate()
  return (
    <div>
      <Header breadcrumbs={[{ label: '404' }]} />
      <div className="flex flex-col items-center justify-center py-24 text-center">
        <h1 className="text-4xl font-bold text-foreground">404</h1>
        <p className="mt-2 text-sm text-muted-foreground">Page not found</p>
        <button
          onClick={() => navigate('/')}
          className="mt-4 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
        >
          Go to Dashboard
        </button>
      </div>
    </div>
  )
}
