import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { toast } from 'sonner'

interface ApiKey {
  id: string
  contrato_id: string
  api_key: string
  descricao: string
  ativo: boolean
  created_at: string
}

interface Cliente {
  id: string
  razao_social: string
}

interface Contrato {
  id: string
  data_inicio: string
  status: string
}

export default function ApiKeysPage() {
  const [clientes, setClientes] = useState<Cliente[]>([])
  const [contratos, setContratos] = useState<Contrato[]>([])
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([])
  const [clienteId, setClienteId] = useState('')
  const [contratoId, setContratoId] = useState('')
  const [novaDescricao, setNovaDescricao] = useState('')
  const [criando, setCriando] = useState(false)
  const [dialogAberto, setDialogAberto] = useState(false)
  const [novaKey, setNovaKey] = useState('')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetch('/api/financeiro/clientes')
      .then(r => r.json())
      .then(setClientes)
      .catch(() => setError('Erro ao carregar clientes'))
  }, [])

  const onClienteChange = async (id: string) => {
    setClienteId(id)
    setContratoId('')
    setApiKeys([])
    if (!id) { setContratos([]); return }
    const res = await fetch(`/api/financeiro/contratos?cliente_id=${id}`)
    setContratos(await res.json())
  }

  const onContratoChange = async (id: string) => {
    setContratoId(id)
    if (!id) { setApiKeys([]); return }
    const res = await fetch(`/api/financeiro/api-keys?contrato_id=${id}`)
    setApiKeys(await res.json())
  }

  const handleCriar = async () => {
    if (!contratoId) return
    setCriando(true)
    try {
      const res = await fetch('/api/financeiro/api-keys', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ contrato_id: contratoId, descricao: novaDescricao }),
      })
      if (!res.ok) throw new Error('Erro ao criar API Key')
      const data = await res.json()
      setNovaKey(data.api_key)
      setDialogAberto(true)
      setNovaDescricao('')
      onContratoChange(contratoId)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Erro')
    } finally {
      setCriando(false)
    }
  }

  const handleToggle = async (key: ApiKey) => {
    await fetch('/api/financeiro/api-keys', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: key.id, ativo: !key.ativo }),
    })
    onContratoChange(contratoId)
  }

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-5xl mx-auto space-y-6">
        <h1 className="text-2xl font-bold">API Keys de Validação</h1>

        {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}

        <Card>
          <CardHeader><CardTitle>Selecionar Contrato</CardTitle></CardHeader>
          <CardContent className="flex gap-4 flex-wrap">
            <div className="flex-1 min-w-48">
              <Label>Cliente</Label>
              <select
                className="mt-1 w-full border border-input bg-background rounded-md px-3 py-2 text-sm"
                value={clienteId}
                onChange={e => onClienteChange(e.target.value)}
              >
                <option value="">Selecione...</option>
                {clientes.map(c => <option key={c.id} value={c.id}>{c.razao_social}</option>)}
              </select>
            </div>
            <div className="flex-1 min-w-48">
              <Label>Contrato</Label>
              <select
                className="mt-1 w-full border border-input bg-background rounded-md px-3 py-2 text-sm"
                value={contratoId}
                onChange={e => onContratoChange(e.target.value)}
                disabled={!clienteId}
              >
                <option value="">Selecione...</option>
                {contratos.map(c => (
                  <option key={c.id} value={c.id}>{c.data_inicio} — {c.status}</option>
                ))}
              </select>
            </div>
          </CardContent>
        </Card>

        {contratoId && (
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle>API Keys</CardTitle>
              <div className="flex gap-2 items-center">
                <Input
                  className="w-48"
                  placeholder="Descrição (opcional)"
                  value={novaDescricao}
                  onChange={e => setNovaDescricao(e.target.value)}
                />
                <Button disabled={criando} onClick={handleCriar}>
                  {criando ? 'Gerando...' : 'Gerar Nova Key'}
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>API Key</TableHead>
                    <TableHead>Descrição</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Criada em</TableHead>
                    <TableHead className="w-24">Ações</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {apiKeys.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                        Nenhuma API Key gerada para este contrato.
                      </TableCell>
                    </TableRow>
                  ) : (
                    apiKeys.map(k => (
                      <TableRow key={k.id}>
                        <TableCell>
                          <code className="text-xs bg-gray-100 px-1 py-0.5 rounded">
                            {k.api_key.slice(0, 16)}…
                          </code>
                        </TableCell>
                        <TableCell>{k.descricao || '—'}</TableCell>
                        <TableCell>
                          <Badge variant={k.ativo ? 'default' : 'secondary'}>
                            {k.ativo ? 'Ativa' : 'Inativa'}
                          </Badge>
                        </TableCell>
                        <TableCell>{k.created_at.slice(0, 10)}</TableCell>
                        <TableCell>
                          <Button size="sm" variant="outline" onClick={() => handleToggle(k)}>
                            {k.ativo ? 'Desativar' : 'Ativar'}
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        )}

        <Dialog open={dialogAberto} onOpenChange={setDialogAberto}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>API Key gerada</DialogTitle>
            </DialogHeader>
            <p className="text-sm text-muted-foreground">
              Copie esta chave agora. Ela não será exibida novamente na íntegra.
            </p>
            <div className="bg-gray-100 rounded p-3 break-all font-mono text-sm select-all">
              {novaKey}
            </div>
            <Button onClick={() => {
              navigator.clipboard.writeText(novaKey)
              toast.success('Copiado!')
            }}>
              Copiar
            </Button>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  )
}
