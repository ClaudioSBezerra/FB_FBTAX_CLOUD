import { useCallback, useEffect, useMemo, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { toast } from 'sonner'

interface Titulo {
  id: string
  cliente_id: string
  cliente: string
  contrato_id: string
  tipo: 'mensalidade' | 'adesao' | 'projeto' | 'outro'
  competencia: string
  descricao: string
  valor: number
  vencimento: string
  status: 'aberto' | 'pago' | 'cancelado'
  dias_atraso: number
  pago_em: string
  valor_pago?: number
  baixa_origem: string
  baixa_obs: string
  n_nfse: string
}

interface Cliente { id: string; razao_social: string; cnpj: string }
interface Contrato { id: string; cliente_id: string; valor_total: number; status: string; itens?: { plano_nome?: string; produto_nome?: string }[] }
interface Candidato { transacao_id: string; data: string; descricao: string; valor: number; categoria: string; pagador: string; cnpj_pagador: string; valor_igual: boolean }

const brl = (v: number) => v.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })
const dia = (s?: string) => (s ? s.slice(0, 10).split('-').reverse().join('/') : '—')
const hoje = () => new Date().toISOString().slice(0, 10)
const TIPOS = { mensalidade: 'Mensalidade', adesao: 'Adesão / setup', projeto: 'Projeto', outro: 'Outro' } as const

async function erroDe(res: Response): Promise<string> {
  const t = await res.text().catch(() => '')
  try {
    const j = JSON.parse(t)
    return j.erro || j.error || `HTTP ${res.status}`
  } catch {
    return t.trim().slice(0, 200) || `HTTP ${res.status}`
  }
}

const vazio = { id: '', cliente_id: '', contrato_id: '', tipo: 'mensalidade', competencia: hoje().slice(0, 7), descricao: '', valor: '', vencimento: '' }

export default function ContasReceberPage() {
  const [titulos, setTitulos] = useState<Titulo[]>([])
  const [clientes, setClientes] = useState<Cliente[]>([])
  const [contratos, setContratos] = useState<Contrato[]>([])
  const [status, setStatus] = useState('aberto')
  const [busca, setBusca] = useState('')
  const [form, setForm] = useState(vazio)
  const [mostrarForm, setMostrarForm] = useState(false)
  const [baixando, setBaixando] = useState<Titulo | null>(null)
  const [baixa, setBaixa] = useState({ data: hoje(), valor: '', obs: '' })
  const [vinculando, setVinculando] = useState<Titulo | null>(null)
  const [cands, setCands] = useState<Candidato[]>([])
  const [ocupado, setOcupado] = useState(false)

  const carregar = useCallback(async () => {
    const q = new URLSearchParams({ limit: '500' })
    if (status) q.set('status', status)
    if (busca.trim()) q.set('q', busca.trim())
    const res = await fetch(`/api/financeiro/titulos?${q}`)
    if (res.ok) setTitulos(await res.json())
  }, [status, busca])

  useEffect(() => { carregar() }, [carregar])
  useEffect(() => {
    fetch('/api/financeiro/clientes?status=ativo').then(r => (r.ok ? r.json() : [])).then(setClientes).catch(() => {})
  }, [])
  useEffect(() => {
    if (!form.cliente_id) return setContratos([])
    fetch(`/api/financeiro/contratos?cliente_id=${form.cliente_id}`).then(r => (r.ok ? r.json() : [])).then(setContratos).catch(() => {})
  }, [form.cliente_id])

  const abertos = titulos.filter(t => t.status === 'aberto')
  const totalAberto = abertos.reduce((s, t) => s + t.valor, 0)
  const vencidos = abertos.filter(t => t.dias_atraso > 0)
  const totalVencido = vencidos.reduce((s, t) => s + t.valor, 0)
  const aVencer7 = abertos.filter(t => t.dias_atraso === 0 && new Date(t.vencimento).getTime() - Date.now() < 7 * 864e5)

  const salvar = async () => {
    const valor = parseFloat(form.valor.replace(/\./g, '').replace(',', '.'))
    if (!form.cliente_id || !(valor > 0) || !form.vencimento || !form.descricao.trim()) return toast.error('Preencha cliente, descrição, valor e vencimento')
    setOcupado(true)
    try {
      const res = await fetch('/api/financeiro/titulos', {
        method: form.id ? 'PUT' : 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...form, valor, competencia: form.tipo === 'mensalidade' ? `${form.competencia}-01` : '' }),
      })
      if (!res.ok) return toast.error(await erroDe(res))
      toast.success(form.id ? 'Título atualizado' : 'Título criado')
      setForm(vazio); setMostrarForm(false); carregar()
    } finally { setOcupado(false) }
  }

  const editar = (t: Titulo) => {
    setForm({ id: t.id, cliente_id: t.cliente_id, contrato_id: t.contrato_id, tipo: t.tipo, competencia: (t.competencia || hoje()).slice(0, 7), descricao: t.descricao, valor: String(t.valor).replace('.', ','), vencimento: t.vencimento })
    setMostrarForm(true)
  }

  const confirmarBaixa = async () => {
    if (!baixando) return
    const valor = parseFloat(baixa.valor.replace(/\./g, '').replace(',', '.'))
    const res = await fetch('/api/financeiro/titulos?action=baixar', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: baixando.id, data: baixa.data, valor, obs: baixa.obs }) })
    if (!res.ok) return toast.error(await erroDe(res))
    const j = await res.json()
    toast.success(j.tokens_renovados > 0 ? `Baixado. ${j.tokens_renovados} token(s) renovado(s).` : 'Título baixado')
    setBaixando(null); carregar()
  }

  const abrirVinculo = async (t: Titulo) => {
    setVinculando(t); setCands([])
    const res = await fetch(`/api/financeiro/titulos?action=candidatos&id=${t.id}`)
    if (res.ok) setCands(await res.json())
  }

  const vincular = async (c: Candidato) => {
    if (!vinculando) return
    const res = await fetch('/api/financeiro/titulos?action=vincular', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: vinculando.id, transacao_id: c.transacao_id }) })
    if (!res.ok) return toast.error(await erroDe(res))
    const j = await res.json()
    toast.success(j.tokens_renovados > 0 ? `Baixado. ${j.tokens_renovados} token(s) renovado(s).` : 'Título baixado')
    setVinculando(null); carregar()
  }

  const cancelar = async (t: Titulo) => {
    const obs = window.prompt(`Cancelar o título "${t.descricao}"? Informe o motivo:`)
    if (obs === null) return
    const res = await fetch('/api/financeiro/titulos?action=cancelar', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: t.id, obs }) })
    if (!res.ok) return toast.error(await erroDe(res))
    toast.success('Título cancelado'); carregar()
  }

  const conciliar = async () => {
    setOcupado(true)
    try {
      const res = await fetch('/api/financeiro/titulos?action=conciliar', { method: 'POST' })
      if (!res.ok) return toast.error(await erroDe(res))
      const j = await res.json()
      toast.success(j.baixados > 0 ? `${j.baixados} título(s) baixado(s) automaticamente` : 'Nenhum pagamento novo identificado')
      carregar()
    } finally { setOcupado(false) }
  }

  const contratosCliente = useMemo(() => contratos.filter(c => c.status === 'ativo'), [contratos])

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-6xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-semibold">Contas a Receber</h1>
          <div className="flex gap-2">
            <Button variant="outline" disabled={ocupado} onClick={conciliar}>Conciliar com o extrato agora</Button>
            <Button onClick={() => { setForm(vazio); setMostrarForm(true) }}>Novo título</Button>
          </div>
        </div>

        <div className="grid grid-cols-3 gap-4">
          <Card><CardHeader><CardTitle className="text-sm text-gray-500">Em aberto</CardTitle></CardHeader><CardContent className="text-xl font-semibold">{brl(totalAberto)}<p className="text-xs font-normal text-gray-500">{abertos.length} título(s)</p></CardContent></Card>
          <Card><CardHeader><CardTitle className="text-sm text-gray-500">Vencido</CardTitle></CardHeader><CardContent className={`text-xl font-semibold ${vencidos.length ? 'text-red-700' : ''}`}>{brl(totalVencido)}<p className="text-xs font-normal text-gray-500">{vencidos.length} título(s)</p></CardContent></Card>
          <Card><CardHeader><CardTitle className="text-sm text-gray-500">Vencem em até 7 dias</CardTitle></CardHeader><CardContent className="text-xl font-semibold">{brl(aVencer7.reduce((s, t) => s + t.valor, 0))}<p className="text-xs font-normal text-gray-500">{aVencer7.length} título(s)</p></CardContent></Card>
        </div>

        {mostrarForm && (
          <Card>
            <CardHeader><CardTitle>{form.id ? 'Editar título' : 'Novo título'}</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              <div className="grid grid-cols-4 gap-3">
                <div className="col-span-2">
                  <Label>Cliente</Label>
                  <select className="w-full border rounded h-10 px-2 bg-white" value={form.cliente_id} onChange={e => setForm({ ...form, cliente_id: e.target.value, contrato_id: '' })}>
                    <option value="">Selecione…</option>
                    {clientes.map(c => <option key={c.id} value={c.id}>{c.razao_social}</option>)}
                  </select>
                </div>
                <div>
                  <Label>Tipo</Label>
                  <select className="w-full border rounded h-10 px-2 bg-white" value={form.tipo} onChange={e => setForm({ ...form, tipo: e.target.value })}>
                    {Object.entries(TIPOS).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
                  </select>
                </div>
                <div>
                  <Label>Contrato {form.tipo === 'mensalidade' ? '(obrigatório)' : '(opcional)'}</Label>
                  <select className="w-full border rounded h-10 px-2 bg-white" value={form.contrato_id} onChange={e => setForm({ ...form, contrato_id: e.target.value })}>
                    <option value="">—</option>
                    {contratosCliente.map(c => <option key={c.id} value={c.id}>{brl(c.valor_total)} · {(c.itens ?? []).map(i => i.produto_nome || i.plano_nome).filter(Boolean).join(' + ') || c.id.slice(0, 8)}</option>)}
                  </select>
                </div>
                {form.tipo === 'mensalidade' && <div><Label>Competência</Label><Input type="month" value={form.competencia} onChange={e => setForm({ ...form, competencia: e.target.value })} /></div>}
                <div className={form.tipo === 'mensalidade' ? 'col-span-1' : 'col-span-2'}><Label>Valor (R$)</Label><Input value={form.valor} onChange={e => setForm({ ...form, valor: e.target.value })} placeholder="4.500,00" /></div>
                <div><Label>Vencimento</Label><Input type="date" value={form.vencimento} onChange={e => setForm({ ...form, vencimento: e.target.value })} /></div>
                <div className="col-span-4"><Label>Descrição</Label><Input value={form.descricao} onChange={e => setForm({ ...form, descricao: e.target.value })} placeholder="Ref. Contrato Manutenção sistemas: SMARTPICK e FAROL DE VENDAS" /></div>
              </div>
              <div className="flex gap-2">
                <Button onClick={salvar} disabled={ocupado}>{form.id ? 'Salvar alterações' : 'Criar título'}</Button>
                <Button variant="outline" onClick={() => { setForm(vazio); setMostrarForm(false) }}>Cancelar</Button>
              </div>
              <p className="text-xs text-gray-500">Sem multa, juros ou desconto. O vencimento é o que você informar aqui. Mensalidade paga renova o token do contrato (vencimento + 45 dias); adesão e projeto não mexem no token.</p>
            </CardContent>
          </Card>
        )}

        {baixando && (
          <Card>
            <CardHeader><CardTitle>Baixar: {baixando.descricao} — {brl(baixando.valor)}</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              <div className="grid grid-cols-3 gap-3">
                <div><Label>Data do pagamento</Label><Input type="date" value={baixa.data} onChange={e => setBaixa({ ...baixa, data: e.target.value })} /></div>
                <div><Label>Valor recebido (R$)</Label><Input value={baixa.valor} onChange={e => setBaixa({ ...baixa, valor: e.target.value })} /></div>
                <div><Label>Observação</Label><Input value={baixa.obs} onChange={e => setBaixa({ ...baixa, obs: e.target.value })} /></div>
              </div>
              <div className="flex gap-2"><Button onClick={confirmarBaixa}>Confirmar baixa</Button><Button variant="outline" onClick={() => setBaixando(null)}>Cancelar</Button></div>
            </CardContent>
          </Card>
        )}

        {vinculando && (
          <Card>
            <CardHeader><CardTitle>Vincular a um lançamento do extrato: {vinculando.descricao} — {brl(vinculando.valor)}</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              <table className="w-full text-sm">
                <thead><tr className="text-left border-b"><th>Data</th><th>Descrição</th><th>Pagador</th><th>Tipo</th><th className="text-right">Valor</th><th></th></tr></thead>
                <tbody>
                  {cands.map(c => (
                    <tr key={c.transacao_id} className="border-b">
                      <td>{dia(c.data)}</td><td>{c.descricao}</td>
                      <td className="text-xs">{c.pagador}{c.cnpj_pagador && <div className="text-gray-500">{c.cnpj_pagador}</div>}</td>
                      <td><Badge variant="outline">{c.categoria}</Badge></td>
                      <td className="text-right">{brl(c.valor)} {c.valor_igual && <Badge>valor igual</Badge>}</td>
                      <td><Button size="sm" onClick={() => vincular(c)}>Vincular</Button></td>
                    </tr>
                  ))}
                  {cands.length === 0 && <tr><td colSpan={6} className="text-gray-500 py-3">Nenhum crédito livre encontrado. Sincronize o extrato em Bancos › Sincronização.</td></tr>}
                </tbody>
              </table>
              <Button variant="outline" onClick={() => setVinculando(null)}>Fechar</Button>
            </CardContent>
          </Card>
        )}

        <Card>
          <CardContent className="space-y-3 pt-6">
            <div className="flex gap-3">
              <select className="border rounded h-10 px-2 bg-white" value={status} onChange={e => setStatus(e.target.value)}>
                <option value="aberto">Em aberto</option><option value="pago">Pagos</option><option value="cancelado">Cancelados</option><option value="">Todos</option>
              </select>
              <Input className="max-w-xs" placeholder="Buscar cliente ou descrição" value={busca} onChange={e => setBusca(e.target.value)} />
            </div>
            <table className="w-full text-sm">
              <thead><tr className="text-left border-b"><th>Vencimento</th><th>Cliente</th><th>Descrição</th><th>Tipo</th><th className="text-right">Valor</th><th>Situação</th><th></th></tr></thead>
              <tbody>
                {titulos.map(t => (
                  <tr key={t.id} className="border-b align-top">
                    <td className="whitespace-nowrap">{dia(t.vencimento)}</td>
                    <td>{t.cliente}</td>
                    <td>{t.descricao}{t.n_nfse && <div className="text-xs text-gray-500">NFS-e {t.n_nfse}</div>}</td>
                    <td><Badge variant="outline">{TIPOS[t.tipo]}</Badge></td>
                    <td className="text-right whitespace-nowrap">{brl(t.valor)}</td>
                    <td className="text-xs">
                      {t.status === 'aberto' && (t.dias_atraso > 0 ? <Badge variant="destructive">vencido há {t.dias_atraso} dia(s)</Badge> : <Badge variant="secondary">a vencer</Badge>)}
                      {t.status === 'pago' && <><Badge>pago {dia(t.pago_em)}</Badge><div className="text-gray-500">{t.baixa_origem === 'automatica' ? 'conciliado automaticamente' : 'baixa manual'}{t.baixa_obs && ` · ${t.baixa_obs}`}</div></>}
                      {t.status === 'cancelado' && <><Badge variant="secondary">cancelado</Badge>{t.baixa_obs && <div className="text-gray-500">{t.baixa_obs}</div>}</>}
                    </td>
                    <td className="whitespace-nowrap space-x-1">
                      {t.status === 'aberto' && <>
                        <Button size="sm" variant="outline" onClick={() => abrirVinculo(t)}>Vincular extrato</Button>
                        <Button size="sm" variant="outline" onClick={() => { setBaixando(t); setBaixa({ data: hoje(), valor: String(t.valor).replace('.', ','), obs: '' }) }}>Baixar</Button>
                        <Button size="sm" variant="outline" onClick={() => editar(t)}>Editar</Button>
                        <Button size="sm" variant="outline" onClick={() => cancelar(t)}>Cancelar</Button>
                      </>}
                    </td>
                  </tr>
                ))}
                {titulos.length === 0 && <tr><td colSpan={7} className="text-gray-500 py-3">Nenhum título.</td></tr>}
              </tbody>
            </table>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
