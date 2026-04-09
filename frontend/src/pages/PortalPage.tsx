import { useQuery } from '@tanstack/react-query'
import { ProductCard, type Product } from '@/components/ProductCard'
import { Skeleton } from '@/components/ui/skeleton'
import { AlertCircle, Sparkles, Zap, Shield, Clock3, Lock } from 'lucide-react'

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

// Símbolo Claude — sol com raios (logo Anthropic/Claude)
function ClaudeSun({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 100 100" fill="currentColor" className={className} aria-hidden="true">
      {/* Raios principais — 4 pontas longas */}
      <path d="M50 4 C51.5 4 52 5 52.5 7 L56 30 C56.5 33 53.5 36 50 36 C46.5 36 43.5 33 44 30 L47.5 7 C48 5 48.5 4 50 4Z"/>
      <path d="M50 96 C48.5 96 48 95 47.5 93 L44 70 C43.5 67 46.5 64 50 64 C53.5 64 56.5 67 56 70 L52.5 93 C52 95 51.5 96 50 96Z"/>
      <path d="M4 50 C4 48.5 5 48 7 47.5 L30 44 C33 43.5 36 46.5 36 50 C36 53.5 33 56.5 30 56 L7 52.5 C5 52 4 51.5 4 50Z"/>
      <path d="M96 50 C96 51.5 95 52 93 52.5 L70 56 C67 56.5 64 53.5 64 50 C64 46.5 67 43.5 70 44 L93 47.5 C95 48 96 48.5 96 50Z"/>
      {/* Raios diagonais — 4 pontas médias */}
      <path d="M17.2 17.2 C18.2 16.2 19.5 16.5 21 17.7 L38 34.7 C40.2 36.9 39.8 40.5 37.5 42.5 C35.2 44.5 31.5 44.1 29.7 41.5 L16.5 21.5 C15.2 19.7 16.2 18.2 17.2 17.2Z"/>
      <path d="M82.8 82.8 C81.8 83.8 80.5 83.5 79 82.3 L62 65.3 C59.8 63.1 60.2 59.5 62.5 57.5 C64.8 55.5 68.5 55.9 70.3 58.5 L83.5 78.5 C84.8 80.3 83.8 81.8 82.8 82.8Z"/>
      <path d="M82.8 17.2 C83.8 18.2 83.5 19.5 82.3 21 L65.3 38 C63.1 40.2 59.5 39.8 57.5 37.5 C55.5 35.2 55.9 31.5 58.5 29.7 L78.5 16.5 C80.3 15.2 81.8 16.2 82.8 17.2Z"/>
      <path d="M17.2 82.8 C16.2 81.8 16.5 80.5 17.7 79 L34.7 62 C36.9 59.8 40.5 60.2 42.5 62.5 C44.5 64.8 44.1 68.5 41.5 70.3 L21.5 83.5 C19.7 84.8 18.2 83.8 17.2 82.8Z"/>
      {/* Centro */}
      <circle cx="50" cy="50" r="10"/>
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
  { icon: Sparkles, title: 'IA Generativa',    desc: 'Projetada e construída com Claude AI da Anthropic — o assistente mais capaz do mercado.' },
  { icon: Zap,      title: 'Alta Performance', desc: 'Backend em Go, latência sub-milissegundo. Frontend React com bundle < 100 kB.' },
  { icon: Lock,     title: 'Segurança',        desc: 'JWT com expiração, refresh tokens, HTTPS obrigatório, prepared statements contra SQL injection, CORS restrito, rate limiting nas APIs e isolamento de dados por tenant.' },
  { icon: Shield,   title: 'Longevidade',      desc: 'Go, React e PostgreSQL — stack madura, sem modismos, dominante há décadas.' },
  { icon: Clock3,   title: 'Sempre evolui',    desc: 'Arquitetura modular. Novas soluções entram sem reescrever o que já funciona.' },
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

          {/* Título + subtítulo (esquerda) e card JC (direita) — mesma linha */}
          <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-6">

            {/* Esquerda: ícone Claude + título + subtítulo */}
            <div className="flex-1">
              <h1 className="text-3xl sm:text-4xl font-bold leading-tight mb-3 flex items-center gap-3 flex-wrap">
                <img src="/claude.png" alt="Claude AI" className="w-10 h-10 sm:w-11 sm:h-11 rounded-xl flex-shrink-0" />
                Soluções <span className="text-blue-400">inteligentes</span> para sua empresa
              </h1>
              <p className="text-slate-300 text-sm leading-relaxed max-w-xl">
                Plataforma integrada com ferramentas especializadas em apuração tributária,
                simulação de cenários fiscais, gestão de RCAs e gestão estratégica de WMS.
              </p>
            </div>

            {/* Direita: card Apresentação JC */}
            <div className="flex-shrink-0">
              <div className="flex items-center gap-2 bg-white/10 rounded-xl px-4 py-2 backdrop-blur-sm">
                <span className="text-xs text-slate-400">Apresentação para</span>
                <img src="/JC.png" alt="JC" className="h-8 w-auto rounded" />
              </div>
            </div>

          </div>
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
          <div className="flex flex-col items-center text-center mb-8">
            <div className="inline-flex items-center gap-2 bg-gradient-to-r from-orange-500 to-amber-400 text-white px-4 py-2 rounded-full font-semibold text-xs shadow-lg mb-2">
              <ClaudeSun className="w-3.5 h-3.5" />
              Powered by Claude AI · Anthropic
            </div>
            <p className="text-slate-400 text-xs max-w-lg leading-relaxed">
              Concebida e construída com IA generativa — do design à arquitetura, do banco ao frontend.
            </p>
          </div>

          {/* Pilares */}
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3 mb-8">
            {PILLARS.map(({ icon: Icon, title, desc }) => (
              <div key={title} className="bg-white rounded-xl border border-slate-100 p-4 shadow-sm">
                <div className="w-7 h-7 rounded-lg bg-slate-900 flex items-center justify-center mb-2">
                  <Icon className="w-3.5 h-3.5 text-white" />
                </div>
                <h3 className="text-xs font-bold text-slate-800 mb-1">{title}</h3>
                <p className="text-[11px] text-slate-400 leading-relaxed">{desc}</p>
              </div>
            ))}
          </div>

          {/* Stack badges */}
          <div className="flex flex-wrap justify-center gap-1.5">
            {STACK.map(({ label, color }) => (
              <span key={label} className={`px-2.5 py-0.5 rounded-full text-[11px] font-semibold ${color}`}>
                {label}
              </span>
            ))}
          </div>
        </div>
      </main>

      {/* ── Footer ── */}
      <footer className="border-t bg-white mt-10">
        <div className="max-w-6xl mx-auto px-6 py-4 flex flex-col sm:flex-row items-center justify-between gap-2 text-xs text-slate-400">
          <div className="flex items-center gap-2">
            <div className="w-6 h-6 rounded-md bg-blue-500 flex items-center justify-center font-bold text-white text-[10px] tracking-wide select-none flex-shrink-0">
              FB
            </div>
            <span>© {new Date().getFullYear()} FBTax Cloud — Todos os direitos reservados</span>
          </div>
          <span className="flex items-center gap-1">
            <ClaudeSun className="w-3 h-3 text-orange-400" />
            Built with Claude AI
          </span>
        </div>
      </footer>
    </div>
  )
}
