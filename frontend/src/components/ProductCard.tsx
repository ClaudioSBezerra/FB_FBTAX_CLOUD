import { ArrowRight, Clock } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export type Product = {
  id: string
  name: string
  description: string
  icon_url: string
  destination_url: string
  contracted: boolean
}

const ACCENTS = [
  { light: 'bg-blue-50',    text: 'text-blue-600',    border: 'border-blue-200',    hover: 'hover:border-blue-400'   },
  { light: 'bg-emerald-50', text: 'text-emerald-600', border: 'border-emerald-200', hover: 'hover:border-emerald-400' },
  { light: 'bg-violet-50',  text: 'text-violet-600',  border: 'border-violet-200',  hover: 'hover:border-violet-400' },
  { light: 'bg-amber-50',   text: 'text-amber-600',   border: 'border-amber-200',   hover: 'hover:border-amber-400'  },
]

type ProductCardProps = Product & { colorIndex?: number }

export function ProductCard({ name, description, icon_url, destination_url, colorIndex = 0 }: ProductCardProps) {
  const accent = ACCENTS[colorIndex % ACCENTS.length]
  const hasLink = !!destination_url

  const inner = (
    <Card
      className={`flex flex-col h-full border-2 transition-all duration-200 ${accent.border} ${
        hasLink ? `${accent.hover} hover:shadow-lg hover:-translate-y-1 cursor-pointer` : ''
      }`}
    >
      <CardHeader className="pb-3">
        <div className={`w-12 h-12 rounded-2xl ${accent.light} flex items-center justify-center mb-3`}>
          {icon_url ? (
            <img src={icon_url} alt={name} className="w-7 h-7 object-contain" />
          ) : (
            <span className={`text-xl font-bold ${accent.text}`}>{name.charAt(0)}</span>
          )}
        </div>
        <CardTitle className="text-base font-semibold text-gray-900 leading-snug">{name}</CardTitle>
      </CardHeader>

      <CardContent className="pt-0 flex flex-col flex-1">
        <p className="text-sm text-muted-foreground leading-relaxed flex-1">{description}</p>

        <div className="mt-4">
          {hasLink ? (
            <span className={`inline-flex items-center gap-1 text-sm font-medium ${accent.text}`}>
              Acessar <ArrowRight className="w-4 h-4" />
            </span>
          ) : (
            <span className="inline-flex items-center gap-1 text-xs text-slate-400">
              <Clock className="w-3 h-3" /> Em breve
            </span>
          )}
        </div>
      </CardContent>
    </Card>
  )

  if (hasLink) {
    return (
      <a href={destination_url} target="_blank" rel="noopener noreferrer" className="block h-full">
        {inner}
      </a>
    )
  }

  return inner
}
