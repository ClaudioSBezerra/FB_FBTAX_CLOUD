import { useState, useEffect, useRef } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { TrendingUp, TrendingDown, Wallet, Building2, Plus, ExternalLink, Info, Bot, Send, Loader2, ChevronDown, ChevronUp, RefreshCw, CheckCircle2, XCircle, Clock, Settings2, Upload } from 'lucide-react'
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

// ── Tipos de chat ─────────────────────────────────────────────────────────────
interface ChatMsg {
  role: 'user' | 'assistant'
  content: string
  sql?: string
  columns?: string[]
  rows?: Record<string, unknown>[]
  truncado?: boolean
  erro?: string
}

interface OFXImportResult {
  importadas: number
  duplicadas: number
  erros: number
  ids_inseridos: string[]
  conta_apelido: string
  detalhes_erros?: { fitid: string; motivo: string }[]
}

interface OFXDetected {
  bankid: string
  acctid: string
  branchid: string
}

const SUGESTOES = [
  'Quais foram as maiores despesas deste mês?',
  'Total de entradas e saídas por mês em 2026',
  'Qual meu saldo atual e resultado líquido do ano?',
  'Gastos por categoria nos últimos 30 dias',
]

// ── Chat IA ───────────────────────────────────────────────────────────────────
function ChatIA() {
  const [msgs, setMsgs] = useState<ChatMsg[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [sqlAberto, setSqlAberto] = useState<Record<number, boolean>>({})
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [msgs])

  const enviar = async (pergunta: string) => {
    if (!pergunta.trim() || loading) return
    const historico = msgs.map(m => ({ role: m.role, content: m.content }))
    setMsgs(prev => [...prev, { role: 'user', content: pergunta }])
    setInput('')
    setLoading(true)
    try {
      const res = await fetch('/api/financeiro/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pergunta, historico }),
      })
      const data = await res.json()
      if (!res.ok || data.erro) {
        setMsgs(prev => [...prev, { role: 'assistant', content: '', erro: data.erro || 'Erro desconhecido' }])
      } else {
        setMsgs(prev => [...prev, {
          role: 'assistant',
          content: data.reply,
          sql: data.sql,
          columns: data.columns,
          rows: data.rows,
          truncado: data.truncado,
        }])
      }
    } catch {
      setMsgs(prev => [...prev, { role: 'assistant', content: '', erro: 'Erro de conexão' }])
    } finally {
      setLoading(false)
    }
  }

  return (
    <Card className="border-violet-100">
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-semibold flex items-center gap-2">
          <Bot className="w-4 h-4 text-violet-600" />
          Assistente Financeiro IA
          <Badge variant="outline" className="text-[10px] text-violet-700 border-violet-300 bg-violet-50 ml-auto">
            GLM-4.5 · Z.AI
          </Badge>
        </CardTitle>
        <CardDescription className="text-xs">
          Faça perguntas sobre suas movimentações financeiras em linguagem natural.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">

        {/* Sugestões — aparecem só quando não há mensagens */}
        {msgs.length === 0 && (
          <div className="flex flex-wrap gap-2">
            {SUGESTOES.map(s => (
              <button
                key={s}
                onClick={() => enviar(s)}
                className="text-xs border border-violet-200 bg-violet-50 text-violet-700 rounded-full px-3 py-1 hover:bg-violet-100 transition-colors text-left"
              >
                {s}
              </button>
            ))}
          </div>
        )}

        {/* Histórico de mensagens */}
        {msgs.length > 0 && (
          <div className="space-y-4 max-h-[480px] overflow-y-auto pr-1">
            {msgs.map((m, i) => (
              <div key={i} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                <div className={`max-w-[85%] space-y-2 ${m.role === 'user' ? 'items-end' : 'items-start'} flex flex-col`}>

                  {/* Balão da mensagem */}
                  <div className={`rounded-2xl px-4 py-2.5 text-sm leading-relaxed ${
                    m.role === 'user'
                      ? 'bg-violet-600 text-white rounded-br-sm'
                      : m.erro
                        ? 'bg-red-50 border border-red-200 text-red-700 rounded-bl-sm'
                        : 'bg-slate-100 text-slate-800 rounded-bl-sm'
                  }`}>
                    {m.erro ? `⚠️ ${m.erro}` : m.content}
                  </div>

                  {/* Tabela de resultado */}
                  {m.columns && m.rows && m.rows.length > 0 && (
                    <div className="w-full overflow-x-auto rounded-lg border text-xs">
                      <table className="w-full">
                        <thead className="bg-slate-50">
                          <tr>
                            {m.columns.map(c => (
                              <th key={c} className="px-3 py-2 text-left font-semibold text-slate-600 whitespace-nowrap">{c}</th>
                            ))}
                          </tr>
                        </thead>
                        <tbody>
                          {m.rows.map((row, ri) => (
                            <tr key={ri} className="border-t hover:bg-slate-50">
                              {m.columns!.map(c => (
                                <td key={c} className="px-3 py-1.5 text-slate-700 whitespace-nowrap">
                                  {String(row[c] ?? '')}
                                </td>
                              ))}
                            </tr>
                          ))}
                        </tbody>
                      </table>
                      {m.truncado && (
                        <p className="text-center text-[10px] text-muted-foreground py-1 border-t">
                          Resultado truncado em 200 linhas.
                        </p>
                      )}
                    </div>
                  )}

                  {/* SQL gerado — colapsável */}
                  {m.sql && (
                    <button
                      onClick={() => setSqlAberto(p => ({ ...p, [i]: !p[i] }))}
                      className="flex items-center gap-1 text-[10px] text-muted-foreground hover:text-slate-600"
                    >
                      {sqlAberto[i] ? <ChevronUp className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />}
                      {sqlAberto[i] ? 'Ocultar SQL' : 'Ver SQL gerado'}
                    </button>
                  )}
                  {m.sql && sqlAberto[i] && (
                    <pre className="text-[10px] bg-slate-900 text-slate-200 rounded-lg p-3 overflow-x-auto w-full">
                      {m.sql}
                    </pre>
                  )}
                </div>
              </div>
            ))}
            {loading && (
              <div className="flex justify-start">
                <div className="bg-slate-100 rounded-2xl rounded-bl-sm px-4 py-2.5 flex items-center gap-2 text-sm text-slate-500">
                  <Loader2 className="w-3.5 h-3.5 animate-spin" /> Analisando...
                </div>
              </div>
            )}
            <div ref={bottomRef} />
          </div>
        )}

        {/* Input */}
        <form
          onSubmit={e => { e.preventDefault(); enviar(input) }}
          className="flex gap-2 pt-1"
        >
          <Input
            value={input}
            onChange={e => setInput(e.target.value)}
            placeholder="Ex: Quais despesas de Uber em janeiro?"
            className="flex-1 text-sm"
            disabled={loading}
          />
          <Button type="submit" size="sm" disabled={loading || !input.trim()} className="bg-violet-600 hover:bg-violet-700">
            {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
          </Button>
        </form>
        <p className="text-[10px] text-muted-foreground text-center">
          As respostas são geradas por IA a partir dos dados reais cadastrados. Verifique valores importantes.
        </p>
      </CardContent>
    </Card>
  )
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

interface ProviderConfig {
  client_id: string
  client_secret: string
  cert_pem?: string
  key_pem?: string
  conta_corrente?: string  // Inter
  agencia?: string         // Itaú
  conta?: string           // Itaú
  base_url?: string
}

interface SyncLog {
  status: string
  iniciado_em: string
  tx_importadas: number
  tx_duplicadas: number
  saldo_final?: number
  erro_detalhe?: string
}

interface BancoStatus {
  id: string
  apelido: string
  banco: string
  provedor: string
  saldo: number
  ultima_sync?: string
  configurado: boolean
  ultimo_sync?: SyncLog
}

const TIPOS_CONTA = ['corrente', 'poupança', 'pagamento', 'investimento']
const CATEGORIAS = ['Receita', 'Folha de pagamento', 'Fornecedores', 'Impostos', 'Serviços', 'Transferência', 'Outros']

export default function PainelFinanceiroPage() {
  const [painel, setPainel] = useState<PainelData | null>(null)
  const [contas, setContas] = useState<Conta[]>([])
  const [transacoes, setTransacoes] = useState<Transacao[]>([])
  const [error, setError] = useState<string | null>(null)

  const [bancos, setBancos] = useState<BancoStatus[]>([])
  const [providers, setProviders] = useState<string[]>([])
  const [listaBancos, setListaBancos] = useState<{ code: number; name: string }[]>([])
  const [syncingConta, setSyncingConta] = useState<string | null>(null)
  const [modalConfig, setModalConfig] = useState<BancoStatus | null>(null)
  const [config, setConfig] = useState<ProviderConfig & { provedor: string }>({
    provedor: '', client_id: '', client_secret: '', cert_pem: '', key_pem: '',
    conta_corrente: '', agencia: '', conta: '', base_url: '',
  })

  const [modalConta, setModalConta] = useState(false)
  const [modalTx, setModalTx] = useState(false)
  const [novaConta, setNovaConta] = useState({ apelido: '', banco: '', agencia: '', conta: '', tipo: 'corrente', saldo: '' })
  const [novaTx, setNovaTx] = useState({ conta_id: '', data_transacao: '', descricao: '', valor: '', tipo: 'credito', categoria: '' })
  const [submitting, setSubmitting] = useState(false)

  // OFX upload states
  const [ofxDialogOpen, setOfxDialogOpen] = useState(false)
  const [ofx409Open, setOfx409Open] = useState(false)
  const [ofxDetected, setOfxDetected] = useState<OFXDetected | null>(null)
  const [ofxFile, setOfxFile] = useState<File | null>(null)
  const [ofxContaSelecionada, setOfxContaSelecionada] = useState('')
  const [ofxImporting, setOfxImporting] = useState(false)

  const handleOFXUpload = async (file: File, contaId?: string) => {
    setOfxImporting(true)
    const fd = new FormData()
    fd.append('file', file)
    if (contaId) fd.append('conta_id', contaId)
    try {
      const res = await fetch('/api/financeiro/ofx/upload', { method: 'POST', body: fd })
      if (res.status === 409) {
        const data = await res.json()
        setOfxDetected(data.detected)
        setOfxFile(file)
        setOfxDialogOpen(false)
        setOfx409Open(true)
        return
      }
      if (!res.ok) {
        const msg = await res.text()
        toast.error(msg || 'Erro ao importar OFX')
        return
      }
      const data: OFXImportResult = await res.json()
      setOfxDialogOpen(false)
      setOfx409Open(false)
      setOfxFile(null)
      setOfxDetected(null)
      toast.success(`${data.importadas} transação(ões) importada(s), ${data.duplicadas} duplicada(s).`)
      if (data.erros > 0) {
        toast.warning(`${data.erros} linha(s) ignorada(s) — verifique os detalhes.`)
      }
      loadAll()
    } catch {
      toast.error('Erro de conexão ao importar OFX')
    } finally {
      setOfxImporting(false)
    }
  }

  const loadAll = () => {
    Promise.all([
      fetch('/api/financeiro/painel').then(r => r.json()),
      fetch('/api/financeiro/contas-financeiras').then(r => r.json()),
      fetch('/api/financeiro/transacoes?limit=30').then(r => r.json()),
      fetch('/api/financeiro/bancos/status').then(r => r.json()),
      fetch('/api/financeiro/bancos/providers').then(r => r.json()),
      fetch('/api/bancos').then(r => r.ok ? r.json() : []).catch(() => []),
    ]).then(([p, c, t, b, prov, lb]) => {
      setPainel(p)
      setContas(c ?? [])
      setTransacoes(t ?? [])
      setBancos(b ?? [])
      setProviders(prov ?? [])
      setListaBancos(lb ?? [])
    }).catch(() => setError('Erro ao carregar dados financeiros'))
  }

  const handleSync = async (contaId: string) => {
    setSyncingConta(contaId)
    try {
      const res = await fetch('/api/financeiro/bancos/sync', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ conta_id: contaId }),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data.erro || 'Erro ao iniciar sync')
      toast.success('Sincronização iniciada — atualize em alguns segundos.')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro desconhecido')
    } finally {
      setSyncingConta(null)
    }
  }

  const handleSaveConfig = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!modalConfig) return
    const { provedor, ...cfg } = config
    try {
      const res = await fetch('/api/financeiro/bancos/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ conta_id: modalConfig.id, provedor, config: cfg }),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data.erro || 'Erro ao salvar')
      toast.success('Configuração salva com sucesso')
      setModalConfig(null)
      loadAll()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro desconhecido')
    }
  }

  const abrirConfig = (banco: BancoStatus) => {
    setConfig({
      provedor: banco.provedor || '',
      client_id: '', client_secret: '', cert_pem: '', key_pem: '',
      conta_corrente: '', agencia: '', conta: '', base_url: '',
    })
    setModalConfig(banco)
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
              <Button variant="ghost" size="sm" className="w-full text-xs" onClick={() => setOfxDialogOpen(true)}>
                <Upload className="w-3 h-3 mr-1" /> Importar OFX
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

        {/* ── Bancos Conectados — multi-banco ── */}
        <Card className="border-slate-200">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-semibold flex items-center gap-2">
              <Building2 className="w-4 h-4 text-slate-600" />
              Bancos Conectados — API Direta
              <Badge variant="outline" className="ml-auto text-xs">
                {bancos.filter(b => b.configurado).length}/{bancos.length} configurado(s)
              </Badge>
            </CardTitle>
            <CardDescription className="text-xs">
              Extrato e saldo via OAuth2 + mTLS direto nos bancos — sem intermediários. Suporta: {providers.join(', ') || 'Inter, Itaú'}.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {bancos.length === 0 ? (
              <div className="text-center py-6 space-y-2">
                <p className="text-sm text-muted-foreground">Nenhuma conta bancária cadastrada ainda.</p>
                <Button variant="outline" size="sm" onClick={() => setModalConta(true)}>
                  <Plus className="w-3.5 h-3.5 mr-1.5" /> Cadastrar conta
                </Button>
              </div>
            ) : (
              <div className="space-y-2">
                {bancos.map(banco => (
                  <div key={banco.id} className="bg-slate-50 rounded-lg border p-3">
                    <div className="flex items-start gap-3">
                      {/* Ícone do banco */}
                      <div className={`w-8 h-8 rounded-lg flex items-center justify-center text-xs font-bold shrink-0 ${
                        banco.provedor === 'inter' ? 'bg-orange-100 text-orange-600' :
                        banco.provedor === 'itau'  ? 'bg-amber-100 text-amber-700' :
                        'bg-slate-200 text-slate-600'
                      }`}>
                        {banco.provedor ? banco.provedor.slice(0,2).toUpperCase() : '??'}
                      </div>

                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <p className="text-sm font-medium">{banco.apelido}</p>
                          <span className="text-xs text-muted-foreground">{banco.banco}</span>
                          {banco.provedor && (
                            <Badge variant="outline" className="text-[10px] capitalize">{banco.provedor}</Badge>
                          )}
                          {banco.configurado ? (
                            <Badge className="text-[10px] bg-emerald-50 text-emerald-700 border-emerald-200">API ok</Badge>
                          ) : banco.provedor ? (
                            <Badge variant="outline" className="text-[10px] text-amber-700 border-amber-300 bg-amber-50">Sem credenciais</Badge>
                          ) : null}
                        </div>

                        <div className="flex items-center gap-3 mt-1">
                          <p className={`text-sm font-semibold ${banco.saldo >= 0 ? 'text-emerald-600' : 'text-red-600'}`}>
                            {fmt(banco.saldo)}
                          </p>
                          {banco.ultima_sync && (
                            <p className="text-[11px] text-muted-foreground flex items-center gap-1">
                              <Clock className="w-3 h-3" />
                              {new Date(banco.ultima_sync).toLocaleString('pt-BR', { day:'2-digit', month:'2-digit', hour:'2-digit', minute:'2-digit' })}
                            </p>
                          )}
                        </div>

                        {/* Último log de sync */}
                        {banco.ultimo_sync && (
                          <div className="flex items-center gap-1.5 mt-1.5">
                            {banco.ultimo_sync.status === 'ok' ? (
                              <CheckCircle2 className="w-3 h-3 text-emerald-500 shrink-0" />
                            ) : banco.ultimo_sync.status === 'erro' ? (
                              <XCircle className="w-3 h-3 text-red-500 shrink-0" />
                            ) : (
                              <Loader2 className="w-3 h-3 text-amber-500 animate-spin shrink-0" />
                            )}
                            <p className="text-[11px] text-muted-foreground">
                              {banco.ultimo_sync.tx_importadas} importadas · {banco.ultimo_sync.tx_duplicadas} duplicadas
                            </p>
                            {banco.ultimo_sync.erro_detalhe && (
                              <p className="text-[11px] text-red-600 truncate max-w-xs">{banco.ultimo_sync.erro_detalhe}</p>
                            )}
                          </div>
                        )}
                      </div>

                      {/* Ações */}
                      <div className="flex items-center gap-1.5 shrink-0">
                        <Button
                          size="sm"
                          variant="ghost"
                          className="h-7 px-2 text-xs"
                          onClick={() => abrirConfig(banco)}
                          title="Configurar API"
                        >
                          <Settings2 className="w-3.5 h-3.5" />
                        </Button>
                        {banco.configurado && (
                          <Button
                            size="sm"
                            variant="outline"
                            className="h-7 px-2 text-xs"
                            onClick={() => handleSync(banco.id)}
                            disabled={syncingConta === banco.id}
                            title="Sincronizar"
                          >
                            {syncingConta === banco.id
                              ? <Loader2 className="w-3.5 h-3.5 animate-spin" />
                              : <RefreshCw className="w-3.5 h-3.5" />
                            }
                          </Button>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* Nota sobre credenciais */}
            {bancos.some(b => !b.configurado && b.provedor) && (
              <div className="rounded-lg border border-dashed border-amber-200 bg-amber-50/50 p-3">
                <p className="text-xs text-amber-800 font-medium mb-1">Contas sem credenciais configuradas</p>
                <p className="text-xs text-muted-foreground">
                  Clique em <Settings2 className="w-3 h-3 inline" /> ao lado da conta para inserir client_id, client_secret e certificado mTLS.
                  As credenciais são criptografadas com AES-256-GCM antes de salvar.
                </p>
                <div className="mt-2 flex flex-wrap gap-2">
                  <a href="https://developers.inter.co" target="_blank" rel="noopener noreferrer"
                    className="text-xs text-orange-600 hover:underline flex items-center gap-0.5">
                    Inter: developers.inter.co <ExternalLink className="w-2.5 h-2.5" />
                  </a>
                  <a href="https://devportal.itau.com.br" target="_blank" rel="noopener noreferrer"
                    className="text-xs text-amber-700 hover:underline flex items-center gap-0.5">
                    Itaú: devportal.itau.com.br <ExternalLink className="w-2.5 h-2.5" />
                  </a>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* ── Chat IA ── */}
        <ChatIA />

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
              <datalist id="bancos-list-painel">
                {listaBancos.map(b => (
                  <option key={b.code} value={`${String(b.code).padStart(3, '0')} - ${b.name}`} />
                ))}
              </datalist>
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

      {/* ── Modal: Importar OFX ── */}
      <Dialog open={ofxDialogOpen} onOpenChange={open => { setOfxDialogOpen(open); if (!open) setOfxImporting(false) }}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Importar extrato OFX</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            {ofxImporting ? (
              <div className="flex flex-col items-center gap-3 py-6">
                <Loader2 className="w-8 h-8 animate-spin text-violet-600" />
                <p className="text-sm text-muted-foreground">Processando...</p>
              </div>
            ) : (
              <>
                <p className="text-sm text-muted-foreground">
                  Selecione um arquivo <strong>.ofx</strong> exportado do seu banco.
                  A conta será identificada automaticamente pelo código BANKID+ACCTID do arquivo.
                </p>
                <Input
                  type="file"
                  accept=".ofx,.ofc"
                  onChange={e => {
                    const file = e.target.files?.[0]
                    if (file) handleOFXUpload(file)
                  }}
                />
              </>
            )}
            <div className="flex justify-end">
              <Button variant="outline" size="sm" onClick={() => { setOfxDialogOpen(false); setOfxImporting(false) }}>
                Cancelar
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* ── Modal: Conta não identificada (409 fallback) ── */}
      <Dialog open={ofx409Open} onOpenChange={open => { if (!open) { setOfx409Open(false); setOfxDetected(null); setOfxFile(null) } }}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Conta não identificada automaticamente</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            {ofxDetected && (
              <div className="rounded-lg bg-amber-50 border border-amber-200 p-3 text-sm text-amber-800 space-y-1">
                <p><strong>BANKID detectado:</strong> {ofxDetected.bankid || '—'}</p>
                <p><strong>ACCTID detectado:</strong> {ofxDetected.acctid || '—'}</p>
                {ofxDetected.branchid && <p><strong>BRANCHID:</strong> {ofxDetected.branchid}</p>}
              </div>
            )}
            <p className="text-sm text-muted-foreground">
              Nenhuma conta cadastrada corresponde aos dados do arquivo. Escolha a conta de destino manualmente:
            </p>
            <select
              className="w-full border border-input bg-background rounded-md px-3 py-2 text-sm"
              value={ofxContaSelecionada}
              onChange={e => setOfxContaSelecionada(e.target.value)}
            >
              <option value="">Selecione uma conta...</option>
              {contas.map(c => (
                <option key={c.id} value={c.id}>{c.apelido} — {c.banco}</option>
              ))}
            </select>
            <div className="flex gap-2 justify-end">
              <Button
                variant="outline"
                size="sm"
                onClick={() => { setOfx409Open(false); setOfxDetected(null); setOfxFile(null) }}
              >
                Cancelar
              </Button>
              <Button
                size="sm"
                disabled={!ofxContaSelecionada || !ofxFile || ofxImporting}
                onClick={() => {
                  if (ofxFile && ofxContaSelecionada) {
                    setOfx409Open(false)
                    handleOFXUpload(ofxFile, ofxContaSelecionada)
                  }
                }}
              >
                {ofxImporting ? <Loader2 className="w-4 h-4 animate-spin mr-1" /> : null}
                Importar para esta conta
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* ── Modal: Configurar API do Banco ── */}
      <Dialog open={!!modalConfig} onOpenChange={v => !v && setModalConfig(null)}>
        <DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Configurar API — {modalConfig?.apelido}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSaveConfig} className="space-y-4">
            <div>
              <Label>Provedor (banco) *</Label>
              <select
                className="w-full border border-input bg-background rounded-md px-3 py-2 text-sm mt-1"
                value={config.provedor}
                onChange={e => setConfig(p => ({ ...p, provedor: e.target.value }))}
                required
              >
                <option value="">Selecione o banco...</option>
                {providers.map(p => (
                  <option key={p} value={p} className="capitalize">{p.charAt(0).toUpperCase() + p.slice(1)}</option>
                ))}
              </select>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label>Client ID *</Label>
                <Input value={config.client_id} onChange={e => setConfig(p => ({ ...p, client_id: e.target.value }))} required placeholder="Seu client_id" className="mt-1" />
              </div>
              <div>
                <Label>Client Secret *</Label>
                <Input type="password" value={config.client_secret} onChange={e => setConfig(p => ({ ...p, client_secret: e.target.value }))} required placeholder="••••••••" className="mt-1" />
              </div>
            </div>

            {/* mTLS — Inter e Itaú */}
            <div className="space-y-2">
              <Label className="text-xs text-muted-foreground uppercase tracking-wide">Certificado mTLS (opcional)</Label>
              <div>
                <Label className="text-xs">Certificado (.crt) — base64 ou PEM</Label>
                <textarea
                  value={config.cert_pem}
                  onChange={e => setConfig(p => ({ ...p, cert_pem: e.target.value }))}
                  className="w-full border border-input bg-background rounded-md px-3 py-2 text-xs font-mono mt-1 h-20 resize-none"
                  placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
                />
              </div>
              <div>
                <Label className="text-xs">Chave privada (.key) — base64 ou PEM</Label>
                <textarea
                  value={config.key_pem}
                  onChange={e => setConfig(p => ({ ...p, key_pem: e.target.value }))}
                  className="w-full border border-input bg-background rounded-md px-3 py-2 text-xs font-mono mt-1 h-20 resize-none"
                  placeholder="-----BEGIN EC PRIVATE KEY-----&#10;...&#10;-----END EC PRIVATE KEY-----"
                />
              </div>
            </div>

            {/* Campos específicos por provider */}
            {config.provedor === 'inter' && (
              <div>
                <Label>Número da Conta Corrente (x-conta-corrente)</Label>
                <Input value={config.conta_corrente} onChange={e => setConfig(p => ({ ...p, conta_corrente: e.target.value }))} placeholder="Ex: 12345678" className="mt-1" />
              </div>
            )}
            {config.provedor === 'itau' && (
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <Label>Agência</Label>
                  <Input value={config.agencia} onChange={e => setConfig(p => ({ ...p, agencia: e.target.value }))} placeholder="0001" className="mt-1" />
                </div>
                <div>
                  <Label>Conta</Label>
                  <Input value={config.conta} onChange={e => setConfig(p => ({ ...p, conta: e.target.value }))} placeholder="12345" className="mt-1" />
                </div>
              </div>
            )}

            <div>
              <Label className="text-xs text-muted-foreground">Base URL (opcional — override do endpoint padrão)</Label>
              <Input value={config.base_url} onChange={e => setConfig(p => ({ ...p, base_url: e.target.value }))} placeholder="Ex: https://sandbox.bancointer.com.br" className="mt-1 text-xs" />
            </div>

            <div className="rounded-lg bg-amber-50 border border-amber-200 p-3 text-xs text-amber-800">
              As credenciais são criptografadas com AES-256-GCM antes de salvar no banco de dados.
              Configure <code className="bg-amber-100 px-1 rounded font-mono">FINANCEIRO_SECRET</code> no Coolify para ativar a criptografia.
            </div>

            <div className="flex gap-2 justify-end pt-1">
              <Button type="button" variant="outline" onClick={() => setModalConfig(null)}>Cancelar</Button>
              <Button type="submit">Salvar configuração</Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
