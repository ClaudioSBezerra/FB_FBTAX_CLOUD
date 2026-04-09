import { useQuery } from '@tanstack/react-query'
import { ProductCard, type Product } from '@/components/ProductCard'
import { Skeleton } from '@/components/ui/skeleton'
import { AlertCircle, Sparkles, Zap, Shield, Clock3 } from 'lucide-react'

function ProductCardSkeleton() {
  return (
    <div className="rounded-2xl border-2 border-slate-100 bg-white p-5 min-h-[220px] flex flex-col gap-4"
      style={{ boxShadow: '4px 4px 0px #e2e8f0' }}>
      <Skeleton className="w-12 h-12 rounded-2xl" />
      <Skeleton className="h-4 w-32" />
      <Skeleton className="h-3 w-full" />
      <Skeleton className="h-3 w-3/4" />
    </div>
  )
}

// Símbolo Claude / Anthropic — 4-pointed star
function ClaudeStar({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={className} aria-hidden="true">
      <path d="M12 2 L13.8 10.2 L22 12 L13.8 13.8 L12 22 L10.2 13.8 L2 12 L10.2 10.2 Z" />
    </svg>
  )
}

const STACK = [
  { label: 'Go',          color: 'bg-cyan-100 text-cyan-800'    },
  { label: 'React 18',    color: 'bg-blue-100 text-blue-800'    },
  { label: 'TypeScript',  color: 'bg-indigo-100 text-indigo-800'},
  { label: 'PostgreSQL',  color: 'bg-sky-100 text-sky-800'      },
  { label: 'Tailwind',    color: 'bg-teal-100 text-teal-800'    },
  { label: 'Docker',      color: 'bg-slate-100 text-slate-700'  },
]

const PILLARS = [
  { icon: Sparkles, title: 'IA Generativa',  desc: 'Toda a plataforma foi projetada e construída com Claude AI da Anthropic — o assistente mais capaz do mercado.' },
  { icon: Zap,      title: 'Alta Performance', desc: 'Backend em Go com zero dependências externas e latência sub-milissegundo. Frontend React com bundle < 100 kB.' },
  { icon: Shield,   title: 'Longevidade',    desc: 'Stack escolhida por maturidade e longevidade — sem modismos. Go, React e PostgreSQL dominam o mercado há décadas.' },
  { icon: Clock3,   title: 'Sempre evolui',  desc: 'Arquitetura modular que cresce junto com o negócio. Novas soluções entram sem reescrever o que já funciona.' },
]

export default function PortalPage() {
  const { data, isPending, isError } = useQuery<Product[]>({
    queryKey: ['products'],
    queryFn: () => fetch('/api/portal/products').then(r => r.json()),
  })

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">

      {/* ── Hero ── */}
      <header className="bg-gradient-to-br from-slate-900 via-blue-950 to-slate-800 text-white">
        <div className="max-w-6xl mx-auto px-6 py-8">
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-lg bg-blue-500 flex items-center justify-center font-bold text-white text-xs tracking-wide select-none">
                FB
              </div>
              <span className="text-xs font-medium text-slate-400 tracking-widest uppercase">FBTax Cloud</span>
            </div>
            <div className="flex items-center gap-2 bg-white/10 rounded-xl px-4 py-2 backdrop-blur-sm">
              <span className="text-xs text-slate-400">Apresentação para</span>
              <img src="/JC.png" alt="JC" className="h-8 w-auto rounded" />
            </div>
          </div>

          <h1 className="text-3xl sm:text-4xl font-bold leading-tight mb-3 max-w-2xl">
            Soluções <span className="text-blue-400">inteligentes</span> para sua empresa
          </h1>
          <p className="text-slate-300 text-sm max-w-2xl leading-relaxed">
            Plataforma integrada com ferramentas especializadas em apuração tributária,
            simulação de cenários fiscais, gestão de RCAs e gestão estratégica de WMS.
          </p>
        </div>
      </header>

      {/* ── Soluções ── */}
      <main className="max-w-6xl mx-auto w-full px-6 py-8 flex-1">
        <div className="mb-6">
          <h2 className="text-lg font-bold text-slate-800">Nossas Soluções</h2>
          <p className="text-slate-400 mt-0.5 text-xs">Clique em uma solução para acessá-la diretamente.</p>
        </div>

        {isError && (
          <div className="flex items-center gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 mb-6">
            <AlertCircle className="w-4 h-4 flex-shrink-0" />
            Não foi possível carregar os produtos. Tente novamente em instantes.
          </div>
        )}

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {isPending
            ? Array.from({ length: 4 }).map((_, i) => <ProductCardSkeleton key={i} />)
            : data?.map((product, index) => (
                <ProductCard key={product.id} {...product} colorIndex={index} />
              ))
          }
        </div>

        {/* ── Tecnologia ── */}
        <div className="mt-16">
          {/* Divisor */}
          <div className="flex items-center gap-4 mb-10">
            <div className="flex-1 border-t border-slate-200" />
            <span className="text-xs font-semibold text-slate-400 tracking-widest uppercase">Tecnologia</span>
            <div className="flex-1 border-t border-slate-200" />
          </div>

          {/* Badge Claude */}
          <div className="flex flex-col items-center text-center mb-10">
            <div className="inline-flex items-center gap-2.5 bg-gradient-to-r from-orange-500 to-amber-400 text-white px-5 py-2.5 rounded-full font-semibold text-sm shadow-lg mb-3">
              <ClaudeStar className="w-4 h-4" />
              Powered by Claude AI · Anthropic
            </div>
            <p className="text-slate-500 text-sm max-w-lg leading-relaxed">
              Esta plataforma foi integralmente concebida e construída com inteligência artificial generativa —
              do design à arquitetura, do banco de dados ao frontend.
            </p>
          </div>

          {/* Pilares */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-10">
            {PILLARS.map(({ icon: Icon, title, desc }) => (
              <div key={title} className="bg-white rounded-2xl border border-slate-100 p-5 shadow-sm">
                <div className="w-9 h-9 rounded-xl bg-slate-900 flex items-center justify-center mb-3">
                  <Icon className="w-4 h-4 text-white" />
                </div>
                <h3 className="text-sm font-bold text-slate-800 mb-1">{title}</h3>
                <p className="text-xs text-slate-500 leading-relaxed">{desc}</p>
              </div>
            ))}
          </div>

          {/* Stack badges */}
          <div className="flex flex-wrap justify-center gap-2">
            {STACK.map(({ label, color }) => (
              <span key={label} className={`px-3 py-1 rounded-full text-xs font-semibold ${color}`}>
                {label}
              </span>
            ))}
          </div>
        </div>
      </main>

      {/* ── Footer ── */}
      <footer className="border-t bg-white mt-10">
        <div className="max-w-6xl mx-auto px-6 py-4 flex flex-col sm:flex-row items-center justify-between gap-1 text-xs text-slate-400">
          <span>© {new Date().getFullYear()} FBTax Cloud — Todos os direitos reservados</span>
          <span className="flex items-center gap-1">
            <ClaudeStar className="w-3 h-3 text-orange-400" />
            Built with Claude AI
          </span>
        </div>
      </footer>
    </div>
  )
}
