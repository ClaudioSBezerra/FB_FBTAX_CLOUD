import { useState, useEffect, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { toast } from 'sonner'
import { buscarCEP } from '@/lib/viacep'

interface Cliente {
  id?: string
  razao_social: string
  cnpj: string
  email?: string
  telefone?: string
  responsavel?: string
  ativo: boolean
  cep?: string
  logradouro?: string
  numero?: string
  complemento?: string
  bairro?: string
  municipio?: string
  uf?: string
}

function formatCNPJ(cnpj: string): string {
  const digits = cnpj.replace(/\D/g, '')
  if (digits.length !== 14) return cnpj
  return digits.replace(/^(\d{2})(\d{3})(\d{3})(\d{4})(\d{2})$/, '$1.$2.$3/$4-$5')
}

const emptyCliente = (): Cliente => ({
  razao_social: '', cnpj: '', ativo: true,
  cep: '', logradouro: '', numero: '', complemento: '',
  bairro: '', municipio: '', uf: '',
  email: '', telefone: '', responsavel: '',
})

export default function ClientesPage() {
  const [clientes, setClientes] = useState<Cliente[]>([])
  const [editando, setEditando] = useState<Cliente | null>(null)
  const [busca, setBusca] = useState('')
  const [statusFiltro, setStatusFiltro] = useState<'todos' | 'ativo' | 'inativo'>('todos')
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [buscandoCEP, setBuscandoCEP] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const fetchClientes = async (q: string, status: string) => {
    const params = new URLSearchParams()
    if (q) params.set('q', q)
    if (status !== 'todos') params.set('status', status)
    const url = '/api/financeiro/clientes' + (params.toString() ? '?' + params.toString() : '')
    const res = await fetch(url)
    if (!res.ok) throw new Error('Erro ao carregar clientes')
    return res.json() as Promise<Cliente[]>
  }

  useEffect(() => {
    if (timerRef.current) clearTimeout(timerRef.current)
    timerRef.current = setTimeout(async () => {
      try {
        const data = await fetchClientes(busca, statusFiltro)
        setClientes(data)
        setError(null)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Erro desconhecido')
      } finally {
        setLoading(false)
      }
    }, busca ? 300 : 0)
    return () => { if (timerRef.current) clearTimeout(timerRef.current) }
  }, [busca, statusFiltro])

  const handleCEPBlur = async () => {
    if (!editando) return
    const digits = (editando.cep || '').replace(/\D/g, '')
    if (digits.length !== 8) return
    setBuscandoCEP(true)
    try {
      const result = await buscarCEP(digits)
      if (result) {
        setEditando(prev => prev ? {
          ...prev,
          logradouro: result.logradouro || prev.logradouro,
          bairro: result.bairro || prev.bairro,
          municipio: result.localidade || prev.municipio,
          uf: result.uf || prev.uf,
        } : prev)
      } else {
        toast.error('CEP não encontrado')
      }
    } finally {
      setBuscandoCEP(false)
    }
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editando) return
    setSubmitting(true)
    try {
      const method = editando.id ? 'PUT' : 'POST'
      const res = await fetch('/api/financeiro/clientes', {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(editando),
      })
      if (!res.ok) {
        const text = await res.text()
        throw new Error(text || 'Erro ao salvar cliente')
      }
      toast.success(editando.id ? 'Cliente atualizado' : 'Cliente cadastrado')
      setEditando(null)
      const data = await fetchClientes(busca, statusFiltro)
      setClientes(data)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Erro desconhecido'
      toast.error(msg)
    } finally {
      setSubmitting(false)
    }
  }

  const upd = (field: keyof Cliente) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setEditando(prev => prev ? { ...prev, [field]: e.target.value } : prev)

  if (loading) {
    return <div className="min-h-screen p-8 flex items-center justify-center">Carregando...</div>
  }

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-5xl mx-auto space-y-6">
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>Clientes</CardTitle>
            <Button onClick={() => setEditando(emptyCliente())}>Novo Cliente</Button>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex gap-3">
              <Input
                className="flex-1"
                placeholder="Buscar por nome ou CNPJ..."
                value={busca}
                onChange={e => setBusca(e.target.value)}
              />
              <select
                className="border border-input bg-background rounded-md px-3 py-2 text-sm"
                value={statusFiltro}
                onChange={e => setStatusFiltro(e.target.value as 'todos' | 'ativo' | 'inativo')}
              >
                <option value="todos">Todos</option>
                <option value="ativo">Ativos</option>
                <option value="inativo">Inativos</option>
              </select>
            </div>

            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Razão Social</TableHead>
                  <TableHead>CNPJ</TableHead>
                  <TableHead>Município / UF</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="w-24">Ações</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {clientes.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                      Nenhum cliente encontrado.
                    </TableCell>
                  </TableRow>
                ) : (
                  clientes.map(c => (
                    <TableRow key={c.id}>
                      <TableCell className="font-medium">{c.razao_social}</TableCell>
                      <TableCell>{formatCNPJ(c.cnpj)}</TableCell>
                      <TableCell className="text-muted-foreground text-sm">
                        {c.municipio ? `${c.municipio}${c.uf ? ' / ' + c.uf : ''}` : '—'}
                      </TableCell>
                      <TableCell>
                        <Badge variant={c.ativo ? 'default' : 'secondary'}>
                          {c.ativo ? 'Ativo' : 'Inativo'}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Button variant="outline" size="sm" onClick={() => setEditando({ ...c })}>
                          Editar
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Dialog open={editando !== null} onOpenChange={open => !open && setEditando(null)}>
          <DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>{editando?.id ? 'Editar Cliente' : 'Novo Cliente'}</DialogTitle>
            </DialogHeader>
            {editando && (
              <form onSubmit={handleSave} className="space-y-5">

                {/* ── Identificação ── */}
                <div className="space-y-3">
                  <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Identificação</p>
                  <div>
                    <Label htmlFor="razao_social">Razão Social *</Label>
                    <Input id="razao_social" value={editando.razao_social} onChange={upd('razao_social')} required />
                  </div>
                  <div>
                    <Label htmlFor="cnpj">CNPJ *</Label>
                    <Input
                      id="cnpj"
                      value={editando.cnpj}
                      onChange={e => setEditando(prev => prev ? { ...prev, cnpj: e.target.value.replace(/\D/g, '') } : prev)}
                      maxLength={14}
                      placeholder="Somente números"
                      required
                    />
                  </div>
                </div>

                {/* ── Endereço ── */}
                <div className="space-y-3">
                  <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Endereço</p>
                  <div>
                    <Label htmlFor="cep">CEP</Label>
                    <Input
                      id="cep"
                      value={editando.cep || ''}
                      onChange={e => setEditando(prev => prev ? { ...prev, cep: e.target.value.replace(/\D/g, '') } : prev)}
                      onBlur={handleCEPBlur}
                      maxLength={8}
                      placeholder={buscandoCEP ? 'Buscando...' : '00000000'}
                      disabled={buscandoCEP}
                    />
                  </div>
                  <div>
                    <Label htmlFor="logradouro">Logradouro</Label>
                    <Input id="logradouro" value={editando.logradouro || ''} onChange={upd('logradouro')} />
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <Label htmlFor="numero">Número</Label>
                      <Input id="numero" value={editando.numero || ''} onChange={upd('numero')} />
                    </div>
                    <div>
                      <Label htmlFor="complemento">Complemento</Label>
                      <Input id="complemento" value={editando.complemento || ''} onChange={upd('complemento')} />
                    </div>
                  </div>
                  <div>
                    <Label htmlFor="bairro">Bairro</Label>
                    <Input id="bairro" value={editando.bairro || ''} onChange={upd('bairro')} />
                  </div>
                  <div className="grid grid-cols-4 gap-3">
                    <div className="col-span-3">
                      <Label htmlFor="municipio">Município</Label>
                      <Input id="municipio" value={editando.municipio || ''} onChange={upd('municipio')} />
                    </div>
                    <div>
                      <Label htmlFor="uf">UF</Label>
                      <Input
                        id="uf"
                        maxLength={2}
                        value={editando.uf || ''}
                        onChange={e => setEditando(prev => prev ? { ...prev, uf: e.target.value.toUpperCase() } : prev)}
                      />
                    </div>
                  </div>
                </div>

                {/* ── Contato ── */}
                <div className="space-y-3">
                  <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Contato</p>
                  <div>
                    <Label htmlFor="responsavel">Responsável / Contato</Label>
                    <Input id="responsavel" value={editando.responsavel || ''} onChange={upd('responsavel')} />
                  </div>
                  <div>
                    <Label htmlFor="email">E-mail do Contato</Label>
                    <Input id="email" type="email" value={editando.email || ''} onChange={upd('email')} />
                  </div>
                  <div>
                    <Label htmlFor="telefone">Fone (com DDD)</Label>
                    <Input id="telefone" value={editando.telefone || ''} onChange={upd('telefone')} placeholder="(00) 00000-0000" />
                  </div>
                </div>

                {/* ── Status (só na edição) ── */}
                {editando.id && (
                  <div className="flex items-center gap-2">
                    <input
                      id="ativo"
                      type="checkbox"
                      checked={editando.ativo}
                      onChange={e => setEditando(prev => prev ? { ...prev, ativo: e.target.checked } : prev)}
                      className="h-4 w-4"
                    />
                    <Label htmlFor="ativo">Cliente ativo</Label>
                  </div>
                )}

                <div className="flex gap-2 justify-end pt-2">
                  <Button type="button" variant="outline" onClick={() => setEditando(null)}>Cancelar</Button>
                  <Button type="submit" disabled={submitting}>
                    {submitting ? 'Salvando...' : 'Salvar'}
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
