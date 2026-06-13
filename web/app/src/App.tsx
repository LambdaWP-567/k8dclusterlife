import { useState, useEffect } from 'react'
import { QueryClient, QueryClientProvider, useQuery } from '@tanstack/react-query'
import { Dashboard } from './pages/Dashboard'
import { Login } from './pages/Login'
import { ClusterAdmin } from './pages/ClusterAdmin'
import { Header } from './components/Header'
import { useAuth } from './hooks/useAuth'
import './index.css'

const queryClient = new QueryClient()

type Page = 'dashboard' | 'clusters'

function AppShell() {
  const [dark, setDark] = useState(() =>
    window.matchMedia('(prefers-color-scheme: dark)').matches
  )
  const [page, setPage] = useState<Page>('dashboard')
  const { user, isLoading: authLoading } = useAuth()

  const { data: providers = [] } = useQuery<string[]>({
    queryKey: ['auth-providers'],
    queryFn: () => fetch('/api/auth/providers').then(r => r.json()),
    staleTime: Infinity,
  })

  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark)
  }, [dark])

  if (authLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="text-gray-500 dark:text-gray-400">Laden…</div>
      </div>
    )
  }

  if (!user && providers.length > 0) {
    return <Login availableProviders={providers} />
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-100">
      <Header darkMode={dark} onToggleDark={() => setDark(d => !d)} user={user} onNavigate={setPage} currentPage={page} />
      <main className="mx-auto max-w-5xl px-4 sm:px-6 lg:px-8 py-6">
        {page === 'dashboard' && <Dashboard />}
        {page === 'clusters' && <ClusterAdmin />}
      </main>
    </div>
  )
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AppShell />
    </QueryClientProvider>
  )
}
