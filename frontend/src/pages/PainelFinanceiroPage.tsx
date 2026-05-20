import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { TrendingUp, TrendingDown, Wallet, Building2, Plus, ExternalLink, Info } from 'lucide-react'
import { toast } from 'sonner'

interface PainelData {
  saldo_total: number
  entradas_30d: number
  saidas_30d: number
  resultado_30d: number
  num_contas: number
  fluxo: { data: string; entrada: number; saida: number }[]
}

interface Conta {
  id: string
  apelido: string
  banco: string
  agencia?: string
  conta?: string
  tipo: string
  provedor?: string
  saldo: number
  ultima_sync?: string
}

interface Transacao {
  id: string
  conta_id: string
  conta_apelido: string
  data_transacao: string
  descricao: string
  valor: number
  tipo: 'credito' | 'debito'
  categoria?: string
  conciliado: boolean
}

function fmt(v: number) {
  return v.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })
}

function SummaryCard({ title, value, icon: Icon, variant }: {
  title: string; value: number; icon: React.ElementType;
  variant?: 'positive' | 'negative' | 'neutral'
}) {
  const color = variant === 'positive' ? 'text-emerald-600' :
                variant === 'negative' ? 'text-red-600' : 'text-slate-800'
  return (
    <Card>
      <CardContent className="pt-5">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-xs text-muted-foreground">{title}</p>
            <p className={`text-2xl font-bold mt-1 ${color}`}>{fmt(value)}</p>
          </div>
          <div className="w-9 h-9 rounded-lg bg-slate-100 flex items-center justify-center">
            <Icon className="w-4 h-4 text-slate-600" />
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

const TIPOS_CONTA = ['corrente', 'poupança', 'pagamento', 'investimento']
const CATEGORIAS = ['Receita', 'Folha de pagamento', 'Fornecedores', 'Impostos', 'Serviços', 'Transferência', 'Outros']

export default function PainelFinanceiroPage() {
  const [painel, setPainel] = useState<PainelData | null>(null)
  const [contas, setContas] = useState<Conta[]>([])
  const [transacoes, setTransacoes] = useState<Transacao[]>([])
  const [error, setError] = useState<string | null>(null)

  const [modalConta, setModalConta] = useState(false)
  const [modalTx, setModalTx] = useState(false)
  const [novaConta, setNovaConta] = useState({ apelido: '', banco: '', agencia: '', conta: '', tipo: 'corrente', saldo: '' })
  const [novaTx, setNovaTx] = useState({ conta_id: '', data_transacao: '', descricao: '', valor: '', tipo: 'credito', categoria: '' })
  const [submitting, setSubmitting] = useState(false)

  const loadAll = () => {
    Promise.all([
      fetch('/api/financeiro/painel').then(r => r.json()),
      fetch('/api/financeiro/contas-financeiras').then(r => r.json()),
      fetch('/api/financeiro/transacoes?limit=30').then(r => r.json()),
    ]).then(([p, c, t]) => {
      setPainel(p)
      setContas(c ?? [])
      setTransacoes(t ?? [])
    }).catch(() => setError('Erro ao carregar dados financeiros'))
  }

  useEffect(() => { loadAll() }, [])

  const handleSaveConta = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    try {
      const res = await fetch('/api/financeiro/contas-financeiras', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...novaConta, saldo: parseFloat(novaConta.saldo) || 0 }),
      })
      if (!res.ok) throw new Error('Erro ao salvar conta')
      toast.success('Conta cadastrada')
      setModalConta(false)
      setNovaConta({ apelido: '', banco: '', agencia: '', conta: '', tipo: 'corrente', saldo: '' })
      loadAll()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro desconhecido')
    } finally {
      setSubmitting(false)
    }
  }

  const handleSaveTx = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    try {
      const res = await fetch('/api/financeiro/transacoes', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...novaTx, valor: parseFloat(novaTx.valor) || 0 }),
      })
      if (!res.ok) throw new Error('Erro ao salvar transação')
      toast.success('Transação registrada')
      setModalTx(false)
      setNovaTx({ conta_id: '', data_transacao: '', descricao: '', valor: '', tipo: 'credito', categoria: '' })
      loadAll()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro desconhecido')
    } finally {
      setSubmitting(false)
    }
  }

  const fluxoFormatado = (painel?.fluxo ?? []).map(f => ({
    ...f,
    data: f.data.slice(5), // MM-DD
  }))

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-6xl mx-auto space-y-6">

        {/* ── Header ── */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-bold">Painel Financeiro</h1>
            <p className="text-sm text-muted-foreground mt-0.5">
              Fluxo de caixa, contas e transações · Open Finance
            </p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={() => setModalConta(true)}>
              <Plus className="w-4 h-4 mr-1" /> Conta
            </Button>
            <Button size="sm" onClick={() => setModalTx(true)} disabled={contas.length === 0}>
              <Plus className="w-4 h-4 mr-1" /> Transação
            </Button>
          </div>
        </div>

        {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}

        {/* ── Cards de resumo ── */}
        {painel && (
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            <SummaryCard title="Saldo total" value={painel.saldo_total} icon={Wallet} variant="neutral" />
            <SummaryCard title="Entradas (30 dias)" value={painel.entradas_30d} icon={TrendingUp} variant="positive" />
            <SummaryCard title="Saídas (30 dias)" value={painel.saidas_30d} icon={TrendingDown} variant="negative" />
            <SummaryCard
              title="Resultado (30 dias)"
              value={painel.resultado_30d}
              icon={painel.resultado_30d >= 0 ? TrendingUp : TrendingDown}
              variant={painel.resultado_30d >= 0 ? 'positive' : 'negative'}
            />
          </div>
        )}

        {/* ── Gráfico fluxo de caixa ── */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-semibold">Fluxo de Caixa — últimos 30 dias</CardTitle>
          </CardHeader>
          <CardContent>
            {fluxoFormatado.length === 0 ? (
              <div className="h-48 flex items-center justify-center text-muted-foreground text-sm">
                Nenhuma transação nos últimos 30 dias.
              </div>
            ) : (
              <ResponsiveContainer width="100%" height={220}>
                <AreaChart data={fluxoFormatado} margin={{ top: 5, right: 10, left: 10, bottom: 5 }}>
                  <defs>
                    <linearGradient id="colorEntrada" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#10b981" stopOpacity={0.15} />
                      <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
                    </linearGradient>
                    <linearGradient id="colorSaida" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#ef4444" stopOpacity={0.15} />
                      <stop offset="95%" stopColor="#ef4444" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                  <XAxis dataKey="data" tick={{ fontSize: 11 }} />
                  <YAxis tick={{ fontSize: 11 }} tickFormatter={v => `R$${(v/1000).toFixed(0)}k`} />
                  <Tooltip formatter={(v: number) => fmt(v)} />
                  <Legend />
                  <Area type="monotone" dataKey="entrada" name="Entradas" stroke="#10b981" strokeWidth={2} fill="url(#colorEntrada)" />
                  <Area type="monotone" dataKey="saida"   name="Saídas"   stroke="#ef4444" strokeWidth={2} fill="url(#colorSaida)" />
                </AreaChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">

          {/* ── Contas cadastradas ── */}
          <Card>
            <CardHeader>
              <CardTitle className="text-sm font-semibold flex items-center gap-2">
                <Building2 className="w-4 h-4" /> Contas ({contas.length})
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {contas.length === 0 ? (
                <p className="text-sm text-muted-foreground">Nenhuma conta cadastrada.</p>
              ) : contas.map(c => (
                <div key={c.id} className="flex items-center justify-between py-2 border-b last:border-0">
                  <div>
                    <p className="text-sm font-medium">{c.apelido}</p>
                    <p className="text-xs text-muted-foreground">{c.banco} · {c.tipo}</p>
                    {c.provedor && (
                      <Badge variant="outline" className="text-[10px] mt-0.5">{c.provedor}</Badge>
                    )}
                  </div>
                  <p className={`text-sm font-semibold ${c.saldo >= 0 ? 'text-emerald-600' : 'text-red-600'}`}>
                    {fmt(c.saldo)}
                  </p>
                </div>
              ))}
              <Button variant="ghost" size="sm" className="w-full text-xs" onClick={() => setModalConta(true)}>
                <Plus className="w-3 h-3 mr-1" /> Nova conta
              </Button>
            </CardContent>
          </Card>

          {/* ── Transações recentes ── */}
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle className="text-sm font-semibold">Transações Recentes</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Data</TableHead>
                    <TableHead>Descrição</TableHead>
                    <TableHead>Conta</TableHead>
                    <TableHead className="text-right">Valor</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {transacoes.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={4} className="text-center text-muted-foreground py-8 text-sm">
                        Nenhuma transação registrada.
                      </TableCell>
                    </TableRow>
                  ) : transacoes.map(t => (
                    <TableRow key={t.id}>
                      <TableCell className="text-xs text-muted-foreground whitespace-nowrap">{t.data_transacao}</TableCell>
                      <TableCell className="text-sm">
                        <div>{t.descricao}</div>
                        {t.categoria && <div className="text-xs text-muted-foreground">{t.categoria}</div>}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">{t.conta_apelido}</TableCell>
                      <TableCell className={`text-right text-sm font-medium ${t.tipo === 'credito' ? 'text-emerald-600' : 'text-red-600'}`}>
                        {t.tipo === 'credito' ? '+' : '-'}{fmt(t.valor)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </div>

        {/* ── Seção Open Finance / Cumbuca ── */}
        <Card className="border-blue-100 bg-blue-50/40">
          <CardHeader>
            <CardTitle className="text-sm font-semibold flex items-center gap-2">
              <Info className="w-4 h-4 text-blue-600" />
              Open Finance — Conexão via Cumbuca
            </CardTitle>
            <CardDescription>
              Acesse saldos e extratos bancários em tempo real sem precisar de licença própria do Banco Central.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className="bg-white rounded-lg border p-4 space-y-1">
                <div className="w-7 h-7 rounded-full bg-blue-100 text-blue-700 text-xs font-bold flex items-center justify-center">1</div>
                <p className="text-sm font-medium">Contratar o Cumbuca</p>
                <p className="text-xs text-muted-foreground">
                  O Cumbuca é um proxy regulado pelo BACEN com licença ITP. Solicite acesso em{' '}
                  <a href="https://www.cumbuca.com" target="_blank" rel="noopener noreferrer"
                    className="text-blue-600 hover:underline inline-flex items-center gap-0.5">
                    cumbuca.com <ExternalLink className="w-3 h-3" />
                  </a>. Retorno em até 1 dia útil.
                </p>
              </div>
              <div className="bg-white rounded-lg border p-4 space-y-1">
                <div className="w-7 h-7 rounded-full bg-blue-100 text-blue-700 text-xs font-bold flex items-center justify-center">2</div>
                <p className="text-sm font-medium">Configurar certificados mTLS</p>
                <p className="text-xs text-muted-foreground">
                  Após aprovação, você recebe um certificado de cliente (mTLS) e uma chave privada para assinar requisições com JWS. Configure as variáveis de ambiente <code className="bg-slate-100 px-1 rounded text-[11px]">CUMBUCA_CERT</code>, <code className="bg-slate-100 px-1 rounded text-[11px]">CUMBUCA_KEY</code> e <code className="bg-slate-100 px-1 rounded text-[11px]">CUMBUCA_HOST</code>.
                </p>
              </div>
              <div className="bg-white rounded-lg border p-4 space-y-1">
                <div className="w-7 h-7 rounded-full bg-blue-100 text-blue-700 text-xs font-bold flex items-center justify-center">3</div>
                <p className="text-sm font-medium">Sincronização automática</p>
                <p className="text-xs text-muted-foreground">
                  Com as credenciais configuradas, o sistema solicita consentimento dos titulares (OAuth 2.0 / FAPI) e passa a sincronizar saldos e extratos automaticamente — as APIs são idênticas ao Open Finance oficial, só o host muda.
                </p>
              </div>
            </div>

            <div className="bg-white rounded-lg border border-dashed border-blue-200 p-4 flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-slate-700">Status da integração</p>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Credenciais Cumbuca não configuradas. Dados financeiros são lançados manualmente.
                </p>
              </div>
              <Badge variant="outline" className="text-amber-700 border-amber-300 bg-amber-50">
                Pendente
              </Badge>
            </div>
          </CardContent>
        </Card>

      </div>

      {/* ── Modal: Nova Conta ── */}
      <Dialog open={modalConta} onOpenChange={setModalConta}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Nova Conta Financeira</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSaveConta} className="space-y-3">
            <div>
              <Label>Apelido *</Label>
              <Input value={novaConta.apelido} onChange={e => setNovaConta(p => ({ ...p, apelido: e.target.value }))} required placeholder="Ex: Conta Principal BB" />
            </div>
            <div>
              <Label>Banco *</Label>
              <Input list="bancos-list-painel" value={novaConta.banco} onChange={e => setNovaConta(p => ({ ...p, banco: e.target.value }))} required placeholder="Ex: 001 - BCO DO BRASIL S.A." />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label>Agência</Label>
                <Input value={novaConta.agencia} onChange={e => setNovaConta(p => ({ ...p, agencia: e.target.value }))} />
              </div>
              <div>
                <Label>Conta</Label>
                <Input value={novaConta.conta} onChange={e => setNovaConta(p => ({ ...p, conta: e.target.value }))} />
              </div>
            </div>
            <div>
              <Label>Tipo</Label>
              <select
                className="w-full border border-input bg-background rounded-md px-3 py-2 text-sm"
                value={novaConta.tipo}
                onChange={e => setNovaConta(p => ({ ...p, tipo: e.target.value }))}
              >
                {TIPOS_CONTA.map(t => <option key={t} value={t}>{t}</option>)}
              </select>
            </div>
            <div>
              <Label>Saldo inicial (R$)</Label>
              <Input type="number" step="0.01" value={novaConta.saldo} onChange={e => setNovaConta(p => ({ ...p, saldo: e.target.value }))} placeholder="0,00" />
            </div>
            <div className="flex gap-2 justify-end pt-2">
              <Button type="button" variant="outline" onClick={() => setModalConta(false)}>Cancelar</Button>
              <Button type="submit" disabled={submitting}>{submitting ? 'Salvando...' : 'Salvar'}</Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>

      {/* ── Modal: Nova Transação ── */}
      <Dialog open={modalTx} onOpenChange={setModalTx}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Nova Transação</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSaveTx} className="space-y-3">
            <div>
              <Label>Conta *</Label>
              <select
                className="w-full border border-input bg-background rounded-md px-3 py-2 text-sm"
                value={novaTx.conta_id}
                onChange={e => setNovaTx(p => ({ ...p, conta_id: e.target.value }))}
                required
              >
                <option value="">Selecione...</option>
                {contas.map(c => <option key={c.id} value={c.id}>{c.apelido}</option>)}
              </select>
            </div>
            <div>
              <Label>Data *</Label>
              <Input type="date" value={novaTx.data_transacao} onChange={e => setNovaTx(p => ({ ...p, data_transacao: e.target.value }))} required />
            </div>
            <div>
              <Label>Descrição *</Label>
              <Input value={novaTx.descricao} onChange={e => setNovaTx(p => ({ ...p, descricao: e.target.value }))} required />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label>Valor (R$) *</Label>
                <Input type="number" step="0.01" min="0.01" value={novaTx.valor} onChange={e => setNovaTx(p => ({ ...p, valor: e.target.value }))} required />
              </div>
              <div>
                <Label>Tipo *</Label>
                <select
                  className="w-full border border-input bg-background rounded-md px-3 py-2 text-sm"
                  value={novaTx.tipo}
                  onChange={e => setNovaTx(p => ({ ...p, tipo: e.target.value }))}
                >
                  <option value="credito">Entrada (crédito)</option>
                  <option value="debito">Saída (débito)</option>
                </select>
              </div>
            </div>
            <div>
              <Label>Categoria</Label>
              <select
                className="w-full border border-input bg-background rounded-md px-3 py-2 text-sm"
                value={novaTx.categoria}
                onChange={e => setNovaTx(p => ({ ...p, categoria: e.target.value }))}
              >
                <option value="">Sem categoria</option>
                {CATEGORIAS.map(c => <option key={c} value={c}>{c}</option>)}
              </select>
            </div>
            <div className="flex gap-2 justify-end pt-2">
              <Button type="button" variant="outline" onClick={() => setModalTx(false)}>Cancelar</Button>
              <Button type="submit" disabled={submitting}>{submitting ? 'Salvando...' : 'Salvar'}</Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
