import { useEffect, useState } from 'react'
import { useTheme } from 'next-themes'
import { Moon, Sun } from 'lucide-react'

type ThemeToggleProps = { className?: string }

/**
 * Alterna entre tema claro e escuro. Renderiza um placeholder do mesmo tamanho
 * até montar — antes disso `resolvedTheme` é indefinido e o ícone piscaria.
 */
export function ThemeToggle({ className = '' }: ThemeToggleProps) {
  const [mounted, setMounted] = useState(false)
  const { resolvedTheme, setTheme } = useTheme()

  useEffect(() => setMounted(true), [])

  const base = `inline-flex items-center justify-center w-8 h-8 rounded-lg border transition-colors ${className}`

  if (!mounted) {
    return <span className={`${base} border-transparent`} aria-hidden="true" />
  }

  const isDark = resolvedTheme === 'dark'

  return (
    <button
      type="button"
      onClick={() => setTheme(isDark ? 'light' : 'dark')}
      className={`${base} border-border text-muted-foreground hover:text-foreground hover:border-foreground/40`}
      aria-label={isDark ? 'Mudar para tema claro' : 'Mudar para tema escuro'}
      title={isDark ? 'Tema claro' : 'Tema escuro'}
    >
      {isDark ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
    </button>
  )
}
