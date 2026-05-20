import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { toast } from 'sonner'

interface ClienteSimples {
  id: string
  razao_social: string
  cnpj: string
  ativo: boolean
}

interface CNPJ {
  id: string
  cnpj: string
  descricao?: string
  is_principal: boolean
}

interface Plano {
  id: string
  nome: string
  preco: number | null
}

interface Produto {
  id: string
  codigo: string
  nome: string
  planos: Plano[]
}

interface Contrato {
  id: string
  cliente_id: string
  data_inicio: string
  periodicidade: string
  valor_total: number
  status: string
  observacoes?: string
  created_at: string
}

interface ItemSelecionado {
  plano_id: string
  valor_item: string
}

interface EditandoContrato {
  id: string
  periodicidade: string
  valor_total: string
  status: string
  observacoes: string
}

function formatMoeda(valor: number): string {
  return valor.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })
}

function statusVariant(status: string): 'default' | 'secondary' | 'destructive' {
  if (status === 'ativo') return 'default'
  if (status === 'encerrado') return 'destructive'
  return 'secondary'
}

export default function ContratosPage() {
  const [searchParams] = useSearchParams()
  const [view, setView] = useState<'lista' | 'novo'>(
    searchParams.get('view') === 'novo' ? 'novo' : 'lista'
  )

  // Lista
  const [clientes, setClientes] = useState<ClienteSimples[]>([])
  const [contratos, setContratos] = useState<Contrato[]>([])
  const [clienteIdFiltro, setClienteIdFiltro] = useState('')
  const [loadingLista, setLoadingLista] = useState(false)

  // Edição
  const [editandoContrato, setEditandoContrato] = useState<EditandoContrato | null>(null)
  const [submittingEdit, setSubmittingEdit] = useState(false)

  // Formulário novo contrato
  const [etapa, setEtapa] = useState<1 | 2 | 3>(1)
  const [clienteSelecionadoId, setClienteSelecionadoId] = useState('')
  const [cnpjsDoGrupo, setCnpjsDoGrupo] = useState<CNPJ[]>([])
  const [cnpjsSelecionados, setCnpjsSelecionados] = useState<string[]>([])
  const [produtos, setProdutos] = useState<Produto[]>([])
  const [itensSelecionados, setItensSelecionados] = useState<ItemSelecionado[]>([])
  const [dataInicio, setDataInicio] = useState('')
  const [periodicidade, setPeriodicidade] = useState<'mensal' | 'trimestral' | 'anual'>('mensal')
  const [valorTotal, setValorTotal] = useState('')
  const [observacoes, setObservacoes] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    Promise.all([
      fetch('/api/financeiro/clientes').then(r => r.json()),
      fetch('/api/financeiro/produtos').then(r => r.json()),
    ]).then(([cli, prod]) => {
      setClientes(cli)
      setProdutos(prod)
    }).catch(err => setError(err instanceof Error ? err.message : 'Erro ao carregar dados'))
  }, [])

  const fetchContratos = async (clienteId: string) => {
    if (!clienteId) { setContratos([]); return }
    setLoadingLista(true)
    try {
      const res = await fetch(`/api/financeiro/contratos?cliente_id=${clienteId}`)
      if (!res.ok) throw new Error('Erro ao carregar contratos')
      setContratos(await res.json())
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro')
    } finally {
      setLoadingLista(false)
    }
  }

  const handleClienteFiltroChange = (id: string) => {
    setClienteIdFiltro(id)
    fetchContratos(id)
  }

  const handleSelecionarCliente = async (id: string) => {
    setClienteSelecionadoId(id)
    setCnpjsDoGrupo([])
    setCnpjsSelecionados([])
    if (!id) return
    try {
      const res = await fetch(`/api/financeiro/clientes?id=${id}`)
      const data = await res.json()
      setCnpjsDoGrupo(data.cnpjs ?? [])
    } catch {
      toast.error('Erro ao carregar CNPJs do cliente')
    }
  }

  const toggleCNPJ = (id: string) => {
    setCnpjsSelecionados(prev =>
      prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id]
    )
  }

  const togglePlano = (planoId: string) => {
    setItensSelecionados(prev => {
      const existe = prev.find(i => i.plano_id === planoId)
      if (existe) return prev.filter(i => i.plano_id !== planoId)
      return [...prev, { plano_id: planoId, valor_item: '' }]
    })
  }

  const setValorItem = (planoId: string, valor: string) => {
    setItensSelecionados(prev =>
      prev.map(i => i.plano_id === planoId ? { ...i, valor_item: valor } : i)
    )
  }

  const handleSubmitContrato = async () => {
    if (itensSelecionados.length === 0 || !dataInicio || !valorTotal) return
    const valorTotalNum = parseFloat(valorTotal)
    if (isNaN(valorTotalNum)) { toast.error('Valor total inválido'); return }
    setSubmitting(true)
    try {
      const body = {
        cliente_id: clienteSelecionadoId,
        data_inicio: dataInicio,
        periodicidade,
        valor_total: valorTotalNum,
        observacoes: observacoes || undefined,
        cnpj_ids: cnpjsSelecionados,
        itens: itensSelecionados.map(i => ({
          plano_id: i.plano_id,
          valor_item: i.valor_item ? parseFloat(i.valor_item) : null,
        })),
      }
      const res = await fetch('/api/financeiro/contratos', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (!res.ok) {
        const text = await res.text()
        throw new Error(text || 'Erro ao criar contrato')
      }
      toast.success('Contrato criado com sucesso')
      setView('lista')
      setEtapa(1)
      setClienteSelecionadoId('')
      setCnpjsSelecionados([])
      setItensSelecionados([])
      setDataInicio('')
      setValorTotal('')
      setObservacoes('')
      setClienteIdFiltro(clienteSelecionadoId)
      fetchContratos(clienteSelecionadoId)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro desconhecido')
    } finally {
      setSubmitting(false)
    }
  }

  const handleEditarContrato = (c: Contrato) => {
    setEditandoContrato({
      id: c.id,
      periodicidade: c.periodicidade,
      valor_total: String(c.valor_total),
      status: c.status,
      observacoes: c.observacoes ?? '',
    })
  }

  const handleSalvarEdicao = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editandoContrato) return
    const valorTotalNum = parseFloat(editandoContrato.valor_total)
    if (isNaN(valorTotalNum)) { toast.error('Valor total inválido'); return }
    setSubmittingEdit(true)
    try {
      const res = await fetch('/api/financeiro/contratos', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          id: editandoContrato.id,
          periodicidade: editandoContrato.periodicidade,
          valor_total: valorTotalNum,
          status: editandoContrato.status,
          observacoes: editandoContrato.observacoes,
        }),
      })
      if (!res.ok) throw new Error('Erro ao atualizar contrato')
      toast.success('Contrato atualizado')
      setEditandoContrato(null)
      fetchContratos(clienteIdFiltro)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro')
    } finally {
      setSubmittingEdit(false)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-4xl mx-auto space-y-6">
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {view === 'lista' && (
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle>Contratos</CardTitle>
              <Button onClick={() => { setView('novo'); setEtapa(1) }}>
                Novo Contrato
              </Button>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label>Filtrar por cliente</Label>
                <select
                  className="mt-1 w-full border border-input bg-background rounded-md px-3 py-2 text-sm"
                  value={clienteIdFiltro}
                  onChange={e => handleClienteFiltroChange(e.target.value)}
                >
                  <option value="">Selecione um cliente...</option>
                  {clientes.map(c => (
                    <option key={c.id} value={c.id}>{c.razao_social}</option>
                  ))}
                </select>
              </div>

              {loadingLista ? (
                <p className="text-sm text-muted-foreground">Carregando...</p>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Data Início</TableHead>
                      <TableHead>Periodicidade</TableHead>
                      <TableHead>Valor Total</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead className="w-24">Ações</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {contratos.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                          {clienteIdFiltro ? 'Nenhum contrato encontrado.' : 'Selecione um cliente para ver os contratos.'}
                        </TableCell>
                      </TableRow>
                    ) : (
                      contratos.map(c => (
                        <TableRow key={c.id}>
                          <TableCell>{c.data_inicio}</TableCell>
                          <TableCell className="capitalize">{c.periodicidade}</TableCell>
                          <TableCell>{formatMoeda(c.valor_total)}</TableCell>
                          <TableCell>
                            <Badge variant={statusVariant(c.status)} className="capitalize">
                              {c.status}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            <Button variant="outline" size="sm" onClick={() => handleEditarContrato(c)}>
                              Editar
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        )}

        {view === 'novo' && (
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle>Novo Contrato</CardTitle>
                <p className="text-sm text-muted-foreground mt-1">Etapa {etapa} de 3</p>
              </div>
              <Button variant="outline" onClick={() => { setView('lista'); setEtapa(1) }}>
                Cancelar
              </Button>
            </CardHeader>
            <CardContent className="space-y-6">

              {etapa === 1 && (
                <div className="space-y-4">
                  <h3 className="font-medium">Selecionar Cliente</h3>
                  <div>
                    <Label>Cliente *</Label>
                    <select
                      className="mt-1 w-full border border-input bg-background rounded-md px-3 py-2 text-sm"
                      value={clienteSelecionadoId}
                      onChange={e => handleSelecionarCliente(e.target.value)}
                    >
                      <option value="">Selecione...</option>
                      {clientes.filter(c => c.ativo).map(c => (
                        <option key={c.id} value={c.id}>{c.razao_social}</option>
                      ))}
                    </select>
                  </div>
                  <Button
                    disabled={!clienteSelecionadoId}
                    onClick={() => setEtapa(2)}
                  >
                    Próximo
                  </Button>
                </div>
              )}

              {etapa === 2 && (
                <div className="space-y-4">
                  <h3 className="font-medium">CNPJs cobertos pelo contrato</h3>
                  {cnpjsDoGrupo.length === 0 ? (
                    <p className="text-sm text-muted-foreground">Nenhum CNPJ cadastrado para este cliente.</p>
                  ) : (
                    <div className="space-y-2">
                      {cnpjsDoGrupo.map(cnpj => (
                        <label key={cnpj.id} className="flex items-center gap-2 cursor-pointer">
                          <input
                            type="checkbox"
                            checked={cnpjsSelecionados.includes(cnpj.id)}
                            onChange={() => toggleCNPJ(cnpj.id)}
                            className="h-4 w-4"
                          />
                          <span className="text-sm">
                            {cnpj.cnpj}
                            {cnpj.is_principal ? ' (Principal)' : cnpj.descricao ? ` — ${cnpj.descricao}` : ''}
                          </span>
                        </label>
                      ))}
                    </div>
                  )}
                  <div className="flex gap-2">
                    <Button variant="outline" onClick={() => setEtapa(1)}>Voltar</Button>
                    <Button
                      disabled={cnpjsSelecionados.length === 0 && cnpjsDoGrupo.length > 0}
                      onClick={() => setEtapa(3)}
                    >
                      Próximo
                    </Button>
                  </div>
                </div>
              )}

              {etapa === 3 && (
                <div className="space-y-6">
                  <h3 className="font-medium">Produtos, Planos e Valores</h3>

                  <div className="space-y-3">
                    {produtos.map(produto => (
                      <details key={produto.id} className="border rounded-md">
                        <summary className="px-4 py-3 cursor-pointer font-medium select-none">
                          {produto.nome} <span className="text-muted-foreground text-sm ml-2">{produto.codigo}</span>
                        </summary>
                        <div className="px-4 pb-4 space-y-2">
                          {produto.planos.map(plano => {
                            const selecionado = itensSelecionados.find(i => i.plano_id === plano.id)
                            return (
                              <div key={plano.id} className="flex items-center gap-3">
                                <input
                                  type="checkbox"
                                  id={`plano-${plano.id}`}
                                  checked={!!selecionado}
                                  onChange={() => togglePlano(plano.id)}
                                  className="h-4 w-4"
                                />
                                <label htmlFor={`plano-${plano.id}`} className="text-sm w-32 cursor-pointer">
                                  {plano.nome}
                                </label>
                                {selecionado && (
                                  <div className="flex items-center gap-1">
                                    <span className="text-muted-foreground text-sm">R$</span>
                                    <Input
                                      className="w-28"
                                      type="text"
                                      placeholder="valor"
                                      value={selecionado.valor_item}
                                      onChange={e => setValorItem(plano.id, e.target.value)}
                                    />
                                  </div>
                                )}
                              </div>
                            )
                          })}
                        </div>
                      </details>
                    ))}
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <Label htmlFor="data_inicio">Data de Início *</Label>
                      <Input
                        id="data_inicio"
                        type="date"
                        value={dataInicio}
                        onChange={e => setDataInicio(e.target.value)}
                        required
                      />
                    </div>
                    <div>
                      <Label htmlFor="periodicidade">Periodicidade *</Label>
                      <select
                        id="periodicidade"
                        className="mt-1 w-full border border-input bg-background rounded-md px-3 py-2 text-sm"
                        value={periodicidade}
                        onChange={e => setPeriodicidade(e.target.value as 'mensal' | 'trimestral' | 'anual')}
                      >
                        <option value="mensal">Mensal</option>
                        <option value="trimestral">Trimestral</option>
                        <option value="anual">Anual</option>
                      </select>
                    </div>
                    <div>
                      <Label htmlFor="valor_total">Valor Total *</Label>
                      <div className="flex items-center gap-2 mt-1">
                        <span className="text-muted-foreground text-sm">R$</span>
                        <Input
                          id="valor_total"
                          type="text"
                          value={valorTotal}
                          onChange={e => setValorTotal(e.target.value)}
                          placeholder="0,00"
                          required
                        />
                      </div>
                    </div>
                    <div>
                      <Label htmlFor="observacoes">Observações</Label>
                      <Input
                        id="observacoes"
                        value={observacoes}
                        onChange={e => setObservacoes(e.target.value)}
                        placeholder="Opcional"
                      />
                    </div>
                  </div>

                  <div className="flex gap-2">
                    <Button variant="outline" onClick={() => setEtapa(2)}>Voltar</Button>
                    <Button
                      disabled={itensSelecionados.length === 0 || !dataInicio || !valorTotal || submitting}
                      onClick={handleSubmitContrato}
                    >
                      {submitting ? 'Criando...' : 'Criar Contrato'}
                    </Button>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        )}

        {/* Dialog de edição de contrato */}
        <Dialog open={editandoContrato !== null} onOpenChange={open => !open && setEditandoContrato(null)}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Editar Contrato</DialogTitle>
            </DialogHeader>
            {editandoContrato && (
              <form onSubmit={handleSalvarEdicao} className="space-y-4">
                <div>
                  <Label htmlFor="edit_periodicidade">Periodicidade</Label>
                  <select
                    id="edit_periodicidade"
                    className="mt-1 w-full border border-input bg-background rounded-md px-3 py-2 text-sm"
                    value={editandoContrato.periodicidade}
                    onChange={e => setEditandoContrato(prev => prev ? { ...prev, periodicidade: e.target.value } : prev)}
                  >
                    <option value="mensal">Mensal</option>
                    <option value="trimestral">Trimestral</option>
                    <option value="anual">Anual</option>
                  </select>
                </div>
                <div>
                  <Label htmlFor="edit_valor_total">Valor Total</Label>
                  <Input
                    id="edit_valor_total"
                    value={editandoContrato.valor_total}
                    onChange={e => setEditandoContrato(prev => prev ? { ...prev, valor_total: e.target.value } : prev)}
                    required
                  />
                </div>
                <div>
                  <Label htmlFor="edit_status">Status</Label>
                  <select
                    id="edit_status"
                    className="mt-1 w-full border border-input bg-background rounded-md px-3 py-2 text-sm"
                    value={editandoContrato.status}
                    onChange={e => setEditandoContrato(prev => prev ? { ...prev, status: e.target.value } : prev)}
                  >
                    <option value="ativo">Ativo</option>
                    <option value="suspenso">Suspenso</option>
                    <option value="encerrado">Encerrado</option>
                  </select>
                </div>
                <div>
                  <Label htmlFor="edit_observacoes">Observações</Label>
                  <Input
                    id="edit_observacoes"
                    value={editandoContrato.observacoes}
                    onChange={e => setEditandoContrato(prev => prev ? { ...prev, observacoes: e.target.value } : prev)}
                    placeholder="Opcional"
                  />
                </div>
                <div className="flex gap-2 justify-end pt-2">
                  <Button type="button" variant="outline" onClick={() => setEditandoContrato(null)}>
                    Cancelar
                  </Button>
                  <Button type="submit" disabled={submittingEdit}>
                    {submittingEdit ? 'Salvando...' : 'Salvar'}
                  </Button>
                </div>
              </form>
            )}
          </DialogContent>
        </Dialog>
      </div>
    </div>
  )
}
