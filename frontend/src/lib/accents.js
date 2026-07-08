/**
 * Accent registry: each accent is a small "character" — a brand colour set
 * (defined in style.css under [data-accent=...]) plus the mascot emoji used
 * in the titlebar, work chip, session counters and Focus button.
 */
export const ACCENTS = {
  duck: { label: 'Duck', emoji: '🦆' },
  tomato: { label: 'Tomato', emoji: '🍅' },
  orange: { label: 'Orange', emoji: '🍊' }
}

export const DEFAULT_ACCENT = 'duck'

export function accentMeta(key) {
  return ACCENTS[key] || ACCENTS[DEFAULT_ACCENT]
}
