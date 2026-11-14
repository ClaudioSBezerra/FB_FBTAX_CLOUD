import { useCallback, useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { toast } from 'sonner'

interface Conta {
  id: string
  apelido: string
  banco: string
  agencia?: string
  conta?: string
  tipo?: string
  provedor: string
  saldo: number
  ultima_sync?: string
  configurado: boolean
}

const contaVazia = { apelido: '', banco: '077 - BANCO INTER S.A.', agencia: '0001', conta: '', tipo: 'corrente', provedor: 'inter' }

interface Meta {
  configurado: boolean
  client_id_final?: string
  tem_secret: boolean
  tem_certificado: boolean
  tem_chave: boolean
  conta_corrente?: string
  cert_validade?: string
  cert_dias_restantes?: number
}

interface Escopo {
  escopo: string
  concedido: boolean
  perigoso?: boolean
  atencao?: string
  indeterminado?: boolean
  detalhe?: string
}

interface Diag {
  cert_validade?: string
  cert_dias_restantes: number
  token_ok: boolean
  escopos: Escopo[]
  saldo_ok: boolean
  saldo_disponivel?: number
  banking_versao?: string
  avisos?: string[]
  atencoes?: string[]
}

async function erroDe(res: Response): Promise<string> {
  const t = await res.text().catch(() => '')
  try {
    const j = JSON.parse(t)
    return j.erro || j.error || `HTTP ${res.status}`
  } catch {
    return t.trim().slice(0, 200) || `HTTP ${res.status}`
  }
}

const lerArquivo = (f: File) => new Promise<string>((ok, falha) => {
  const r = new FileReader()
  r.onload = () => ok(String(r.result))
  r.onerror = () => falha(r.error)
  r.readAsText(f)
})

const brl = (v: number) => v.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })

export default function BancosPage() {
  const [contas, setContas] = useState<Conta[]>([])
  const [contaId, setContaId] = useState('')
  const [meta, setMeta] = useState<Meta | null>(null)
  const [clientId, setClientId] = useState('')
  const [secret, setSecret] = useState('')
  const [contaCorrente, setContaCorrente] = useState('')
  const [crt, setCrt] = useState('')
  const [key, setKey] = useState('')
  const [nomeCrt, setNomeCrt] = useState('')
  const [nomeKey, setNomeKey] = useState('')
  const [diag, setDiag] = useState<Diag | null>(null)
  const [ocupado, setOcupado] = useState(false)

  const [form, setForm] = useState<typeof contaVazia & { id?: string }>(contaVazia)
  const [editando, setEditando] = useState(false)

  // Contas (com agência/conta/tipo) + indicador de credenciais vindo do status dos bancos.
  const carregarContas = useCallback(async () => {
    const [a, b] = await Promise.all([fetch('/api/financeiro/contas-financeiras'), fetch('/api/financeiro/bancos/status')])
    if (!a.ok) return
    const contasBase: Conta[] = await a.json()
    const status: { id: string; configurado: boolean }[] = b.ok ? await b.json() : []
    const cfg = new Map(status.map(s => [s.id, s.configurado]))
    const lista = contasBase.map(c => ({ ...c, configurado: cfg.get(c.id) ?? false }))
    setContas(lista)
    setContaId(atual => atual || (lista.length === 1 ? lista[0].id : ''))
  }, [])

  const salvarConta = async () => {
    if (!form.apelido.trim() || !form.banco.trim()) return toast.error('Informe o apelido e o banco')
    const res = await fetch('/api/financeiro/contas-financeiras', {
      method: editando ? 'PUT' : 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...form, ativa: true }),
    })
    if (!res.ok) return toast.error(await erroDe(res))
    toast.success(editando ? 'Conta atualizada' : 'Conta criada')
    setForm(contaVazia); setEditando(false)
    carregarContas()
  }

  const desativarConta = async (c: Conta) => {
    if (!window.confirm(`Desativar a conta "${c.apelido}"? Os lançamentos já importados são mantidos.`)) return
    const res = await fetch(`/api/financeiro/contas-financeiras?id=${c.id}`, { method: 'DELETE' })
    if (!res.ok) return toast.error(await erroDe(res))
    if (contaId === c.id) setContaId('')
    toast.success('Conta desativada')
    carregarContas()
  }

  const carregarMeta = useCallback(async (id: string) => {
    setDiag(null)
    if (!id) return setMeta(null)
    const res = await fetch(`/api/financeiro/bancos/config?conta_id=${id}`)
    if (res.ok) {
      const m: Meta = await res.json()
      setMeta(m)
      setContaCorrente(m.conta_corrente || '')
    }
  }, [])

  useEffect(() => { carregarContas() }, [carregarContas])
  useEffect(() => { carregarMeta(contaId) }, [contaId, carregarMeta])

  const salvar = async () => {
    if (!contaId) return toast.error('Escolha a conta')
    setOcupado(true)
    try {
      const res = await fetch('/api/financeiro/bancos/config', {
        method: 'PUT', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ conta_id: contaId, provedor: 'inter', config: { client_id: clientId.trim(), client_secret: secret.trim(), conta_corrente: contaCorrente.trim(), cert_pem: crt, key_pem: key } }),
      })
      if (!res.ok) return toast.error(await erroDe(res))
      toast.success('Credenciais salvas (criptografadas)')
      setSecret(''); setClientId(''); setCrt(''); setKey(''); setNomeCrt(''); setNomeKey('')
      carregarMeta(contaId); carregarContas()
    } finally { setOcupado(false) }
  }

  const testar = async () => {
    setOcupado(true); setDiag(null)
    try {
      const res = await fetch('/api/financeiro/bancos/testar', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ conta_id: contaId }) })
      if (!res.ok) return toast.error(await erroDe(res))
      setDiag(await res.json())
    } finally { setOcupado(false) }
  }

  const sincronizar = async () => {
    const res = await fetch('/api/financeiro/bancos/sync', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ conta_id: contaId }) })
    toast[res.ok ? 'success' : 'error'](res.ok ? 'Sincronização iniciada' : await erroDe(res))
  }

  const concedidos = diag?.escopos.filter(e => e.concedido) ?? []
  const naoConcedidos = diag?.escopos.filter(e => !e.concedido && !e.perigoso && !e.indeterminado) ?? []
  const indeterminados = diag?.escopos.filter(e => e.indeterminado) ?? []

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-3xl mx-auto space-y-6">
        <h1 className="text-2xl font-semibold">Manutenção do banco</h1>

        <Card>
          <CardHeader><CardTitle>1. Contas bancárias</CardTitle></CardHeader>
          <CardContent className="space-y-4">
            <table className="w-full text-sm">
              <thead><tr className="text-left border-b"><th>Conta</th><th>Banco</th><th>Agência / Conta</th><th>API</th><th>Última sync</th><th className="text-right">Saldo</th><th></th></tr></thead>
              <tbody>
                {contas.map(c => (
                  <tr key={c.id} className={`border-b cursor-pointer ${contaId === c.id ? 'bg-teal-50' : ''}`} onClick={() => setContaId(c.id)}>
                    <td className="font-medium">{c.apelido}</td>
                    <td>{c.banco}</td>
                    <td>{c.agencia || '-'} / {c.conta || '-'}</td>
                    <td>{c.provedor ? <Badge variant={c.configurado ? 'default' : 'secondary'}>{c.provedor}{c.configurado ? ' ✓' : ' sem credenciais'}</Badge> : <span className="text-gray-400">manual</span>}</td>
                    <td>{c.ultima_sync ? new Date(c.ultima_sync).toLocaleString('pt-BR') : '—'}</td>
                    <td className="text-right">{brl(c.saldo)}</td>
                    <td className="whitespace-nowrap space-x-1" onClick={e => e.stopPropagation()}>
                      <Button size="sm" variant="outline" onClick={() => { setContaId(c.id); setForm({ id: c.id, apelido: c.apelido, banco: c.banco, agencia: c.agencia || '', conta: c.conta || '', tipo: c.tipo || 'corrente', provedor: c.provedor || '' }); setEditando(true) }}>Editar</Button>
                      <Button size="sm" variant="outline" onClick={() => desativarConta(c)}>Desativar</Button>
                    </td>
                  </tr>
                ))}
                {contas.length === 0 && <tr><td colSpan={7} className="text-gray-500 py-3">Nenhuma conta cadastrada.</td></tr>}
              </tbody>
            </table>
            <div className="border rounded p-3 space-y-3">
              <p className="text-sm font-medium">{editando ? 'Editar conta' : 'Nova conta'}</p>
              <div className="grid grid-cols-4 gap-3">
                <div className="col-span-2"><Label>Apelido</Label><Input value={form.apelido} onChange={e => setForm({ ...form, apelido: e.target.value })} placeholder="Inter PJ — principal" /></div>
                <div className="col-span-2"><Label>Banco</Label><Input value={form.banco} onChange={e => setForm({ ...form, banco: e.target.value })} /></div>
                <div><Label>Agência</Label><Input value={form.agencia} onChange={e => setForm({ ...form, agencia: e.target.value })} /></div>
                <div><Label>Conta</Label><Input value={form.conta} onChange={e => setForm({ ...form, conta: e.target.value })} /></div>
                <div>
                  <Label>Tipo</Label>
                  <select className="w-full border rounded h-10 px-2 bg-white" value={form.tipo} onChange={e => setForm({ ...form, tipo: e.target.value })}>
                    <option value="corrente">Corrente</option><option value="poupanca">Poupança</option><option value="investimento">Investimento</option>
                  </select>
                </div>
                <div>
                  <Label>Integração (API)</Label>
                  <select className="w-full border rounded h-10 px-2 bg-white" value={form.provedor} onChange={e => setForm({ ...form, provedor: e.target.value })}>
                    <option value="">Manual (sem API)</option><option value="inter">Banco Inter</option>
                  </select>
                </div>
              </div>
              <div className="flex gap-2">
                <Button onClick={salvarConta}>{editando ? 'Salvar alterações' : 'Criar conta'}</Button>
                {editando && <Button variant="outline" onClick={() => { setForm(contaVazia); setEditando(false) }}>Cancelar</Button>}
              </div>
            </div>
            {!contaId && contas.length > 0 && <Alert><AlertDescription>Clique numa conta da lista para abrir <b>Credenciais</b> e <b>Testar conexão</b>.</AlertDescription></Alert>}
            <p className="text-xs text-gray-500">Clique numa conta para configurar as credenciais da API e testar a integração. Para ver o extrato e sincronizar, use <b>Bancos › Sincronização</b>.</p>
          </CardContent>
        </Card>

        {contaId && (
          <Card>
            <CardHeader><CardTitle>2. Credenciais da integração — {contas.find(c => c.id === contaId)?.apelido}</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              {meta?.configurado && (
                <p className="text-sm">
                  Salvo: Client ID …{meta.client_id_final} · {meta.tem_secret ? 'secret ✓' : 'sem secret'} · {meta.tem_certificado && meta.tem_chave ? 'certificado e chave ✓' : 'certificado/chave incompletos'}
                  {meta.cert_validade && <> · certificado até {meta.cert_validade} <Badge variant={(meta.cert_dias_restantes ?? 0) <= 90 ? 'destructive' : 'secondary'}>{meta.cert_dias_restantes} dias</Badge></>}
                </p>
              )}
              <Alert><AlertDescription>
                Cole só o que mudou: campos em branco <b>mantêm</b> o valor já salvo. Client ID, Secret, certificado e chave <b>nunca são exibidos de volta</b> e ficam criptografados.
              </AlertDescription></Alert>
              <div className="grid grid-cols-2 gap-3">
                <div><Label>Client ID</Label><Input autoComplete="off" value={clientId} onChange={e => setClientId(e.target.value)} placeholder={meta?.configurado ? 'em branco = manter' : ''} /></div>
                <div><Label>Client Secret</Label><Input type="password" autoComplete="new-password" value={secret} onChange={e => setSecret(e.target.value)} placeholder={meta?.tem_secret ? 'em branco = manter' : ''} /></div>
                <div><Label>Conta corrente (número da conta no Inter)</Label><Input value={contaCorrente} onChange={e => setContaCorrente(e.target.value)} /></div>
                <div />
                <div>
                  <Label>Certificado (.crt)</Label>
                  <Input type="file" accept=".crt,.pem,.cer" onChange={async e => { const f = e.target.files?.[0]; if (f) { setCrt(await lerArquivo(f)); setNomeCrt(f.name) } }} />
                  {nomeCrt && <p className="text-xs text-green-700">{nomeCrt} carregado</p>}
                </div>
                <div>
                  <Label>Chave privada (.key)</Label>
                  <Input type="file" accept=".key,.pem" onChange={async e => { const f = e.target.files?.[0]; if (f) { setKey(await lerArquivo(f)); setNomeKey(f.name) } }} />
                  {nomeKey && <p className="text-xs text-green-700">{nomeKey} carregado</p>}
                </div>
              </div>
              <Button onClick={salvar} disabled={ocupado}>Salvar credenciais</Button>
            </CardContent>
          </Card>
        )}

        {contaId && meta?.configurado && (
          <Card>
            <CardHeader><CardTitle>3. Testar a integração</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              <p className="text-sm text-gray-600">Somente leitura: confere o certificado, descobre quais permissões a integração realmente tem e consulta o saldo. Não movimenta nada.</p>
              <div className="flex gap-3">
                <Button onClick={testar} disabled={ocupado}>{ocupado ? 'Testando…' : 'Testar conexão'}</Button>
                <Button variant="outline" onClick={sincronizar}>Sincronizar extrato agora</Button>
              </div>
              {diag && (
                <div className="space-y-3 text-sm">
                  {(diag.avisos ?? []).map((a, i) => <Alert key={i} variant="destructive"><AlertDescription>{a}</AlertDescription></Alert>)}
                  {(diag.atencoes ?? []).map((a, i) => <Alert key={`at${i}`}><AlertDescription>{a}</AlertDescription></Alert>)}
                  <p>Token: {diag.token_ok ? '✓ emitido' : '✗ não emitido'} · Saldo: {diag.saldo_ok ? `✓ ${brl(diag.saldo_disponivel ?? 0)} (banking ${diag.banking_versao})` : '✗'} · Certificado: {diag.cert_validade ? `até ${diag.cert_validade} (${diag.cert_dias_restantes} dias)` : '—'}</p>
                  <div>
                    <p className="font-medium">Permissões concedidas ({concedidos.length})</p>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {concedidos.map(e => <Badge key={e.escopo} variant={e.perigoso ? 'destructive' : e.atencao ? 'secondary' : 'default'} title={e.atencao}>{e.escopo}{e.atencao ? ' ⚠' : ''}</Badge>)}
                      {concedidos.length === 0 && <span className="text-gray-500">nenhuma</span>}
                    </div>
                  </div>
                  <div>
                    <p className="font-medium">Sem permissão (ou nome desconhecido)</p>
                    <div className="flex flex-wrap gap-1 mt-1">{naoConcedidos.map(e => <Badge key={e.escopo} variant="secondary">{e.escopo}</Badge>)}</div>
                  </div>
                  {indeterminados.length > 0 && (
                    <div>
                      <p className="font-medium">Indeterminado (limite de requisições do Inter — teste de novo em alguns minutos)</p>
                      <div className="flex flex-wrap gap-1 mt-1">{indeterminados.map(e => <Badge key={e.escopo} variant="outline">{e.escopo}</Badge>)}</div>
                    </div>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
