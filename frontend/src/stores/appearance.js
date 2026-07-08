import { writable, derived } from 'svelte/store'
import { DEFAULT_ACCENT, accentMeta } from '../lib/accents.js'

/**
 * Appearance store. App.svelte sets `accent` from config; components derive
 * the mascot emoji from it so the whole UI follows the chosen accent.
 */
export const accent = writable(DEFAULT_ACCENT)

export const mascot = derived(accent, (a) => accentMeta(a).emoji)
