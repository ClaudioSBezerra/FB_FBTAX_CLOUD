import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'

// Leads do Assistente FbTax (bot de vendas da landing apuracaoassistida.ia.br) — issue #1.

interface Mensagem { papel: 'visitante' | 'assistente'; texto: string }
interface Lead {
  id: number
  criado_em: string
  atualizado_em: string
  nome: string
  empresa: string
  cargo: string
  email: string
  whatsapp: string
  faixa_faturamento: string
  erp: string
  num_empresas: string
  prazo: string
  resumo: string
  origem: string
  status: Status
  observacao: string
  mensagens: number
  conversa: Mensagem[]
}
type Status = 'conversa' | 'qualificado' | 'quente' | 'contatado' | 'descartado'

const ROTULO: Record<Status, string> = {
  quente: 'Quente', qualificado: 'Qualificado', conversa: 'Só conversa', contatado: 'Contatado', descartado: 'Descartado',
}
const COR: Record<Status, string> = {
  quente: 'bg-red-100 text-red-700', qualificado: 'bg-amber-100 text-amber-800', conversa: 'bg-gray-100 text-gray-700',
  contatado: 'bg-green-100 text-green-700', descartado: 'bg-gray-100 text-gray-400',
}

function quando(iso: string) {
  const d = new Date(iso)
  return isNaN(d.getTime()) ? '' : d.toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })
}
function linkWhats(n: string) {
  const d = n.replace(/\D/g, '')
  return d ? `https://wa.me/${d.length <= 11 ? '55' + d : d}` : ''
}

export default function LeadsPage() {
  const [filtro, setFiltro] = useState<Status | ''>('')
  const [leads, setLeads] = useState<Lead[]>([])
  const [carregando, setCarregando] = useState(true)
  const [aberto, setAberto] = useState<number | null>(null)
  const [obs, setObs] = useState<Record<number, string>>({})

  const carregar = async () => {
    setCarregando(true)
    try {
      const res = await fetch(`/api/vendas/leads${filtro ? `?status=${filtro}` : ''}`)
      if (!res.ok) throw new Error('Erro ao carregar os leads')
      setLeads(await res.json())
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Erro ao carregar os leads')
    } finally {
      setCarregando(false)
    }
  }
  useEffect(() => { carregar() }, [filtro])

  const mudar = async (l: Lead, status: Status) => {
    try {
      const corpo: { status: Status; observacao?: string } = { status }
      if (obs[l.id] !== undefined) corpo.observacao = obs[l.id]
      const res = await fetch(`/api/vendas/leads?id=${l.id}`, {
        method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(corpo),
      })
      if (!res.ok) throw new Error('Erro ao atualizar')
      toast.success(`Lead #${l.id}: ${ROTULO[status]}`)
      carregar()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Erro ao atualizar')
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-5xl mx-auto space-y-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 className="text-2xl font-bold">Leads do site</h1>
            <p className="text-sm text-muted-foreground">Conversas do Assistente FbTax em apuracaoassistida.ia.br.</p>
          </div>
          <div className="flex flex-wrap gap-1">
            {(['', 'quente', 'qualificado', 'contatado', 'conversa', 'descartado'] as (Status | '')[]).map(s => (
              <Button key={s || 'todos'} size="sm" variant={filtro === s ? 'default' : 'outline'} onClick={() => setFiltro(s)}>
                {s ? ROTULO[s] : 'Todos'}
              </Button>
            ))}
          </div>
        </div>

        {carregando ? <p className="text-sm text-muted-foreground py-8 text-center">Carregando…</p>
          : leads.length === 0 ? <p className="text-sm text-muted-foreground py-8 text-center">Nenhum lead.</p>
          : leads.map(l => (
            <Card key={l.id} className={l.status === 'quente' ? 'border-red-300' : ''}>
              <CardHeader className="pb-2">
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div>
                    <CardTitle className="text-base flex items-center gap-2">
                      #{l.id} · {l.nome || 'Visitante'}{l.empresa && ` · ${l.empresa}`}
                      <span className={`text-xs px-2 py-0.5 rounded ${COR[l.status]}`}>{ROTULO[l.status]}</span>
                    </CardTitle>
                    <CardDescription className="mt-1">
                      {[l.cargo, l.faixa_faturamento && `faturamento ${l.faixa_faturamento}`, l.erp && `ERP ${l.erp}`,
                        l.num_empresas && `${l.num_empresas} empresa(s)`, l.prazo && `prazo ${l.prazo}`].filter(Boolean).join(' · ') || 'sem qualificação ainda'}
                    </CardDescription>
                    <CardDescription>{quando(l.atualizado_em)} · {l.mensagens} mensagem(ns){l.origem && ` · origem: ${l.origem}`}</CardDescription>
                  </div>
                  <div className="flex gap-1 shrink-0">
                    {l.whatsapp && <Button size="sm" variant="outline" asChild><a href={linkWhats(l.whatsapp)} target="_blank" rel="noopener noreferrer">WhatsApp</a></Button>}
                    {l.email && <Button size="sm" variant="outline" asChild><a href={`mailto:${l.email}?subject=${encodeURIComponent('FBTax - demonstração')}`}>E-mail</a></Button>}
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-3 text-sm">
                {l.resumo && <p><span className="text-muted-foreground">Interesse:</span> {l.resumo}</p>}
                <button type="button" className="text-xs text-primary hover:underline" onClick={() => setAberto(aberto === l.id ? null : l.id)}>
                  {aberto === l.id ? 'Esconder conversa' : 'Ver conversa'}
                </button>
                {aberto === l.id && (
                  <div className="rounded-md border bg-white p-3 space-y-2 max-h-80 overflow-y-auto">
                    {l.conversa.map((m, i) => (
                      <p key={i}><strong className={m.papel === 'visitante' ? 'text-gray-900' : 'text-teal-700'}>
                        {m.papel === 'visitante' ? 'Visitante' : 'Assistente'}:</strong> {m.texto}</p>
                    ))}
                  </div>
                )}
                <div className="flex flex-col sm:flex-row gap-2 sm:items-end">
                  <Textarea rows={2} className="text-sm" placeholder="Observação (ex.: demo marcada para 25/11 às 10h)"
                    value={obs[l.id] ?? l.observacao} onChange={e => setObs(o => ({ ...o, [l.id]: e.target.value }))} />
                  <div className="flex gap-1">
                    <Button size="sm" onClick={() => mudar(l, 'contatado')} disabled={l.status === 'contatado'}>Contatado</Button>
                    <Button size="sm" variant="outline" onClick={() => mudar(l, 'descartado')} disabled={l.status === 'descartado'}>Descartar</Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
      </div>
    </div>
  )
}
