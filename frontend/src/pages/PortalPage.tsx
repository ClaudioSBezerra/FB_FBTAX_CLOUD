import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useSearchParams, Link } from 'react-router-dom'
import { ProductCard, type Product } from '@/components/ProductCard'
import { ThemeToggle } from '@/components/ThemeToggle'
import { Skeleton } from '@/components/ui/skeleton'
import { AlertCircle, Sparkles, Zap, Shield, Clock3, Lock } from 'lucide-react'

// Mapa de clientes para apresentação — adicionar logo em /public e entrada aqui
const CLIENTE_MAP: Record<string, { logo: string; name: string }> = {
  jc:        { logo: '/JC.png',        name: 'JC'              },
  ferreira:  { logo: '/logo-ferreira-costa.png', name: 'Ferreira Costa' },
  // adicionar novos clientes aqui: slug: { logo: '/arquivo.png', name: 'Nome' }
}

function ProductCardSkeleton() {
  return (
    <div className="rounded-2xl border-2 border-border bg-card p-5 min-h-[220px] flex flex-col gap-4 shadow-[4px_4px_0px_hsl(var(--border))]">
      <Skeleton className="w-16 h-16 rounded-2xl" />
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
      <path d="M50 4 C51.5 4 52 5 52.5 7 L56 30 C56.5 33 53.5 36 50 36 C46.5 36 43.5 33 44 30 L47.5 7 C48 5 48.5 4 50 4Z"/>
      <path d="M50 96 C48.5 96 48 95 47.5 93 L44 70 C43.5 67 46.5 64 50 64 C53.5 64 56.5 67 56 70 L52.5 93 C52 95 51.5 96 50 96Z"/>
      <path d="M4 50 C4 48.5 5 48 7 47.5 L30 44 C33 43.5 36 46.5 36 50 C36 53.5 33 56.5 30 56 L7 52.5 C5 52 4 51.5 4 50Z"/>
      <path d="M96 50 C96 51.5 95 52 93 52.5 L70 56 C67 56.5 64 53.5 64 50 C64 46.5 67 43.5 70 44 L93 47.5 C95 48 96 48.5 96 50Z"/>
      <path d="M17.2 17.2 C18.2 16.2 19.5 16.5 21 17.7 L38 34.7 C40.2 36.9 39.8 40.5 37.5 42.5 C35.2 44.5 31.5 44.1 29.7 41.5 L16.5 21.5 C15.2 19.7 16.2 18.2 17.2 17.2Z"/>
      <path d="M82.8 82.8 C81.8 83.8 80.5 83.5 79 82.3 L62 65.3 C59.8 63.1 60.2 59.5 62.5 57.5 C64.8 55.5 68.5 55.9 70.3 58.5 L83.5 78.5 C84.8 80.3 83.8 81.8 82.8 82.8Z"/>
      <path d="M82.8 17.2 C83.8 18.2 83.5 19.5 82.3 21 L65.3 38 C63.1 40.2 59.5 39.8 57.5 37.5 C55.5 35.2 55.9 31.5 58.5 29.7 L78.5 16.5 C80.3 15.2 81.8 16.2 82.8 17.2Z"/>
      <path d="M17.2 82.8 C16.2 81.8 16.5 80.5 17.7 79 L34.7 62 C36.9 59.8 40.5 60.2 42.5 62.5 C44.5 64.8 44.1 68.5 41.5 70.3 L21.5 83.5 C19.7 84.8 18.2 83.8 17.2 82.8Z"/>
      <circle cx="50" cy="50" r="10"/>
    </svg>
  )
}

const STACK = [
  { label: 'Go',          color: 'bg-cyan-100 text-cyan-800 dark:bg-cyan-500/15 dark:text-cyan-300'       },
  { label: 'React 18',    color: 'bg-blue-100 text-blue-800 dark:bg-blue-500/15 dark:text-blue-300'       },
  { label: 'TypeScript',  color: 'bg-indigo-100 text-indigo-800 dark:bg-indigo-500/15 dark:text-indigo-300' },
  { label: 'PostgreSQL',  color: 'bg-sky-100 text-sky-800 dark:bg-sky-500/15 dark:text-sky-300'           },
  { label: 'Tailwind',    color: 'bg-teal-100 text-teal-800 dark:bg-teal-500/15 dark:text-teal-300'       },
  { label: 'Docker',      color: 'bg-slate-100 text-slate-700 dark:bg-slate-500/15 dark:text-slate-300'   },
]

const PILLARS = [
  { icon: Sparkles, title: 'IA Generativa',    desc: 'Projetada e construída com Claude AI da Anthropic — o assistente mais capaz do mercado.' },
  { icon: Zap,      title: 'Alta Performance', desc: 'Backend em Go, latência sub-milissegundo. Frontend React com bundle < 100 kB.' },
  { icon: Lock,     title: 'Segurança',        desc: 'JWT com expiração, refresh tokens, HTTPS obrigatório, prepared statements contra SQL injection, CORS restrito, rate limiting nas APIs e isolamento de dados por tenant.' },
  { icon: Shield,   title: 'Longevidade',      desc: 'Go, React e PostgreSQL — stack madura, sem modismos, dominante há décadas.' },
  { icon: Clock3,   title: 'Sempre evolui',    desc: 'Arquitetura modular. Novas soluções entram sem reescrever o que já funciona.' },
]

export default function PortalPage() {
  const [showTech, setShowTech] = useState(false)
  const [searchParams] = useSearchParams()

  // ?cliente=jc → mostra badge de apresentação
  const clienteSlug = searchParams.get('cliente')?.toLowerCase() ?? ''
  const cliente = clienteSlug ? CLIENTE_MAP[clienteSlug] : null

  const { data, isPending, isError } = useQuery<Product[]>({
    queryKey: ['products'],
    queryFn: () => fetch('/api/portal/products').then(r => r.json()),
  })

  return (
    <div className="min-h-screen bg-background flex flex-col">

      {/* ── Hero ── */}
      <header className="relative overflow-hidden text-white bg-gradient-to-br from-slate-900 via-blue-950 to-slate-800 dark:from-[#0b1620] dark:via-[#0a1a2b] dark:to-[#0e1a2a]">
        {/* Orbes de luz ambiente — só aparecem no tema escuro */}
        <div className="ambient-light -top-24 left-[8%]" aria-hidden="true" />
        <div className="ambient-light-2 -bottom-40 right-[5%]" aria-hidden="true" />

        <div className="relative z-10 max-w-6xl mx-auto px-6 pt-4 pb-8">

          {/* ── Barra superior: tema + link admin ── */}
          <div className="flex justify-end items-center gap-3 mb-6">
            <ThemeToggle className="border-slate-600 text-slate-300 hover:text-white hover:border-slate-400" />
            {/* Página estática servida de public/portfolio — fora do React Router */}
            <a
              href="/portfolio/"
              className="text-sm text-slate-300 hover:text-white border border-slate-600 hover:border-slate-400 dark:hover:border-teal-400/60 px-4 py-1.5 rounded-lg transition-colors"
            >
              Portfólio
            </a>
            <Link
              to="/admin/login"
              className="text-sm text-slate-300 hover:text-white border border-slate-600 hover:border-slate-400 dark:hover:border-teal-400/60 px-4 py-1.5 rounded-lg transition-colors"
            >
              Área Administrativa
            </Link>
          </div>

          <div className="flex flex-col lg:flex-row lg:items-start lg:justify-between gap-6">

            {/* Esquerda: ícone Claude + título + subtítulo */}
            <div className="flex-1">
              <div className="flex items-center gap-3 mb-1">
                <img src="/claude.png" alt="Claude AI" className="w-10 h-10 sm:w-11 sm:h-11 rounded-xl flex-shrink-0" />
                <div>
                  <h1 className="text-3xl sm:text-4xl font-bold leading-tight text-glow">
                    Soluções <span className="text-blue-400 dark:text-teal-300">inteligentes</span>
                  </h1>
                  <p className="text-xl sm:text-2xl font-semibold text-slate-300 leading-tight">
                    para sua empresa
                  </p>
                </div>
              </div>
              <p className="text-slate-400 text-sm leading-relaxed mt-3 lg:whitespace-nowrap">
                Plataforma integrada com ferramentas especializadas em apuração tributária, simulação de cenários fiscais, gestão de RCAs e gestão estratégica de WMS.
              </p>
            </div>

            {/* Direita: badge de apresentação — só aparece com ?cliente=slug */}
            {cliente && (
              <div className="flex-shrink-0">
                <div className="flex items-center gap-2 bg-white/10 rounded-xl px-4 py-2 backdrop-glow border border-white/10">
                  <span className="text-xs text-slate-400">Apresentação para</span>
                  <img src={cliente.logo} alt={cliente.name} className="h-8 w-auto rounded" />
                </div>
              </div>
            )}

          </div>
        </div>
      </header>

      {/* ── Soluções ── */}
      <main className="max-w-6xl mx-auto w-full px-6 py-8 flex-1">
        <div className="mb-6">
          <h2 className="text-lg font-bold text-foreground">Nossas Soluções</h2>
          <p className="text-muted-foreground mt-0.5 text-xs">Clique em uma solução para acessá-la diretamente.</p>
        </div>

        {isError && (
          <div className="flex items-center gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300 mb-6">
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

        {/* ── Tecnologia (descortina ao clicar em Dúvidas?) ── */}
        <div
          className="overflow-hidden transition-all duration-500 ease-in-out"
          style={{ maxHeight: showTech ? '800px' : '0px', opacity: showTech ? 1 : 0 }}
        >
          <div className="mt-12">
            <div className="flex items-center gap-4 mb-8">
              <div className="flex-1 border-t border-border" />
              <span className="text-xs font-semibold text-muted-foreground tracking-widest uppercase">Tecnologia</span>
              <div className="flex-1 border-t border-border" />
            </div>

            <div className="flex flex-col items-center text-center mb-8">
              <div className="inline-flex items-center gap-2 bg-gradient-to-r from-orange-500 to-amber-400 text-white px-4 py-2 rounded-full font-semibold text-xs shadow-lg mb-2">
                <ClaudeSun className="w-3.5 h-3.5" />
                Powered by Claude AI · Anthropic
              </div>
              <p className="text-muted-foreground text-xs max-w-lg leading-relaxed">
                Concebida e construída com IA generativa — do design à arquitetura, do banco ao frontend.
              </p>
            </div>

            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3 mb-8">
              {PILLARS.map(({ icon: Icon, title, desc }) => (
                <div key={title} className="bg-card rounded-xl border border-border p-4 shadow-sm glow-card">
                  <div className="w-7 h-7 rounded-lg bg-slate-900 dark:bg-teal-500/20 dark:border dark:border-teal-400/40 flex items-center justify-center mb-2">
                    <Icon className="w-3.5 h-3.5 text-white dark:text-teal-300" />
                  </div>
                  <h3 className="text-xs font-bold text-foreground mb-1">{title}</h3>
                  <p className="text-[11px] text-muted-foreground leading-relaxed">{desc}</p>
                </div>
              ))}
            </div>

            <div className="flex flex-wrap justify-center gap-1.5 pb-10">
              {STACK.map(({ label, color }) => (
                <span key={label} className={`px-2.5 py-0.5 rounded-full text-[11px] font-semibold ${color}`}>
                  {label}
                </span>
              ))}
            </div>
          </div>
        </div>

        {/* ── Botão Dúvidas? ── */}
        <div className="flex justify-center mt-6">
          <button
            onClick={() => setShowTech(v => !v)}
            className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            <span>{showTech ? 'Fechar' : 'Dúvidas?'}</span>
            <svg
              className={`w-3 h-3 transition-transform duration-300 ${showTech ? 'rotate-180' : ''}`}
              viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="2"
            >
              <path d="M2 4l4 4 4-4" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
          </button>
        </div>
      </main>

      {/* ── Footer ── */}
      <footer className="border-t border-border bg-card mt-6">
        <div className="max-w-6xl mx-auto px-6 py-4 flex flex-col sm:flex-row items-center justify-between gap-2 text-xs text-muted-foreground">
          <div className="flex items-center gap-2">
            <div className="w-6 h-6 rounded-md bg-blue-500 dark:bg-gradient-to-br dark:from-teal-500 dark:to-cyan-400 flex items-center justify-center font-bold text-white dark:text-[#0b1620] text-[10px] tracking-wide select-none flex-shrink-0">
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
