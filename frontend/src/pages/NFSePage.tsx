import { useCallback, useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { toast } from 'sonner'
import { buscarCEP } from '@/lib/viacep'

const FRASE_PRODUCAO = 'EMITIR EM PRODUCAO'

interface CertStatus {
  cadastrado: boolean
  nome_arquivo?: string
  cnpj_titular?: string
  validade?: string
  dias_restantes?: number
  cnpj_confere?: boolean
  vencido?: boolean
}

interface Prestador {
  cnpj: string
  cod_ibge: string
  telefone: string
  email: string
  inscricao_municipal: string
  op_simp_nac: number
  reg_ap_trib_sn: number
  reg_esp_trib: number
  nfse_ambiente: number
}

interface ClienteNF {
  id: string
  razao_social: string
  cnpj: string
  cep: string
  municipio: string
  uf: string
  cod_ibge: string
}

interface Nota {
  id: string
  cliente: string
  ambiente: number
  n_dps: number
  competencia: string
  valor: number
  status: string
  chave_acesso: string
  n_nfse: string
  erro: string
}

async function lerErro(res: Response): Promise<string> {
  try {
    const j = await res.json()
    return j.erro || j.error || `HTTP ${res.status}`
  } catch {
    return `HTTP ${res.status}`
  }
}

const brl = (v: number) => v.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })

export default function NFSePage() {
  const [cert, setCert] = useState<CertStatus | null>(null)
  const [prest, setPrest] = useState<Prestador | null>(null)
  const [clientes, setClientes] = useState<ClienteNF[]>([])
  const [notas, setNotas] = useState<Nota[]>([])
  const [erro, setErro] = useState<string | null>(null)

  const [pfx, setPfx] = useState<File | null>(null)
  const [senha, setSenha] = useState('')

  const [clienteId, setClienteId] = useState('')
  const [valor, setValor] = useState('1,00')
  const [descricao, setDescricao] = useState('Teste de emissão — NFS-e de validação do sistema')
  const [competencia, setCompetencia] = useState('')
  const [previa, setPrevia] = useState<string | null>(null)
  const [confirmacao, setConfirmacao] = useState('')
  const [ocupado, setOcupado] = useState(false)
  const [convenio, setConvenio] = useState<string | null>(null)

  const carregar = useCallback(async () => {
    try {
      const [c, p, cl, n] = await Promise.all([
        fetch('/api/financeiro/nfse/certificado'),
        fetch('/api/financeiro/nfse/prestador'),
        fetch('/api/financeiro/nfse/clientes'),
        fetch('/api/financeiro/nfse'),
      ])
      if (c.ok) setCert(await c.json())
      if (p.ok) setPrest(await p.json())
      else setErro(await lerErro(p))
      if (cl.ok) setClientes(await cl.json())
      if (n.ok) setNotas(await n.json())
    } catch (e) {
      setErro(e instanceof Error ? e.message : 'Erro ao carregar')
    }
  }, [])

  useEffect(() => { carregar() }, [carregar])

  const producao = prest?.nfse_ambiente === 1
  const cliente = clientes.find(c => c.id === clienteId)
  const centavos = Math.round(parseFloat(valor.replace(/\./g, '').replace(',', '.')) * 100)

  const enviarCertificado = async () => {
    if (!pfx || !senha) return toast.error('Selecione o .pfx e informe a senha')
    const fd = new FormData()
    fd.append('pfx', pfx)
    fd.append('senha', senha)
    setOcupado(true)
    try {
      const res = await fetch('/api/financeiro/nfse/certificado', { method: 'POST', body: fd })
      if (!res.ok) return toast.error(await lerErro(res))
      toast.success('Certificado salvo')
      setPfx(null)
      setSenha('')
      carregar()
    } finally {
      setOcupado(false)
    }
  }

  const salvarPrestador = async (ambiente?: number) => {
    if (!prest) return
    const body = { ...prest, nfse_ambiente: ambiente ?? prest.nfse_ambiente, confirmar_producao: ambiente === 1 ? FRASE_PRODUCAO : undefined }
    const res = await fetch('/api/financeiro/nfse/prestador', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })
    if (!res.ok) return toast.error(await lerErro(res))
    toast.success('Parâmetros salvos')
    carregar()
  }

  const verificarConvenio = async () => {
    setConvenio('Consultando…')
    const res = await fetch('/api/financeiro/nfse/convenio')
    setConvenio(res.ok ? JSON.stringify(await res.json(), null, 2) : await lerErro(res))
  }

  const preencherIBGE = async () => {
    if (!cliente) return
    const r = await buscarCEP(cliente.cep)
    if (!r?.ibge) return toast.error('Não foi possível obter o IBGE pelo CEP do cliente')
    const res = await fetch('/api/financeiro/nfse/clientes', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ cliente_id: cliente.id, cod_ibge: r.ibge }) })
    if (!res.ok) return toast.error(await lerErro(res))
    toast.success(`IBGE ${r.ibge} salvo para ${cliente.razao_social}`)
    carregar()
  }

  const emitir = async (enviar: boolean) => {
    if (!clienteId || !(centavos > 0)) return toast.error('Escolha o cliente e informe um valor válido')
    setOcupado(true)
    try {
      const res = await fetch('/api/financeiro/nfse/emitir', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ cliente_id: clienteId, valor_centavos: centavos, descricao, competencia: competencia || undefined, enviar, confirmar_producao: enviar && producao ? FRASE_PRODUCAO : undefined }),
      })
      if (!res.ok) return toast.error(await lerErro(res))
      const j = await res.json()
      if (!enviar) {
        setPrevia(j.dps_xml)
        return toast.success('Prévia gerada — nada foi enviado')
      }
      toast.success(`NFS-e autorizada${j.n_nfse ? ` nº ${j.n_nfse}` : ''}`)
      setConfirmacao('')
      setPrevia(null)
      carregar()
    } finally {
      setOcupado(false)
    }
  }

  // Âncora comum não envia o token de login: baixa com fetch (que leva o Authorization) e salva o blob.
  const baixar = async (url: string, nome: string) => {
    const res = await fetch(url)
    if (!res.ok) return toast.error(await lerErro(res))
    const blob = await res.blob()
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = nome
    a.click()
    URL.revokeObjectURL(a.href)
  }

  const acao = async (n: Nota, tipo: 'consultar' | 'cancelar') => {
    let body: string | undefined
    if (tipo === 'cancelar') {
      const justificativa = window.prompt('Justificativa do cancelamento (15 a 255 caracteres):')
      if (!justificativa) return
      body = JSON.stringify({ id: n.id, motivo: 1, justificativa, confirmar_producao: n.ambiente === 1 ? FRASE_PRODUCAO : undefined })
    }
    const url = tipo === 'consultar' ? `/api/financeiro/nfse/consultar?id=${n.id}` : '/api/financeiro/nfse/cancelar'
    const res = await fetch(url, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body })
    if (!res.ok) return toast.error(await lerErro(res))
    toast.success(tipo === 'cancelar' ? 'Nota cancelada' : 'Consulta concluída')
    carregar()
  }

  const podeEnviar = !producao || confirmacao === FRASE_PRODUCAO

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-4xl mx-auto space-y-6">
        <h1 className="text-2xl font-semibold">NFS-e Nacional</h1>
        {erro && <Alert variant="destructive"><AlertDescription>{erro}</AlertDescription></Alert>}

        <Card>
          <CardHeader><CardTitle>1. Certificado digital A1</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {cert?.cadastrado ? (
              <p className="text-sm">
                {cert.nome_arquivo} — CNPJ {cert.cnpj_titular} — válido até {cert.validade}{' '}
                {cert.vencido ? <Badge variant="destructive">vencido</Badge> : <Badge>{cert.dias_restantes} dias</Badge>}{' '}
                {!cert.cnpj_confere && <Badge variant="destructive">CNPJ diferente do prestador</Badge>}
              </p>
            ) : <p className="text-sm text-gray-600">Nenhum certificado cadastrado.</p>}
            <div className="grid grid-cols-2 gap-3">
              <div><Label>Arquivo .pfx / .p12</Label><Input type="file" accept=".pfx,.p12" onChange={e => setPfx(e.target.files?.[0] ?? null)} /></div>
              <div><Label>Senha do certificado</Label><Input type="password" autoComplete="off" value={senha} onChange={e => setSenha(e.target.value)} /></div>
            </div>
            <Button onClick={enviarCertificado} disabled={ocupado}>Enviar certificado</Button>
            <p className="text-xs text-gray-500">O arquivo e a senha são guardados criptografados e nunca são exibidos de volta.</p>
          </CardContent>
        </Card>

        {prest && (
          <Card>
            <CardHeader><CardTitle>2. Parâmetros fiscais do prestador</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              <div className="grid grid-cols-3 gap-3">
                <div><Label>CNPJ</Label><Input value={prest.cnpj} disabled /></div>
                <div><Label>IBGE do município</Label><Input value={prest.cod_ibge} onChange={e => setPrest({ ...prest, cod_ibge: e.target.value })} placeholder="5201405" /></div>
                <div><Label>Inscrição municipal (opcional)</Label><Input value={prest.inscricao_municipal} onChange={e => setPrest({ ...prest, inscricao_municipal: e.target.value })} /></div>
                <div><Label>Telefone</Label><Input value={prest.telefone} onChange={e => setPrest({ ...prest, telefone: e.target.value })} /></div>
                <div className="col-span-2"><Label>E-mail</Label><Input value={prest.email} onChange={e => setPrest({ ...prest, email: e.target.value })} /></div>
              </div>
              <div className="flex items-center gap-3">
                <Button variant="outline" onClick={() => salvarPrestador()}>Salvar parâmetros</Button>
                <span className="text-sm">Ambiente: <Badge variant={producao ? 'destructive' : 'secondary'}>{producao ? 'PRODUÇÃO' : 'Produção restrita (teste)'}</Badge></span>
                {!producao && <Button variant="destructive" onClick={() => window.confirm('Ativar PRODUÇÃO? As notas passarão a valer fiscalmente.') && salvarPrestador(1)}>Ativar produção</Button>}
                {producao && <Button variant="outline" onClick={() => salvarPrestador(2)}>Voltar para teste</Button>}
                <Button variant="outline" onClick={verificarConvenio}>Verificar município nos ambientes</Button>
              </div>
              {convenio && <pre className="text-xs overflow-auto max-h-64 bg-white border p-2">{convenio}</pre>}
            </CardContent>
          </Card>
        )}

        <Card>
          <CardHeader><CardTitle>3. Emitir NFS-e</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            <div className="grid grid-cols-3 gap-3">
              <div className="col-span-2">
                <Label>Cliente (tomador)</Label>
                <select className="w-full border rounded h-10 px-2 bg-white" value={clienteId} onChange={e => { setClienteId(e.target.value); setPrevia(null) }}>
                  <option value="">Selecione…</option>
                  {clientes.map(c => <option key={c.id} value={c.id}>{c.razao_social} — {c.cnpj}</option>)}
                </select>
              </div>
              <div><Label>Valor (R$)</Label><Input value={valor} onChange={e => setValor(e.target.value)} /></div>
              <div className="col-span-3"><Label>Descrição do serviço</Label><Textarea value={descricao} onChange={e => setDescricao(e.target.value)} rows={3} /></div>
              <div><Label>Competência (opcional)</Label><Input type="date" value={competencia} onChange={e => setCompetencia(e.target.value)} /></div>
            </div>
            {cliente && !cliente.cod_ibge && (
              <Alert><AlertDescription>
                Este cliente não tem código IBGE (necessário no DPS). <Button size="sm" variant="outline" onClick={preencherIBGE}>Buscar pelo CEP {cliente.cep}</Button>
              </AlertDescription></Alert>
            )}
            <div className="flex gap-3">
              <Button variant="outline" disabled={ocupado} onClick={() => emitir(false)}>Gerar prévia (não envia)</Button>
            </div>
            {previa && (
              <div className="space-y-3">
                <p className="text-sm text-green-700">Prévia válida e assinada. Nenhum número de DPS foi gasto.</p>
                <details><summary className="text-sm cursor-pointer">Ver XML</summary><pre className="text-xs overflow-auto max-h-64 bg-white border p-2">{previa}</pre></details>
                {producao && (
                  <div>
                    <Label>Ambiente de PRODUÇÃO: digite <b>{FRASE_PRODUCAO}</b> para liberar o envio</Label>
                    <Input value={confirmacao} onChange={e => setConfirmacao(e.target.value)} />
                  </div>
                )}
                <Button variant={producao ? 'destructive' : 'default'} disabled={ocupado || !podeEnviar} onClick={() => emitir(true)}>
                  Emitir de verdade ({brl(centavos / 100)}) {producao ? '— PRODUÇÃO' : '— produção restrita'}
                </Button>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle>Notas emitidas</CardTitle></CardHeader>
          <CardContent>
            <table className="w-full text-sm">
              <thead><tr className="text-left border-b"><th>NFS-e</th><th>Cliente</th><th>Competência</th><th>Valor</th><th>Status</th><th></th></tr></thead>
              <tbody>
                {notas.map(n => (
                  <tr key={n.id} className="border-b align-top">
                    <td>{n.n_nfse || `DPS ${n.n_dps}`}{n.ambiente === 2 && <Badge variant="secondary" className="ml-1">teste</Badge>}</td>
                    <td>{n.cliente}</td><td>{n.competencia}</td><td>{brl(n.valor)}</td>
                    <td><Badge variant={n.status === 'autorizada' ? 'default' : n.status === 'rejeitada' ? 'destructive' : 'secondary'}>{n.status}</Badge>{n.erro && <div className="text-xs text-red-600 max-w-xs">{n.erro}</div>}</td>
                    <td className="space-x-1 whitespace-nowrap">
                      {n.status === 'enviada' && <Button size="sm" variant="outline" onClick={() => acao(n, 'consultar')}>Consultar</Button>}
                      {(n.status === 'autorizada' || n.status === 'cancelada') && <>
                        <Button size="sm" variant="outline" onClick={() => baixar(`/api/financeiro/nfse/xml?id=${n.id}&tipo=nfse`, `nfse-${n.chave_acesso}.xml`)}>XML</Button>
                        <Button size="sm" variant="outline" onClick={() => baixar(`/api/financeiro/nfse/danfse?id=${n.id}`, `danfse-${n.chave_acesso}.pdf`)}>DANFSe</Button>
                        {n.status === 'autorizada' && <Button size="sm" variant="outline" onClick={() => acao(n, 'cancelar')}>Cancelar</Button>}
                      </>}
                    </td>
                  </tr>
                ))}
                {notas.length === 0 && <tr><td colSpan={6} className="text-gray-500 py-3">Nenhuma nota ainda.</td></tr>}
              </tbody>
            </table>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
