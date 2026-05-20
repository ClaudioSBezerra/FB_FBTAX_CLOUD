import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { toast } from 'sonner'

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

export default function ProdutosPage() {
  const [produtos, setProdutos] = useState<Produto[]>([])
  const [loading, setLoading] = useState(true)
  const [salvando, setSalvando] = useState<Record<string, boolean>>({})
  const [precoEditado, setPrecoEditado] = useState<Record<string, string>>({})
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetch('/api/financeiro/produtos')
      .then(r => r.json())
      .then((data: Produto[]) => {
        setProdutos(data)
        const precos: Record<string, string> = {}
        data.forEach(p => p.planos.forEach(pl => {
          precos[pl.id] = pl.preco != null ? String(pl.preco) : ''
        }))
        setPrecoEditado(precos)
      })
      .catch(err => setError(err instanceof Error ? err.message : 'Erro ao carregar produtos'))
      .finally(() => setLoading(false))
  }, [])

  const handleSalvarPreco = async (plano: Plano) => {
    const valor = precoEditado[plano.id]
    let preco: number | null = null
    if (valor !== '') {
      preco = parseFloat(valor)
      if (isNaN(preco)) {
        toast.error('Preço inválido')
        return
      }
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
        <h1 className="text-2xl font-bold">Produtos e Planos</h1>

        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {produtos.map(produto => (
          <Card key={produto.id}>
            <CardHeader className="flex flex-row items-center gap-3 pb-3">
              <CardTitle className="flex-1">{produto.nome}</CardTitle>
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
                            onChange={e => setPrecoEditado(prev => ({
                              ...prev,
                              [plano.id]: e.target.value,
                            }))}
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
  )
}
