import type { CSSProperties } from 'react'
import type { AccentColor } from '../types'

/**
 * One accent definition. `rgb` is the "r g b" triple injected into
 * `--accent-rgb`; every tint in index.css derives from it, so a single
 * `data-accent` switch repaints the whole UI.
 */
export interface AccentDefinition {
  name: AccentColor
  label: string
  hex: string
  hover: string
  rgb: string
}

/** Accent applied when no project is selected: the Equativ brand orange. */
export const BRAND_ACCENT = 'equativ'

/** Accent used when a project carries no color of its own. */
export const DEFAULT_PROJECT_ACCENT: AccentColor = 'indigo'

export const ACCENT_COLORS: AccentDefinition[] = [
  { name: 'indigo', label: 'Indigo', hex: '#6366f1', hover: '#4f46e5', rgb: '99 102 241' },
  { name: 'violet', label: 'Violet', hex: '#8b5cf6', hover: '#7c3aed', rgb: '139 92 246' },
  { name: 'emerald', label: 'Émeraude', hex: '#10b981', hover: '#059669', rgb: '16 185 129' },
  { name: 'amber', label: 'Ambre', hex: '#f59e0b', hover: '#d97706', rgb: '245 158 11' },
  { name: 'rose', label: 'Rose', hex: '#f43f5e', hover: '#e11d48', rgb: '244 63 94' },
  { name: 'cyan', label: 'Cyan', hex: '#06b6d4', hover: '#0891b2', rgb: '6 182 212' },
  { name: 'blue', label: 'Bleu', hex: '#3b82f6', hover: '#2563eb', rgb: '59 130 246' },
  { name: 'orange', label: 'Orange', hex: '#f97316', hover: '#ea580c', rgb: '249 115 22' },
  { name: 'neon-cyan', label: '⚡ Cyber Cyan', hex: '#00f0ff', hover: '#00c8d6', rgb: '0 240 255' },
  { name: 'neon-purple', label: '🔮 Synthwave', hex: '#d946ef', hover: '#c026d3', rgb: '217 70 239' },
  { name: 'neon-green', label: '🟢 Matrix Green', hex: '#10f070', hover: '#0bc95c', rgb: '16 240 112' },
  { name: 'neon-amber', label: '✨ Laser Gold', hex: '#ffd000', hover: '#e6b800', rgb: '255 208 0' },
]

const BY_NAME: Record<string, AccentDefinition> = Object.fromEntries(
  ACCENT_COLORS.map(c => [c.name, c])
)

/**
 * Tailwind family names stored by older projects (and by the seeded demo
 * projects) that the palette does not carry under that exact name.
 */
const ALIASES: Record<string, AccentColor> = {
  purple: 'violet',
  fuchsia: 'neon-purple',
  green: 'emerald',
  teal: 'cyan',
  sky: 'blue',
  red: 'rose',
  pink: 'rose',
  yellow: 'amber',
}

/** True when `value` is one of the accents index.css knows how to paint. */
export function isAccentColor(value?: string | null): value is AccentColor {
  return !!value && value in BY_NAME
}

/** The palette name for a stored project color, or null when there is none. */
export function normalizeAccentColor(color?: string | null): AccentColor | null {
  if (!color) return null
  if (isAccentColor(color)) return color
  return ALIASES[color] ?? DEFAULT_PROJECT_ACCENT
}

/**
 * The `data-accent` value to put on <html>/<body> for a given project color.
 * With no project color at all (the "all projects" view), the Equativ brand
 * accent defined on :root is kept.
 */
export function resolveAccentAttribute(color?: string | null): string {
  return normalizeAccentColor(color) ?? BRAND_ACCENT
}

/** The definition for a project color, or the default project accent. */
export function accentDefinition(color?: string | null): AccentDefinition {
  return BY_NAME[normalizeAccentColor(color) ?? DEFAULT_PROJECT_ACCENT]
}

/**
 * Inline style for a project badge (icon chip). Inline styles are used instead
 * of `bg-${color}-500/20` classes because Tailwind cannot generate classes it
 * never sees in the source, and because index.css remaps the raw Tailwind
 * families onto the Equativ palette with `!important`.
 */
export function accentBadgeStyle(color?: string | null): CSSProperties {
  const def = accentDefinition(color)
  return {
    color: def.hex,
    backgroundColor: `rgb(${def.rgb} / 0.2)`,
    borderColor: `rgb(${def.rgb} / 0.3)`,
  }
}

/** Inline style for text painted with a project color. */
export function accentTextStyle(color?: string | null): CSSProperties {
  return { color: accentDefinition(color).hex }
}
