import { BrowserRouter, Routes, Route, Navigate, useNavigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from '@/components/ui/sonner'
import Login from './pages/Login'
import PortalPage from './pages/PortalPage'
import EmpresaPage from './pages/EmpresaPage'
import ClientesPage from './pages/ClientesPage'
import ProdutosPage from './pages/ProdutosPage'
import ContratosPage from './pages/ContratosPage'
import TokensPage from './pages/TokensPage'
import ApiKeysPage from './pages/ApiKeysPage'
import DashboardFinanceiroPage from './pages/DashboardFinanceiroPage'
import PainelFinanceiroPage from './pages/PainelFinanceiroPage'
import PortalClientesAdminPage from './pages/PortalClientesAdminPage'
import PortalLoginPage from './pages/PortalLoginPage'
import PortalDashboardPage from './pages/PortalDashboardPage'
import FinanceiroLayout from './components/FinanceiroLayout'
import { AuthProvider, useAuth } from './contexts/AuthContext'
import { Button } from '@/components/ui/button'

const queryClient = new QueryClient()

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, loading } = useAuth()
  if (loading) return null
  if (!isAuthenticated) return <Navigate to="/admin/login" replace />
  return <>{children}</>
}

function FinanceiroRoute({ children }: { children: React.ReactNode }) {
  return (
    <ProtectedRoute>
      <FinanceiroLayout>{children}</FinanceiroLayout>
    </ProtectedRoute>
  )
}

function AdminIndex() {
  const navigate = useNavigate()
  return (
    <div className="min-h-screen bg-gray-50 flex flex-col items-center justify-center gap-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold">FBTax Cloud</h1>
        <p className="text-muted-foreground mt-1">Painel Administrativo</p>
      </div>
      <Button onClick={() => navigate('/admin/financeiro')}>
        Acessar Fortes Bezerra
      </Button>
    </div>
  )
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <AuthProvider>
          <Routes>
            {/* Portal público FBTax */}
            <Route path="/" element={<PortalPage />} />

            {/* Admin FBTax */}
            <Route path="/admin/login" element={<Login />} />
            <Route path="/admin" element={
              <ProtectedRoute><AdminIndex /></ProtectedRoute>
            } />

            {/* Financeiro — Fortes Bezerra (com sidebar) */}
            <Route path="/admin/financeiro" element={
              <FinanceiroRoute><DashboardFinanceiroPage /></FinanceiroRoute>
            } />
            <Route path="/admin/financeiro/painel" element={
              <FinanceiroRoute><PainelFinanceiroPage /></FinanceiroRoute>
            } />
            <Route path="/admin/financeiro/empresa" element={
              <FinanceiroRoute><EmpresaPage /></FinanceiroRoute>
            } />
            <Route path="/admin/financeiro/clientes" element={
              <FinanceiroRoute><ClientesPage /></FinanceiroRoute>
            } />
            <Route path="/admin/financeiro/produtos" element={
              <FinanceiroRoute><ProdutosPage /></FinanceiroRoute>
            } />
            <Route path="/admin/financeiro/contratos" element={
              <FinanceiroRoute><ContratosPage /></FinanceiroRoute>
            } />
            <Route path="/admin/financeiro/tokens" element={
              <FinanceiroRoute><TokensPage /></FinanceiroRoute>
            } />
            <Route path="/admin/financeiro/api-keys" element={
              <FinanceiroRoute><ApiKeysPage /></FinanceiroRoute>
            } />
            <Route path="/admin/financeiro/portal-clientes" element={
              <FinanceiroRoute><PortalClientesAdminPage /></FinanceiroRoute>
            } />

            {/* Portal do cliente (auth própria) */}
            <Route path="/portal/login" element={<PortalLoginPage />} />
            <Route path="/portal/dashboard" element={<PortalDashboardPage />} />

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
