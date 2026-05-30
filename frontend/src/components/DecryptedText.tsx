import { useEffect, useState, useRef, useMemo, useCallback } from 'react'
import { motion } from 'motion/react'

interface DecryptedTextProps {
  text: string
  speed?: number
  maxIterations?: number
  sequential?: boolean
  revealDirection?: 'start' | 'end' | 'center'
  useOriginalCharsOnly?: boolean
  characters?: string
  className?: string
  encryptedClassName?: string
  parentClassName?: string
  animateOn?: 'view' | 'hover'
}

export default function DecryptedText({
  text,
  speed = 50,
  maxIterations = 10,
  sequential = false,
  revealDirection = 'start',
  useOriginalCharsOnly = false,
  characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz!@#$%^&*()_+',
  className = '',
  encryptedClassName = '',
  parentClassName = '',
  animateOn = 'view',
}: DecryptedTextProps) {
  const [displayText, setDisplayText] = useState(text)
  const [isAnimating, setIsAnimating] = useState(false)
  const [revealedIndices, setRevealedIndices] = useState<Set<number>>(new Set())
  const [hasAnimated, setHasAnimated] = useState(false)
  const [isDecrypted, setIsDecrypted] = useState(false)

  const containerRef = useRef<HTMLSpanElement>(null)

  const availableChars = useMemo(
    () =>
      useOriginalCharsOnly
        ? Array.from(new Set(text.split(''))).filter((c) => c !== ' ')
        : characters.split(''),
    [useOriginalCharsOnly, text, characters]
  )

  const shuffleText = useCallback(
    (orig: string, revealed: Set<number>) =>
      orig
        .split('')
        .map((char, i) => {
          if (char === ' ') return ' '
          if (revealed.has(i)) return orig[i]
          return availableChars[Math.floor(Math.random() * availableChars.length)]
        })
        .join(''),
    [availableChars]
  )

  const getNextIndex = useCallback(
    (revealed: Set<number>) => {
      const len = text.length
      if (revealDirection === 'start') return revealed.size
      if (revealDirection === 'end') return len - 1 - revealed.size
      const mid = Math.floor(len / 2)
      const off = Math.floor(revealed.size / 2)
      const idx = revealed.size % 2 === 0 ? mid + off : mid - off - 1
      if (idx >= 0 && idx < len && !revealed.has(idx)) return idx
      for (let i = 0; i < len; i++) if (!revealed.has(i)) return i
      return 0
    },
    [text.length, revealDirection]
  )

  const triggerDecrypt = useCallback(() => {
    setRevealedIndices(new Set())
    setIsDecrypted(false)
    setDisplayText(text)
    setIsAnimating(true)
  }, [text])

  useEffect(() => {
    if (!isAnimating) return
    let iteration = 0
    const interval = setInterval(() => {
      setRevealedIndices((prev) => {
        if (sequential) {
          if (prev.size < text.length) {
            const next = new Set(prev)
            next.add(getNextIndex(prev))
            setDisplayText(shuffleText(text, next))
            return next
          }
          clearInterval(interval)
          setIsAnimating(false)
          setIsDecrypted(true)
          return prev
        }
        setDisplayText(shuffleText(text, prev))
        iteration++
        if (iteration >= maxIterations) {
          clearInterval(interval)
          setIsAnimating(false)
          setDisplayText(text)
          setIsDecrypted(true)
        }
        return prev
      })
    }, speed)
    return () => clearInterval(interval)
  }, [isAnimating, text, speed, maxIterations, sequential, shuffleText, getNextIndex])

  useEffect(() => {
    if (animateOn !== 'view') return
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting && !hasAnimated) {
            triggerDecrypt()
            setHasAnimated(true)
          }
        })
      },
      { threshold: 0.1 }
    )
    const el = containerRef.current
    if (el) observer.observe(el)
    return () => {
      if (el) observer.unobserve(el)
    }
  }, [animateOn, hasAnimated, triggerDecrypt])

  const hoverProps =
    animateOn === 'hover'
      ? {
          onMouseEnter: () => {
            if (!isAnimating) triggerDecrypt()
          },
          onMouseLeave: () => {
            setIsAnimating(false)
            setRevealedIndices(new Set())
            setDisplayText(text)
            setIsDecrypted(true)
          },
        }
      : {}

  return (
    <motion.span
      ref={containerRef}
      className={`inline-block whitespace-pre-wrap ${parentClassName}`}
      {...hoverProps}
    >
      <span className="sr-only">{displayText}</span>
      <span aria-hidden="true">
        {displayText.split('').map((char, i) => {
          const revealed = revealedIndices.has(i) || (!isAnimating && isDecrypted)
          return (
            <span key={i} className={revealed ? className : encryptedClassName}>
              {char}
            </span>
          )
        })}
      </span>
    </motion.span>
  )
}
