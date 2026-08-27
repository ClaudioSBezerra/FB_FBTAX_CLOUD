import { useState, useEffect, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { FileText, Download, Upload, Plus, Trash2, ArrowLeft, Save } from 'lucide-react'
import { toast } from 'sonner'

// ── Tipos ────────────────────────────────────────────────────────────────────

interface ClienteSimples { id: string; razao_social: string; cnpj: string }
interface ProdutoSimples { id: string; codigo: string; nome: string }

interface Item { descricao: string; detalhe: string; situacao: string }
interface Pendencia { descricao: string; responsavel: string; prazo: string }
interface Signatario {
  papel: string; nome: string; cargo: string; organizacao: string; documento: string
}

interface Entrega {
  id?: string
  numero?: string
  cliente_id: string
  contrato_id: string
  produto_id: string
  versao: string
  titulo: string
  projeto: string
  escopo_resumo: string
  data_inicio: string
  data_golive: string
  data_entrega: string
  garantia_dias: number
  assistida_dias: number
  ambiente: string
  responsavel_tecnico: string
  responsavel_comercial: string
  observacoes: string
  linhas_livres: number
  status: string
  cliente_nome?: string
  produto_nome?: string
  produto_codigo?: string
  assinado_em?: string
  itens: Item[]
  pendencias: Pendencia[]
  signatarios: Signatario[]
}

const SITUACOES = [
  { valor: 'entregue', rotulo: 'Entregue' },
  { valor: 'parcial', rotulo: 'Entregue parcialmente' },
  { valor: 'pendente', rotulo: 'Pendente' },
  { valor: 'nao_aplicavel', rotulo: 'Não aplicável' },
]

// Escopo de partida do SmartPick, transcrito dos entregáveis da proposta
// comercial assinada (docs/templates/Proposta_SmartPick_Farol_JC_assinado.pdf,
// seção 2.2). O termo abre já com estas linhas para que a equipe ajuste o texto
// em vez de digitar tudo do zero a cada implantação.
const ESCOPO_SMARTPICK: Item[] = [
  {
    descricao: 'Dashboard Interativo',
    detalhe: 'Filtros por filial, rua, classe e tipo de ofensor; KPIs em tempo real, gráficos e tabelas paginadas',
    situacao: 'entregue',
  },
  {
    descricao: 'Motor de Cálculo',
    detalhe: 'Recalibragem por giro real, curva ABC com cobertura diferenciada, trava sazonal, safety factor e múltiplo da norma palete',
    situacao: 'entregue',
  },
  {
    descricao: 'Plano de Ação',
    detalhe: 'Priorização automática: Classe A falta → B falta → otimização de espaço',
    situacao: 'entregue',
  },
  {
    descricao: 'Apresentação CEO',
    detalhe: 'Relatório executivo navegável com dados reais para aprovação da diretoria',
    situacao: 'entregue',
  },
  {
    descricao: 'Documento Técnico',
    detalhe: 'Benchmark global de picking em WMS',
    situacao: 'entregue',
  },
  {
    descricao: 'Processo de 6 Etapas',
    detalhe: 'Extrair → Calcular → Analisar → Validar → Executar → Monitorar',
    situacao: 'entregue',
  },
  {
    descricao: 'Integração com o ambiente Winthor/Oracle',
    detalhe: 'Consultas de leitura para dados de picking e vendas',
    situacao: 'entregue',
  },
  {
    descricao: 'Treinamento da equipe na ferramenta',
    detalhe: '',
    situacao: 'entregue',
  },
]

const SIGNATARIOS_PADRAO: Signatario[] = [
  { papel: 'PELA CONTRATADA', nome: '', cargo: '', organizacao: '', documento: '' },
  { papel: 'PELO CLIENTE', nome: '', cargo: '', organizacao: '', documento: '' },
  { papel: 'GESTOR DO CD', nome: '', cargo: '', organizacao: '', documento: '' },
  { papel: 'TESTEMUNHA', nome: '', cargo: '', organizacao: '', documento: '' },
]

function entregaVazia(): Entrega {
  return {
    cliente_id: '', contrato_id: '', produto_id: '',
    versao: '1.0',
    titulo: 'Termo de Entrega e Aceite de Projeto',
    projeto: '', escopo_resumo: '',
    data_inicio: '', data_golive: '',
    data_entrega: new Date().toISOString().slice(0, 10),
    garantia_dias: 30, assistida_dias: 14,
    ambiente: '', responsavel_tecnico: '', responsavel_comercial: '',
    observacoes: '', linhas_livres: 4, status: 'rascunho',
    itens: [], pendencias: [], signatarios: [],
  }
}

function statusVariant(s: string): 'default' | 'secondary' | 'outline' {
  if (s === 'assinado') return 'default'
  if (s === 'emitido') return 'secondary'
  return 'outline'
}

// ── Página ───────────────────────────────────────────────────────────────────

export default function EntregasPage() {
  const [view, setView] = useState<'lista' | 'form'>('lista')
  const [entregas, setEntregas] = useState<Entrega[]>([])
  const [clientes, setClientes] = useState<ClienteSimples[]>([])
  const [produtos, setProdutos] = useState<ProdutoSimples[]>([])
  const [form, setForm] = useState<Entrega>(entregaVazia())
  const [loading, setLoading] = useState(true)
  const [salvando, setSalvando] = useState(false)

  const carregarLista = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/financeiro/entregas')
      if (!res.ok) throw new Error(await res.text())
      setEntregas(await res.json())
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro ao carregar termos')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    carregarLista()
    fetch('/api/financeiro/clientes').then(r => r.json()).then(setClientes).catch(() => {})
    fetch('/api/financeiro/produtos').then(r => r.json()).then(setProdutos).catch(() => {})
  }, [carregarLista])

  // ── Manipulação das linhas dinâmicas ──────────────────────────────────────
  //
  // Cada bloco do termo (escopo, pendências, signatários) é uma lista de linhas
  // que o usuário abre, preenche e remove livremente. O que ficar em branco não
  // é enviado; o PDF imprime linha pautada no lugar.

  function atualizarLinha<T>(campo: 'itens' | 'pendencias' | 'signatarios',
                             indice: number, patch: Partial<T>) {
    setForm(f => ({
      ...f,
      [campo]: (f[campo] as T[]).map((l, i) => i === indice ? { ...l, ...patch } : l),
    }))
  }

  function removerLinha(campo: 'itens' | 'pendencias' | 'signatarios', indice: number) {
    setForm(f => ({ ...f, [campo]: (f[campo] as unknown[]).filter((_, i) => i !== indice) }))
  }

  const adicionarItem = () =>
    setForm(f => ({ ...f, itens: [...f.itens, { descricao: '', detalhe: '', situacao: 'entregue' }] }))
  const adicionarPendencia = () =>
    setForm(f => ({ ...f, pendencias: [...f.pendencias, { descricao: '', responsavel: '', prazo: '' }] }))
  const adicionarSignatario = () =>
    setForm(f => ({ ...f, signatarios: [...f.signatarios, { papel: '', nome: '', cargo: '', organizacao: '', documento: '' }] }))

  // ── Ações ─────────────────────────────────────────────────────────────────

  function novoTermo(preset: 'smartpick' | 'vazio') {
    const base = entregaVazia()
    if (preset === 'smartpick') {
      const smartpick = produtos.find(p => p.codigo === 'FB_SMARTPICK')
      base.produto_id = smartpick?.id ?? ''
      base.projeto = 'Implantação FB_SMARTPICK'
      base.ambiente = 'ERP Winthor/TOTVS (Oracle) — leitura'
      base.assistida_dias = 15
      base.escopo_resumo =
        'Implantação da plataforma SmartPick para recalibragem inteligente da área de ' +
        'picking do Centro de Distribuição, calculando a capacidade ideal de cada ' +
        'endereço a partir do giro real, da classe de venda ABC, da trava sazonal e do ' +
        'múltiplo da norma palete, com detecção automática de ofensores de falta e de espaço.'
      base.itens = ESCOPO_SMARTPICK.map(i => ({ ...i }))
      base.signatarios = SIGNATARIOS_PADRAO.map(s => ({ ...s }))
    }
    setForm(base)
    setView('form')
  }

  async function editarTermo(id: string) {
    try {
      const res = await fetch(`/api/financeiro/entregas/detalhe?id=${id}`)
      if (!res.ok) throw new Error(await res.text())
      const dados: Entrega = await res.json()
      setForm({ ...entregaVazia(), ...dados })
      setView('form')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro ao abrir termo')
    }
  }

  // Devolve o termo gravado, ou null se a gravação falhou — quem encadeia
  // ações (salvar e imprimir) precisa distinguir os dois casos.
  async function salvar(): Promise<Entrega | null> {
    if (!form.cliente_id) { toast.error('Selecione o cliente'); return null }
    setSalvando(true)
    try {
      const res = await fetch('/api/financeiro/entregas', {
        method: form.id ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      })
      if (!res.ok) throw new Error(await res.text())
      const salvo: Entrega = await res.json()
      setForm(f => ({ ...f, id: salvo.id, numero: salvo.numero }))
      toast.success(form.id ? 'Termo atualizado' : `Termo ${salvo.numero} criado`)
      carregarLista()
      return salvo
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro ao salvar')
      return null
    } finally {
      setSalvando(false)
    }
  }

  async function baixarPDF(id: string, numero?: string) {
    try {
      const res = await fetch(`/api/financeiro/entregas/pdf?id=${id}`)
      if (!res.ok) throw new Error(await res.text())
      const blob = await res.blob()
      const href = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = href
      a.download = `Termo_Entrega_${numero ?? id.slice(0, 8)}.pdf`
      a.click()
      URL.revokeObjectURL(href)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro ao gerar PDF')
    }
  }

  // Salva e já entrega o PDF pronto para impressão e coleta de assinaturas.
  // Se a gravação falhar, não baixa nada: um PDF da versão anterior passaria a
  // impressão de que as alterações da tela foram salvas.
  async function salvarEImprimir() {
    const salvo = await salvar()
    if (salvo?.id) await baixarPDF(salvo.id, salvo.numero)
  }

  async function enviarAssinado(id: string, file: File) {
    try {
      const dados = new FormData()
      dados.append('entrega_id', id)
      dados.append('arquivo', file)
      const res = await fetch('/api/financeiro/entregas/upload-assinado', { method: 'POST', body: dados })
      if (!res.ok) throw new Error(await res.text())
      toast.success('Termo assinado enviado')
      carregarLista()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro ao enviar')
    }
  }

  async function excluir(id: string) {
    if (!confirm('Excluir este termo? Esta ação não pode ser desfeita.')) return
    try {
      const res = await fetch(`/api/financeiro/entregas?id=${id}`, { method: 'DELETE' })
      if (!res.ok) throw new Error(await res.text())
      toast.success('Termo excluído')
      carregarLista()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro ao excluir')
    }
  }

  // ── Lista ─────────────────────────────────────────────────────────────────

  if (view === 'lista') {
    return (
      <div className="space-y-6">
        <div className="flex items-start justify-between gap-4 flex-wrap">
          <div>
            <h1 className="text-2xl font-semibold">Termos de Entrega</h1>
            <p className="text-sm text-muted-foreground mt-1">
              Documento de entrega e aceite de projeto, no Padrão de Documentos FBTECH.
            </p>
          </div>
          <div className="flex gap-2">
            <Button onClick={() => novoTermo('smartpick')}>
              <Plus className="w-4 h-4 mr-2" /> Novo termo SmartPick
            </Button>
            <Button variant="outline" onClick={() => novoTermo('vazio')}>
              Termo em branco
            </Button>
          </div>
        </div>

        <Card>
          <CardContent className="pt-6">
            {loading ? (
              <p className="text-sm text-muted-foreground">Carregando…</p>
            ) : entregas.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                Nenhum termo emitido. Comece por <strong>Novo termo SmartPick</strong>.
              </p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Número</TableHead>
                    <TableHead>Cliente</TableHead>
                    <TableHead>Produto</TableHead>
                    <TableHead>Projeto</TableHead>
                    <TableHead>Entrega</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="text-right">Ações</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {entregas.map(e => (
                    <TableRow key={e.id}>
                      <TableCell className="font-medium">{e.numero}</TableCell>
                      <TableCell>{e.cliente_nome}</TableCell>
                      <TableCell>{e.produto_nome}</TableCell>
                      <TableCell>{e.projeto}</TableCell>
                      <TableCell>
                        {e.data_entrega ? new Date(e.data_entrega + 'T12:00:00').toLocaleDateString('pt-BR') : '—'}
                      </TableCell>
                      <TableCell><Badge variant={statusVariant(e.status)}>{e.status}</Badge></TableCell>
                      <TableCell className="text-right space-x-1">
                        <Button size="sm" variant="ghost" title="Editar"
                                onClick={() => e.id && editarTermo(e.id)}>
                          <FileText className="w-4 h-4" />
                        </Button>
                        <Button size="sm" variant="ghost" title="Baixar PDF"
                                onClick={() => e.id && baixarPDF(e.id, e.numero)}>
                          <Download className="w-4 h-4" />
                        </Button>
                        <label title="Enviar assinado">
                          <Button size="sm" variant="ghost" asChild>
                            <span className="cursor-pointer"><Upload className="w-4 h-4" /></span>
                          </Button>
                          <input type="file" accept="application/pdf,image/*" className="hidden"
                                 onChange={ev => {
                                   const f = ev.target.files?.[0]
                                   if (f && e.id) enviarAssinado(e.id, f)
                                   ev.target.value = ''
                                 }} />
                        </label>
                        <Button size="sm" variant="ghost" title="Excluir"
                                onClick={() => e.id && excluir(e.id)}>
                          <Trash2 className="w-4 h-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    )
  }

  // ── Formulário ────────────────────────────────────────────────────────────

  const set = <K extends keyof Entrega>(campo: K, valor: Entrega[K]) =>
    setForm(f => ({ ...f, [campo]: valor }))

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div>
          <Button variant="ghost" size="sm" className="mb-2 -ml-2"
                  onClick={() => { setView('lista'); carregarLista() }}>
            <ArrowLeft className="w-4 h-4 mr-1" /> Voltar
          </Button>
          <h1 className="text-2xl font-semibold">
            {form.numero ? `Termo ${form.numero}` : 'Novo Termo de Entrega'}
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            Campos deixados em branco são impressos como linha pautada, para preenchimento
            à mão no ato da assinatura.
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={salvar} disabled={salvando}>
            <Save className="w-4 h-4 mr-2" /> {salvando ? 'Salvando…' : 'Salvar'}
          </Button>
          <Button onClick={salvarEImprimir} disabled={salvando}>
            <Download className="w-4 h-4 mr-2" /> Salvar e gerar PDF
          </Button>
        </div>
      </div>

      {/* ── Identificação ────────────────────────────────────────────────── */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">1. Identificação</CardTitle>
          <CardDescription>Dados impressos na capa e na tabela de identificação.</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <div className="space-y-1.5">
            <Label>Cliente *</Label>
            <select className="w-full h-9 rounded-md border border-input bg-background px-3 text-sm"
                    value={form.cliente_id} onChange={e => set('cliente_id', e.target.value)}>
              <option value="">Selecione…</option>
              {clientes.map(c => <option key={c.id} value={c.id}>{c.razao_social}</option>)}
            </select>
          </div>
          <div className="space-y-1.5">
            <Label>Produto / Módulo</Label>
            <select className="w-full h-9 rounded-md border border-input bg-background px-3 text-sm"
                    value={form.produto_id} onChange={e => set('produto_id', e.target.value)}>
              <option value="">Selecione…</option>
              {produtos.map(p => <option key={p.id} value={p.id}>{p.nome} ({p.codigo})</option>)}
            </select>
          </div>
          <div className="space-y-1.5">
            <Label>Projeto</Label>
            <Input value={form.projeto} onChange={e => set('projeto', e.target.value)}
                   placeholder="Implantação FB_SMARTPICK — Grupo JC" />
          </div>
          <div className="space-y-1.5">
            <Label>Ambiente</Label>
            <Input value={form.ambiente} onChange={e => set('ambiente', e.target.value)}
                   placeholder="WMS Winthor (Totvs) — CD Goiânia" />
          </div>
          <div className="space-y-1.5">
            <Label>Início da implantação</Label>
            <Input type="date" value={form.data_inicio} onChange={e => set('data_inicio', e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <Label>Go-live</Label>
            <Input type="date" value={form.data_golive} onChange={e => set('data_golive', e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <Label>Data da entrega</Label>
            <Input type="date" value={form.data_entrega} onChange={e => set('data_entrega', e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <Label>Versão do documento</Label>
            <Input value={form.versao} onChange={e => set('versao', e.target.value)} placeholder="1.0" />
          </div>
          <div className="space-y-1.5 md:col-span-2">
            <Label>Resumo do escopo entregue</Label>
            <Textarea rows={3} value={form.escopo_resumo}
                      onChange={e => set('escopo_resumo', e.target.value)}
                      placeholder="Parágrafo de abertura da seção de escopo." />
          </div>
        </CardContent>
      </Card>

      {/* ── Escopo entregue ──────────────────────────────────────────────── */}
      <Card>
        <CardHeader className="flex-row items-start justify-between space-y-0">
          <div>
            <CardTitle className="text-base">2. Escopo entregue</CardTitle>
            <CardDescription>
              Uma linha por entregável. Sem nenhuma linha, o PDF imprime oito linhas
              pautadas para relacionar o escopo à mão.
            </CardDescription>
          </div>
          <Button size="sm" variant="outline" onClick={adicionarItem}>
            <Plus className="w-4 h-4 mr-1" /> Linha
          </Button>
        </CardHeader>
        <CardContent className="space-y-3">
          {form.itens.length === 0 && (
            <p className="text-sm text-muted-foreground">Nenhum entregável informado.</p>
          )}
          {form.itens.map((it, i) => (
            <div key={i} className="grid gap-2 md:grid-cols-12 items-start">
              <div className="md:col-span-4">
                {i === 0 && <Label className="text-xs mb-1 block">Entregável</Label>}
                <Input value={it.descricao} placeholder="Integração com o WMS"
                       onChange={e => atualizarLinha<Item>('itens', i, { descricao: e.target.value })} />
              </div>
              <div className="md:col-span-4">
                {i === 0 && <Label className="text-xs mb-1 block">Detalhamento</Label>}
                <Input value={it.detalhe} placeholder="Carga diária de posições e giro"
                       onChange={e => atualizarLinha<Item>('itens', i, { detalhe: e.target.value })} />
              </div>
              <div className="md:col-span-3">
                {i === 0 && <Label className="text-xs mb-1 block">Situação</Label>}
                <select className="w-full h-9 rounded-md border border-input bg-background px-3 text-sm"
                        value={it.situacao}
                        onChange={e => atualizarLinha<Item>('itens', i, { situacao: e.target.value })}>
                  {SITUACOES.map(s => <option key={s.valor} value={s.valor}>{s.rotulo}</option>)}
                </select>
              </div>
              <div className="md:col-span-1 flex md:justify-end">
                {i === 0 && <Label className="text-xs mb-1 block md:hidden">&nbsp;</Label>}
                <Button size="icon" variant="ghost" className={i === 0 ? 'md:mt-5' : ''}
                        onClick={() => removerLinha('itens', i)}>
                  <Trash2 className="w-4 h-4" />
                </Button>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>

      {/* ── Pendências ───────────────────────────────────────────────────── */}
      <Card>
        <CardHeader className="flex-row items-start justify-between space-y-0">
          <div>
            <CardTitle className="text-base">3. Pendências e ressalvas</CardTitle>
            <CardDescription>
              Pendências acordadas que não impedem o aceite.
            </CardDescription>
          </div>
          <Button size="sm" variant="outline" onClick={adicionarPendencia}>
            <Plus className="w-4 h-4 mr-1" /> Linha
          </Button>
        </CardHeader>
        <CardContent className="space-y-3">
          {form.pendencias.length === 0 && (
            <p className="text-sm text-muted-foreground">Nenhuma pendência registrada.</p>
          )}
          {form.pendencias.map((p, i) => (
            <div key={i} className="grid gap-2 md:grid-cols-12 items-start">
              <div className="md:col-span-6">
                {i === 0 && <Label className="text-xs mb-1 block">Pendência</Label>}
                <Input value={p.descricao} placeholder="Exportação XLSX do painel executivo"
                       onChange={e => atualizarLinha<Pendencia>('pendencias', i, { descricao: e.target.value })} />
              </div>
              <div className="md:col-span-3">
                {i === 0 && <Label className="text-xs mb-1 block">Responsável</Label>}
                <Input value={p.responsavel} placeholder="FBTECH"
                       onChange={e => atualizarLinha<Pendencia>('pendencias', i, { responsavel: e.target.value })} />
              </div>
              <div className="md:col-span-2">
                {i === 0 && <Label className="text-xs mb-1 block">Prazo</Label>}
                <Input type="date" value={p.prazo}
                       onChange={e => atualizarLinha<Pendencia>('pendencias', i, { prazo: e.target.value })} />
              </div>
              <div className="md:col-span-1 flex md:justify-end">
                <Button size="icon" variant="ghost" className={i === 0 ? 'md:mt-5' : ''}
                        onClick={() => removerLinha('pendencias', i)}>
                  <Trash2 className="w-4 h-4" />
                </Button>
              </div>
            </div>
          ))}
          <Separator />
          <div className="space-y-1.5 max-w-xs">
            <Label>Linhas pautadas em branco no PDF</Label>
            <Input type="number" min={0} max={20} value={form.linhas_livres}
                   onChange={e => set('linhas_livres', Number(e.target.value) || 0)} />
            <p className="text-xs text-muted-foreground">
              Espaço para ressalvas anotadas à mão durante a reunião de aceite.
            </p>
          </div>
        </CardContent>
      </Card>

      {/* ── Prazos e responsáveis ────────────────────────────────────────── */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">4. Prazos e responsáveis</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <div className="space-y-1.5">
            <Label>Operação assistida (dias)</Label>
            <Input type="number" min={0} value={form.assistida_dias}
                   onChange={e => set('assistida_dias', Number(e.target.value) || 0)} />
          </div>
          <div className="space-y-1.5">
            <Label>Garantia (dias)</Label>
            <Input type="number" min={0} value={form.garantia_dias}
                   onChange={e => set('garantia_dias', Number(e.target.value) || 0)} />
          </div>
          <div className="space-y-1.5">
            <Label>Responsável técnico</Label>
            <Input value={form.responsavel_tecnico}
                   onChange={e => set('responsavel_tecnico', e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <Label>Responsável comercial</Label>
            <Input value={form.responsavel_comercial}
                   onChange={e => set('responsavel_comercial', e.target.value)} />
          </div>
          <div className="space-y-1.5 md:col-span-2">
            <Label>Observações</Label>
            <Textarea rows={3} value={form.observacoes}
                      onChange={e => set('observacoes', e.target.value)} />
          </div>
        </CardContent>
      </Card>

      {/* ── Assinaturas ──────────────────────────────────────────────────── */}
      <Card>
        <CardHeader className="flex-row items-start justify-between space-y-0">
          <div>
            <CardTitle className="text-base">5. Assinaturas</CardTitle>
            <CardDescription>
              Um bloco de assinatura por linha. Nome, cargo e CPF em branco viram
              linha pautada no PDF.
            </CardDescription>
          </div>
          <Button size="sm" variant="outline" onClick={adicionarSignatario}>
            <Plus className="w-4 h-4 mr-1" /> Linha
          </Button>
        </CardHeader>
        <CardContent className="space-y-3">
          {form.signatarios.length === 0 && (
            <p className="text-sm text-muted-foreground">
              Sem signatários informados, o PDF imprime os blocos padrão
              (CONTRATADA e CLIENTE).
            </p>
          )}
          {form.signatarios.map((s, i) => (
            <div key={i} className="grid gap-2 md:grid-cols-12 items-start">
              <div className="md:col-span-3">
                {i === 0 && <Label className="text-xs mb-1 block">Papel</Label>}
                <Input value={s.papel} placeholder="PELO CLIENTE"
                       onChange={e => atualizarLinha<Signatario>('signatarios', i, { papel: e.target.value })} />
              </div>
              <div className="md:col-span-3">
                {i === 0 && <Label className="text-xs mb-1 block">Nome</Label>}
                <Input value={s.nome}
                       onChange={e => atualizarLinha<Signatario>('signatarios', i, { nome: e.target.value })} />
              </div>
              <div className="md:col-span-2">
                {i === 0 && <Label className="text-xs mb-1 block">Cargo</Label>}
                <Input value={s.cargo}
                       onChange={e => atualizarLinha<Signatario>('signatarios', i, { cargo: e.target.value })} />
              </div>
              <div className="md:col-span-2">
                {i === 0 && <Label className="text-xs mb-1 block">Empresa</Label>}
                <Input value={s.organizacao}
                       onChange={e => atualizarLinha<Signatario>('signatarios', i, { organizacao: e.target.value })} />
              </div>
              <div className="md:col-span-1">
                {i === 0 && <Label className="text-xs mb-1 block">CPF</Label>}
                <Input value={s.documento}
                       onChange={e => atualizarLinha<Signatario>('signatarios', i, { documento: e.target.value })} />
              </div>
              <div className="md:col-span-1 flex md:justify-end">
                <Button size="icon" variant="ghost" className={i === 0 ? 'md:mt-5' : ''}
                        onClick={() => removerLinha('signatarios', i)}>
                  <Trash2 className="w-4 h-4" />
                </Button>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}
