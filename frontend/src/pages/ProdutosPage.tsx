import { useState, useEffect, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Calculator, Plus, Trash2, Pencil, Check } from 'lucide-react'
import { toast } from 'sonner'

// ── Tipos ──────────────────────────────────────────────────────────────────────

interface Plano {
  id: string
  produto_id: string
  nome: string
  preco: number | null
  ativo: boolean
}

interface Produto {
  id: string
  codigo: string
  nome: string
  descricao?: string
  ativo: boolean
  planos: Plano[]
}

interface FaixaPreco {
  id: string
  produto_id: string | null
  mb_min: number
  mb_max: number | null
  preco: number | null
  descricao: string
  ativo: boolean
  ordem: number
}

// ── Formatadores ──────────────────────────────────────────────────────────────

function fmtBRL(v: number) {
  return v.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })
}

function fmtMB(min: number, max: number | null) {
  if (max === null) return `Acima de ${min} MB`
  if (min === 0) return `Até ${max} MB`
  return `De ${min} MB até ${max} MB`
}

// ── Calculadora ───────────────────────────────────────────────────────────────

function Calculadora({ faixas }: { faixas: FaixaPreco[] }) {
  const [mb, setMb] = useState('')

  const faixaAtual = mb !== '' && !isNaN(parseFloat(mb))
    ? faixas.find(f => {
        const v = parseFloat(mb)
        return v >= f.mb_min && (f.mb_max === null || v < f.mb_max)
      }) ?? null
    : null

  return (
    <Card className="border-violet-100 bg-violet-50/30">
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-semibold flex items-center gap-2">
          <Calculator className="w-4 h-4 text-violet-600" />
          Calculadora de Preço por Volume
        </CardTitle>
        <CardDescription className="text-xs">
          Informe a quantidade de MB de dados processados por mês para consultar o valor.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-end gap-3">
          <div className="flex-1 max-w-xs">
            <Label className="text-xs mb-1 block">Volume mensal de dados</Label>
            <div className="relative">
              <Input
                type="number"
                min="0"
                step="0.1"
                placeholder="Ex: 150"
                value={mb}
                onChange={e => setMb(e.target.value)}
                className="pr-10"
              />
              <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">MB</span>
            </div>
          </div>
        </div>

        {mb !== '' && !isNaN(parseFloat(mb)) && (
          <div className={`rounded-lg border p-4 transition-all ${
            faixaAtual
              ? faixaAtual.preco !== null
                ? 'bg-emerald-50 border-emerald-200'
                : 'bg-amber-50 border-amber-200'
              : 'bg-slate-50 border-slate-200'
          }`}>
            {faixaAtual ? (
              <>
                <p className="text-xs text-muted-foreground mb-1">{faixaAtual.descricao}</p>
                {faixaAtual.preco !== null ? (
                  <p className="text-3xl font-bold text-emerald-700">
                    {fmtBRL(faixaAtual.preco)}
                    <span className="text-sm font-normal text-muted-foreground ml-2">/mês</span>
                  </p>
                ) : (
                  <div>
                    <p className="text-xl font-bold text-amber-700">Sob Consulta</p>
                    <p className="text-xs text-muted-foreground mt-1">
                      Para volumes acima de {faixaAtual.mb_min} MB, entre em contato para uma proposta personalizada.
                    </p>
                  </div>
                )}
              </>
            ) : (
              <p className="text-sm text-muted-foreground">Nenhuma faixa encontrada para este volume.</p>
            )}
          </div>
        )}

        {/* Tabela resumo das faixas */}
        <div className="rounded-lg border overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50">
              <tr>
                <th className="px-4 py-2 text-left text-xs font-semibold text-slate-600">Volume</th>
                <th className="px-4 py-2 text-right text-xs font-semibold text-slate-600">Preço mensal</th>
              </tr>
            </thead>
            <tbody>
              {faixas.map((f, i) => {
                const v = mb !== '' && !isNaN(parseFloat(mb)) ? parseFloat(mb) : null
                const ativa = v !== null && v >= f.mb_min && (f.mb_max === null || v < f.mb_max)
                return (
                  <tr key={f.id} className={`border-t ${ativa ? 'bg-violet-50' : i % 2 === 0 ? '' : 'bg-slate-50/50'}`}>
                    <td className="px-4 py-2.5 text-slate-700">
                      {ativa && <span className="inline-block w-1.5 h-1.5 rounded-full bg-violet-500 mr-2 align-middle" />}
                      {fmtMB(f.mb_min, f.mb_max)}
                    </td>
                    <td className={`px-4 py-2.5 text-right font-medium ${ativa ? 'text-violet-700' : ''}`}>
                      {f.preco !== null ? fmtBRL(f.preco) : (
                        <span className="text-amber-600 text-xs">Sob Consulta</span>
                      )}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  )
}

// ── Gestão de faixas ──────────────────────────────────────────────────────────

function GestaoFaixas({ faixas, onRefresh }: { faixas: FaixaPreco[]; onRefresh: () => void }) {
  const [editando, setEditando] = useState<FaixaPreco | null>(null)
  const [nova, setNova] = useState<Partial<FaixaPreco>>({})
  const [modalNova, setModalNova] = useState(false)
  const [salvando, setSalvando] = useState(false)

  const handleDelete = async (id: string) => {
    if (!confirm('Remover esta faixa?')) return
    await fetch(`/api/financeiro/faixas-preco?id=${id}`, { method: 'DELETE' })
    toast.success('Faixa removida')
    onRefresh()
  }

  const handleSaveEdit = async () => {
    if (!editando) return
    setSalvando(true)
    try {
      const res = await fetch('/api/financeiro/faixas-preco', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(editando),
      })
      if (!res.ok) throw new Error('Erro ao salvar')
      toast.success('Faixa atualizada')
      setEditando(null)
      onRefresh()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro')
    } finally {
      setSalvando(false)
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setSalvando(true)
    try {
      const payload = {
        ...nova,
        mb_min: parseFloat(String(nova.mb_min ?? 0)),
        mb_max: nova.mb_max !== undefined && nova.mb_max !== null && String(nova.mb_max) !== ''
          ? parseFloat(String(nova.mb_max))
          : null,
        preco: nova.preco !== undefined && nova.preco !== null && String(nova.preco) !== ''
          ? parseFloat(String(nova.preco))
          : null,
        ordem: faixas.length + 1,
      }
      const res = await fetch('/api/financeiro/faixas-preco', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!res.ok) throw new Error('Erro ao criar faixa')
      toast.success('Faixa criada')
      setModalNova(false)
      setNova({})
      onRefresh()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro')
    } finally {
      setSalvando(false)
    }
  }

  return (
    <Card>
      <CardHeader className="pb-3 flex flex-row items-center">
        <div className="flex-1">
          <CardTitle className="text-sm font-semibold">Gerenciar Faixas de Preço</CardTitle>
          <CardDescription className="text-xs mt-0.5">
            Edite os limites de MB e valores de cada faixa. Preco vazio = Sob Consulta.
          </CardDescription>
        </div>
        <Button size="sm" variant="outline" onClick={() => setModalNova(true)}>
          <Plus className="w-3.5 h-3.5 mr-1.5" /> Nova faixa
        </Button>
      </CardHeader>
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Descrição</TableHead>
              <TableHead className="text-right">MB mín.</TableHead>
              <TableHead className="text-right">MB máx.</TableHead>
              <TableHead className="text-right">Preço</TableHead>
              <TableHead className="w-20" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {faixas.map(f => (
              <TableRow key={f.id}>
                {editando?.id === f.id ? (
                  <>
                    <TableCell>
                      <Input
                        value={editando.descricao}
                        onChange={e => setEditando(p => p && ({ ...p, descricao: e.target.value }))}
                        className="h-8 text-sm"
                      />
                    </TableCell>
                    <TableCell className="text-right">
                      <Input
                        type="number" min="0" step="0.1"
                        value={editando.mb_min}
                        onChange={e => setEditando(p => p && ({ ...p, mb_min: parseFloat(e.target.value) || 0 }))}
                        className="h-8 text-sm w-24 ml-auto"
                      />
                    </TableCell>
                    <TableCell className="text-right">
                      <Input
                        type="number" min="0" step="0.1"
                        value={editando.mb_max ?? ''}
                        placeholder="sem limite"
                        onChange={e => setEditando(p => p && ({ ...p, mb_max: e.target.value !== '' ? parseFloat(e.target.value) : null }))}
                        className="h-8 text-sm w-24 ml-auto"
                      />
                    </TableCell>
                    <TableCell className="text-right">
                      <Input
                        type="number" min="0" step="0.01"
                        value={editando.preco ?? ''}
                        placeholder="sob consulta"
                        onChange={e => setEditando(p => p && ({ ...p, preco: e.target.value !== '' ? parseFloat(e.target.value) : null }))}
                        className="h-8 text-sm w-32 ml-auto"
                      />
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        <Button size="sm" variant="ghost" className="h-8 w-8 p-0" onClick={handleSaveEdit} disabled={salvando}>
                          <Check className="w-3.5 h-3.5 text-emerald-600" />
                        </Button>
                        <Button size="sm" variant="ghost" className="h-8 w-8 p-0" onClick={() => setEditando(null)}>
                          ×
                        </Button>
                      </div>
                    </TableCell>
                  </>
                ) : (
                  <>
                    <TableCell className="text-sm">{f.descricao || fmtMB(f.mb_min, f.mb_max)}</TableCell>
                    <TableCell className="text-right text-sm text-muted-foreground">{f.mb_min} MB</TableCell>
                    <TableCell className="text-right text-sm text-muted-foreground">
                      {f.mb_max !== null ? `${f.mb_max} MB` : '—'}
                    </TableCell>
                    <TableCell className="text-right text-sm font-medium">
                      {f.preco !== null ? fmtBRL(f.preco) : (
                        <span className="text-amber-600 text-xs font-normal">Sob Consulta</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        <Button size="sm" variant="ghost" className="h-8 w-8 p-0" onClick={() => setEditando({ ...f })}>
                          <Pencil className="w-3.5 h-3.5" />
                        </Button>
                        <Button size="sm" variant="ghost" className="h-8 w-8 p-0 text-red-500 hover:text-red-700" onClick={() => handleDelete(f.id)}>
                          <Trash2 className="w-3.5 h-3.5" />
                        </Button>
                      </div>
                    </TableCell>
                  </>
                )}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>

      {/* Modal nova faixa */}
      <Dialog open={modalNova} onOpenChange={setModalNova}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>Nova Faixa de Preço</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleCreate} className="space-y-3">
            <div>
              <Label>Descrição</Label>
              <Input value={nova.descricao ?? ''} onChange={e => setNova(p => ({ ...p, descricao: e.target.value }))} placeholder="Ex: Até 20 MB" className="mt-1" />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label>MB mínimo *</Label>
                <Input type="number" min="0" step="0.1" required value={nova.mb_min ?? ''} onChange={e => setNova(p => ({ ...p, mb_min: parseFloat(e.target.value) }))} className="mt-1" />
              </div>
              <div>
                <Label>MB máximo</Label>
                <Input type="number" min="0" step="0.1" value={nova.mb_max ?? ''} placeholder="sem limite" onChange={e => setNova(p => ({ ...p, mb_max: e.target.value !== '' ? parseFloat(e.target.value) : null }))} className="mt-1" />
              </div>
            </div>
            <div>
              <Label>Preço (R$)</Label>
              <Input type="number" min="0" step="0.01" value={nova.preco ?? ''} placeholder="vazio = Sob Consulta" onChange={e => setNova(p => ({ ...p, preco: e.target.value !== '' ? parseFloat(e.target.value) : null }))} className="mt-1" />
            </div>
            <div className="flex justify-end gap-2 pt-1">
              <Button type="button" variant="outline" onClick={() => setModalNova(false)}>Cancelar</Button>
              <Button type="submit" disabled={salvando}>{salvando ? 'Salvando...' : 'Criar faixa'}</Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </Card>
  )
}

// ── Página principal ──────────────────────────────────────────────────────────

export default function ProdutosPage() {
  const [produtos, setProdutos] = useState<Produto[]>([])
  const [faixas, setFaixas] = useState<FaixaPreco[]>([])
  const [loading, setLoading] = useState(true)
  const [salvando, setSalvando] = useState<Record<string, boolean>>({})
  const [precoEditado, setPrecoEditado] = useState<Record<string, string>>({})
  const [error, setError] = useState<string | null>(null)

  const loadFaixas = useCallback(() => {
    fetch('/api/financeiro/faixas-preco')
      .then(r => r.json())
      .then(setFaixas)
      .catch(() => {})
  }, [])

  useEffect(() => {
    Promise.all([
      fetch('/api/financeiro/produtos').then(r => r.json()),
      fetch('/api/financeiro/faixas-preco').then(r => r.json()),
    ]).then(([data, fx]) => {
      setProdutos(data)
      setFaixas(fx ?? [])
      const precos: Record<string, string> = {}
      data.forEach((p: Produto) => p.planos.forEach(pl => {
        precos[pl.id] = pl.preco != null ? String(pl.preco) : ''
      }))
      setPrecoEditado(precos)
    }).catch(err => setError(err instanceof Error ? err.message : 'Erro ao carregar'))
      .finally(() => setLoading(false))
  }, [])

  const handleSalvarPreco = async (plano: Plano) => {
    const valor = precoEditado[plano.id]
    let preco: number | null = null
    if (valor !== '') {
      preco = parseFloat(valor)
      if (isNaN(preco)) { toast.error('Preço inválido'); return }
    }
    setSalvando(prev => ({ ...prev, [plano.id]: true }))
    try {
      const res = await fetch('/api/financeiro/planos', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: plano.id, preco }),
      })
      if (!res.ok) throw new Error('Erro ao atualizar preço')
      toast.success('Preço atualizado')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro desconhecido')
    } finally {
      setSalvando(prev => ({ ...prev, [plano.id]: false }))
    }
  }

  if (loading) {
    return <div className="min-h-screen p-8 flex items-center justify-center">Carregando...</div>
  }

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-4xl mx-auto space-y-6">
        <div>
          <h1 className="text-2xl font-bold">Produtos e Planos</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Gerencie os produtos FB, planos fixos e a tabela de preços por volume de dados.
          </p>
        </div>

        {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}

        {/* ── Calculadora ── */}
        <Calculadora faixas={faixas} />

        <Separator />

        {/* ── Gestão das faixas ── */}
        <GestaoFaixas faixas={faixas} onRefresh={loadFaixas} />

        <Separator />

        {/* ── Planos por produto (preço fixo legado) ── */}
        <div>
          <h2 className="text-base font-semibold mb-4">Planos por Produto</h2>
          <div className="space-y-4">
            {produtos.map(produto => (
              <Card key={produto.id}>
                <CardHeader className="flex flex-row items-center gap-3 pb-3">
                  <CardTitle className="flex-1 text-base">{produto.nome}</CardTitle>
                  <Badge variant="outline">{produto.codigo}</Badge>
                  <Badge variant={produto.ativo ? 'default' : 'secondary'}>
                    {produto.ativo ? 'Ativo' : 'Inativo'}
                  </Badge>
                </CardHeader>
                <CardContent>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Plano</TableHead>
                        <TableHead>Preço Mensal</TableHead>
                        <TableHead className="w-24">Ações</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {produto.planos.map(plano => (
                        <TableRow key={plano.id}>
                          <TableCell className="font-medium">{plano.nome}</TableCell>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              <span className="text-muted-foreground text-sm">R$</span>
                              <Input
                                className="w-32"
                                type="text"
                                placeholder="—"
                                value={precoEditado[plano.id] ?? ''}
                                onChange={e => setPrecoEditado(prev => ({ ...prev, [plano.id]: e.target.value }))}
                              />
                            </div>
                          </TableCell>
                          <TableCell>
                            <Button
                              size="sm"
                              variant="outline"
                              disabled={salvando[plano.id]}
                              onClick={() => handleSalvarPreco(plano)}
                            >
                              {salvando[plano.id] ? 'Salvando...' : 'Salvar'}
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
