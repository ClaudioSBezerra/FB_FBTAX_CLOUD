import { useCallback, useEffect, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { toast } from 'sonner'

interface Conta {
  id: string
  apelido: string
  banco: string
  provedor: string
  saldo: number
  ultima_sync?: string
  configurado: boolean
}

interface Lancamento {
  id: string
  data_transacao: string
  descricao: string
  valor: number
  tipo: 'credito' | 'debito'
  categoria?: string
  detalhes?: Record<string, unknown>
}

interface LogSync {
  id: string
  status: 'running' | 'ok' | 'erro'
  iniciado_em: string
  encerrado_em?: string
  data_inicio?: string
  data_fim?: string
  tx_importadas: number
  tx_duplicadas: number
  saldo_final?: number
  erro?: string
}

const brl = (v: number) => v.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })
const iso = (d: Date) => d.toISOString().slice(0, 10)
const mascaraDoc = (s: string) => {
  const d = s.replace(/\D/g, '')
  if (d.length === 14) return d.replace(/^(\d{2})(\d{3})(\d{3})(\d{4})(\d{2})$/, '$1.$2.$3/$4-$5')
  if (d.length === 11) return d.replace(/^(\d{3})(\d{3})(\d{3})(\d{2})$/, '$1.$2.$3-$4')
  return s
}

// Contraparte do lançamento a partir do detalhe do extrato enriquecido (quem pagou / quem recebeu).
function contraparte(l: Lancamento): { nome: string; doc: string } | null {
  // O JSON guardado é o lançamento inteiro; pagador/recebedor ficam no objeto interno "detalhes".
  const raiz = l.detalhes
  if (!raiz) return null
  const d = (raiz.detalhes && typeof raiz.detalhes === 'object' ? raiz.detalhes : raiz) as Record<string, unknown>
  const t = (k: string) => (typeof d[k] === 'string' ? (d[k] as string).trim() : '')
  const credito = l.tipo === 'credito'
  const nome = credito ? t('nomePagador') : (t('nomeRecebedor') || t('nomeDestinatario'))
  const doc = credito ? t('cpfCnpjPagador') : (t('cpfCnpjRecebedor') || t('cpfCnpj'))
  return nome || doc ? { nome, doc } : null
}

const dia = (s?: string) => (s ? s.slice(0, 10).split('-').reverse().join('/') : '—')

async function erroDe(res: Response): Promise<string> {
  const t = await res.text().catch(() => '')
  try {
    const j = JSON.parse(t)
    return j.erro || j.error || `HTTP ${res.status}`
  } catch {
    return t.trim().slice(0, 200) || `HTTP ${res.status}`
  }
}

export default function BancosSincronizacaoPage() {
  const hoje = new Date()
  const [contas, setContas] = useState<Conta[]>([])
  const [contaId, setContaId] = useState('')
  const [inicio, setInicio] = useState(iso(new Date(hoje.getTime() - 30 * 864e5)))
  const [fim, setFim] = useState(iso(hoje))
  const [tipo, setTipo] = useState('')
  const [busca, setBusca] = useState('')
  const [lanc, setLanc] = useState<Lancamento[]>([])
  const [logs, setLogs] = useState<LogSync[]>([])
  const [saldoVivo, setSaldoVivo] = useState<{ disponivel: number; bloqueado: number; consultado_em: string } | null>(null)
  const [ocupado, setOcupado] = useState(false)
  const [sincronizando, setSincronizando] = useState(false)
  const [amostra, setAmostra] = useState<string | null>(null)
  const [aberto, setAberto] = useState<string | null>(null)
  const timer = useRef<number | undefined>()

  const conta = contas.find(c => c.id === contaId)

  const carregarContas = useCallback(async () => {
    const [a, b] = await Promise.all([fetch('/api/financeiro/contas-financeiras'), fetch('/api/financeiro/bancos/status')])
    if (!a.ok) return
    const base: Conta[] = await a.json()
    const st: { id: string; configurado: boolean }[] = b.ok ? await b.json() : []
    const cfg = new Map(st.map(s => [s.id, s.configurado]))
    setContas(base.map(c => ({ ...c, configurado: cfg.get(c.id) ?? false })))
  }, [])

  const carregarLogs = useCallback(async (id: string) => {
    const res = await fetch(`/api/financeiro/bancos/sync-log?conta_id=${id}&limit=15`)
    if (res.ok) {
      const l: LogSync[] = await res.json()
      setLogs(l)
      return l
    }
    return []
  }, [])

  const carregarLancamentos = useCallback(async () => {
    if (!contaId) return setLanc([])
    const q = new URLSearchParams({ conta_id: contaId, data_inicio: inicio, data_fim: fim, limit: '1000' })
    if (tipo) q.set('tipo', tipo)
    if (busca.trim()) q.set('q', busca.trim())
    const res = await fetch(`/api/financeiro/transacoes?${q}`)
    if (res.ok) setLanc(await res.json())
  }, [contaId, inicio, fim, tipo, busca])

  useEffect(() => { carregarContas() }, [carregarContas])
  useEffect(() => { setSaldoVivo(null); if (contaId) carregarLogs(contaId) }, [contaId, carregarLogs])
  useEffect(() => { carregarLancamentos() }, [carregarLancamentos])
  useEffect(() => () => window.clearInterval(timer.current), [])

  const consultarSaldo = async () => {
    setOcupado(true)
    try {
      const res = await fetch(`/api/financeiro/bancos/saldo?conta_id=${contaId}`)
      if (!res.ok) return toast.error(await erroDe(res))
      setSaldoVivo(await res.json())
      carregarContas()
    } finally { setOcupado(false) }
  }

  // Diagnóstico: mostra o que o Inter devolve para o período, sem gravar nada.
  const verAmostra = async () => {
    setOcupado(true); setAmostra('Consultando o Inter…')
    try {
      const res = await fetch(`/api/financeiro/bancos/extrato-amostra?conta_id=${contaId}&data_inicio=${inicio}&data_fim=${fim}`)
      setAmostra(res.ok ? JSON.stringify(await res.json(), null, 2) : await erroDe(res))
    } finally { setOcupado(false) }
  }

  // Dispara a sincronização e acompanha o histórico até terminar.
  const sincronizar = async () => {
    const res = await fetch('/api/financeiro/bancos/sync', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ conta_id: contaId, data_inicio: inicio, data_fim: fim }) })
    if (!res.ok) return toast.error(await erroDe(res))
    toast.success('Sincronização iniciada')
    setSincronizando(true)
    let tentativas = 0
    window.clearInterval(timer.current)
    timer.current = window.setInterval(async () => {
      tentativas++
      const l = await carregarLogs(contaId)
      const ultimo = l[0]
      if ((ultimo && ultimo.status !== 'running') || tentativas > 60) {
        window.clearInterval(timer.current)
        setSincronizando(false)
        carregarContas(); carregarLancamentos()
        if (ultimo?.status === 'ok') toast.success(`Concluída: ${ultimo.tx_importadas} novas, ${ultimo.tx_duplicadas} já existiam`)
        else if (ultimo?.status === 'erro') toast.error(ultimo.erro || 'A sincronização falhou')
      }
    }, 2000)
  }

  const entradas = lanc.filter(l => l.tipo === 'credito').reduce((s, l) => s + l.valor, 0)
  const saidas = lanc.filter(l => l.tipo === 'debito').reduce((s, l) => s + l.valor, 0)
  const dias = Math.round((new Date(fim).getTime() - new Date(inicio).getTime()) / 864e5)
  const periodoInvalido = !inicio || !fim || dias < 0 || dias > 90

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-5xl mx-auto space-y-6">
        <h1 className="text-2xl font-semibold">Sincronização bancária</h1>

        <Card>
          <CardHeader><CardTitle>Conta e período</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            <div className="grid grid-cols-4 gap-3">
              <div className="col-span-2">
                <Label>Conta</Label>
                <select className="w-full border rounded h-10 px-2 bg-white" value={contaId} onChange={e => setContaId(e.target.value)}>
                  <option value="">Selecione…</option>
                  {contas.map(c => <option key={c.id} value={c.id}>{c.apelido} — {c.banco}{c.configurado ? '' : ' (sem credenciais)'}</option>)}
                </select>
              </div>
              <div><Label>De</Label><Input type="date" value={inicio} onChange={e => setInicio(e.target.value)} /></div>
              <div><Label>Até</Label><Input type="date" value={fim} onChange={e => setFim(e.target.value)} /></div>
            </div>
            {periodoInvalido && <Alert variant="destructive"><AlertDescription>O período deve ter no máximo 90 dias (limite do Inter) e a data final não pode ser anterior à inicial.</AlertDescription></Alert>}
            {conta && !conta.configurado && <Alert variant="destructive"><AlertDescription>Esta conta ainda não tem credenciais da API. Configure em <b>Bancos › Manutenção do banco</b>.</AlertDescription></Alert>}
            <div className="flex flex-wrap gap-3">
              <Button variant="outline" disabled={!conta?.configurado || ocupado} onClick={consultarSaldo}>Consultar saldo agora</Button>
              <Button disabled={!conta?.configurado || periodoInvalido || sincronizando} onClick={sincronizar}>{sincronizando ? 'Sincronizando…' : 'Sincronizar extrato do período'}</Button>
              <Button variant="outline" disabled={!conta?.configurado || periodoInvalido || ocupado} onClick={verAmostra}>Diagnóstico: ver resposta do Inter</Button>
            </div>
            {amostra && <pre className="text-xs overflow-auto max-h-80 bg-white border p-2 whitespace-pre-wrap">{amostra}</pre>}
            <p className="text-xs text-gray-500">O Inter permite 10 consultas por minuto e, no máximo, 90 dias por extrato. Os lançamentos já importados não são duplicados.</p>
          </CardContent>
        </Card>

        {conta && (
          <div className="grid grid-cols-4 gap-4">
            <Card><CardHeader><CardTitle className="text-sm text-gray-500">Saldo guardado</CardTitle></CardHeader><CardContent className="text-xl font-semibold">{brl(conta.saldo)}<p className="text-xs font-normal text-gray-500">sync: {conta.ultima_sync ? new Date(conta.ultima_sync).toLocaleString('pt-BR') : 'nunca'}</p></CardContent></Card>
            <Card><CardHeader><CardTitle className="text-sm text-gray-500">Saldo no banco agora</CardTitle></CardHeader><CardContent className="text-xl font-semibold">{saldoVivo ? brl(saldoVivo.disponivel) : '—'}{saldoVivo && saldoVivo.bloqueado > 0 && <p className="text-xs font-normal text-gray-500">bloqueado {brl(saldoVivo.bloqueado)}</p>}</CardContent></Card>
            <Card><CardHeader><CardTitle className="text-sm text-gray-500">Entradas no período</CardTitle></CardHeader><CardContent className="text-xl font-semibold text-green-700">{brl(entradas)}</CardContent></Card>
            <Card><CardHeader><CardTitle className="text-sm text-gray-500">Saídas no período</CardTitle></CardHeader><CardContent className="text-xl font-semibold text-red-700">{brl(saidas)}<p className="text-xs font-normal text-gray-500">resultado {brl(entradas - saidas)}</p></CardContent></Card>
          </div>
        )}

        {contaId && (
          <Card>
            <CardHeader><CardTitle>Lançamentos ({lanc.length})</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              <div className="flex gap-3">
                <select className="border rounded h-10 px-2 bg-white" value={tipo} onChange={e => setTipo(e.target.value)}>
                  <option value="">Entradas e saídas</option><option value="credito">Só entradas</option><option value="debito">Só saídas</option>
                </select>
                <Input className="max-w-xs" placeholder="Buscar na descrição / tipo (PIX, BOLETO…); clique numa linha para ver o detalhe" value={busca} onChange={e => setBusca(e.target.value)} />
              </div>
              <div className="max-h-96 overflow-auto">
                <table className="w-full text-sm">
                  <thead className="sticky top-0 bg-white"><tr className="text-left border-b"><th>Data</th><th>Descrição</th><th>Pagador / destinatário</th><th>Tipo</th><th className="text-right">Valor</th></tr></thead>
                  <tbody>
                    {lanc.map(l => {
                      const c = contraparte(l)
                      return (
                        <>
                          <tr key={l.id} className="border-b cursor-pointer hover:bg-gray-50" onClick={() => setAberto(aberto === l.id ? null : l.id)}>
                            <td className="whitespace-nowrap">{dia(l.data_transacao)}</td>
                            <td>{l.descricao}</td>
                            <td className="text-xs">{c ? <>{c.nome}{c.doc && <div className="text-gray-500">{mascaraDoc(c.doc)}</div>}</> : <span className="text-gray-400">—</span>}</td>
                            <td>{l.categoria && <Badge variant="outline">{l.categoria}</Badge>}</td>
                            <td className={`text-right whitespace-nowrap ${l.tipo === 'credito' ? 'text-green-700' : 'text-red-700'}`}>{l.tipo === 'debito' ? '−' : '+'} {brl(l.valor)}</td>
                          </tr>
                          {aberto === l.id && (
                            <tr key={`${l.id}-d`} className="border-b bg-gray-50">
                              <td colSpan={5}><pre className="text-xs whitespace-pre-wrap p-2">{l.detalhes ? JSON.stringify(l.detalhes, null, 2) : 'Sem detalhes do banco para este lançamento (extrato simples).'}</pre></td>
                            </tr>
                          )}
                        </>
                      )
                    })}
                    {lanc.length === 0 && <tr><td colSpan={5} className="text-gray-500 py-3">Nenhum lançamento no período. Clique em "Sincronizar extrato do período".</td></tr>}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>
        )}

        {contaId && (
          <Card>
            <CardHeader><CardTitle>Histórico de sincronizações</CardTitle></CardHeader>
            <CardContent>
              <table className="w-full text-sm">
                <thead><tr className="text-left border-b"><th>Início</th><th>Período</th><th>Status</th><th>Novas</th><th>Já existiam</th><th>Detalhe</th></tr></thead>
                <tbody>
                  {logs.map(l => (
                    <tr key={l.id} className="border-b align-top">
                      <td className="whitespace-nowrap">{new Date(l.iniciado_em).toLocaleString('pt-BR')}</td>
                      <td className="whitespace-nowrap">{l.data_inicio ? `${dia(l.data_inicio)} a ${dia(l.data_fim)}` : 'padrão'}</td>
                      <td><Badge variant={l.status === 'ok' ? 'default' : l.status === 'erro' ? 'destructive' : 'secondary'}>{l.status === 'running' ? 'em andamento' : l.status}</Badge></td>
                      <td>{l.tx_importadas}</td><td>{l.tx_duplicadas}</td>
                      <td className="text-xs text-red-600 max-w-xs">{l.erro}</td>
                    </tr>
                  ))}
                  {logs.length === 0 && <tr><td colSpan={6} className="text-gray-500 py-3">Nenhuma sincronização ainda.</td></tr>}
                </tbody>
              </table>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
