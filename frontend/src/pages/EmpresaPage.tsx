import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { toast } from 'sonner'
import { buscarCEP } from '@/lib/viacep'

interface Banco {
  code: number
  name: string
}

interface Empresa {
  id?: string
  razao_social: string
  nome_fantasia?: string
  cnpj: string
  cep: string
  logradouro: string
  numero: string
  complemento?: string
  bairro: string
  municipio: string
  uf: string
}

interface DadosBancarios {
  id?: string
  empresa_id: string
  banco: string
  agencia: string
  conta: string
  tipo_conta: string
  titular?: string
}

export default function EmpresaPage() {
  const [empresa, setEmpresa] = useState<Empresa>({
    razao_social: '',
    cnpj: '',
    cep: '',
    logradouro: '',
    numero: '',
    bairro: '',
    municipio: '',
    uf: '',
  })
  const [dadosBancarios, setDadosBancarios] = useState<DadosBancarios>({
    empresa_id: '',
    banco: '',
    agencia: '',
    conta: '',
    tipo_conta: '',
  })
  const [bancos, setBancos] = useState<Banco[]>([])
  const [loading, setLoading] = useState(true)
  const [buscandoCEP, setBuscandoCEP] = useState(false)
  const [submittingEmpresa, setSubmittingEmpresa] = useState(false)
  const [submittingDados, setSubmittingDados] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetch('/api/bancos').then(r => r.ok ? r.json() : []).then(setBancos).catch(() => {})
  }, [])

  useEffect(() => {
    const fetchData = async () => {
      try {
        const resEmp = await fetch('/api/financeiro/empresa')
        if (resEmp.ok) {
          const data = await resEmp.json()
          setEmpresa(data)
        }
        const resDad = await fetch('/api/financeiro/dados-bancarios')
        if (resDad.ok) {
          const data = await resDad.json()
          setDadosBancarios(data)
        }
      } catch (err) {
        const msg = err instanceof Error ? err.message : 'Erro ao carregar dados'
        setError(msg)
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [])

  const handleCEPBlur = async () => {
    const digits = empresa.cep.replace(/\D/g, '')
    if (digits.length !== 8) return
    setBuscandoCEP(true)
    try {
      const result = await buscarCEP(digits)
      if (result) {
        setEmpresa(prev => ({
          ...prev,
          logradouro: result.logradouro || prev.logradouro,
          bairro: result.bairro || prev.bairro,
          municipio: result.localidade || prev.municipio,
          uf: result.uf || prev.uf,
        }))
      } else {
        toast.error('CEP não encontrado')
      }
    } finally {
      setBuscandoCEP(false)
    }
  }

  const handleSaveEmpresa = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmittingEmpresa(true)
    setError(null)
    try {
      const method = empresa.id ? 'PUT' : 'POST'
      const res = await fetch('/api/financeiro/empresa', {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(empresa),
      })
      const data = await res.json()
      if (!res.ok) {
        throw new Error(typeof data === 'string' ? data : 'Erro ao salvar empresa')
      }
      if (!empresa.id) {
        setEmpresa(prev => ({ ...prev, id: data.id }))
      }
      toast.success('Empresa salva com sucesso')
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Erro desconhecido'
      setError(msg)
      toast.error(msg)
    } finally {
      setSubmittingEmpresa(false)
    }
  }

  const handleSaveDadosBancarios = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!empresa.id) {
      toast.error('Salve os dados da empresa primeiro')
      return
    }
    setSubmittingDados(true)
    setError(null)
    try {
      const method = dadosBancarios.id ? 'PUT' : 'POST'
      const res = await fetch('/api/financeiro/dados-bancarios', {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...dadosBancarios, empresa_id: empresa.id }),
      })
      const data = await res.json()
      if (!res.ok) {
        throw new Error(typeof data === 'string' ? data : 'Erro ao salvar dados bancários')
      }
      if (!dadosBancarios.id) {
        setDadosBancarios(prev => ({ ...prev, id: data.id }))
      }
      toast.success('Dados bancários salvos com sucesso')
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Erro desconhecido'
      setError(msg)
      toast.error(msg)
    } finally {
      setSubmittingDados(false)
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen p-8 flex items-center justify-center">
        Carregando...
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-2xl mx-auto space-y-8">
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <Card>
          <CardHeader>
            <CardTitle>Dados da Empresa</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSaveEmpresa} className="space-y-4">
              <div>
                <Label htmlFor="razao_social">Razão Social *</Label>
                <Input
                  id="razao_social"
                  value={empresa.razao_social}
                  onChange={e => setEmpresa(prev => ({ ...prev, razao_social: e.target.value }))}
                  required
                />
              </div>
              <div>
                <Label htmlFor="nome_fantasia">Nome Fantasia</Label>
                <Input
                  id="nome_fantasia"
                  value={empresa.nome_fantasia || ''}
                  onChange={e => setEmpresa(prev => ({ ...prev, nome_fantasia: e.target.value }))}
                />
              </div>
              <div>
                <Label htmlFor="cnpj">CNPJ *</Label>
                <Input
                  id="cnpj"
                  value={empresa.cnpj}
                  onChange={e => setEmpresa(prev => ({ ...prev, cnpj: e.target.value }))}
                  required
                />
              </div>

              {/* CEP logo após CNPJ — busca automática ao sair do campo */}
              <div>
                <Label htmlFor="cep">CEP *</Label>
                <Input
                  id="cep"
                  value={empresa.cep}
                  onChange={e => setEmpresa(prev => ({ ...prev, cep: e.target.value.replace(/\D/g, '') }))}
                  onBlur={handleCEPBlur}
                  maxLength={8}
                  placeholder={buscandoCEP ? 'Buscando...' : '00000000'}
                  disabled={buscandoCEP}
                  required
                />
              </div>

              <div>
                <Label htmlFor="logradouro">Logradouro *</Label>
                <Input
                  id="logradouro"
                  value={empresa.logradouro}
                  onChange={e => setEmpresa(prev => ({ ...prev, logradouro: e.target.value }))}
                  required
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <Label htmlFor="numero">Número *</Label>
                  <Input
                    id="numero"
                    value={empresa.numero}
                    onChange={e => setEmpresa(prev => ({ ...prev, numero: e.target.value }))}
                    required
                  />
                </div>
                <div>
                  <Label htmlFor="complemento">Complemento</Label>
                  <Input
                    id="complemento"
                    value={empresa.complemento || ''}
                    onChange={e => setEmpresa(prev => ({ ...prev, complemento: e.target.value }))}
                  />
                </div>
              </div>
              <div>
                <Label htmlFor="bairro">Bairro *</Label>
                <Input
                  id="bairro"
                  value={empresa.bairro}
                  onChange={e => setEmpresa(prev => ({ ...prev, bairro: e.target.value }))}
                  required
                />
              </div>
              <div className="grid grid-cols-4 gap-3">
                <div className="col-span-3">
                  <Label htmlFor="municipio">Município *</Label>
                  <Input
                    id="municipio"
                    value={empresa.municipio}
                    onChange={e => setEmpresa(prev => ({ ...prev, municipio: e.target.value }))}
                    required
                  />
                </div>
                <div>
                  <Label htmlFor="uf">UF *</Label>
                  <Input
                    id="uf"
                    maxLength={2}
                    value={empresa.uf}
                    onChange={e => setEmpresa(prev => ({ ...prev, uf: e.target.value.toUpperCase() }))}
                    required
                  />
                </div>
              </div>

              <Button type="submit" disabled={submittingEmpresa} className="w-full">
                {submittingEmpresa ? 'Salvando...' : 'Salvar Empresa'}
              </Button>
            </form>
          </CardContent>
        </Card>

        <Separator />

        <Card>
          <CardHeader>
            <CardTitle>Dados Bancários</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSaveDadosBancarios} className="space-y-4">
              <div>
                <Label htmlFor="banco">Banco *</Label>
                <Input
                  id="banco"
                  list="bancos-list"
                  value={dadosBancarios.banco}
                  onChange={e => setDadosBancarios(prev => ({ ...prev, banco: e.target.value }))}
                  disabled={!empresa.id}
                  placeholder="Digite o código ou nome do banco..."
                  required
                />
                <datalist id="bancos-list">
                  {bancos.map(b => (
                    <option
                      key={b.code}
                      value={`${String(b.code).padStart(3, '0')} - ${b.name}`}
                    />
                  ))}
                </datalist>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <Label htmlFor="agencia">Agência *</Label>
                  <Input
                    id="agencia"
                    value={dadosBancarios.agencia}
                    onChange={e => setDadosBancarios(prev => ({ ...prev, agencia: e.target.value }))}
                    disabled={!empresa.id}
                    required
                  />
                </div>
                <div>
                  <Label htmlFor="conta">Conta *</Label>
                  <Input
                    id="conta"
                    value={dadosBancarios.conta}
                    onChange={e => setDadosBancarios(prev => ({ ...prev, conta: e.target.value }))}
                    disabled={!empresa.id}
                    required
                  />
                </div>
              </div>
              <div>
                <Label htmlFor="tipo_conta">Tipo de Conta *</Label>
                <Input
                  id="tipo_conta"
                  value={dadosBancarios.tipo_conta}
                  onChange={e => setDadosBancarios(prev => ({ ...prev, tipo_conta: e.target.value }))}
                  disabled={!empresa.id}
                  required
                  placeholder="Ex: corrente, poupança"
                />
              </div>
              <div>
                <Label htmlFor="titular">Titular</Label>
                <Input
                  id="titular"
                  value={dadosBancarios.titular || ''}
                  onChange={e => setDadosBancarios(prev => ({ ...prev, titular: e.target.value }))}
                  disabled={!empresa.id}
                />
              </div>
              <Button
                type="submit"
                disabled={submittingDados || !empresa.id}
                className="w-full"
              >
                {submittingDados ? 'Salvando...' : 'Salvar Dados Bancários'}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
