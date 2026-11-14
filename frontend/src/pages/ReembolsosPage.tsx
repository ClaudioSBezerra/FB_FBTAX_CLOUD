import { useCallback, useEffect, useMemo, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { toast } from 'sonner'

interface Cliente { id: string; razao_social: string; cnpj: string }
interface Despesa {
  id: string; cliente_id: string; data: string; tipo_despesa: string; descricao: string; valor: number
  status: 'pendente' | 'aprovada' | 'descartada'; obs: string
  comprovante_id: string; comprovante_nome: string; nota_debito_id: string; nota_codigo: string
}
interface Nota {
  id: string; cliente: string; codigo: string; emissao: string; vencimento: string; periodo_ini: string; periodo_fim: string
  total: number; status: 'emitida' | 'cancelada'; titulo_status: string; despesas: number; enviada_em: string
}
interface EmailForm {
  id: string; codigo: string; remetente: string; para: string; cc: string; assunto: string; mensagem: string
  comprovantes: boolean; jaEnviada: string; smtpOk: boolean
}

const brl = (v: number) => v.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })
const dia = (s?: string) => (s ? s.slice(0, 10).split('-').reverse().join('/') : '—')
const hoje = () => new Date().toISOString().slice(0, 10)

async function erroDe(res: Response): Promise<string> {
  const t = await res.text().catch(() => '')
  try {
    const j = JSON.parse(t)
    return j.erro || j.error || `HTTP ${res.status}`
  } catch {
    return t.trim().slice(0, 200) || `HTTP ${res.status}`
  }
}

const COR: Record<string, string> = {
  pendente: 'bg-amber-500 hover:bg-amber-500 text-white',
  aprovada: 'bg-emerald-600 hover:bg-emerald-600 text-white',
  descartada: 'bg-gray-400 hover:bg-gray-400 text-white',
}

const vazio = { id: '', data: hoje(), tipo_despesa: '', descricao: '', valor: '', obs: '' }

function lembrado(chave: string, padrao = ''): string {
  try { return localStorage.getItem(chave) ?? padrao } catch { return padrao }
}
function lembrar(chave: string, v: string) {
  try { localStorage.setItem(chave, v) } catch { /* sem armazenamento: segue sem lembrar */ }
}

export default function ReembolsosPage() {
  const [clientes, setClientes] = useState<Cliente[]>([])
  const [clienteId, setClienteId] = useState('')
  const [despesas, setDespesas] = useState<Despesa[]>([])
  const [tipos, setTipos] = useState<string[]>([])
  const [notas, setNotas] = useState<Nota[]>([])
  const [filtro, setFiltro] = useState('')
  const [sel, setSel] = useState<Set<string>>(new Set())
  const [form, setForm] = useState(vazio)
  const [mostrarForm, setMostrarForm] = useState(false)
  const [nota, setNota] = useState({
    destino: lembrado('reemb.destino'), solicitante: lembrado('reemb.solicitante'),
    cc_desc: lembrado('reemb.cc_desc'), cc_num: lembrado('reemb.cc_num'),
    cont_desc: lembrado('reemb.cont_desc'), cont_num: lembrado('reemb.cont_num'),
  })
  const [ocupado, setOcupado] = useState(false)
  const [cnpjDestino, setCnpjDestino] = useState(lembrado('reemb.cnpj_destino'))
  const [consultando, setConsultando] = useState(false)
  const [email, setEmail] = useState<EmailForm | null>(null)
  const [semDespesa, setSemDespesa] = useState<{ id: string; nome: string }[]>([])
  const [iaOk, setIaOk] = useState(true)
  const [progresso, setProgresso] = useState('')

  const carregar = useCallback(async () => {
    if (clienteId) {
      const res = await fetch(`/api/financeiro/despesas?cliente_id=${clienteId}`)
      if (res.ok) {
        const j = await res.json()
        setDespesas(j.despesas ?? [])
        setTipos(j.tipos ?? [])
        setSemDespesa(j.comprovantes_sem_despesa ?? [])
        setIaOk(j.ia_configurada !== false)
      }
    } else {
      setDespesas([])
    }
    const rn = await fetch('/api/financeiro/notas-debito')
    if (rn.ok) setNotas((await rn.json()).notas ?? [])
    setSel(new Set())
  }, [clienteId])

  useEffect(() => { carregar() }, [carregar])
  useEffect(() => {
    fetch('/api/financeiro/clientes?status=ativo').then(r => (r.ok ? r.json() : [])).then(setClientes).catch(() => {})
  }, [])

  const visiveis = useMemo(() => despesas.filter(d => !filtro || d.status === filtro), [despesas, filtro])
  const selecionadas = useMemo(() => despesas.filter(d => sel.has(d.id)), [despesas, sel])
  const aprovadasSel = selecionadas.filter(d => d.status === 'aprovada' && !d.nota_debito_id)
  const totalAprovadas = aprovadasSel.reduce((s, d) => s + d.valor, 0)
  const totais = useMemo(() => {
    const t = { pendente: 0, aprovada: 0, descartada: 0 }
    despesas.forEach(d => { t[d.status] += d.valor })
    return t
  }, [despesas])

  const alternar = (id: string) => setSel(s => { const n = new Set(s); if (n.has(id)) n.delete(id); else n.add(id); return n })
  const livres = visiveis.filter(d => !d.nota_debito_id)
  const todosMarcados = livres.length > 0 && livres.every(d => sel.has(d.id))

  const baixar = async (url: string, nome: string) => {
    const res = await fetch(url)
    if (!res.ok) return toast.error(await erroDe(res))
    const blob = await res.blob()
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = nome
    a.click()
    URL.revokeObjectURL(a.href)
  }
  const abrirComprovante = async (id: string) => {
    const res = await fetch(`/api/financeiro/despesas/comprovantes?id=${id}`)
    if (!res.ok) return toast.error(await erroDe(res))
    const url = URL.createObjectURL(await res.blob())
    window.open(url, '_blank', 'noopener')
    setTimeout(() => URL.revokeObjectURL(url), 60000)
  }

  const enviarComprovantes = async (files: FileList | null) => {
    if (!files || files.length === 0) return
    setOcupado(true)
    try {
      const fd = new FormData()
      Array.from(files).forEach(f => fd.append('arquivo', f))
      const res = await fetch('/api/financeiro/despesas/comprovantes', { method: 'POST', body: fd })
      if (!res.ok) return toast.error(await erroDe(res))
      const j = await res.json()
      toast.success(`${j.comprovantes.length} comprovante(s) guardado(s)`)
      if (clienteId && iaOk) await lerComIA(j.comprovantes)
      else carregar()
    } finally { setOcupado(false) }
  }

  // Lê um comprovante por vez: cada leitura leva dezenas de segundos e o proxy corta pedidos longos.
  const lerComIA = async (lista: { id: string; nome: string }[]) => {
    if (!clienteId) return toast.error('Escolha o cliente antes de ler os comprovantes')
    let total = 0
    const avisos: string[] = []
    for (let i = 0; i < lista.length; i++) {
      setProgresso(`Lendo ${i + 1} de ${lista.length}: ${lista[i].nome}…`)
      try {
        const res = await fetch('/api/financeiro/despesas?action=ler', {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ cliente_id: clienteId, comprovante_id: lista[i].id }),
        })
        if (!res.ok) { avisos.push(await erroDe(res)); continue }
        const j = await res.json()
        total += j.lidas
        ;(j.ignoradas ?? []).forEach((m: string) => avisos.push(`${lista[i].nome}: ${m}`))
      } catch { avisos.push(`${lista[i].nome}: falha de conexão`) }
    }
    setProgresso('')
    toast.success(`${total} despesa(s) lida(s) — revise e aprove`)
    avisos.slice(0, 6).forEach(a => toast.warning(a, { duration: 12000 }))
    carregar()
  }

  const importar = async (file: File | null) => {
    if (!file || !clienteId) return toast.error('Escolha o cliente antes de importar')
    setOcupado(true)
    try {
      let lista: unknown
      try {
        const j = JSON.parse(await file.text())
        lista = Array.isArray(j) ? j : j.despesas
      } catch { return toast.error('Arquivo JSON inválido') }
      const res = await fetch('/api/financeiro/despesas?action=importar', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ cliente_id: clienteId, despesas: lista }),
      })
      if (!res.ok) return toast.error(await erroDe(res))
      const j = await res.json()
      toast.success(`${j.importadas} despesa(s) importada(s) — revise e aprove`)
      if (j.comprovantes_nao_encontrados?.length) toast.warning(`Envie estes comprovantes e importe de novo se faltar: ${j.comprovantes_nao_encontrados.join(', ')}`)
      carregar()
    } finally { setOcupado(false) }
  }

  const mudarStatus = async (status: 'pendente' | 'aprovada' | 'descartada') => {
    const ids = selecionadas.filter(d => !d.nota_debito_id).map(d => d.id)
    if (ids.length === 0) return toast.error('Marque despesas que ainda não estão em nota')
    const res = await fetch('/api/financeiro/despesas?action=status', {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ids, status }),
    })
    if (!res.ok) return toast.error(await erroDe(res))
    carregar()
  }

  const salvar = async () => {
    if (!clienteId) return toast.error('Escolha o cliente')
    const body = { id: form.id || undefined, cliente_id: clienteId, data: form.data, tipo_despesa: form.tipo_despesa, descricao: form.descricao,
      valor: parseFloat(String(form.valor).replace(/\./g, '').replace(',', '.')), obs: form.obs,
      comprovante_id: despesas.find(d => d.id === form.id)?.comprovante_id || undefined,
      status: despesas.find(d => d.id === form.id)?.status || 'pendente' }
    const res = await fetch('/api/financeiro/despesas', { method: form.id ? 'PUT' : 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })
    if (!res.ok) return toast.error(await erroDe(res))
    setForm(vazio); setMostrarForm(false); carregar()
  }

  const excluir = async (d: Despesa) => {
    if (!window.confirm(`Excluir "${d.descricao}" de ${dia(d.data)}?`)) return
    const res = await fetch(`/api/financeiro/despesas?id=${d.id}`, { method: 'DELETE' })
    if (!res.ok) return toast.error(await erroDe(res))
    carregar()
  }

  const gerarNota = async () => {
    if (aprovadasSel.length === 0) return toast.error('Marque despesas aprovadas para a nota')
    if (aprovadasSel.length !== selecionadas.length) return toast.error('Há despesas marcadas que não estão aprovadas ou já estão em nota')
    if (!window.confirm(`Gerar a nota de débito de ${brl(totalAprovadas)} com ${aprovadasSel.length} despesa(s)? Será criado o título no Contas a Receber.`)) return
    setOcupado(true)
    try {
      Object.entries({ destino: nota.destino, solicitante: nota.solicitante, cc_desc: nota.cc_desc, cc_num: nota.cc_num, cont_desc: nota.cont_desc, cont_num: nota.cont_num })
        .forEach(([k, v]) => lembrar(`reemb.${k}`, v))
      const res = await fetch('/api/financeiro/notas-debito', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ cliente_id: clienteId, despesa_ids: aprovadasSel.map(d => d.id), destino: nota.destino, solicitante: nota.solicitante,
          centro_custo_desc: nota.cc_desc, centro_custo_num: nota.cc_num, conta_contabil_desc: nota.cont_desc, conta_contabil_num: nota.cont_num }),
      })
      if (!res.ok) return toast.error(await erroDe(res))
      const j = await res.json()
      toast.success(`Nota ${j.codigo} gerada (${brl(j.total)}, vence ${dia(j.vencimento)}). Título criado no Contas a Receber.`)
      carregar()
    } finally { setOcupado(false) }
  }

  const abrirEmail = async (n: Nota) => {
    const res = await fetch(`/api/financeiro/notas-debito?action=email-previa&id=${n.id}`)
    if (!res.ok) return toast.error(await erroDe(res))
    const j = await res.json()
    setEmail({
      id: n.id, codigo: n.codigo, remetente: lembrado('reemb.remetente', j.remetente ?? ''), para: lembrado('reemb.para', j.para ?? ''),
      cc: lembrado('reemb.cc', j.cc ?? ''), assunto: j.assunto, mensagem: j.mensagem, comprovantes: true,
      jaEnviada: j.enviada_em ? `${j.enviada_em} para ${j.enviada_para}` : '', smtpOk: !!j.smtp_configurado,
    })
  }

  const enviarEmail = async () => {
    if (!email) return
    const lista = (v: string) => v.split(/[;,]/).map(x => x.trim()).filter(Boolean)
    if (lista(email.para).length === 0) return toast.error('Informe o e-mail do destinatário')
    if (email.jaEnviada && !window.confirm(`Esta nota já foi enviada em ${email.jaEnviada}. Enviar de novo?`)) return
    if (!window.confirm(`Enviar a nota ${email.codigo} para ${lista(email.para).join(', ')}?`)) return
    setOcupado(true)
    try {
      lembrar('reemb.remetente', email.remetente); lembrar('reemb.para', email.para); lembrar('reemb.cc', email.cc)
      const res = await fetch('/api/financeiro/notas-debito?action=enviar', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: email.id, remetente: email.remetente, para: lista(email.para), cc: lista(email.cc), assunto: email.assunto,
          mensagem: email.mensagem, incluir_comprovantes: email.comprovantes, confirmar_reenvio: !!email.jaEnviada }),
      })
      if (!res.ok) return toast.error(await erroDe(res))
      toast.success('E-mail enviado')
      setEmail(null)
      carregar()
    } finally { setOcupado(false) }
  }

  const cancelarNota = async (n: Nota) => {
    if (!window.confirm(`Cancelar a nota ${n.codigo}? O título em aberto será cancelado e as despesas voltam a ficar livres.`)) return
    const res = await fetch('/api/financeiro/notas-debito?action=cancelar', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: n.id }) })
    if (!res.ok) return toast.error(await erroDe(res))
    toast.success('Nota cancelada')
    carregar()
  }

  const buscarCNPJ = async () => {
    const digitos = cnpjDestino.replace(/\D/g, '')
    if (digitos.length !== 14) return toast.error('Informe o CNPJ do destino com 14 dígitos')
    setConsultando(true)
    try {
      const res = await fetch(`/api/financeiro/cnpj?cnpj=${digitos}`)
      if (!res.ok) return toast.error(await erroDe(res))
      const j = await res.json()
      setNota(n => ({ ...n, destino: j.destino }))
      lembrar('reemb.cnpj_destino', cnpjDestino)
      if (j.situacao && j.situacao !== 'ATIVA') toast.warning(`Situação cadastral do CNPJ: ${j.situacao}`)
      else toast.success(`Destino preenchido: ${j.razao_social}`)
    } finally { setConsultando(false) }
  }

  const campoNota = (k: keyof typeof nota, rotulo: string, ph = '') => (
    <div><Label>{rotulo}</Label><Input value={nota[k]} placeholder={ph} onChange={e => setNota({ ...nota, [k]: e.target.value })} /></div>
  )

  return (
    <div className="p-6 space-y-6 max-w-6xl">
      <h1 className="text-2xl font-semibold">Reembolso de despesas</h1>

      <Card>
        <CardContent className="pt-6 grid grid-cols-1 md:grid-cols-3 gap-4 items-end">
          <div className="md:col-span-2">
            <Label>Cliente</Label>
            <select className="w-full border rounded h-10 px-2 bg-white" value={clienteId} onChange={e => setClienteId(e.target.value)}>
              <option value="">Selecione…</option>
              {clientes.map(c => <option key={c.id} value={c.id}>{c.razao_social} — {c.cnpj}</option>)}
            </select>
          </div>
          <div className="text-sm text-gray-600">
            Pendentes {brl(totais.pendente)} · Aprovadas {brl(totais.aprovada)} · Descartadas {brl(totais.descartada)}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle className="text-base">1. Comprovantes e importação</CardTitle></CardHeader>
        <CardContent className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <Label>Enviar comprovantes (PDF, PNG ou JPEG; vários de uma vez)</Label>
            <Input type="file" multiple accept="application/pdf,image/png,image/jpeg,image/webp" disabled={ocupado} onChange={e => { enviarComprovantes(e.target.files); e.target.value = '' }} />
          </div>
          <div>
            <Label>Importar lista de despesas (JSON) para o cliente escolhido</Label>
            <Input type="file" accept="application/json,.json" disabled={ocupado || !clienteId} onChange={e => { importar(e.target.files?.[0] ?? null); e.target.value = '' }} />
          </div>
          <div className="md:col-span-2 space-y-2">
            {semDespesa.length > 0 && (
              <div className="flex flex-wrap items-center gap-3 border rounded p-3 bg-blue-50 text-sm">
                <span>{semDespesa.length} comprovante(s) enviado(s) ainda sem despesa.</span>
                <Button size="sm" disabled={ocupado || !clienteId || !iaOk} onClick={async () => { setOcupado(true); try { await lerComIA(semDespesa) } finally { setOcupado(false) } }}>
                  Ler com IA
                </Button>
              </div>
            )}
            {progresso && <p className="text-sm text-blue-700">{progresso}</p>}
            {!iaOk && <p className="text-xs text-red-700">A leitura por IA não está configurada (ANTHROPIC_API_KEY). Use a importação por JSON ou cadastre à mão.</p>}
            <p className="text-xs text-gray-500">Ao enviar os comprovantes com o cliente escolhido, a IA lê cada um e cria as despesas como <b>pendentes</b>, com avisos (itens pessoais, contingência, data ilegível). Os arquivos são enviados à Anthropic para a leitura. Você revisa e aprova abaixo. A importação por JSON continua disponível.</p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-base">2. Despesas</CardTitle>
          <div className="flex gap-2 items-center">
            <select className="border rounded h-9 px-2 bg-white text-sm" value={filtro} onChange={e => setFiltro(e.target.value)}>
              <option value="">Todas</option>
              <option value="pendente">Pendentes</option>
              <option value="aprovada">Aprovadas</option>
              <option value="descartada">Descartadas</option>
            </select>
            <Button size="sm" variant="outline" disabled={!clienteId} onClick={() => { setForm(vazio); setMostrarForm(v => !v) }}>Nova despesa</Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          {mostrarForm && (
            <div className="grid grid-cols-1 md:grid-cols-6 gap-3 border rounded p-3 bg-gray-50">
              <div><Label>Data</Label><Input type="date" value={form.data} onChange={e => setForm({ ...form, data: e.target.value })} /></div>
              <div className="md:col-span-2">
                <Label>Tipo de despesa</Label>
                <select className="w-full border rounded h-10 px-2 bg-white" value={form.tipo_despesa} onChange={e => setForm({ ...form, tipo_despesa: e.target.value })}>
                  <option value="">Selecione…</option>
                  {tipos.map(t => <option key={t} value={t}>{t}</option>)}
                </select>
              </div>
              <div className="md:col-span-2"><Label>Descrição</Label><Input value={form.descricao} onChange={e => setForm({ ...form, descricao: e.target.value })} /></div>
              <div><Label>Valor (R$)</Label><Input value={form.valor} onChange={e => setForm({ ...form, valor: e.target.value })} /></div>
              <div className="md:col-span-5"><Label>Observação</Label><Input value={form.obs} onChange={e => setForm({ ...form, obs: e.target.value })} /></div>
              <div className="flex items-end gap-2"><Button onClick={salvar}>{form.id ? 'Salvar' : 'Adicionar'}</Button><Button variant="ghost" onClick={() => { setForm(vazio); setMostrarForm(false) }}>Cancelar</Button></div>
            </div>
          )}

          <div className="flex flex-wrap gap-2 items-center text-sm">
            <span>{sel.size} marcada(s) · aprovadas marcadas: {brl(totalAprovadas)}</span>
            <Button size="sm" variant="outline" onClick={() => mudarStatus('aprovada')}>Aprovar</Button>
            <Button size="sm" variant="outline" onClick={() => mudarStatus('descartada')}>Descartar</Button>
            <Button size="sm" variant="outline" onClick={() => mudarStatus('pendente')}>Voltar a pendente</Button>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left border-b">
                  <th className="p-2"><input type="checkbox" checked={todosMarcados} onChange={e => setSel(e.target.checked ? new Set(livres.map(d => d.id)) : new Set())} /></th>
                  <th className="p-2">Data</th><th className="p-2">Tipo</th><th className="p-2">Descrição</th>
                  <th className="p-2 text-right">Valor</th><th className="p-2">Situação</th><th className="p-2">Comprovante</th><th className="p-2"></th>
                </tr>
              </thead>
              <tbody>
                {visiveis.length === 0 && <tr><td colSpan={8} className="p-4 text-center text-gray-500">{clienteId ? 'Nenhuma despesa.' : 'Escolha o cliente.'}</td></tr>}
                {visiveis.map(d => (
                  <tr key={d.id} className="border-b align-top">
                    <td className="p-2"><input type="checkbox" disabled={!!d.nota_debito_id} checked={sel.has(d.id)} onChange={() => alternar(d.id)} /></td>
                    <td className="p-2 whitespace-nowrap">{dia(d.data)}</td>
                    <td className="p-2">{d.tipo_despesa}</td>
                    <td className="p-2">{d.descricao}{d.obs && <div className="text-xs text-amber-700">{d.obs}</div>}</td>
                    <td className="p-2 text-right whitespace-nowrap">{brl(d.valor)}</td>
                    <td className="p-2">
                      <Badge className={COR[d.status]}>{d.status}</Badge>
                      {d.nota_codigo && <div className="text-xs text-gray-500">{d.nota_codigo}</div>}
                    </td>
                    <td className="p-2">{d.comprovante_id ? <button className="text-blue-700 underline text-xs" onClick={() => abrirComprovante(d.comprovante_id)}>{d.comprovante_nome}</button> : <span className="text-xs text-gray-400">sem comprovante</span>}</td>
                    <td className="p-2 whitespace-nowrap">
                      {!d.nota_debito_id && <>
                        <Button size="sm" variant="ghost" onClick={() => { setForm({ id: d.id, data: d.data.slice(0, 10), tipo_despesa: d.tipo_despesa, descricao: d.descricao, valor: String(d.valor).replace('.', ','), obs: d.obs }); setMostrarForm(true) }}>Editar</Button>
                        <Button size="sm" variant="ghost" onClick={() => excluir(d)}>Excluir</Button>
                      </>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle className="text-base">3. Gerar a nota de débito (FOR.FIN.0002)</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <div>
              <Label>CNPJ do destino</Label>
              <div className="flex gap-2">
                <Input value={cnpjDestino} placeholder="00.000.000/0000-00" onChange={e => setCnpjDestino(e.target.value)}
                  onBlur={() => { if (cnpjDestino.replace(/\D/g, '').length === 14 && !nota.destino) buscarCNPJ() }} />
                <Button type="button" variant="outline" disabled={consultando} onClick={buscarCNPJ}>{consultando ? 'Buscando…' : 'Buscar'}</Button>
              </div>
            </div>
            <div className="md:col-span-2"><Label>Destino (preenchido pelo CNPJ; pode ajustar)</Label>
              <Input value={nota.destino} onChange={e => setNota({ ...nota, destino: e.target.value })} /></div>
            {campoNota('solicitante', 'Solicitante')}
            <div />
            <div />
            {campoNota('cc_desc', 'Centro de custo (descrição)')}
            {campoNota('cc_num', 'Nº do centro de custo')}
            <div />
            {campoNota('cont_desc', 'Conta contábil (descrição)')}
            {campoNota('cont_num', 'Nº da conta contábil')}
          </div>
          <p className="text-sm text-gray-600">
            Marque, na tabela, as despesas <b>aprovadas</b> que entram na nota. As do mesmo tipo são somadas em uma linha; o vencimento é a emissão + 10 dias e o título é criado no Contas a Receber.
          </p>
          <Button disabled={ocupado || aprovadasSel.length === 0} onClick={gerarNota}>Gerar nota de {brl(totalAprovadas)}</Button>
        </CardContent>
      </Card>

      {email && (
        <Card>
          <CardHeader><CardTitle className="text-base">Enviar a nota {email.codigo} por e-mail</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {!email.smtpOk && <p className="text-sm text-red-700">O e-mail do sistema não está configurado (SMTP_PASSWORD): o envio vai falhar.</p>}
            {email.jaEnviada && <p className="text-sm text-amber-700">Já enviada em {email.jaEnviada}.</p>}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
              <div><Label>Remetente (as respostas chegam aqui)</Label><Input value={email.remetente} onChange={e => setEmail({ ...email, remetente: e.target.value })} /></div>
              <div><Label>Destinatário(s)</Label><Input value={email.para} placeholder="financeiro@cliente.com.br" onChange={e => setEmail({ ...email, para: e.target.value })} /></div>
              <div><Label>Cc</Label><Input value={email.cc} placeholder="separe vários por vírgula" onChange={e => setEmail({ ...email, cc: e.target.value })} /></div>
            </div>
            <div><Label>Assunto</Label><Input value={email.assunto} onChange={e => setEmail({ ...email, assunto: e.target.value })} /></div>
            <div><Label>Mensagem</Label><Textarea rows={9} value={email.mensagem} onChange={e => setEmail({ ...email, mensagem: e.target.value })} /></div>
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={email.comprovantes} onChange={e => setEmail({ ...email, comprovantes: e.target.checked })} />
              Anexar o ZIP dos comprovantes (a planilha vai sempre)
            </label>
            <p className="text-xs text-gray-500">O e-mail sai pela conta de e-mail do sistema; o remetente informado vai como "responder para".</p>
            <div className="flex gap-2">
              <Button disabled={ocupado} onClick={enviarEmail}>Enviar e-mail</Button>
              <Button variant="ghost" onClick={() => setEmail(null)}>Fechar</Button>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader><CardTitle className="text-base">Notas de débito</CardTitle></CardHeader>
        <CardContent className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead><tr className="text-left border-b"><th className="p-2">Nota</th><th className="p-2">Cliente</th><th className="p-2">Período</th><th className="p-2">Vencimento</th><th className="p-2 text-right">Total</th><th className="p-2">Situação</th><th className="p-2"></th></tr></thead>
            <tbody>
              {notas.length === 0 && <tr><td colSpan={7} className="p-4 text-center text-gray-500">Nenhuma nota gerada.</td></tr>}
              {notas.map(n => (
                <tr key={n.id} className="border-b">
                  <td className="p-2 whitespace-nowrap">{n.codigo}<div className="text-xs text-gray-500">{dia(n.emissao)} · {n.despesas} despesa(s){n.enviada_em && ` · enviada ${dia(n.enviada_em)}`}</div></td>
                  <td className="p-2">{n.cliente}</td>
                  <td className="p-2 whitespace-nowrap">{dia(n.periodo_ini)} a {dia(n.periodo_fim)}</td>
                  <td className="p-2">{dia(n.vencimento)}</td>
                  <td className="p-2 text-right whitespace-nowrap">{brl(n.total)}</td>
                  <td className="p-2">
                    <Badge className={n.status === 'emitida' ? (n.titulo_status === 'pago' ? COR.aprovada : COR.pendente) : COR.descartada}>
                      {n.status === 'cancelada' ? 'cancelada' : n.titulo_status === 'pago' ? 'paga' : 'a receber'}
                    </Badge>
                  </td>
                  <td className="p-2 whitespace-nowrap">
                    <Button size="sm" variant="outline" onClick={() => baixar(`/api/financeiro/notas-debito?action=xlsx&id=${n.id}`, `${n.codigo}.xlsx`)}>Planilha</Button>{' '}
                    <Button size="sm" variant="outline" onClick={() => baixar(`/api/financeiro/notas-debito?action=zip&id=${n.id}`, `${n.codigo}-comprovantes.zip`)}>Comprovantes</Button>{' '}
                    {n.status === 'emitida' && <Button size="sm" variant="outline" onClick={() => abrirEmail(n)}>E-mail</Button>}{' '}
                    {n.status === 'emitida' && n.titulo_status !== 'pago' && <Button size="sm" variant="ghost" onClick={() => cancelarNota(n)}>Cancelar</Button>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </CardContent>
      </Card>
    </div>
  )
}
