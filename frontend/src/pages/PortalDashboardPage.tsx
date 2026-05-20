import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

interface ClienteInfo {
  id: string
  razao_social: string
  cnpj: string
  email: string
}

interface Contrato {
  id: string
  data_inicio: string
  periodicidade: string
  valor_total: number
  status: string
  observacoes?: string
}

interface Token {
  id: string
  contrato_id: string
  token: string
  status: string
  valid_from: string
  valid_until: string
  plano_nome: string
  produto_nome: string
  produto_codigo: string
}

function statusVariant(s: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  if (s === 'ativo') return 'default'
  if (s === 'em_carencia') return 'outline'
  if (s === 'suspenso') return 'destructive'
  return 'secondary'
}

function diasRestantes(validUntil: string): number {
  return Math.ceil((new Date(validUntil).getTime() - Date.now()) / 86400000)
}

function portalFetch(url: string, init?: RequestInit): Promise<Response> {
  const token = localStorage.getItem('portal_token')
  return fetch(url, {
    ...init,
    headers: {
      ...(init?.headers ?? {}),
      Authorization: token ? `Bearer ${token}` : '',
    },
  })
}

export default function PortalDashboardPage() {
  const [cliente, setCliente] = useState<ClienteInfo | null>(null)
  const [contratos, setContratos] = useState<Contrato[]>([])
  const [tokens, setTokens] = useState<Token[]>([])
  const [error, setError] = useState<string | null>(null)
  const navigate = useNavigate()

  useEffect(() => {
    const token = localStorage.getItem('portal_token')
    if (!token) { navigate('/portal/login'); return }

    Promise.all([
      portalFetch('/api/financeiro/portal/me').then(r => {
        if (r.status === 401) { navigate('/portal/login'); return null }
        return r.json()
      }),
      portalFetch('/api/financeiro/portal/contratos').then(r => r.json()),
      portalFetch('/api/financeiro/portal/tokens').then(r => r.json()),
    ]).then(([c, co, t]) => {
      if (c) setCliente(c)
      setContratos(co ?? [])
      setTokens(t ?? [])
    }).catch(() => setError('Erro ao carregar dados'))
  }, [navigate])

  const handleLogout = () => {
    localStorage.removeItem('portal_token')
    localStorage.removeItem('portal_cliente_id')
    navigate('/portal/login')
  }

  const tokensSuspensos = tokens.filter(t => t.status === 'suspenso')
  const tokensCarencia = tokens.filter(t => t.status === 'em_carencia')
  const tokensProximos = tokens.filter(t => t.status === 'ativo' && diasRestantes(t.valid_until) <= 15)

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-4xl mx-auto space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-bold">Portal Fortes Bezerra</h1>
            {cliente && (
              <p className="text-sm text-muted-foreground">{cliente.razao_social} — {cliente.cnpj}</p>
            )}
          </div>
          <Button variant="outline" size="sm" onClick={handleLogout}>Sair</Button>
        </div>

        {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}

        {/* Alertas */}
        {(tokensSuspensos.length > 0 || tokensCarencia.length > 0 || tokensProximos.length > 0) && (
          <Alert variant="destructive">
            <AlertDescription className="space-y-1">
              {tokensSuspensos.length > 0 && (
                <div><strong>{tokensSuspensos.length} acesso(s) suspenso(s).</strong> Entre em contato com a Fortes Bezerra para regularizar.</div>
              )}
              {tokensCarencia.length > 0 && (
                <div><strong>{tokensCarencia.length} acesso(s) em carência</strong> — válidos por mais alguns dias. Regularize para não perder o acesso.</div>
              )}
              {tokensProximos.length > 0 && (
                <div><strong>{tokensProximos.length} acesso(s) vencem em até 15 dias.</strong></div>
              )}
            </AlertDescription>
          </Alert>
        )}

        {/* Tokens de acesso */}
        <Card>
          <CardHeader><CardTitle>Meus Acessos</CardTitle></CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Produto / Plano</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Válido até</TableHead>
                  <TableHead>Dias restantes</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tokens.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center text-muted-foreground py-8">
                      Nenhum acesso ativo.
                    </TableCell>
                  </TableRow>
                ) : (
                  tokens.map(t => {
                    const dias = diasRestantes(t.valid_until)
                    return (
                      <TableRow key={t.id}>
                        <TableCell>
                          <div className="font-medium">{t.produto_nome}</div>
                          <div className="text-xs text-muted-foreground">{t.plano_nome}</div>
                        </TableCell>
                        <TableCell>
                          <Badge variant={statusVariant(t.status)} className="capitalize">
                            {t.status.replace('_', ' ')}
                          </Badge>
                        </TableCell>
                        <TableCell>{t.valid_until}</TableCell>
                        <TableCell>
                          <span className={dias < 15 ? 'text-red-600 font-medium' : ''}>
                            {dias > 0 ? `${dias} dias` : `Vencido há ${-dias} dias`}
                          </span>
                        </TableCell>
                      </TableRow>
                    )
                  })
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        {/* Contratos */}
        <Card>
          <CardHeader><CardTitle>Meus Contratos</CardTitle></CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Início</TableHead>
                  <TableHead>Periodicidade</TableHead>
                  <TableHead>Valor</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {contratos.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center text-muted-foreground py-8">
                      Nenhum contrato encontrado.
                    </TableCell>
                  </TableRow>
                ) : (
                  contratos.map(c => (
                    <TableRow key={c.id}>
                      <TableCell>{c.data_inicio}</TableCell>
                      <TableCell className="capitalize">{c.periodicidade}</TableCell>
                      <TableCell>
                        {c.valor_total.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}
                      </TableCell>
                      <TableCell>
                        <Badge variant={c.status === 'ativo' ? 'default' : 'secondary'} className="capitalize">
                          {c.status}
                        </Badge>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <p className="text-xs text-center text-muted-foreground">
          Dúvidas ou irregularidades? Entre em contato com a Fortes Bezerra Tecnologia.
        </p>
      </div>
    </div>
  )
}
