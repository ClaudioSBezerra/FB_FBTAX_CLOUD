import { NavLink, useNavigate } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'

const navItems = [
  { label: 'Dashboard', path: '/admin/financeiro' },
  { label: 'Painel Financeiro', path: '/admin/financeiro/painel' },
  { label: 'Empresa', path: '/admin/financeiro/empresa' },
  { label: 'Clientes', path: '/admin/financeiro/clientes' },
  { label: 'Produtos', path: '/admin/financeiro/produtos' },
  { label: 'Contratos', path: '/admin/financeiro/contratos' },
  { label: 'Tokens', path: '/admin/financeiro/tokens' },
  { label: 'API Keys', path: '/admin/financeiro/api-keys' },
  { label: 'Acesso Portal', path: '/admin/financeiro/portal-clientes' },
]

export default function FinanceiroLayout({ children }: { children: React.ReactNode }) {
  const { logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    navigate('/admin/login')
  }

  return (
    <div className="flex min-h-screen">
      {/* Sidebar */}
      <aside className="w-52 bg-gray-900 text-white flex flex-col shrink-0">
        <div className="px-4 py-5 border-b border-gray-700">
          <div className="text-sm font-bold text-white">Fortes Bezerra</div>
          <div className="text-xs text-gray-400 mt-0.5">Módulo Financeiro</div>
        </div>

        <nav className="flex-1 py-4 space-y-0.5 px-2">
          {navItems.map(item => (
            <NavLink
              key={item.path}
              to={item.path}
              end={item.path === '/admin/financeiro'}
              className={({ isActive }) =>
                `block px-3 py-2 rounded-md text-sm transition-colors ${
                  isActive
                    ? 'bg-gray-700 text-white font-medium'
                    : 'text-gray-300 hover:bg-gray-800 hover:text-white'
                }`
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>

        <div className="px-2 py-4 border-t border-gray-700 space-y-1">
          <button
            onClick={() => navigate('/admin')}
            className="w-full text-left px-3 py-2 rounded-md text-xs text-gray-400 hover:bg-gray-800 hover:text-white transition-colors"
          >
            ← FBTax Cloud
          </button>
          <button
            onClick={handleLogout}
            className="w-full text-left px-3 py-2 rounded-md text-xs text-gray-400 hover:bg-gray-800 hover:text-white transition-colors"
          >
            Sair
          </button>
        </div>
      </aside>

      {/* Conteúdo principal */}
      <main className="flex-1 overflow-auto">
        {children}
      </main>
    </div>
  )
}
