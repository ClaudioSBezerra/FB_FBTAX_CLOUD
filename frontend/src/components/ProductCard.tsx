import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export type Product = {
  id: string
  name: string
  description: string
  icon_url: string
  destination_url: string
  contracted: boolean
}

type ProductCardProps = Product

export function ProductCard({ name, description, icon_url }: ProductCardProps) {
  return (
    <Card className="flex flex-col h-full hover:shadow-md transition-shadow">
      <CardHeader className="flex flex-row items-center gap-3 pb-2">
        <div className="w-12 h-12 rounded-full bg-blue-50 flex items-center justify-center flex-shrink-0">
          {icon_url ? (
            <img src={icon_url} alt={name} className="w-8 h-8 object-contain" />
          ) : (
            <span className="text-2xl" aria-hidden="true">📦</span>
          )}
        </div>
        <CardTitle className="text-base leading-tight">{name}</CardTitle>
      </CardHeader>
      <CardContent className="pt-0">
        <p className="text-sm text-muted-foreground">{description}</p>
      </CardContent>
    </Card>
  )
}
