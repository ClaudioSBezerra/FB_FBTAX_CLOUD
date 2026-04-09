import { useQuery } from '@tanstack/react-query'
import { ProductCard, type Product } from '@/components/ProductCard'
import { Skeleton } from '@/components/ui/skeleton'
import { AlertCircle } from 'lucide-react'

function ProductCardSkeleton() {
  return (
    <div className="rounded-xl border-2 border-slate-100 bg-white p-4 flex flex-col gap-3">
      <Skeleton className="w-10 h-10 rounded-xl" />
      <Skeleton className="h-4 w-32" />
      <Skeleton className="h-3 w-full" />
      <Skeleton className="h-3 w-3/4" />
    </div>
  )
}

export default function PortalPage() {
  const { data, isPending, isError } = useQuery<Product[]>({
    queryKey: ['products'],
    queryFn: () => fetch('/api/portal/products').then(r => r.json()),
  })

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">

      {/* Hero Header */}
      <header className="bg-gradient-to-br from-slate-900 via-blue-950 to-slate-800 text-white">
        <div className="max-w-6xl mx-auto px-6 py-8">

          {/* Topo: logo FB + logo cliente */}
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-lg bg-blue-500 flex items-center justify-center font-bold text-white text-xs tracking-wide select-none">
                FB
              </div>
              <span className="text-xs font-medium text-slate-400 tracking-widest uppercase">
                FBTax Cloud
              </span>
            </div>

            {/* Logo cliente — apresentação */}
            <div className="flex items-center gap-2 bg-white/10 rounded-xl px-4 py-2 backdrop-blur-sm">
              <span className="text-xs text-slate-400">Apresentação para</span>
              <img src="/JC.png" alt="JC" className="h-8 w-auto rounded" />
            </div>
          </div>

          {/* Título */}
          <h1 className="text-3xl sm:text-4xl font-bold leading-tight mb-3 max-w-2xl">
            Soluções <span className="text-blue-400">inteligentes</span>{' '}
            para sua empresa
          </h1>
          <p className="text-slate-300 text-sm max-w-2xl leading-relaxed">
            Plataforma integrada com ferramentas especializadas em apuração tributária,
            simulação de cenários fiscais, gestão de RCAs e gestão estratégica de WMS.
          </p>
        </div>
      </header>

      {/* Main */}
      <main className="max-w-6xl mx-auto w-full px-6 py-6 flex-1">
        <div className="mb-5">
          <h2 className="text-lg font-bold text-slate-800">Nossas Soluções</h2>
          <p className="text-slate-400 mt-0.5 text-xs">
            Clique em uma solução para acessá-la diretamente.
          </p>
        </div>

        {isError && (
          <div className="flex items-center gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 mb-6">
            <AlertCircle className="w-4 h-4 flex-shrink-0" />
            Não foi possível carregar os produtos. Tente novamente em instantes.
          </div>
        )}

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {isPending
            ? Array.from({ length: 4 }).map((_, i) => <ProductCardSkeleton key={i} />)
            : data?.map((product, index) => (
                <ProductCard key={product.id} {...product} colorIndex={index} />
              ))
          }
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t bg-white">
        <div className="max-w-6xl mx-auto px-6 py-4 flex flex-col sm:flex-row items-center justify-between gap-1 text-xs text-slate-400">
          <span>© {new Date().getFullYear()} FBTax Cloud — Todos os direitos reservados</span>
          <span>Soluções inteligentes para sua empresa</span>
        </div>
      </footer>

    </div>
  )
}
