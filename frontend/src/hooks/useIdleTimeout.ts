import { useEffect, useRef, useCallback } from 'react'

const IDLE_TIMEOUT = 15 * 60 * 1000 // 15 minutes

const ACTIVITY_EVENTS: (keyof WindowEventMap)[] = [
  'mousedown',
  'mousemove',
  'keydown',
  'scroll',
  'touchstart',
  'click',
]

export function useIdleTimeout(onTimeout: () => void, timeout = IDLE_TIMEOUT) {
  const timerRef = useRef<ReturnType<typeof setTimeout>>()

  const resetTimer = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current)
    timerRef.current = setTimeout(onTimeout, timeout)
  }, [onTimeout, timeout])

  useEffect(() => {
    resetTimer()

    for (const event of ACTIVITY_EVENTS) {
      window.addEventListener(event, resetTimer, { passive: true })
    }

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
      for (const event of ACTIVITY_EVENTS) {
        window.removeEventListener(event, resetTimer)
      }
    }
  }, [resetTimer])
}
