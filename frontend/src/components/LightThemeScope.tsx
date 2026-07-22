import { useEffect, useRef } from 'react'
import { useTheme } from 'next-themes'

/**
 * Mantém as telas envolvidas no tema claro, mesmo com o tema global em escuro.
 *
 * Existe porque o tema "glass green" foi aplicado só ao portal público e às telas
 * de login. O módulo financeiro e o painel FBTax ainda têm cores claras fixas no
 * markup (bg-gray-50, bg-white), que ficariam ilegíveis sob os tokens escuros.
 * Remover este wrapper é o passo final de quando essas telas forem retemadas.
 *
 * next-themes 0.4 ignora ThemeProvider aninhado (`useContext(L) ? children : ...`),
 * então `forcedTheme` por rota não é uma opção — daí a manipulação direta da classe.
 * O MutationObserver é necessário porque efeitos de filho rodam antes dos do pai:
 * sem ele, o provider reaplicaria `dark` logo após a montagem.
 */
export function LightThemeScope({ children }: { children: React.ReactNode }) {
  const { resolvedTheme } = useTheme()
  const themeRef = useRef(resolvedTheme)
  themeRef.current = resolvedTheme

  useEffect(() => {
    const el = document.documentElement

    const forceLight = () => {
      if (!el.classList.contains('dark')) return
      el.classList.remove('dark')
      el.classList.add('light')
      el.style.colorScheme = 'light'
    }

    forceLight()
    const observer = new MutationObserver(forceLight)
    observer.observe(el, { attributes: true, attributeFilter: ['class'] })

    return () => {
      observer.disconnect()
      // Devolve o controle ao next-themes com o tema que ele acredita estar ativo
      if (themeRef.current === 'dark') {
        el.classList.remove('light')
        el.classList.add('dark')
        el.style.colorScheme = 'dark'
      }
    }
  }, [])

  return <>{children}</>
}
