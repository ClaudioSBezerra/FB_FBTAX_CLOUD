import { useQuery } from '@tanstack/react-query'
import { ProductCard, type Product } from '@/components/ProductCard'
import { Skeleton } from '@/components/ui/skeleton'
import { AlertCircle } from 'lucide-react'

function ProductCardSkeleton() {
  return (
    <div className="rounded-xl border-2 border-slate-100 bg-white p-5 flex flex-col gap-4">
      <Skeleton className="w-12 h-12 rounded-2xl" />
      <Skeleton className="h-4 w-36" />
      <Skeleton className="h-3 w-full" />
      <Skeleton className="h-3 w-3/4" />
      <Skeleton className="h-3 w-16 mt-2" />
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
        <div className="max-w-6xl mx-auto px-6 py-14">
          {/* Logo mark */}
          <div className="flex items-center gap-3 mb-8">
            <div className="w-9 h-9 rounded-xl bg-blue-500 flex items-center justify-center font-bold text-white text-sm tracking-wide select-none">
              FB
            </div>
            <span className="text-sm font-medium text-slate-300 tracking-widest uppercase">
              FBTax Cloud
            </span>
          </div>

          <h1 className="text-4xl sm:text-5xl font-bold leading-tight mb-4 max-w-2xl">
            Soluções fiscais <span className="text-blue-400">inteligentes</span> para sua empresa
          </h1>
          <p className="text-slate-300 text-lg max-w-xl leading-relaxed">
            Plataforma integrada com ferramentas especializadas em apuração tributária,
            simulação de cenários fiscais e gestão de portfólio.
          </p>
        </div>
      </header>

      {/* Main */}
      <main className="max-w-6xl mx-auto w-full px-6 py-12 flex-1">
        <div className="mb-8">
          <h2 className="text-2xl font-bold text-slate-800">Nossas Soluções</h2>
          <p className="text-slate-500 mt-1 text-sm">
            Clique em uma solução para acessá-la diretamente.
          </p>
        </div>

        {isError && (
          <div className="flex items-center gap-3 rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700 mb-8">
            <AlertCircle className="w-5 h-5 flex-shrink-0" />
            Não foi possível carregar os produtos. Tente novamente em instantes.
          </div>
        )}

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
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
        <div className="max-w-6xl mx-auto px-6 py-5 flex flex-col sm:flex-row items-center justify-between gap-2 text-xs text-slate-400">
          <span>© {new Date().getFullYear()} FBTax Cloud — Todos os direitos reservados</span>
          <span>Soluções fiscais inteligentes</span>
        </div>
      </footer>

    </div>
  )
}
