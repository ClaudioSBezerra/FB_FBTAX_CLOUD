import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { toast } from 'sonner'

interface PortalCliente {
  id: string
  cliente_id: string
  razao_social: string
  email: string
  ativo: boolean
  created_at: string
}

interface Cliente {
  id: string
  razao_social: string
}

export default function PortalClientesAdminPage() {
  const [lista, setLista] = useState<PortalCliente[]>([])
  const [clientes, setClientes] = useState<Cliente[]>([])
  const [dialogNovo, setDialogNovo] = useState(false)
  const [dialogEditar, setDialogEditar] = useState<PortalCliente | null>(null)
  const [form, setForm] = useState({ cliente_id: '', email: '', password: '' })
  const [editForm, setEditForm] = useState({ ativo: true, password: '' })
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchLista = () =>
    fetch('/api/financeiro/portal-clientes')
      .then(r => r.json())
      .then(setLista)
      .catch(() => setError('Erro ao carregar usuários'))

  useEffect(() => {
    fetchLista()
    fetch('/api/financeiro/clientes')
      .then(r => r.json())
      .then(setClientes)
  }, [])

  const handleCriar = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    try {
      const res = await fetch('/api/financeiro/portal-clientes', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      })
      if (!res.ok) throw new Error('Erro ao criar acesso')
      toast.success('Acesso ao portal criado')
      setDialogNovo(false)
      setForm({ cliente_id: '', email: '', password: '' })
      fetchLista()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro')
    } finally {
      setSubmitting(false)
    }
  }

  const handleAtualizar = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!dialogEditar) return
    setSubmitting(true)
    try {
      const body: Record<string, unknown> = { id: dialogEditar.id, ativo: editForm.ativo }
      if (editForm.password) body.password = editForm.password
      const res = await fetch('/api/financeiro/portal-clientes', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (!res.ok) throw new Error('Erro ao atualizar')
      toast.success('Acesso atualizado')
      setDialogEditar(null)
      fetchLista()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-5xl mx-auto space-y-6">
        <h1 className="text-2xl font-bold">Acesso ao Portal do Cliente</h1>
        <p className="text-sm text-muted-foreground">
          Gerencie as credenciais de login dos clientes no portal externo.
        </p>

        {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}

        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>Usuários do Portal</CardTitle>
            <Button onClick={() => setDialogNovo(true)}>Novo Acesso</Button>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Cliente</TableHead>
                  <TableHead>E-mail de acesso</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Criado em</TableHead>
                  <TableHead className="w-24">Ações</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {lista.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                      Nenhum acesso ao portal cadastrado.
                    </TableCell>
                  </TableRow>
                ) : (
                  lista.map(p => (
                    <TableRow key={p.id}>
                      <TableCell className="font-medium">{p.razao_social}</TableCell>
                      <TableCell>{p.email}</TableCell>
                      <TableCell>
                        <Badge variant={p.ativo ? 'default' : 'secondary'}>
                          {p.ativo ? 'Ativo' : 'Inativo'}
                        </Badge>
                      </TableCell>
                      <TableCell>{p.created_at.slice(0, 10)}</TableCell>
                      <TableCell>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => {
                            setDialogEditar(p)
                            setEditForm({ ativo: p.ativo, password: '' })
                          }}
                        >
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

        {/* Dialog novo acesso */}
        <Dialog open={dialogNovo} onOpenChange={setDialogNovo}>
          <DialogContent>
            <DialogHeader><DialogTitle>Novo Acesso ao Portal</DialogTitle></DialogHeader>
            <form onSubmit={handleCriar} className="space-y-4">
              <div>
                <Label>Cliente *</Label>
                <select
                  className="mt-1 w-full border border-input bg-background rounded-md px-3 py-2 text-sm"
                  value={form.cliente_id}
                  onChange={e => setForm(f => ({ ...f, cliente_id: e.target.value }))}
                  required
                >
                  <option value="">Selecione...</option>
                  {clientes.map(c => <option key={c.id} value={c.id}>{c.razao_social}</option>)}
                </select>
              </div>
              <div>
                <Label htmlFor="email_novo">E-mail de acesso *</Label>
                <Input
                  id="email_novo"
                  type="email"
                  value={form.email}
                  onChange={e => setForm(f => ({ ...f, email: e.target.value }))}
                  required
                />
              </div>
              <div>
                <Label htmlFor="senha_novo">Senha *</Label>
                <Input
                  id="senha_novo"
                  type="password"
                  value={form.password}
                  onChange={e => setForm(f => ({ ...f, password: e.target.value }))}
                  required
                />
              </div>
              <div className="flex gap-2 justify-end">
                <Button type="button" variant="outline" onClick={() => setDialogNovo(false)}>Cancelar</Button>
                <Button type="submit" disabled={submitting}>{submitting ? 'Criando...' : 'Criar Acesso'}</Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>

        {/* Dialog editar acesso */}
        <Dialog open={dialogEditar !== null} onOpenChange={open => !open && setDialogEditar(null)}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Editar Acesso — {dialogEditar?.razao_social}</DialogTitle>
            </DialogHeader>
            <form onSubmit={handleAtualizar} className="space-y-4">
              <div className="flex items-center gap-2">
                <input
                  id="ativo_edit"
                  type="checkbox"
                  checked={editForm.ativo}
                  onChange={e => setEditForm(f => ({ ...f, ativo: e.target.checked }))}
                  className="h-4 w-4"
                />
                <Label htmlFor="ativo_edit">Acesso ativo</Label>
              </div>
              <div>
                <Label htmlFor="senha_edit">Nova senha (deixe em branco para manter)</Label>
                <Input
                  id="senha_edit"
                  type="password"
                  value={editForm.password}
                  onChange={e => setEditForm(f => ({ ...f, password: e.target.value }))}
                  placeholder="Nova senha (opcional)"
                />
              </div>
              <div className="flex gap-2 justify-end">
                <Button type="button" variant="outline" onClick={() => setDialogEditar(null)}>Cancelar</Button>
                <Button type="submit" disabled={submitting}>{submitting ? 'Salvando...' : 'Salvar'}</Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  )
}
