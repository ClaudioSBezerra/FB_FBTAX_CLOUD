import { useEffect } from 'react'

const SEGMENTS = 18
const EASE_RING = 0.22
const EASE_SNAKE = 0.28

const RING_IDLE = {
  size: 28,
  border: 'rgba(45, 212, 191, 0.85)',
  shadow: '0 0 0 2px rgba(13,148,136,0.15), 0 0 30px rgba(20,184,166,0.25)',
}
const RING_HOVER = {
  size: 44,
  border: 'rgba(94, 234, 212, 1)',
  shadow: '0 0 0 3px rgba(94,234,212,0.15), 0 0 60px rgba(20,184,166,0.35)',
}

/** Elementos que fazem o anel crescer ao passar o mouse. */
const HOVER_SELECTOR =
  'a, button, [role="button"], input, select, textarea, .hover-lift, .hover-float, .glow-border, .button-hover, .accent-card-interactive'

const segmentSize = (i: number) => Math.max(4, 12 - i * 0.5)
const segmentOpacity = (i: number) => Math.max(0.15, 0.9 - i * 0.04)

/**
 * Cursor com rastro em teal, portado do design system glass-green-effect.
 *
 * Só entra em cena no tema escuro, em ponteiro fino (mouse) e viewport md+.
 * Observa a classe do <html> em vez de usar useTheme porque LightThemeScope
 * manipula a classe direto no DOM — só o observer enxerga as duas fontes.
 */
export function GlowCursor() {
  useEffect(() => {
    const finePointer = window.matchMedia('(pointer: fine) and (min-width: 768px)')
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)')
    const root = document.documentElement

    // Recursos vivos enquanto o cursor está ativo; nulos quando desativado.
    let teardown: (() => void) | null = null

    const activate = () => {
      if (teardown) return

      let targetX = window.innerWidth / 2
      let targetY = window.innerHeight / 2
      let ringX = targetX
      let ringY = targetY
      let sizeBoost = 0
      let rafId = 0

      const make = (css: string) => {
        const el = document.createElement('div')
        el.style.cssText = `position:fixed;top:0;left:0;pointer-events:none;border-radius:9999px;will-change:transform;${css}`
        document.body.appendChild(el)
        return el
      }

      const ring = make(
        `width:${RING_IDLE.size}px;height:${RING_IDLE.size}px;z-index:60;` +
          `border:1.5px solid ${RING_IDLE.border};box-shadow:${RING_IDLE.shadow};` +
          'backdrop-filter:blur(1px);opacity:.9;' +
          'transition:width .2s,height .2s,border-color .2s,box-shadow .2s,opacity .2s;',
      )
      const dot = make(
        'width:6px;height:6px;z-index:60;background:rgba(94,234,212,.95);' +
          'box-shadow:0 0 10px rgba(94,234,212,.6);opacity:.9;transition:opacity .2s;',
      )

      const segs: HTMLDivElement[] = []
      const segX = new Array<number>(SEGMENTS).fill(targetX)
      const segY = new Array<number>(SEGMENTS).fill(targetY)
      for (let i = 0; i < SEGMENTS; i++) {
        const s = segmentSize(i)
        segs.push(
          make(
            `width:${s}px;height:${s}px;z-index:50;opacity:${segmentOpacity(i)};` +
              `background:rgba(94,234,212,${Math.min(0.95, 0.65 + 0.02 * (SEGMENTS - i))});` +
              'box-shadow:0 0 0 2px rgba(13,148,136,.08),0 0 18px rgba(20,184,166,.25),' +
              'inset 0 0 6px rgba(255,255,255,.06);transition:width .2s,height .2s,opacity .2s;',
          ),
        )
      }

      const place = (el: HTMLElement, x: number, y: number, scale = 1) => {
        el.style.transform = `translate3d(${x}px,${y}px,0) translate(-50%,-50%) scale(${scale})`
      }

      let pressed = false

      const animate = () => {
        ringX += (targetX - ringX) * EASE_RING
        ringY += (targetY - ringY) * EASE_RING
        place(ring, ringX, ringY, pressed ? 0.88 : 1)
        place(dot, targetX, targetY)

        segX[0] += (targetX - segX[0]) * EASE_SNAKE
        segY[0] += (targetY - segY[0]) * EASE_SNAKE
        for (let i = 1; i < SEGMENTS; i++) {
          segX[i] += (segX[i - 1] - segX[i]) * EASE_SNAKE
          segY[i] += (segY[i - 1] - segY[i]) * EASE_SNAKE
        }
        for (let i = 0; i < SEGMENTS; i++) place(segs[i], segX[i], segY[i])

        rafId = requestAnimationFrame(animate)
      }

      const onMove = (e: MouseEvent) => {
        targetX = e.clientX
        targetY = e.clientY
      }

      const applyBoost = () => {
        for (let i = 0; i < SEGMENTS; i++) {
          const s = segmentSize(i) + sizeBoost
          segs[i].style.width = `${s}px`
          segs[i].style.height = `${s}px`
        }
      }

      const setRing = (state: typeof RING_IDLE, boost: number) => {
        ring.style.width = `${state.size}px`
        ring.style.height = `${state.size}px`
        ring.style.borderColor = state.border
        ring.style.boxShadow = state.shadow
        sizeBoost = boost
        applyBoost()
      }

      // Delegação em vez de um listener por elemento: o conteúdo é React e muda
      // a cada render, então uma lista capturada na montagem ficaria obsoleta.
      const onOver = (e: MouseEvent) => {
        if ((e.target as Element)?.closest?.(HOVER_SELECTOR)) setRing(RING_HOVER, 2)
      }
      const onOut = (e: MouseEvent) => {
        if ((e.target as Element)?.closest?.(HOVER_SELECTOR)) setRing(RING_IDLE, 0)
      }

      const setVisible = (visible: boolean) => {
        ring.style.opacity = visible ? '.9' : '0'
        dot.style.opacity = visible ? '.9' : '0'
        for (let i = 0; i < SEGMENTS; i++) {
          segs[i].style.opacity = visible ? String(segmentOpacity(i)) : '0'
        }
      }

      const onDown = () => { pressed = true }
      const onUp = () => { pressed = false }
      const onEnter = () => setVisible(true)
      const onLeave = () => setVisible(false)

      // Pausa fora da aba — rAF em aba oculta é desperdício de bateria
      const onVisibility = () => {
        cancelAnimationFrame(rafId)
        if (!document.hidden) rafId = requestAnimationFrame(animate)
      }

      document.body.classList.add('glow-cursor-active')
      window.addEventListener('mousemove', onMove, { passive: true })
      window.addEventListener('mouseover', onOver, { passive: true })
      window.addEventListener('mouseout', onOut, { passive: true })
      window.addEventListener('mousedown', onDown, { passive: true })
      window.addEventListener('mouseup', onUp, { passive: true })
      document.addEventListener('mouseenter', onEnter)
      document.addEventListener('mouseleave', onLeave)
      document.addEventListener('visibilitychange', onVisibility)
      rafId = requestAnimationFrame(animate)

      teardown = () => {
        cancelAnimationFrame(rafId)
        window.removeEventListener('mousemove', onMove)
        window.removeEventListener('mouseover', onOver)
        window.removeEventListener('mouseout', onOut)
        window.removeEventListener('mousedown', onDown)
        window.removeEventListener('mouseup', onUp)
        document.removeEventListener('mouseenter', onEnter)
        document.removeEventListener('mouseleave', onLeave)
        document.removeEventListener('visibilitychange', onVisibility)
        document.body.classList.remove('glow-cursor-active')
        ring.remove()
        dot.remove()
        for (const s of segs) s.remove()
      }
    }

    const deactivate = () => {
      teardown?.()
      teardown = null
    }

    const sync = () => {
      const wanted =
        root.classList.contains('dark') && finePointer.matches && !reducedMotion.matches
      if (wanted) activate()
      else deactivate()
    }

    sync()
    const observer = new MutationObserver(sync)
    observer.observe(root, { attributes: true, attributeFilter: ['class'] })
    finePointer.addEventListener('change', sync)
    reducedMotion.addEventListener('change', sync)

    return () => {
      observer.disconnect()
      finePointer.removeEventListener('change', sync)
      reducedMotion.removeEventListener('change', sync)
      deactivate()
    }
  }, [])

  return null
}
