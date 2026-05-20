import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Label } from '@/components/ui/label'
import { toast } from 'sonner'

interface Token {
  id: string
  contrato_id: string
  plano_id: string
  plano_nome: string
  produto_nome: string
  token: string
  status: string
  valid_from: string
  valid_until: string
  alerta_enviado: boolean
  predecessor_id?: string
  created_at: string
}

interface Cliente {
  id: string
  razao_social: string
}

interface Contrato {
  id: string
  cliente_id: string
  data_inicio: string
  status: string
}

function statusVariant(s: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  if (s === 'ativo') return 'default'
  if (s === 'em_carencia') return 'outline'
  if (s === 'suspenso' || s === 'encerrado') return 'destructive'
  return 'secondary'
}

function diasRestantes(validUntil: string): number {
  const diff = new Date(validUntil).getTime() - Date.now()
  return Math.ceil(diff / (1000 * 60 * 60 * 24))
}

export default function TokensPage() {
  const [clientes, setClientes] = useState<Cliente[]>([])
  const [contratos, setContratos] = useState<Contrato[]>([])
  const [tokens, setTokens] = useState<Token[]>([])
  const [clienteId, setClienteId] = useState('')
  const [contratoId, setContratoId] = useState('')
  const [loading, setLoading] = useState(false)
  const [reativando, setReativando] = useState<Record<string, boolean>>({})
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetch('/api/financeiro/clientes')
      .then(r => r.json())
      .then(setClientes)
      .catch(() => setError('Erro ao carregar clientes'))
  }, [])

  const onClienteChange = async (id: string) => {
    setClienteId(id)
    setContratoId('')
    setTokens([])
    if (!id) { setContratos([]); return }
    const res = await fetch(`/api/financeiro/contratos?cliente_id=${id}`)
    setContratos(await res.json())
  }

  const onContratoChange = async (id: string) => {
    setContratoId(id)
    setTokens([])
    if (!id) return
    setLoading(true)
    try {
      const res = await fetch(`/api/financeiro/tokens?contrato_id=${id}`)
      setTokens(await res.json())
    } finally {
      setLoading(false)
    }
  }

  const handleReativar = async (tokenId: string) => {
    setReativando(prev => ({ ...prev, [tokenId]: true }))
    try {
      const res = await fetch('/api/financeiro/tokens', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token_id: tokenId }),
      })
      if (!res.ok) throw new Error('Erro ao reativar token')
      toast.success('Token reativado com sucesso')
      onContratoChange(contratoId)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro')
    } finally {
      setReativando(prev => ({ ...prev, [tokenId]: false }))
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-5xl mx-auto space-y-6">
        <h1 className="text-2xl font-bold">Tokens de Acesso</h1>

        {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}

        <Card>
          <CardHeader><CardTitle>Filtrar Tokens</CardTitle></CardHeader>
          <CardContent className="flex gap-4 flex-wrap">
            <div className="flex-1 min-w-48">
              <Label>Cliente</Label>
              <select
                className="mt-1 w-full border border-input bg-background rounded-md px-3 py-2 text-sm"
                value={clienteId}
                onChange={e => onClienteChange(e.target.value)}
              >
                <option value="">Selecione um cliente...</option>
                {clientes.map(c => (
                  <option key={c.id} value={c.id}>{c.razao_social}</option>
                ))}
              </select>
            </div>
            <div className="flex-1 min-w-48">
              <Label>Contrato</Label>
              <select
                className="mt-1 w-full border border-input bg-background rounded-md px-3 py-2 text-sm"
                value={contratoId}
                onChange={e => onContratoChange(e.target.value)}
                disabled={!clienteId}
              >
                <option value="">Selecione um contrato...</option>
                {contratos.map(c => (
                  <option key={c.id} value={c.id}>
                    {c.data_inicio} — {c.status}
                  </option>
                ))}
              </select>
            </div>
          </CardContent>
        </Card>

        {contratoId && (
          <Card>
            <CardHeader><CardTitle>Tokens do Contrato</CardTitle></CardHeader>
            <CardContent>
              {loading ? (
                <p className="text-sm text-muted-foreground">Carregando...</p>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Produto / Plano</TableHead>
                      <TableHead>Token</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Vence em</TableHead>
                      <TableHead>Dias restantes</TableHead>
                      <TableHead className="w-24">Ações</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {tokens.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                          Nenhum token encontrado.
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
                              <code className="text-xs bg-gray-100 px-1 py-0.5 rounded break-all">
                                {t.token.slice(0, 16)}…
                              </code>
                            </TableCell>
                            <TableCell>
                              <Badge variant={statusVariant(t.status)} className="capitalize">
                                {t.status.replace('_', ' ')}
                              </Badge>
                            </TableCell>
                            <TableCell>{t.valid_until}</TableCell>
                            <TableCell>
                              <span className={dias < 15 ? 'text-red-600 font-medium' : ''}>
                                {dias > 0 ? `${dias} dias` : `${-dias} dias atrás`}
                              </span>
                            </TableCell>
                            <TableCell>
                              {(t.status === 'suspenso' || t.status === 'em_carencia' || t.status === 'encerrado') && (
                                <Button
                                  size="sm"
                                  variant="outline"
                                  disabled={reativando[t.id]}
                                  onClick={() => handleReativar(t.id)}
                                >
                                  {reativando[t.id] ? '...' : 'Reativar'}
                                </Button>
                              )}
                            </TableCell>
                          </TableRow>
                        )
                      })
                    )}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
