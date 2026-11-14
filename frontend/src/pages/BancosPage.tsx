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
  provedor: string
  saldo: number
  configurado: boolean
}

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

  const carregarContas = useCallback(async () => {
    const res = await fetch('/api/financeiro/bancos/status')
    if (res.ok) setContas(await res.json())
  }, [])

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
        <h1 className="text-2xl font-semibold">Bancos — integração Inter</h1>

        <Card>
          <CardHeader><CardTitle>1. Conta</CardTitle></CardHeader>
          <CardContent className="space-y-2">
            <select className="w-full border rounded h-10 px-2 bg-white" value={contaId} onChange={e => setContaId(e.target.value)}>
              <option value="">Selecione a conta…</option>
              {contas.map(c => <option key={c.id} value={c.id}>{c.apelido} — {c.banco}{c.configurado ? ' (configurada)' : ''}</option>)}
            </select>
            <p className="text-xs text-gray-500">A conta é criada no Painel Financeiro. Aqui só se guardam as credenciais da API.</p>
          </CardContent>
        </Card>

        {contaId && (
          <Card>
            <CardHeader><CardTitle>2. Credenciais da integração</CardTitle></CardHeader>
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
                  <p>Token: {diag.token_ok ? '✓ emitido' : '✗ não emitido'} · Saldo: {diag.saldo_ok ? `✓ ${brl(diag.saldo_disponivel ?? 0)} (banking ${diag.banking_versao})` : '✗'} · Certificado: {diag.cert_validade ? `até ${diag.cert_validade} (${diag.cert_dias_restantes} dias)` : '—'}</p>
                  <div>
                    <p className="font-medium">Permissões concedidas ({concedidos.length})</p>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {concedidos.map(e => <Badge key={e.escopo} variant={e.perigoso ? 'destructive' : 'default'}>{e.escopo}</Badge>)}
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
