import { ArrowUpRight, Clock, TrendingUp, Trophy, Warehouse } from 'lucide-react'

export type Product = {
  id: string
  name: string
  description: string
  icon_url: string
  destination_url: string
  contracted: boolean
}

const ACCENTS = [
  { bg: 'bg-blue-50',    text: 'text-blue-700',    label: 'bg-blue-100 text-blue-700',    border: '#93c5fd', shadow: '#bfdbfe' },
  { bg: 'bg-emerald-50', text: 'text-emerald-700',  label: 'bg-emerald-100 text-emerald-700', border: '#6ee7b7', shadow: '#a7f3d0' },
  { bg: 'bg-violet-50',  text: 'text-violet-700',   label: 'bg-violet-100 text-violet-700',   border: '#c4b5fd', shadow: '#ddd6fe' },
  { bg: 'bg-amber-50',   text: 'text-amber-700',    label: 'bg-amber-100 text-amber-700',     border: '#fcd34d', shadow: '#fde68a' },
]

// Ícone por produto — imagem pública ou componente lucide
function ProductIcon({ name, accent }: { name: string; accent: typeof ACCENTS[0] }) {
  const n = name.toLowerCase()

  if (n.includes('apura')) {
    return <img src="/receita-apuracao.png" alt="Receita Federal" className="w-full h-full object-cover rounded-2xl" />
  }
  if (n.includes('simulador')) {
    return <TrendingUp className={`w-6 h-6 ${accent.text}`} strokeWidth={2.5} />
  }
  if (n.includes('farol')) {
    return <img src="/farol.png" alt="Farol" className="w-9 h-9 object-contain" />
  }
  if (n.includes('smart') || n.includes('pick')) {
    return <Warehouse className={`w-6 h-6 ${accent.text}`} strokeWidth={2} />
  }
  // fallback genérico
  return <span className={`text-xl font-black ${accent.text}`}>{name.charAt(0)}</span>
}

type ProductCardProps = Product & { colorIndex?: number }

export function ProductCard({ name, description, icon_url, destination_url, colorIndex = 0 }: ProductCardProps) {
  const accent = ACCENTS[colorIndex % ACCENTS.length]
  const hasLink = !!destination_url

  const card = (
    <div
      className={`
        relative flex flex-col h-full min-h-[220px] rounded-2xl border-2 bg-white p-5
        transition-all duration-200
        ${hasLink
          ? 'cursor-pointer hover:-translate-y-2 hover:-translate-x-1 group'
          : 'opacity-80'}
      `}
      style={{
        borderColor: accent.border,
        boxShadow: hasLink
          ? `5px 5px 0px ${accent.shadow}`
          : `3px 3px 0px ${accent.shadow}`,
      }}
    >
      {/* Ícone */}
      <div className={`w-12 h-12 rounded-2xl ${accent.bg} flex items-center justify-center mb-4 flex-shrink-0`}>
        {icon_url
          ? <img src={icon_url} alt={name} className="w-7 h-7 object-contain" />
          : <ProductIcon name={name} accent={accent} />
        }
      </div>

      {/* Conteúdo */}
      <div className="flex flex-col flex-1">
        <h3 className="text-sm font-bold text-slate-900 mb-2 leading-snug">{name}</h3>
        <p className="text-xs text-slate-500 leading-relaxed flex-1">{description}</p>
      </div>

      {/* Rodapé */}
      <div className="mt-4 pt-3 border-t border-slate-100 flex items-center justify-between">
        {hasLink ? (
          <>
            <span className={`text-xs font-semibold px-2 py-0.5 rounded-full ${accent.label}`}>
              Disponível
            </span>
            <span className={`flex items-center gap-1 text-xs font-medium ${accent.text} group-hover:gap-2 transition-all`}>
              Acessar <ArrowUpRight className="w-3.5 h-3.5" />
            </span>
          </>
        ) : (
          <span className="flex items-center gap-1.5 text-xs text-slate-400">
            <Clock className="w-3 h-3" /> Em breve
          </span>
        )}
      </div>
    </div>
  )

  if (hasLink) {
    return (
      <a href={destination_url} target="_blank" rel="noopener noreferrer" className="block h-full">
        {card}
      </a>
    )
  }

  return card
}
