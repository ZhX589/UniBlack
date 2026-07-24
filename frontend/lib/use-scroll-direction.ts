'use client'

import { useEffect, useRef, useState } from 'react'

/**
 * Returns 'up' | 'down' | 'top' based on scroll direction.
 * 'top' when within 60px of page top, 'up' when scrolling upward, 'down' when scrolling down.
 */
export function useScrollDirection() {
  const [dir, setDir] = useState<'top' | 'up' | 'down'>('top')
  const lastY = useRef(0)

  useEffect(() => {
    let ticking = false

    function update() {
      const y = window.scrollY
      if (y <= 60) {
        setDir('top')
      } else if (y < lastY.current) {
        setDir('up')
      } else if (y > lastY.current + 4) {
        setDir('down')
      }
      lastY.current = y
      ticking = false
    }

    function onScroll() {
      if (!ticking) {
        requestAnimationFrame(update)
        ticking = true
      }
    }

    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  return dir
}
