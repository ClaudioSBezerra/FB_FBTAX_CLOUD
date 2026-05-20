import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { useNavigate } from 'react-router-dom'

interface DashboardStats {
  total_clientes: number
  total_contratos: number
  contratos_ativos: number
  tokens_ativos: number
  tokens_em_carencia: number
  tokens_suspensos: number
  tokens_proximos_vencimento: number
}

function StatCard({ title, value, variant }: { title: string; value: number; variant?: 'warn' | 'danger' | 'ok' }) {
  const bg = variant === 'danger' ? 'border-red-200 bg-red-50' :
             variant === 'warn'   ? 'border-yellow-200 bg-yellow-50' : ''
  return (
    <Card className={bg}>
      <CardContent className="pt-6">
        <div className="text-3xl font-bold">{value}</div>
        <div className="text-sm text-muted-foreground mt-1">{title}</div>
      </CardContent>
    </Card>
  )
}

const nav = [
  { label: 'Empresa', path: '/admin/financeiro/empresa' },
  { label: 'Clientes', path: '/admin/financeiro/clientes' },
  { label: 'Produtos', path: '/admin/financeiro/produtos' },
  { label: 'Contratos', path: '/admin/financeiro/contratos' },
  { label: 'Tokens', path: '/admin/financeiro/tokens' },
  { label: 'API Keys', path: '/admin/financeiro/api-keys' },
  { label: 'Acesso Portal', path: '/admin/financeiro/portal-clientes' },
]

export default function DashboardFinanceiroPage() {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [error, setError] = useState<string | null>(null)
  const navigate = useNavigate()

  useEffect(() => {
    fetch('/api/financeiro/dashboard')
      .then(r => r.json())
      .then(setStats)
      .catch(() => setError('Erro ao carregar dashboard'))
  }, [])

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-5xl mx-auto space-y-8">
        <div>
          <h1 className="text-2xl font-bold">Portal Fortes Bezerra</h1>
          <p className="text-muted-foreground text-sm mt-1">Módulo Financeiro — Painel Administrativo</p>
        </div>

        {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}

        {stats && (
          <>
            {(stats.tokens_suspensos > 0 || stats.tokens_em_carencia > 0) && (
              <Alert variant="destructive">
                <AlertDescription>
                  {stats.tokens_suspensos > 0 && (
                    <span className="block"><strong>{stats.tokens_suspensos} token(s) suspenso(s)</strong> — clientes sem acesso.</span>
                  )}
                  {stats.tokens_em_carencia > 0 && (
                    <span className="block"><strong>{stats.tokens_em_carencia} token(s) em carência</strong> — vencidos, ainda válidos por até 15 dias.</span>
                  )}
                </AlertDescription>
              </Alert>
            )}

            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <StatCard title="Clientes ativos" value={stats.total_clientes} />
              <StatCard title="Contratos totais" value={stats.total_contratos} />
              <StatCard title="Contratos ativos" value={stats.contratos_ativos} />
              <StatCard title="Tokens ativos" value={stats.tokens_ativos} variant="ok" />
            </div>
            <div className="grid grid-cols-3 gap-4">
              <StatCard
                title="Vencendo em 15 dias"
                value={stats.tokens_proximos_vencimento}
                variant={stats.tokens_proximos_vencimento > 0 ? 'warn' : undefined}
              />
              <StatCard
                title="Em carência"
                value={stats.tokens_em_carencia}
                variant={stats.tokens_em_carencia > 0 ? 'warn' : undefined}
              />
              <StatCard
                title="Suspensos"
                value={stats.tokens_suspensos}
                variant={stats.tokens_suspensos > 0 ? 'danger' : undefined}
              />
            </div>
          </>
        )}

        <Card>
          <CardHeader><CardTitle>Navegação rápida</CardTitle></CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-3">
              {nav.map(item => (
                <Badge
                  key={item.path}
                  variant="outline"
                  className="cursor-pointer px-4 py-2 text-sm hover:bg-gray-100"
                  onClick={() => navigate(item.path)}
                >
                  {item.label}
                </Badge>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
