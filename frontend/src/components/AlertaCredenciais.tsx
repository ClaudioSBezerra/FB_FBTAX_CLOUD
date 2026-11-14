import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

interface Credencial {
  chave: string
  tipo: string
  nome: string
  validade: string
  dias_restantes: number
  nivel: 'ok' | 'atencao' | 'critico' | 'vencida'
  acao: string
  link: string
}

const dia = (s: string) => s.split('-').reverse().join('/')

// Faixa no topo de todas as telas do Financeiro: certificado do Inter e A1 da NFS-e a vencer.
// Aparece a partir de 30 dias; amarela até 30, vermelha com 7 dias ou menos ou vencida.
export default function AlertaCredenciais() {
  const [itens, setItens] = useState<Credencial[]>([])

  useEffect(() => {
    let vivo = true
    const carregar = () =>
      fetch('/api/financeiro/credenciais/vencimentos')
        .then(r => (r.ok ? r.json() : []))
        .then((l: Credencial[]) => vivo && setItens(l.filter(c => c.nivel !== 'ok')))
        .catch(() => {})
    carregar()
    const t = window.setInterval(carregar, 30 * 60 * 1000)
    return () => { vivo = false; window.clearInterval(t) }
  }, [])

  if (itens.length === 0) return null
  const grave = itens.some(c => c.nivel === 'critico' || c.nivel === 'vencida')

  return (
    <div className={`border-b px-6 py-3 text-sm space-y-1 ${grave ? 'bg-red-50 border-red-300 text-red-900' : 'bg-amber-50 border-amber-300 text-amber-900'}`} role="alert">
      <p className="font-semibold">{grave ? 'Atenção: credencial vencida ou prestes a vencer' : 'Credencial vencendo em breve'} — sem ela o sistema perde o acesso ao banco e à emissão de NFS-e.</p>
      {itens.map(c => (
        <p key={c.chave}>
          <b>{c.nome}</b>:{' '}
          {c.dias_restantes < 0 ? `VENCIDA há ${-c.dias_restantes} dia(s)` : c.dias_restantes === 0 ? 'vence HOJE' : `vence em ${c.dias_restantes} dia(s)`} ({dia(c.validade)}).{' '}
          {c.acao} <Link className="underline font-medium" to={c.link}>Abrir</Link>
        </p>
      ))}
    </div>
  )
}
