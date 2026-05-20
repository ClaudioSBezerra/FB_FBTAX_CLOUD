import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from '@/components/ui/sonner'
import Login from './pages/Login'
import PortalPage from './pages/PortalPage'
import EmpresaPage from './pages/EmpresaPage'
import ClientesPage from './pages/ClientesPage'
import ProdutosPage from './pages/ProdutosPage'
import ContratosPage from './pages/ContratosPage'
import { AuthProvider, useAuth } from './contexts/AuthContext'

const queryClient = new QueryClient()

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, loading } = useAuth()
  if (loading) return null
  if (!isAuthenticated) return <Navigate to="/admin/login" replace />
  return <>{children}</>
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <AuthProvider>
          <Routes>
            {/* Portal público */}
            <Route path="/" element={<PortalPage />} />

            {/* Admin */}
            <Route path="/admin/login" element={<Login />} />
            <Route path="/admin" element={
              <ProtectedRoute>
                <div>AdminDashboard — em breve</div>
              </ProtectedRoute>
            } />

            {/* Financeiro */}
            <Route path="/admin/financeiro/empresa" element={
              <ProtectedRoute>
                <EmpresaPage />
              </ProtectedRoute>
            } />
            <Route path="/admin/financeiro/clientes" element={
              <ProtectedRoute>
                <ClientesPage />
              </ProtectedRoute>
            } />
            <Route path="/admin/financeiro/produtos" element={
              <ProtectedRoute>
                <ProdutosPage />
              </ProtectedRoute>
            } />
            <Route path="/admin/financeiro/contratos" element={
              <ProtectedRoute>
                <ContratosPage />
              </ProtectedRoute>
            } />

            {/* Fallback */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
          <Toaster />
        </AuthProvider>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

export default App
