import { ArrowUpRight, Clock, TrendingUp, Trophy, Warehouse } from 'lucide-react'

export type Product = {
  id: string
  name: string
  description: string
  icon_url: string
  destination_url: string
  contracted: boolean
}

// Cada acento tem par claro/escuro. No dark a sombra sólida "5px 5px 0" dá lugar
// a um glow da própria cor, seguindo o design system glass-green-effect.
const ACCENTS = [
  {
    bg: 'bg-blue-50 dark:bg-blue-500/10',
    text: 'text-blue-700 dark:text-blue-300',
    label: 'bg-blue-100 text-blue-700 dark:bg-blue-500/15 dark:text-blue-300',
    border: '#93c5fd', shadow: '#bfdbfe',
    borderDark: 'rgba(96, 165, 250, 0.35)', glowDark: 'rgba(59, 130, 246, 0.22)',
  },
  {
    bg: 'bg-emerald-50 dark:bg-teal-500/10',
    text: 'text-emerald-700 dark:text-teal-300',
    label: 'bg-emerald-100 text-emerald-700 dark:bg-teal-500/15 dark:text-teal-300',
    border: '#6ee7b7', shadow: '#a7f3d0',
    borderDark: 'rgba(45, 212, 191, 0.4)', glowDark: 'rgba(20, 184, 166, 0.28)',
  },
  {
    bg: 'bg-violet-50 dark:bg-violet-500/10',
    text: 'text-violet-700 dark:text-violet-300',
    label: 'bg-violet-100 text-violet-700 dark:bg-violet-500/15 dark:text-violet-300',
    border: '#c4b5fd', shadow: '#ddd6fe',
    borderDark: 'rgba(167, 139, 250, 0.35)', glowDark: 'rgba(139, 92, 246, 0.22)',
  },
  {
    bg: 'bg-amber-50 dark:bg-amber-500/10',
    text: 'text-amber-700 dark:text-amber-300',
    label: 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300',
    border: '#fcd34d', shadow: '#fde68a',
    borderDark: 'rgba(251, 191, 36, 0.35)', glowDark: 'rgba(245, 158, 11, 0.2)',
  },
]

// Ícone por produto — imagem pública ou componente lucide
function ProductIcon({ name, accent }: { name: string; accent: typeof ACCENTS[0] }) {
  const n = name.toLowerCase()

  if (n.includes('apura')) {
    return <img src="/receita-apuracao.png" alt="Receita Federal" className="w-full h-full object-contain" />
  }
  if (n.includes('simulador')) {
    return <TrendingUp className={`w-10 h-10 ${accent.text}`} strokeWidth={2} />
  }
  if (n.includes('farol')) {
    return <img src="/farol.png" alt="Farol" className="w-full h-full object-contain" />
  }
  if (n.includes('smart') || n.includes('pick')) {
    return <Warehouse className={`w-10 h-10 ${accent.text}`} strokeWidth={1.75} />
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
        accent-card relative flex flex-col h-full min-h-[220px] rounded-2xl border-2 bg-card p-5
        ${hasLink
          ? 'accent-card-interactive cursor-pointer hover:-translate-y-2 hover:-translate-x-1 group'
          : ''}
      `}
      style={{
        '--accent-border': accent.border,
        '--accent-shadow': accent.shadow,
        '--accent-border-dark': accent.borderDark,
        '--accent-glow-dark': accent.glowDark,
        '--accent-offset': hasLink ? '5px 5px' : '3px 3px',
      } as React.CSSProperties}
    >
      {/* Ícone */}
      <div className={`w-16 h-16 rounded-2xl ${accent.bg} flex items-center justify-center mb-4 flex-shrink-0 overflow-hidden`}>
        {icon_url
          ? <img src={icon_url} alt={name} className="w-7 h-7 object-contain" />
          : <ProductIcon name={name} accent={accent} />
        }
      </div>

      {/* Conteúdo */}
      <div className="flex flex-col flex-1">
        <h3 className="text-sm font-bold text-foreground mb-2 leading-snug">{name}</h3>
        <p className="text-xs text-muted-foreground leading-relaxed flex-1">{description}</p>
      </div>

      {/* Rodapé */}
      <div className="mt-4 pt-3 border-t border-border flex items-center justify-between">
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
          <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <Clock className="w-3 h-3" /> Em breve
          </span>
        )}
      </div>
    </div>
  )

  if (hasLink) {
    return (
      <a href={destination_url} target="_blank" rel="noopener noreferrer" className="block h-full" aria-label={`Acessar ${name}`}>
        {card}
      </a>
    )
  }

  return card
}
